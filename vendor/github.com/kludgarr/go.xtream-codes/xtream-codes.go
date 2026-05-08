// Package xtreamcodes provides a Golang interface to the Xtream-Codes IPTV Server API.
package xtreamcodes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var defaultUserAgent = "go.xtream-codes (Go-http-client/1.1)"

// HTTPError is returned by the Xtream API client when a request returns an
// HTTP status >= 400, or when a 2xx response body for a JSON-expected
// endpoint fails the non-JSON error heuristic (empty body, leading '<' for
// HTML, exact-match sentinel words). 3xx redirects are followed by the
// underlying http.Client and do not surface as HTTPError. Callers can
// type-assert to access StatusCode, ContentType, and Body for diagnostic
// inspection.
type HTTPError struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("xtreamcodes: http %d (%s)", e.StatusCode, e.ContentType)
}

// AuthError is returned by NewClient when authentication fails. Failure is
// detected via three signals: HTTP error status from the auth request,
// a 2xx response whose body fails the non-JSON heuristic, or a 2xx response
// whose parsed AuthenticationResponse lacks user_info (null envelope or
// missing field). StatusCode, ContentType, and Body capture the response
// for diagnosis.
type AuthError struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("xtreamcodes: authentication failed (status %d, content-type %q)", e.StatusCode, e.ContentType)
}

// XtreamClient is the client used to communicate with a Xtream-Codes server.
type XtreamClient struct {
	Username  string
	Password  string
	BaseURL   string
	UserAgent string

	ServerInfo ServerInfo
	UserInfo   UserInfo

	// Our HTTP client to communicate with Xtream
	HTTP    *http.Client
	Context context.Context

	// We store an internal map of Streams for use with GetStreamURL
	streams map[int]Stream
}

// NewClient returns an initialized XtreamClient with the given values.
func NewClient(username, password, baseURL string) (*XtreamClient, error) {

	_, parseURLErr := url.Parse(baseURL)
	if parseURLErr != nil {
		return nil, fmt.Errorf("error parsing url: %s", parseURLErr.Error())
	}

	client := &XtreamClient{
		Username:  username,
		Password:  password,
		BaseURL:   baseURL,
		UserAgent: defaultUserAgent,

		// 90s accommodates legitimate large responses (XMLTV can exceed a minute)
		// while still failing visibly on a hung upstream rather than blocking forever.
		HTTP:    &http.Client{Timeout: 90 * time.Second},
		Context: context.Background(),

		streams: make(map[int]Stream),
	}

	authData, authErr := client.sendRequest("", nil)
	if authErr != nil {
		// Translate HTTPError from sendRequest into an AuthError for auth-specific semantics.
		if httpErr, ok := authErr.(*HTTPError); ok {
			return nil, &AuthError{
				StatusCode:  httpErr.StatusCode,
				ContentType: httpErr.ContentType,
				Body:        httpErr.Body,
			}
		}
		return nil, fmt.Errorf("error sending authentication request: %s", authErr.Error())
	}

	a := &AuthenticationResponse{}

	if jsonErr := json.Unmarshal(authData, &a); jsonErr != nil {
		return nil, fmt.Errorf("error unmarshaling json: %s", jsonErr.Error())
	}

	// Shape detection: {"user_info": null, "error": ...} or body lacking user_info
	// both leave UserInfo zero-valued. Empty Username is the reliable signal.
	if a.UserInfo.Username == "" {
		return nil, &AuthError{
			StatusCode:  200,
			ContentType: "application/json",
			Body:        authData,
		}
	}

	client.ServerInfo = a.ServerInfo
	client.UserInfo = a.UserInfo

	return client, nil
}

// NewClientWithContext returns an initialized XtreamClient with the given values.
func NewClientWithContext(ctx context.Context, username, password, baseURL string) (*XtreamClient, error) {
	c, err := NewClient(username, password, baseURL)
	if err != nil {
		return nil, err
	}
	c.Context = ctx

	return c, nil
}

// NewClientWithUserAgent returns an initialized XtreamClient with the given values.
func NewClientWithUserAgent(ctx context.Context, username, password, baseURL, userAgent string) (*XtreamClient, error) {
	c, err := NewClient(username, password, baseURL)
	if err != nil {
		return nil, err
	}
	c.UserAgent = userAgent
	c.Context = ctx

	return c, nil
}

// GetStreamURL will return a stream URL string for the given streamID and wantedFormat.
func (c *XtreamClient) GetStreamURL(streamID int, wantedFormat string) (string, error) {

	// For Live Streams the main format is
	// http(s)://domain:port/live/username/password/streamID.ext ( In allowed_output_formats element you have the available ext )
	// For VOD Streams the format is:
	// http(s)://domain:port/movie/username/password/streamID.ext ( In target_container element you have the available ext )
	// For Series Streams the format is
	// http(s)://domain:port/series/username/password/streamID.ext ( In target_container element you have the available ext )

	validFormat := false

	for _, allowedFormat := range c.UserInfo.AllowedOutputFormats {
		if wantedFormat == allowedFormat {
			validFormat = true
		}
	}

	if !validFormat {
		return "", fmt.Errorf("%s is not an allowed output format", wantedFormat)
	}

	if _, ok := c.streams[streamID]; !ok {
		return "", fmt.Errorf("%d is not a valid stream id", streamID)
	}

	stream := c.streams[streamID]

	return fmt.Sprintf("%s/%s/%s/%s/%d.%s", c.BaseURL, stream.Type, c.Username, c.Password, stream.ID.Int64(), wantedFormat), nil
}

