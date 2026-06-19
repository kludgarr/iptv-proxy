package xtreamcodes

import "encoding/json"

// TODO: Add more flex types on IDs if needed
// for future potential provider issues.

// ServerInfo describes the state of the Xtream-Codes server.
type ServerInfo struct {
	HTTPSPort    FlexInt            `json:"https_port"`
	IE           ConvertibleBoolean `json:"ie,omitzero"`
	IEAuth       ConvertibleBoolean `json:"ie_auth,omitzero"`
	Port         FlexInt            `json:"port"`
	Process      bool               `json:"process"`
	RTMPPort     FlexInt            `json:"rtmp_port"`
	Protocol     string             `json:"server_protocol"`
	TimeNow      string             `json:"time_now"`
	TimestampNow Timestamp          `json:"timestamp_now,string"`
	Timezone     string             `json:"timezone"`
	URL          string             `json:"url"`
	Version      string             `json:"version,omitempty"`
	Revision     string             `json:"revision,omitempty"`
	XUI          bool               `json:"xui,omitempty"`
}

// UserInfo is the current state of the user as it relates to the Xtream-Codes server.
type UserInfo struct {
	ActiveConnections    FlexInt            `json:"active_cons"`
	AllowedOutputFormats []string           `json:"allowed_output_formats"`
	Auth                 ConvertibleBoolean `json:"auth"`
	CreatedAt            Timestamp          `json:"created_at"`
	ExpDate              *Timestamp         `json:"exp_date"`
	IsTrial              ConvertibleBoolean `json:"is_trial,string"`
	MaxConnections       FlexInt            `json:"max_connections"`
	Message              string             `json:"message"`
	Password             string             `json:"password"`
	PlaylistName         string             `json:"playlist_name,omitempty"`
	Status               string             `json:"status"`
	Username             string             `json:"username"`
}

// AuthenticationResponse is a container for what the server returns after the initial authentication.
type AuthenticationResponse struct {
	ServerInfo ServerInfo `json:"server_info"`
	UserInfo   UserInfo   `json:"user_info"`
}

// Category describes a grouping of Stream.
type Category struct {
	ID     FlexInt `json:"category_id"`
	Name   string  `json:"category_name"`
	Parent FlexInt `json:"parent_id"`

	// Set by us, not Xtream.
	Type string `json:"-"`
}

// Stream is a streamble video source.
type Stream struct {
	Added              *Timestamp         `json:"added"`
	CategoryID         FlexInt            `json:"category_id"`
	CategoryIDs        []FlexInt          `json:"category_ids,omitempty"`
	CategoryName       string             `json:"category_name"`
	ContainerExtension string             `json:"container_extension"`
	CustomSid          string             `json:"custom_sid"`
	DirectSource       string             `json:"direct_source,omitempty"`
	EPGChannelID       string             `json:"epg_channel_id"`
	Icon               string             `json:"stream_icon"`
	ID                 FlexInt            `json:"stream_id"`
	ImdbID             string             `json:"imdb,omitempty"`
	IsAdult            ConvertibleBoolean `json:"is_adult"`
	Live               ConvertibleBoolean `json:"live"`
	Name               string             `json:"name"`
	Number             FlexInt            `json:"num"`
	Rating             FlexFloat          `json:"rating"`
	Rating5based       FlexFloat          `json:"rating_5based"`
	SeriesNo           *FlexInt           `json:"series_no,omitempty"`
	TmdbID             FlexInt            `json:"tmdb,omitempty"`
	Trailer            string             `json:"trailer,omitempty"`
	TvdbID             FlexInt            `json:"tvdb,omitzero"`
	TVArchive          ConvertibleBoolean `json:"tv_archive"`
	TVArchiveDuration  *FlexInt           `json:"tv_archive_duration"`
	Type               string             `json:"stream_type"`
	TypeName           string             `json:"type_name,omitempty"`
}

// streamAlias prevents UnmarshalJSON recursion.
type streamAlias Stream

