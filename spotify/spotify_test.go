package spotify

import (
	"context"
	"fmt"
	"io"
	"melodex/config"
	"net/http"
	"strings"
	"testing"
)

type tokenTransport func(*http.Request) (*http.Response, error)

func (f tokenTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestSpotifyRefreshesExpiredToken(t *testing.T) {
	prior := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = prior })
	tokens := 0
	http.DefaultTransport = tokenTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"id":"track"}`
		if r.URL.Host == "accounts.spotify.com" {
			tokens++
			expiry := 1
			if tokens > 1 {
				expiry = 3600
			}
			body = fmt.Sprintf(`{"access_token":"test-%d","token_type":"Bearer","expires_in":%d}`, tokens, expiry)
		} else if r.Header.Get("Authorization") != "Bearer test-2" {
			t.Error("expired token reused")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	client := ProvideSpotify(config.Config{SpotifyID: "test", SpotifySecret: "test"})
	if _, err := client.Client.GetTrack(context.Background(), "track"); err != nil {
		t.Fatal(err)
	}
	if tokens != 2 {
		t.Fatalf("token requests=%d", tokens)
	}
}
