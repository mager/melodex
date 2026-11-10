package scrapers

import (
	"context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPitchforkCurrentCards(t *testing.T) {
	page := `<div class="summary-item"><a class="summary-item__hed-link" href="/reviews/tracks/star-moles-born-again/"><h3 class="summary-item__hed">“Born Again”</h3></a><div class="summary-item__sub-hed">Star Moles</div></div><div class="summary-item"><a class="summary-item__hed-link" href="/reviews/albums/another/"><h3 class="summary-item__hed">An album</h3></a><div class="summary-item__sub-hed">Someone</div></div>`
	songs, err := parsePitchfork(strings.NewReader(page))
	if err != nil || len(songs) != 1 || songs[0].Title != "Born Again" || songs[0].Artist != "Star Moles" {
		t.Fatalf("bad card parse: %+v %v", songs, err)
	}
	if _, err := parsePitchfork(strings.NewReader(`<html>temporarily unavailable</html>`)); err == nil {
		t.Fatal("empty page reported success")
	}
}

func TestPitchforkLiveCards(t *testing.T) {
	if os.Getenv("BEATBRAIN_LIVE_METADATA_TEST") != "1" {
		t.Skip("explicit live source check")
	}
	songs, err := ScrapePitchforkBestNewTracks(context.Background(), httptest.NewRecorder())
	if err != nil || len(songs) == 0 {
		t.Fatalf("no live tracks: %v", err)
	}
	t.Logf("Parsed %d current Pitchfork tracks", len(songs))
}