// UnmarshalJSON accepts tmdb_id/imdb_id/tvdb_id as input aliases for the
// canonical short-form tmdb/imdb/tvdb keys — ProviderC (Dispatcharr) emits
// the long forms on stream records while A and B emit the short forms.
func (s *Stream) UnmarshalJSON(b []byte) error {
	type wire struct {
		*streamAlias
		TmdbIDAlt FlexInt `json:"tmdb_id"`
		ImdbIDAlt string  `json:"imdb_id"`
		TvdbIDAlt FlexInt `json:"tvdb_id"`
	}
	w := wire{streamAlias: (*streamAlias)(s)}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	if s.TmdbID.Int64() == 0 {
		s.TmdbID = w.TmdbIDAlt
	}
	if s.ImdbID == "" {
		s.ImdbID = w.ImdbIDAlt
	}
	if s.TvdbID.Int64() == 0 {
		s.TvdbID = w.TvdbIDAlt
	}
	return nil
}

// SeriesInfo contains information about a TV series.
type SeriesInfo struct {
	Added          *Timestamp       `json:"added,omitempty"`
	BackdropPath   *JSONStringSlice `json:"backdrop_path,omitempty"`
	Cast           string           `json:"cast"`
	CategoryID     *FlexInt         `json:"category_id"`
	CategoryIDs    []FlexInt        `json:"category_ids,omitempty"`
	Cover          string           `json:"cover"`
	Director       string           `json:"director"`
	EpisodeRunTime FlexInt          `json:"episode_run_time"`
	Genre          string           `json:"genre"`
	ImdbID         string           `json:"imdb,omitempty"`
	LastModified   *Timestamp       `json:"last_modified,omitempty"`
	Name           string           `json:"name"`
	Num            FlexInt          `json:"num"`
	Plot           string           `json:"plot"`
	Rating         FlexFloat        `json:"rating"`
	Rating5        FlexFloat        `json:"rating_5based"`
	ReleaseDate    string           `json:"releaseDate"`
	SeriesID       FlexInt          `json:"series_id"`
	StreamType     string           `json:"stream_type"`
	TmdbID         FlexInt          `json:"tmdb,omitempty"`
	TvdbID         FlexInt          `json:"tvdb,omitzero"`
	YoutubeTrailer string           `json:"youtube_trailer"`
}

// seriesInfoAlias prevents UnmarshalJSON recursion.
type seriesInfoAlias SeriesInfo

// UnmarshalJSON coalesces provider key variants:
//   - releaseDate / release_date — ProviderA/B emit releaseDate, C emits release_date
//   - tmdb / tmdb_id, imdb / imdb_id, tvdb / tvdb_id — short form is canonical
//     on this struct (matches ProviderA/B); long forms accepted for ProviderC
//     compatibility (Dispatcharr emits _id-suffixed forms on the list shape).
func (s *SeriesInfo) UnmarshalJSON(b []byte) error {
	type wire struct {
		*seriesInfoAlias
		ReleaseDateSnake string  `json:"release_date"`
		TmdbIDAlt        FlexInt `json:"tmdb_id"`
		ImdbIDAlt        string  `json:"imdb_id"`
		TvdbIDAlt        FlexInt `json:"tvdb_id"`
	}
	w := wire{seriesInfoAlias: (*seriesInfoAlias)(s)}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	if s.ReleaseDate == "" {
		s.ReleaseDate = w.ReleaseDateSnake
	}
	if s.TmdbID.Int64() == 0 {
		s.TmdbID = w.TmdbIDAlt
	}
	if s.ImdbID == "" {
		s.ImdbID = w.ImdbIDAlt
	}
	if s.TvdbID.Int64() == 0 {
		s.TvdbID = w.TvdbIDAlt
	}
	return nil
}

