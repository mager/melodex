package handlers

import (
	"context"
	"errors"
	spot "github.com/zmb3/spotify/v2"
	"io"
	"log"
	fs "melodex/firestore"
	localspotify "melodex/spotify"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type memoryMusicStore struct {
	saved    []fs.Track
	writes   int
	snapshot musicSnapshot
}

func (s *memoryMusicStore) load(context.Context, string, string) (musicSnapshot, error) {
	return s.snapshot, nil
}
func (s *memoryMusicStore) save(_ context.Context, _, _ string, tracks []fs.Track) error {
	s.writes++
	s.saved = tracks
	return nil
}

func TestChartRefreshRetriesMissingMetadataAndUsesCurrentRanks(t *testing.T) {
	oldTime := time.Now().Add(-24 * time.Hour)
	store := &memoryMusicStore{snapshot: musicSnapshot{Tracks: []fs.Track{
		{Artist: "A", Title: "One", Rank: 10, SpotifyID: strings.Repeat("a", 22), Thumb: "art", CreatedAt: oldTime},
		{Artist: "B", Title: "Two", Rank: 20},
	}}}
	requests := 0
	client := spot.New(&http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		if !strings.Contains(r.URL.Query().Get("q"), "Two") {
			t.Error("repeated complete lookup")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"tracks":{"items":[{"id":"bbbbbbbbbbbbbbbbbbbbbb","name":"Two","artists":[{"name":"B"}],"album":{"images":[{"url":"https://i.scdn.co/image/art2","width":300}]}}]}}`))}, nil
	})})
	h := NewScrapeHandler(nil, &localspotify.SpotifyClient{Client: client}, nil)
	h.store = store
	fetch := func(context.Context, http.ResponseWriter) ([]fs.Song, error) {
		return []fs.Song{{Artist: "A", Title: "One", Rank: 1}, {Artist: "B", Title: "Two", Rank: 2}}, nil
	}
	w := httptest.NewRecorder()
	h.handleMusicChart(w, httptest.NewRequest("POST", "/scrape", nil), "billboard", fetch, false)
	if w.Code != 200 || store.writes != 1 || len(store.saved) != 2 || requests != 1 {
		t.Fatalf("status=%d saved=%v requests=%d", w.Code, store.saved, requests)
	}
	if store.saved[0].Rank != 1 || !store.saved[0].CreatedAt.Equal(oldTime) || store.saved[1].SpotifyID != strings.Repeat("b", 22) {
		t.Fatalf("bad metadata reuse: %+v", store.saved)
	}
}

func TestChartProviderFailureDoesNotReplaceGoodSnapshot(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(map[bool]string{false: "lookup failure", true: "empty chart"}[empty], func(t *testing.T) {
			store := &memoryMusicStore{}
			client := spot.New(&http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) { return nil, errors.New("provider unavailable") })})
			h := NewScrapeHandler(nil, &localspotify.SpotifyClient{Client: client}, nil)
			h.store = store
			fetch := func(context.Context, http.ResponseWriter) ([]fs.Song, error) {
				if empty {
					return nil, nil
				}
				return []fs.Song{{Artist: "B", Title: "Two", Rank: 2}}, nil
			}
			w := httptest.NewRecorder()
			h.handleMusicChart(w, httptest.NewRequest("POST", "/scrape", nil), "billboard", fetch, false)
			if w.Code != 502 || store.writes != 0 {
				t.Fatalf("bad source saved: status=%d writes=%d", w.Code, store.writes)
			}
		})
	}
}

func TestCompletedChartSkipsUntilExplicitRefresh(t *testing.T) {
	store := &memoryMusicStore{snapshot: musicSnapshot{Version: 2, Tracks: []fs.Track{{SpotifyID: strings.Repeat("a", 22), Thumb: "art"}}}}
	h := NewScrapeHandler(nil, nil, nil)
	h.store = store
	called := false
	fetch := func(context.Context, http.ResponseWriter) ([]fs.Song, error) {
		called = true
		return nil, errors.New("offline")
	}
	w := httptest.NewRecorder()
	h.handleMusicChart(w, httptest.NewRequest("POST", "/scrape", nil), "billboard", fetch, false)
	if w.Code != 409 || called {
		t.Fatal("completed source was repeated")
	}
	w = httptest.NewRecorder()
	h.handleMusicChart(w, httptest.NewRequest("POST", "/scrape?refresh=true", nil), "billboard", fetch, false)
	if w.Code != 502 || !called || store.writes != 0 {
		t.Fatal("explicit refresh lost good snapshot")
	}
}

func TestNewReleasesProviderFailureIsNotSuccess(t *testing.T) {
	client := spot.New(&http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"albums":{"items":[{"id":"album"}]}}`
		status := 200
		if strings.Contains(r.URL.Path, "/albums/") {
			status = 429
			body = `{"error":{"status":429,"message":"limited"}}`
		}
		return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})})
	h := NewScrapeHandler(nil, &localspotify.SpotifyClient{Client: client}, nil)
	w := httptest.NewRecorder()
	h.HandleSpotifyNewReleases(w, httptest.NewRequest("POST", "/scrape?debug=true", nil))
	if w.Code < 500 {
		t.Fatalf("incomplete releases reported success: %d", w.Code)
	}
}

func TestTargetedRefreshRecordsItsOutcome(t *testing.T) {
	var output strings.Builder
	prior := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(prior)
	client := spot.New(&http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) { return nil, errors.New("provider unavailable") })})
	h := NewScrapeHandler(nil, &localspotify.SpotifyClient{Client: client}, nil)
	w := httptest.NewRecorder()
	h.Handle(w, httptest.NewRequest("POST", "/scrape?debug=true", strings.NewReader(`{"target":"spotify-new-releases"}`)))
	if w.Code != 502 || !strings.Contains(output.String(), `"outcome":"failed"`) || !strings.Contains(output.String(), `"source":"spotify-new-releases"`) {
		t.Fatalf("targeted refresh lost status: %d %s", w.Code, output.String())
	}
}
