// v4 release

/*
 * Iptv-Proxy is a project to proxyfie an m3u file and to proxyfie an Xtream iptv service (client API).
 * Copyright (C) 2026  warrentc3
 * Copyright (C) 2020  Pierre-Emmanuel Jacquier
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/qdm12/gosettings/reader"

	"github.com/kludgarr/iptv-proxy/pkg/config"
	"github.com/kludgarr/iptv-proxy/pkg/httptrace"
	"github.com/kludgarr/iptv-proxy/pkg/server"
)

func main() {
	conf, err := buildConfig()
	if err != nil {
		log.Fatal(err)
	}
	httptrace.Init(conf.DebugHTTP, conf.DebugHTTPFile)
	srv, err := server.NewServer(conf)
	if err != nil {
		log.Fatal(err)
	}
	if err := srv.Serve(); err != nil {
		log.Fatal(err)
	}
}

// buildConfig reads all iptv-proxy configuration from the environment.
//
// Env var naming follows three categorical prefixes that name the surface
// the variable operates on:
//
//   - SOURCE_*   — what the proxy reads from (M3U URL, XC provider creds
//     and behaviors, UA sent to the source).
//   - PROXY_*    — what the proxy exposes to the player (auth credentials
//     and the served M3U filename).
//   - REWRITE_*  — what the proxy emits in URLs (hostname, ports, scheme).
//
// Most variables carry retroactive key support for the upstream-original
// names; gosettings logs a deprecation warning when an old name is used.
// Two have no retro by design: SOURCE_UA_OVERRIDE (net-new — pierre had
// no UA override) and REWRITE_HOSTNAME (deliberately no back-compat for
// HOSTNAME, which Docker auto-sets and silently consumed in the old code).
//
// Parse errors on integer / boolean values are surfaced as startup errors
// rather than silently falling back to a default. Bad config fails loudly.
func buildConfig() (*config.ProxyConfig, error) {
	r := reader.New(reader.Settings{
		HandleDeprecatedKey: func(source, deprecatedKey, currentKey string) {
			log.Printf("[iptv-proxy] DEPRECATED: %s env var %q is deprecated, use %q instead",
				source, deprecatedKey, currentKey)
		},
		DefaultOptions: []reader.Option{reader.ForceLowercase(false)},
	})

	m3uURL := r.String("SOURCE_M3U_URL", reader.RetroKeys("M3U_URL"))
	// url.Parse("") returns a non-nil zero-value *url.URL, NOT an error.
	// Preserves pierre's pre-modernization invariant that ProxyConfig.RemoteURL
	// is always non-nil; downstream code in pkg/server checks RemoteURL.String()
	// for emptiness rather than nil-comparing the pointer.
	remoteURL, err := url.Parse(m3uURL)
	if err != nil {
		return nil, fmt.Errorf("invalid SOURCE_M3U_URL: %w", err)
	}

	xtreamUser := r.String("SOURCE_XC_USER", reader.RetroKeys("XTREAM_USER"))
	xtreamPassword := r.String("SOURCE_XC_PASSWORD", reader.RetroKeys("XTREAM_PASSWORD"))
	xtreamBaseURL := r.String("SOURCE_XC_BASE_URL", reader.RetroKeys("XTREAM_BASE_URL"))

	// Auto-detect Xtream credentials from M3U URL when /get.php is present.
	if remoteURL != nil && strings.Contains(m3uURL, "/get.php") {
		if xtreamBaseURL == "" && xtreamUser == "" && xtreamPassword == "" {
			q := remoteURL.Query()
			if u := q.Get("username"); u != "" {
				xtreamUser = u
			}
			if p := q.Get("password"); p != "" {
				xtreamPassword = p
			}
			xtreamBaseURL = fmt.Sprintf("%s://%s", remoteURL.Scheme, remoteURL.Host)
			if xtreamUser != "" {
				log.Printf("[iptv-proxy] INFO: Xtream provider detected; base=%q user=%q", xtreamBaseURL, xtreamUser)
			}
		}
	}

	port, err := r.Int("REWRITE_PORT", reader.RetroKeys("PORT"))
	if err != nil {
		return nil, fmt.Errorf("invalid REWRITE_PORT: %w", err)
	}
	if port == 0 {
		port = 8080
	}

	advertisedPort, err := r.Int("REWRITE_REVPROXY_PORT", reader.RetroKeys("ADVERTISED_PORT"))
	if err != nil {
		return nil, fmt.Errorf("invalid REWRITE_REVPROXY_PORT: %w", err)
	}
	if advertisedPort == 0 {
		advertisedPort = port
	}

	cachedM3UTTL, err := r.Int("SOURCE_XC_CACHED_M3U_TTL", reader.RetroKeys("M3U_CACHE_EXPIRATION"))
	if err != nil {
		return nil, fmt.Errorf("invalid SOURCE_XC_CACHED_M3U_TTL: %w", err)
	}
	if cachedM3UTTL == 0 {
		cachedM3UTTL = 1
	}

	httpsPtr, err := r.BoolPtr("REWRITE_HTTPS", reader.RetroKeys("HTTPS"))
	if err != nil {
		return nil, fmt.Errorf("invalid REWRITE_HTTPS: %w", err)
	}
	https := false
	if httpsPtr != nil {
		https = *httpsPtr
	}

	// SOURCE_XC_APIGET_NOBACKCOMPAT preserves the value semantics of the
	// upstream XTREAM_API_GET flag (true = use /apiget which includes
	// Series and VOD; default false = back-compat to the simpler /get.php
	// behavior). The rename names the existence-of-the-flag as itself a
	// back-compat affordance — three years on, the legacy default is the
	// one most operators no longer want, but we keep the toggle for those
	// who do.
	apigetNoBackcompatPtr, err := r.BoolPtr("SOURCE_XC_APIGET_NOBACKCOMPAT", reader.RetroKeys("XTREAM_API_GET"))
	if err != nil {
		return nil, fmt.Errorf("invalid SOURCE_XC_APIGET_NOBACKCOMPAT: %w", err)
	}
	apigetNoBackcompat := false
	if apigetNoBackcompatPtr != nil {
		apigetNoBackcompat = *apigetNoBackcompatPtr
	}

	proxyUser := r.String("PROXY_USER", reader.RetroKeys("USER"))
	if proxyUser == "" {
		proxyUser = "usertest"
	}
	proxyPassword := r.String("PROXY_PASSWORD", reader.RetroKeys("PASSWORD"))
	if proxyPassword == "" {
		proxyPassword = "passwordtest"
	}
	m3uFileName := r.String("PROXY_M3U", reader.RetroKeys("M3U_FILE_NAME"))
	if m3uFileName == "" {
		m3uFileName = "iptv.m3u"
	}

	debugHTTPPtr, err := r.BoolPtr("DEBUG_HTTP")
	if err != nil {
		return nil, fmt.Errorf("invalid DEBUG_HTTP: %w", err)
	}
	debugHTTP := false
	if debugHTTPPtr != nil {
		debugHTTP = *debugHTTPPtr
	}
	debugHTTPFile := r.String("DEBUG_HTTP_FILE")

	return &config.ProxyConfig{
		HostConfig: &config.HostConfiguration{
			Hostname: r.String("REWRITE_HOSTNAME"),
			Port:     port,
		},
		RemoteURL:            remoteURL,
		XtreamUser:           config.CredentialString(xtreamUser),
		XtreamPassword:       config.CredentialString(xtreamPassword),
		XtreamBaseURL:        xtreamBaseURL,
		XtreamUserAgent:      r.String("SOURCE_UA_OVERRIDE"),
		XtreamGenerateApiGet: apigetNoBackcompat,
		M3UCacheExpiration:   cachedM3UTTL,
		User:                 config.CredentialString(proxyUser),
		Password:             config.CredentialString(proxyPassword),
		AdvertisedPort:       advertisedPort,
		HTTPS:                https,
		M3UFileName:          m3uFileName,
		DebugHTTP:            debugHTTP,
		DebugHTTPFile:        debugHTTPFile,
	}, nil
}
