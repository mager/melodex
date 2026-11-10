package handlers

import (
	"encoding/json"
	spot "github.com/zmb3/spotify/v2"
	"io"
	fs "melodex/firestore"
	localspotify "melodex/spotify"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type catalogTransport func(*http.Request) (*http.Response, error)

func (f catalogTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestNewReleasesDoNotRequireMusicBrainz(t *testing.T) {
	calls := 0
	client := spot.New(&http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var body string
		switch r.URL.Path {
		case "/v1/browse/new-releases":
			body = `{"albums":{"items":[{"id":"album","images":[{"height":300,"width":300,"url":"https://i.scdn.co/image/art"}]}]}}`
		case "/v1/albums/album/tracks":
			body = `{"items":[{"id":"track","name":"A new track","artists":[{"name":"A new artist"}]}]}`
		case "/v1/tracks/track":
			body = `{"id":"track","external_ids":{"isrc":"TEST123"}}`
		default:
			t.Fatalf("unexpected enrichment request: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}, Request: r}, nil
	})})
	h := NewScrapeHandler(nil, &localspotify.SpotifyClient{Client: client}, nil)
	w := httptest.NewRecorder()
	h.HandleSpotifyNewReleases(w, httptest.NewRequest("POST", "/scrape?debug=true", strings.NewReader(`{}`)))
	var tracks []fs.Track
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &tracks) != nil || len(tracks) != 1 || tracks[0].SpotifyID != "track" || calls != 3 {
		t.Fatalf("valid new track lost: %d %s", w.Code, w.Body.String())
	}
}

func TestChartMatchingRejectsWrongSong(t *testing.T) {
	song := fs.Song{Title: "A song", Artist: "Artist A & Artist B"}
	var tracks []spot.FullTrack
	json.Unmarshal([]byte(`[{"id":"wrong","name":"A song (Live)","artists":[{"name":"Artist A"},{"name":"Artist B"}]},{"id":"wrong2","name":"A song","artists":[{"name":"Another artist"}]},{"id":"right","name":"A song","artists":[{"name":"Artist A"},{"name":"Artist B"}]}]`), &tracks)
	if got := matchChartTrack(song, tracks); got == nil || got.ID != "right" {
		t.Fatal("wrong recording selected")
	}
	if got := matchChartTrack(song, tracks[:2]); got != nil {
		t.Fatal("wrong edition accepted")
	}
}
