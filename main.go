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
	"os"
	"strconv"
	"strings"

	"github.com/pierre-emmanuelJ/iptv-proxy/pkg/config"
	"github.com/pierre-emmanuelJ/iptv-proxy/pkg/server"
)

func main() {
	conf, err := buildConfig()
	if err != nil {
		log.Fatal(err)
	}
	srv, err := server.NewServer(conf)
	if err != nil {
		log.Fatal(err)
	}
	if err := srv.Serve(); err != nil {
		log.Fatal(err)
	}
}

func buildConfig() (*config.ProxyConfig, error) {
	m3uURL := getenv("M3U_URL", "")
	var remoteURL *url.URL
	if m3uURL != "" {
		u, err := url.Parse(m3uURL)
		if err != nil {
			return nil, fmt.Errorf("invalid M3U_URL: %w", err)
		}
		remoteURL = u
	}

	xtreamUser := getenv("XC_USER", "")
	xtreamPassword := getenv("XC_PASSWORD", "")
	xtreamBaseURL := getenv("XC_BASE_URL", "")

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

	port := getenvInt("XCPort", getenvInt("XC_PORT", 8080))
	advertisedPort := getenvInt("XC_ADVERTISED_PORT", 0)
	if advertisedPort == 0 {
		advertisedPort = port
	}

	return &config.ProxyConfig{
		HostConfig: &config.HostConfiguration{
			Hostname: getenv("XC_HOST", ""),
			Port:     port,
		},
		RemoteURL:            remoteURL,
		XtreamUser:           config.CredentialString(xtreamUser),
		XtreamPassword:       config.CredentialString(xtreamPassword),
		XtreamBaseURL:        xtreamBaseURL,
		XtreamUserAgent:      getenv("XC_USER_AGENT", ""),
		XtreamGenerateApiGet: getenvBool("XC_XTREAM_API_GET", false),
		M3UCacheExpiration:   getenvInt("XC_M3U_CACHE_EXPIRATION", 1),
		User:                 config.CredentialString(getenv("XC_PROXY_USER", "usertest")),
		Password:             config.CredentialString(getenv("XC_PROXY_PASSWORD", "passwordtest")),
		AdvertisedPort:       advertisedPort,
		HTTPS:                getenvBool("XC_HTTPS", false),
		M3UFileName:          getenv("XC_M3U_FILE_NAME", "iptv.m3u"),
		CustomEndpoint:       getenv("XC_CUSTOM_ENDPOINT", ""),
		CustomId:             getenv("XC_CUSTOM_ID", ""),
	}, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getenvBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1" || v == "yes"
}