// EpisodeInfo contains the metadata block embedded within a SeriesEpisode.
type EpisodeInfo struct {
	AirDate        string           `json:"air_date,omitempty"`
	BackdropPath   *JSONStringSlice `json:"backdrop_path,omitempty"`
	Bitrate        FlexInt          `json:"bitrate,omitempty"`
	Crew           string           `json:"crew,omitempty"`
	DirectedBy     string           `json:"directed_by,omitempty"`
	Duration       string           `json:"duration,omitempty"`
	DurationSecs   FlexInt          `json:"duration_secs,omitempty"`
	ID             FlexInt          `json:"id,omitempty"`
	ImdbID         string           `json:"imdb_id,omitempty"`
	MovieImage     string           `json:"movie_image"`
	MovieImageTmdb string           `json:"movie_image_tmdb,omitempty"`
	Name           string           `json:"name,omitempty"`
	Overview       string           `json:"overview,omitempty"`
	Plot           string           `json:"plot"`
	Rating         FlexFloat        `json:"rating"`
	ReleaseDate    string           `json:"releasedate"`
	TmdbID         FlexInt          `json:"tmdb_id,omitempty"`
	TvdbID         FlexInt          `json:"tvdb_id,omitzero"`
}

// episodeInfoAlias prevents UnmarshalJSON recursion.
type episodeInfoAlias EpisodeInfo

// UnmarshalJSON coalesces provider key variants:
//   - releasedate / release_date — struct uses releasedate; C emits release_date
//   - tmdb / tmdb_id, imdb / imdb_id, tvdb / tvdb_id — long form is canonical
//     on this struct; short forms accepted for cross-provider tolerance.
func (e *EpisodeInfo) UnmarshalJSON(b []byte) error {
	type wire struct {
		*episodeInfoAlias
		ReleaseDateSnake string  `json:"release_date"`
		TmdbAlt          FlexInt `json:"tmdb"`
		ImdbAlt          string  `json:"imdb"`
		TvdbAlt          FlexInt `json:"tvdb"`
	}
	w := wire{episodeInfoAlias: (*episodeInfoAlias)(e)}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	if e.ReleaseDate == "" {
		e.ReleaseDate = w.ReleaseDateSnake
	}
	if e.TmdbID.Int64() == 0 {
		e.TmdbID = w.TmdbAlt
	}
	if e.ImdbID == "" {
		e.ImdbID = w.ImdbAlt
	}
	if e.TvdbID.Int64() == 0 {
		e.TvdbID = w.TvdbAlt
	}
	return nil
}

type SeriesEpisode struct {
	Added              Timestamp   `json:"added"`
	ContainerExtension string      `json:"container_extension"`
	CustomSid          string      `json:"custom_sid"`
	DirectSource       string      `json:"direct_source"`
	EpisodeNum         FlexInt     `json:"episode_num"`
	ID                 FlexInt     `json:"id"`
	ImdbID             string      `json:"imdb,omitempty"`
	Info               EpisodeInfo `json:"info"`
	Season             FlexInt     `json:"season"`
	Title              string      `json:"title"`
	TmdbID             FlexInt     `json:"tmdb,omitempty"`
	TvdbID             FlexInt     `json:"tvdb,omitzero"`
}

// seriesEpisodeAlias prevents UnmarshalJSON recursion.
type seriesEpisodeAlias SeriesEpisode

// UnmarshalJSON accepts tmdb_id/imdb_id/tvdb_id as input aliases for the
// canonical short-form tmdb/imdb/tvdb keys on this struct.
func (s *SeriesEpisode) UnmarshalJSON(b []byte) error {
	type wire struct {
		*seriesEpisodeAlias
		TmdbIDAlt FlexInt `json:"tmdb_id"`
		ImdbIDAlt string  `json:"imdb_id"`
		TvdbIDAlt FlexInt `json:"tvdb_id"`
	}
	w := wire{seriesEpisodeAlias: (*seriesEpisodeAlias)(s)}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	if s.TmdbID.Int64() == 0 {
		s.TmdbID = w.TmdbIDAlt
	}
	if s.ImdbID == "" {
		s.ImdbID = w.ImdbIDAlt
	}
	if s.TvdbID.Int64() == 0 {
		s.TvdbID = w.TvdbIDAlt
	}
	return nil
}

