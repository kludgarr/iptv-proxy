package server

import (
	"errors"
	"testing"
)

func TestResolveHLSRequestURLUsesConfiguredPlaylistURI(t *testing.T) {
	const trackURI = "https://source:secret@provider.example/live/index.m3u8?playlist_token=one"

	got, err := resolveHLSRequestURL(trackURI, "index.m3u8", "player_query=ignored")
	if err != nil {
		t.Fatalf("resolveHLSRequestURL returned an error: %v", err)
	}
	if got.String() != trackURI {
		t.Fatalf("playlist URI changed: got %q, want %q", got.String(), trackURI)
	}
}

func TestResolveHLSRequestURLResolvesChildReference(t *testing.T) {
	got, err := resolveHLSRequestURL(
		"https://source:secret@provider.example/live/index.m3u8?playlist_token=one",
		"segment001.ts",
		"segment_token=two",
	)
	if err != nil {
		t.Fatalf("resolveHLSRequestURL returned an error: %v", err)
	}

	const want = "https://source:secret@provider.example/live/segment001.ts?segment_token=two"
	if got.String() != want {
		t.Fatalf("resolved URI mismatch: got %q, want %q", got.String(), want)
	}
}

func TestResolveHLSRequestURLPreservesConfiguredAuthority(t *testing.T) {
	got, err := resolveHLSRequestURL(
		"http://segment.ts/live/segment.ts?token=secret",
		"169.254.169.254",
		"",
	)
	if err != nil {
		t.Fatalf("resolveHLSRequestURL returned an error: %v", err)
	}

	if got.Scheme != "http" {
		t.Fatalf("scheme changed: got %q", got.Scheme)
	}
	if got.Host != "segment.ts" {
		t.Fatalf("host changed: got %q", got.Host)
	}
	if got.Path != "/live/169.254.169.254" {
		t.Fatalf("path mismatch: got %q", got.Path)
	}
	if got.RawQuery != "" {
		t.Fatalf("playlist query leaked into child request: got %q", got.RawQuery)
	}
}

func TestResolveHLSRequestURLRejectsInvalidReferences(t *testing.T) {
	tests := []string{
		"",
		".",
		"..",
		"../admin",
		`..\admin`,
		"//169.254.169.254",
		"segment.ts?redirect=http://169.254.169.254",
		"segment.ts#fragment",
		"segment.ts\r\nX-Test: injected",
	}

	for _, segment := range tests {
		t.Run(segment, func(t *testing.T) {
			_, err := resolveHLSRequestURL("https://provider.example/live/index.m3u8", segment, "")
			if !errors.Is(err, errInvalidHLSReference) {
				t.Fatalf("expected invalid-reference error for %q, got %v", segment, err)
			}
		})
	}
}

func TestResolveHLSRequestURLRejectsInvalidBase(t *testing.T) {
	tests := []string{
		"",
		"/local/index.m3u8",
		"ftp://provider.example/index.m3u8",
		"http://",
	}

	for _, trackURI := range tests {
		t.Run(trackURI, func(t *testing.T) {
			_, err := resolveHLSRequestURL(trackURI, "index.m3u8", "")
			if !errors.Is(err, errInvalidHLSBaseURL) {
				t.Fatalf("expected invalid-base error for %q, got %v", trackURI, err)
			}
		})
	}
}
