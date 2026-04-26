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

package server

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

func (c *Config) routes(r *gin.RouterGroup) {
	r = r.Group(xcpNamespace)

	//Xtream service endopoints
	if c.ProxyConfig.XtreamBaseURL != "" {
		c.xtreamRoutes(r)
		if strings.Contains(c.XtreamBaseURL, c.RemoteURL.Host) &&
			c.XtreamUser.String() == c.RemoteURL.Query().Get("username") &&
			c.XtreamPassword.String() == c.RemoteURL.Query().Get("password") {

			r.GET("/"+c.M3UFileName, c.authenticate, c.xtreamGetAuto)
			// XXX Private need: for external Android app
			r.POST("/"+c.M3UFileName, c.authenticate, c.xtreamGetAuto)

			return
		}
	}

	c.m3uRoutes(r)
}

func (c *Config) xtreamRoutes(r *gin.RouterGroup) {
	getphp := gin.HandlerFunc(c.xtreamGet)
	if c.XtreamGenerateApiGet {
		getphp = c.xtreamApiGet
	}
	r.GET("/get.php", c.authenticate, getphp)
	r.POST("/get.php", c.authenticate, getphp)
	r.GET("/apiget", c.authenticate, c.xtreamApiGet)
	r.GET("/player_api.php", c.authenticate, c.xtreamPlayerAPIGET)
	r.POST("/player_api.php", c.appAuthenticate, c.xtreamPlayerAPIPOST)
	r.GET("/xmltv.php", c.authenticate, c.xtreamXMLTV)
	r.GET(fmt.Sprintf("/%s/%s/:id", c.User, c.Password), c.xtreamStreamHandler)
	r.GET(fmt.Sprintf("/live/%s/%s/:id", c.User, c.Password), c.xtreamStreamLive)
	r.GET(fmt.Sprintf("/timeshift/%s/%s/:duration/:start/:id", c.User, c.Password), c.xtreamStreamTimeshift)
	r.GET(fmt.Sprintf("/movie/%s/%s/:id", c.User, c.Password), c.xtreamStreamMovie)
	r.GET(fmt.Sprintf("/series/%s/%s/:id", c.User, c.Password), c.xtreamStreamSeries)
	r.GET(fmt.Sprintf("/hlsr/:token/%s/%s/:channel/:hash/:chunk", c.User, c.Password), c.xtreamHlsrStream)
	r.GET("/hls/:chunk", c.xtreamHlsStream)
	r.GET("/play/:token/:type", c.xtreamStreamPlay)
}

func (c *Config) m3uRoutes(r *gin.RouterGroup) {
	r.GET("/"+c.M3UFileName, c.authenticate, c.getM3U)
	// XXX Private need: for external Android app
	r.POST("/"+c.M3UFileName, c.authenticate, c.getM3U)

	// Local-scope dedup: gin panics on duplicate route registration within a
	// single router; the dedup only needs to scope to this build pass. Keeping
	// the map local eliminates cross-instance leakage if multiple Server
	// instances ever share a process.
	registered := map[string]struct{}{}

	for i, track := range c.playlist.Tracks {
		u, err := url.Parse(track.URI)
		if err != nil {
			continue
		}

		trackConfig := &Config{
			ProxyConfig: c.ProxyConfig,
			track:       &c.playlist.Tracks[i],
		}

		if strings.HasSuffix(track.URI, ".m3u8") {
			key := fmt.Sprintf("/%s/%s/%d/:id", c.User, c.Password, i)
			if _, exists := registered[key]; !exists {
				registered[key] = struct{}{}
				r.GET(key, trackConfig.m3u8ReverseProxy)
			}
		} else {
			// path.Base(u.Path) — single-segment basename from the parsed URL.
			// Drops query string (avoids the original SOURCE bug of collapsing
			// query-distinct URIs into the same key) and avoids embedding
			// multi-segment paths from u.Path that wouldn't match the
			// player-requested URL emitted by replaceURL (which uses
			// path.Base on the URI path).
			key := fmt.Sprintf("/%s/%s/%d/%s", c.User, c.Password, i, path.Base(u.Path))
			if _, exists := registered[key]; !exists {
				registered[key] = struct{}{}
				r.GET(key, trackConfig.reverseProxy)
			}
		}
	}
}
