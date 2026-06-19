# iptv-proxy

A small reverse proxy for IPTV M3U playlists and Xtream-Codes-compatible
provider APIs. Translates upstream credentials and stream URLs into
proxy-served credentials and URLs so players talk to one stable
endpoint regardless of changes upstream.

## About this fork

This is a community-maintained fork of
[`pierre-emmanuelJ/iptv-proxy`](https://github.com/pierre-emmanuelJ/iptv-proxy),
which has been dormant since 2023. The fork inherits GPLv3 and
preserves the original copyright in every source file. Focus is
stability, current dependencies, and closing recurring failure
classes that have accumulated in the upstream issue tracker —
not feature expansion.

## Configuration

All configuration is via environment variables. Names follow three
categorical prefixes that describe which surface the variable
operates on:

- **`SOURCE_*`** — what the proxy reads from (M3U URL, Xtream provider
  credentials and behavior, user-agent sent to the source).
- **`PROXY_*`** — what the proxy exposes to the player (auth credentials,
  served M3U filename).
- **`REWRITE_*`** — what the proxy emits in URLs (hostname, ports, scheme).

Most variables accept the upstream-original name as a deprecated
alias and log a deprecation warning when used. Two variables have no
back-compat alias by design and are noted below.

### `SOURCE_*` — upstream connection

| Var | Deprecated alias | Default | Notes |
|---|---|---|---|
| `SOURCE_M3U_URL` | `M3U_URL` | `""` | Remote URL or local file path of the source M3U. Empty disables M3U mode. |
| `SOURCE_XC_BASE_URL` | `XTREAM_BASE_URL` | `""` | Base URL of the Xtream provider (e.g., `http://provider.tv:1234`). |
| `SOURCE_XC_USER` | `XTREAM_USER` | `""` | Xtream provider username. |
| `SOURCE_XC_PASSWORD` | `XTREAM_PASSWORD` | `""` | Xtream provider password. |
| `SOURCE_XC_CACHED_M3U_TTL` | `M3U_CACHE_EXPIRATION` | `1` | Hours to cache the rebuilt M3U before re-fetching from upstream. |
| `SOURCE_XC_APIGET_NOBACKCOMPAT` | `XTREAM_API_GET` | `false` | When `true`, `/get.php` builds the M3U from the provider's API endpoints (includes Series and VOD). Default keeps the legacy behavior of forwarding the provider's own `/get.php` response. |
| `SOURCE_UA_OVERRIDE` | — | `""` (passthrough) | User-agent sent to the upstream. Empty passes the client's UA through unchanged; set when a provider gates on a specific UA. |

### `PROXY_*` — what the proxy presents to the player

| Var | Deprecated alias | Default | Notes |
|---|---|---|---|
| `PROXY_USER` | `USER` | `usertest` | Username clients use to authenticate to this proxy. **Set this.** |
| `PROXY_PASSWORD` | `PASSWORD` | `passwordtest` | Password for the same. **Set this.** |
| `PROXY_M3U` | `M3U_FILE_NAME` | `iptv.m3u` | Filename served as the player-facing M3U. |

### `REWRITE_*` — what the proxy emits in URLs

| Var | Deprecated alias | Default | Notes |
|---|---|---|---|
| `REWRITE_HOSTNAME` | — (no alias) | `""` | Hostname or IP this proxy advertises in rewritten URLs. **No back-compat for `HOSTNAME`** — Docker auto-sets that variable in containers, and silently consuming it caused operator pain. Set `REWRITE_HOSTNAME` explicitly. |
| `REWRITE_PORT` | `PORT` | `8080` | TCP port the proxy listens on. |
| `REWRITE_REVPROXY_PORT` | `ADVERTISED_PORT` | listen port | Port advertised in rewritten URLs. Set when behind a reverse proxy on a different external port. |
| `REWRITE_HTTPS` | `HTTPS` | `false` | Emit `https://` in rewritten URLs. Use behind a TLS-terminating reverse proxy. |

Bad values for integer / boolean variables fail at startup with a
clear message rather than silently falling back to a default.

All proxy routes are mounted under `/xcp/` — operators previously
using `CUSTOM_ENDPOINT` to set a path prefix should handle path
routing at their reverse proxy now. The `xcp` namespace is part of
the proxy's identity.

## Quickstart

Minimal `docker-compose.yml` for an Xtream upstream:

```yaml
version: "3"
services:
  iptv-proxy:
    build: .
    container_name: iptv-proxy
    restart: on-failure
    ports:
      - 8080:8080
    environment:
      # Upstream Xtream provider
      SOURCE_XC_BASE_URL: "http://provider.example.tv:8080"
      SOURCE_XC_USER: your_provider_username
      SOURCE_XC_PASSWORD: your_provider_password

      # Credentials the player uses to connect to this proxy
      PROXY_USER: your_proxy_username
      PROXY_PASSWORD: your_proxy_password

      # How the proxy advertises itself in rewritten URLs
      REWRITE_HOSTNAME: iptv-proxy.example.lan
      REWRITE_PORT: 8080

      GIN_MODE: release
```

Player then connects using `PROXY_USER` / `PROXY_PASSWORD` against
`http://iptv-proxy.example.lan:8080/xcp/get.php` (or the
player's Xtream-Codes setup screen).

## Modes

### M3U passthrough

Operator provides a source M3U via `SOURCE_M3U_URL` (remote URL or
local file path). The proxy parses the M3U, registers a route per
track under `/xcp/<PROXY_USER>/<PROXY_PASSWORD>/<index>/<basename>`,
and serves a rewritten M3U at `/xcp/<PROXY_M3U>` containing those
proxy-side URLs. Players see a stable proxy endpoint instead of
the upstream provider's URLs.

### Xtream-Codes proxy

Operator provides upstream Xtream credentials via `SOURCE_XC_*`. The
proxy presents itself as an Xtream-Codes-compatible API at the
standard endpoints (`/xcp/player_api.php`, `/xcp/xmltv.php`,
`/xcp/get.php`, etc.) using `PROXY_USER` / `PROXY_PASSWORD`. Player
authenticates against the proxy; the proxy translates calls to the
upstream provider.

Both modes can be configured simultaneously.

## Building from source

```bash
# Build the binary
go build -mod=vendor

# Build a multi-arch container image
docker buildx build --platform linux/amd64,linux/arm64 -t iptv-proxy:local .
```

## License

GPLv3, inherited from the original
[`pierre-emmanuelJ/iptv-proxy`](https://github.com/pierre-emmanuelJ/iptv-proxy).
Pierre-Emmanuel Jacquier's copyright is preserved in every modified
source file (GPLv3 §5(a)). Modifications carry an additional
`Copyright (C) 2026 warrentc3` line above the original.

## Powered by

- [gin](https://github.com/gin-gonic/gin) — HTTP routing.
- [gosettings](https://github.com/qdm12/gosettings) by qdm12 — env var
  reader with deprecation-warning support.
- [go.xtream-codes](https://github.com/kludgarr/go.xtream-codes) —
  Xtream-Codes API client (a maintained fork of `tellytv/go.xtream-codes`).