// Season describes a single season within a series.
type Season struct {
	AirDate      string  `json:"air_date"`
	Cover        string  `json:"cover"`
	CoverBig     string  `json:"cover_big"`
	CoverTmdb    string  `json:"cover_tmdb"`
	Duration     FlexInt `json:"duration"`
	EpisodeCount FlexInt `json:"episode_count"`
	ImdbID       string  `json:"imdb,omitempty"`
	Name         string  `json:"name"`
	Overview     string  `json:"overview"`
	ReleaseDate  string  `json:"releaseDate"`
	SeasonNumber FlexInt `json:"season_number"`
	TmdbID       FlexInt `json:"tmdb,omitempty"`
	TvdbID       FlexInt `json:"tvdb,omitzero"`
}

// seasonAlias prevents UnmarshalJSON recursion.
type seasonAlias Season

// UnmarshalJSON coalesces releaseDate/release_date into ReleaseDate and
// accepts tmdb_id/imdb_id/tvdb_id as aliases for the canonical short-form
// tmdb/imdb/tvdb keys on this struct.
func (s *Season) UnmarshalJSON(b []byte) error {
	type wire struct {
		*seasonAlias
		ReleaseDateSnake string  `json:"release_date"`
		TmdbIDAlt        FlexInt `json:"tmdb_id"`
		ImdbIDAlt        string  `json:"imdb_id"`
		TvdbIDAlt        FlexInt `json:"tvdb_id"`
	}
	w := wire{seasonAlias: (*seasonAlias)(s)}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	if s.ReleaseDate == "" {
		s.ReleaseDate = w.ReleaseDateSnake
	}
	if s.TmdbID.Int64() == 0 {
		s.TmdbID = w.TmdbIDAlt
	}
	if s.ImdbID == "" {
		s.ImdbID = w.ImdbIDAlt
	}
	if s.TvdbID.Int64() == 0 {
		s.TvdbID = w.TvdbIDAlt
	}
	return nil
}

type Series struct {
	Episodes map[string][]SeriesEpisode `json:"episodes"`
	Info     SeriesInfo                 `json:"info"`
	Seasons  []Season                   `json:"seasons"`
}

// VODInfo is the metadata block within a VideoOnDemandInfo response.
type VODInfo struct {
	Actors         string    `json:"actors"`
	Age            string    `json:"age"`
	BackdropPath   []string  `json:"backdrop_path"`
	Bitrate        FlexInt   `json:"bitrate"`
	Cast           string    `json:"cast"`
	Country        string    `json:"country"`
	Cover          string    `json:"cover,omitempty"`
	CoverBig       string    `json:"cover_big"`
	Description    string    `json:"description"`
	Director       string    `json:"director"`
	Duration       string    `json:"duration"`
	DurationSecs   FlexInt   `json:"duration_secs"`
	EpisodeRunTime *FlexInt  `json:"episode_run_time,omitempty"`
	Genre          string    `json:"genre"`
	ImdbID         string    `json:"imdb_id,omitempty"`
	KinopoiskURL   string    `json:"kinopoisk_url,omitempty"`
	MovieImage     string    `json:"movie_image"`
	Name           string    `json:"name"`
	OriginalName   string    `json:"o_name"`
	Plot           string    `json:"plot"`
	Rating         FlexFloat `json:"rating"`
	ReleaseDate    string    `json:"releasedate"`
	Runtime        string    `json:"runtime,omitempty"`
	Status         string    `json:"status"`
	TmdbID         FlexInt   `json:"tmdb_id"`
	TvdbID         FlexInt   `json:"tvdb_id,omitzero"`
	Year           FlexInt   `json:"year,omitempty"`
	YoutubeTrailer string    `json:"youtube_trailer"`
}