// GetLiveCategories will return a slice of categories for live streams.
func (c *XtreamClient) GetLiveCategories() ([]Category, error) {
	return c.GetCategories("live")
}

// GetVideoOnDemandCategories will return a slice of categories for VOD streams.
func (c *XtreamClient) GetVideoOnDemandCategories() ([]Category, error) {
	return c.GetCategories("vod")
}

// GetSeriesCategories will return a slice of categories for series streams.
func (c *XtreamClient) GetSeriesCategories() ([]Category, error) {
	return c.GetCategories("series")
}

// GetCategories is a helper function used by GetLiveCategories, GetVideoOnDemandCategories and
// GetSeriesCategories to reduce duplicate code.
func (c *XtreamClient) GetCategories(catType string) ([]Category, error) {
	catData, catErr := c.sendRequest(fmt.Sprintf("get_%s_categories", catType), nil)
	if catErr != nil {
		return nil, catErr
	}

	cats := make([]Category, 0)

	if err := json.Unmarshal(catData, &cats); err != nil {
		return nil, err
	}

	for idx := range cats {
		cats[idx].Type = catType
	}

	return cats, nil
}

// GetLiveStreams will return a slice of live streams.
// You can also optionally provide a categoryID to limit the output to members of that category.
func (c *XtreamClient) GetLiveStreams(categoryID string) ([]Stream, error) {
	return c.GetStreams("live", categoryID)
}

// GetVideoOnDemandStreams will return a slice of VOD streams.
// You can also optionally provide a categoryID to limit the output to members of that category.
func (c *XtreamClient) GetVideoOnDemandStreams(categoryID string) ([]Stream, error) {
	return c.GetStreams("vod", categoryID)
}

// GetStreams is a helper function used by GetLiveStreams and GetVideoOnDemandStreams
// to reduce duplicate code.
func (c *XtreamClient) GetStreams(streamAction, categoryID string) ([]Stream, error) {
	var params url.Values
	if categoryID != "" {
		params = url.Values{}
		params.Add("category_id", categoryID)
	}

	// For whatever reason, unlike live and vod, series streams action doesn't have "_streams".
	if streamAction != "series" {
		streamAction = fmt.Sprintf("%s_streams", streamAction)
	}

	streamData, streamErr := c.sendRequest(fmt.Sprintf("get_%s", streamAction), params)
	if streamErr != nil {
		return nil, streamErr
	}

	streams := make([]Stream, 0)

	if jsonErr := json.Unmarshal(streamData, &streams); jsonErr != nil {
		return nil, jsonErr
	}

	for _, stream := range streams {
		c.streams[stream.ID.Int()] = stream
	}

	return streams, nil
}

// GetSeries will return a slice of all available Series.
// You can also optionally provide a categoryID to limit the output to members of that category.
func (c *XtreamClient) GetSeries(categoryID string) ([]SeriesInfo, error) {
	var params url.Values
	if categoryID != "" {
		params = url.Values{}
		params.Add("category_id", categoryID)
	}

	seriesData, seriesErr := c.sendRequest("get_series", params)
	if seriesErr != nil {
		return nil, seriesErr
	}

	seriesInfos := make([]SeriesInfo, 0)

	if jsonErr := json.Unmarshal(seriesData, &seriesInfos); jsonErr != nil {
		return nil, jsonErr
	}

	return seriesInfos, nil
}

// GetSeriesInfo will return a series info for the given seriesID.
func (c *XtreamClient) GetSeriesInfo(seriesID string) (*Series, error) {
	if seriesID == "" {
		return nil, fmt.Errorf("series ID can not be empty")
	}

	seriesData, seriesErr := c.sendRequest("get_series_info", url.Values{"series_id": []string{seriesID}})
	if seriesErr != nil {
		return nil, seriesErr
	}

	seriesInfo := &Series{}
	if err := json.Unmarshal(seriesData, &seriesInfo); err != nil {
		return nil, err
	}

	return seriesInfo, nil
}

// GetVideoOnDemandInfo will return VOD info for the given vodID.
func (c *XtreamClient) GetVideoOnDemandInfo(vodID string) (*VideoOnDemandInfo, error) {
	if vodID == "" {
		return nil, fmt.Errorf("vod ID can not be empty")
	}

	vodData, vodErr := c.sendRequest("get_vod_info", url.Values{"vod_id": []string{vodID}})
	if vodErr != nil {
		return nil, vodErr
	}

	vodInfo := &VideoOnDemandInfo{}
	if err := json.Unmarshal(vodData, &vodInfo); err != nil {
		return nil, err
	}

	return vodInfo, nil
}