// vodInfoAlias prevents UnmarshalJSON recursion.
type vodInfoAlias VODInfo

// UnmarshalJSON coalesces provider key variants:
//   - releasedate / release_date — struct uses releasedate; C emits release_date
//   - tmdb / tmdb_id, imdb / imdb_id, tvdb / tvdb_id — long form is canonical
//     on this struct (matches existing tmdb_id precedent across providers);
//     short forms accepted for cross-provider tolerance.
func (v *VODInfo) UnmarshalJSON(b []byte) error {
	type wire struct {
		*vodInfoAlias
		ReleaseDateSnake string  `json:"release_date"`
		TmdbAlt          FlexInt `json:"tmdb"`
		ImdbAlt          string  `json:"imdb"`
		TvdbAlt          FlexInt `json:"tvdb"`
	}
	w := wire{vodInfoAlias: (*vodInfoAlias)(v)}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	if v.ReleaseDate == "" {
		v.ReleaseDate = w.ReleaseDateSnake
	}
	if v.TmdbID.Int64() == 0 {
		v.TmdbID = w.TmdbAlt
	}
	if v.ImdbID == "" {
		v.ImdbID = w.ImdbAlt
	}
	if v.TvdbID.Int64() == 0 {
		v.TvdbID = w.TvdbAlt
	}
	return nil
}

// VODMovieData is the stream identity block within a VideoOnDemandInfo response.
type VODMovieData struct {
	Added              Timestamp `json:"added"`
	CategoryID         FlexInt   `json:"category_id"`
	CategoryIDs        []FlexInt `json:"category_ids,omitempty"`
	ContainerExtension string    `json:"container_extension"`
	CustomSid          string    `json:"custom_sid"`
	DirectSource       string    `json:"direct_source"`
	Name               string    `json:"name"`
	StreamID           FlexInt   `json:"stream_id"`
}

// VideoOnDemandInfo contains information about a video on demand stream.
type VideoOnDemandInfo struct {
	Info      VODInfo      `json:"info"`
	MovieData VODMovieData `json:"movie_data"`
}

// UnmarshalJSON tolerates providers that emit "info": [] (empty array) when
// the metadata block is absent for a given record — treat as a zero-value
// VODInfo rather than failing the whole decode.
func (v *VideoOnDemandInfo) UnmarshalJSON(b []byte) error {
	type wire struct {
		Info      json.RawMessage `json:"info"`
		MovieData json.RawMessage `json:"movie_data"`
	}
	var w wire
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	if len(w.Info) > 0 {
		if err := json.Unmarshal(w.Info, &v.Info); err != nil {
			// Tolerate "info": [] specifically — the empty-array sentinel
			// some providers emit when the metadata block is absent.
			// A non-empty array is real data we don't know how to shape;
			// surface the original error rather than silently dropping it.
			var placeholder []json.RawMessage
			if arrErr := json.Unmarshal(w.Info, &placeholder); arrErr != nil || len(placeholder) != 0 {
				return err
			}
		}
	}
	if len(w.MovieData) > 0 {
		if err := json.Unmarshal(w.MovieData, &v.MovieData); err != nil {
			return err
		}
	}
	return nil
}

type epgContainer struct {
	EPGListings []EPGInfo `json:"epg_listings"`
}

// EPGInfo describes electronic programming guide information of a stream.
type EPGInfo struct {
	ChannelID      string             `json:"channel_id"`
	Description    Base64Value        `json:"description"`
	End            string             `json:"end"`
	EPGID          FlexInt            `json:"epg_id"`
	HasArchive     ConvertibleBoolean `json:"has_archive"`
	ID             FlexInt            `json:"id"`
	Lang           string             `json:"lang"`
	NowPlaying     ConvertibleBoolean `json:"now_playing"`
	Start          string             `json:"start"`
	StartTimestamp Timestamp          `json:"start_timestamp"`
	StopTimestamp  Timestamp          `json:"stop_timestamp"`
	Title          Base64Value        `json:"title"`
}