// GetShortEPG returns a short version of the EPG for the given streamID. If no limit is provided, the next 4 items in the EPG will be returned.
func (c *XtreamClient) GetShortEPG(streamID string, limit int) ([]EPGInfo, error) {
	return c.getEPG("get_short_epg", streamID, limit)
}

// GetEPG returns the full EPG for the given streamID.
func (c *XtreamClient) GetEPG(streamID string) ([]EPGInfo, error) {
	return c.getEPG("get_simple_data_table", streamID, 0)
}

// GetXMLTV will return a slice of bytes for the XMLTV EPG file available from the provider.
func (c *XtreamClient) GetXMLTV() ([]byte, error) {
	xmlTVData, xmlTVErr := c.sendRequest("xmltv.php", nil)
	if xmlTVErr != nil {
		return nil, xmlTVErr
	}

	return xmlTVData, xmlTVErr
}

func (c *XtreamClient) getEPG(action, streamID string, limit int) ([]EPGInfo, error) {
	if streamID == "" {
		return nil, fmt.Errorf("stream ID can not be empty")
	}

	params := url.Values{"stream_id": []string{streamID}}
	if limit > 0 {
		params.Add("limit", strconv.Itoa(limit))
	}

	epgData, epgErr := c.sendRequest(action, params)
	if epgErr != nil {
		return nil, epgErr
	}

	epgContainer := &epgContainer{}
	if err := json.Unmarshal(epgData, &epgContainer); err != nil {
		return nil, err
	}

	return epgContainer.EPGListings, nil
}

func (c *XtreamClient) sendRequest(action string, parameters url.Values) ([]byte, error) {
	// XMLTV is exposed at /xmltv.php directly — the file path encodes the
	// endpoint, so no &action query param is appended (and no JSON-shape
	// expectation applies on the response — see the heuristic guard below).
	xmltvEndpoint := action == "xmltv.php"
	file := "player_api.php"
	if xmltvEndpoint {
		file = action
	}
	// Username/password are interpolated unescaped: empirically, Xtream
	// credentials are uniformly [a-zA-Z0-9]{10} across providers and the
	// upstream/Dispatcharr clients use the same raw concatenation. Revisit
	// if a provider ever issues credentials with reserved URL characters.
	requestURL := fmt.Sprintf("%s/%s?username=%s&password=%s", c.BaseURL, file, c.Username, c.Password)
	if action != "" && !xmltvEndpoint {
		requestURL = fmt.Sprintf("%s&action=%s", requestURL, action)
	}
	if parameters != nil {
		requestURL = fmt.Sprintf("%s&%s", requestURL, parameters.Encode())
	}

	request, httpErr := http.NewRequestWithContext(c.Context, "GET", requestURL, nil)
	if httpErr != nil {
		return nil, httpErr
	}

	request.Header.Set("User-Agent", c.UserAgent)

	response, httpErr := c.HTTP.Do(request)
	if httpErr != nil {
		return nil, fmt.Errorf("cannot reach server: %w", httpErr)
	}
	defer response.Body.Close()

	body, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return nil, fmt.Errorf("cannot read response: %w", readErr)
	}

	contentType := response.Header.Get("Content-Type")

	if response.StatusCode >= 400 {
		return nil, &HTTPError{
			StatusCode:  response.StatusCode,
			ContentType: contentType,
			Body:        body,
		}
	}

	// Heuristic: 2xx responses that clearly aren't JSON, applied only to
	// JSON-expected endpoints. Skipped for /xmltv.php since XMLTV payloads
	// legitimately start with '<'. The sentinel list is exact-match and is
	// sourced secondhand from Dispatcharr's Python XC client; tighten when
	// we have concrete examples of the failure modes in our sample data.
	if !xmltvEndpoint && isNonJSONErrorBody(body) {
		return nil, &HTTPError{
			StatusCode:  response.StatusCode,
			ContentType: contentType,
			Body:        body,
		}
	}

	return body, nil
}

// isNonJSONErrorBody flags 2xx responses that clearly aren't JSON: empty
// body, leading '<' (HTML/XML error page), or an exact-match against a
// short list of plain-text sentinels. The sentinel match is bytes.Equal
// (whole body, case-folded) — not substring-match — to keep false-positive
// risk low against legitimate JSON payloads that contain these words.
func isNonJSONErrorBody(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return true
	}
	if trimmed[0] == '<' {
		return true
	}
	lower := bytes.ToLower(trimmed)
	for _, sentinel := range [][]byte{
		[]byte("blocked"),
		[]byte("forbidden"),
		[]byte("access denied"),
		[]byte("unauthorized"),
	} {
		if bytes.Equal(lower, sentinel) {
			return true
		}
	}
	return false
}
