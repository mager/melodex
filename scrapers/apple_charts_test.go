package scrapers

import (
	"strings"
	"testing"
)

func TestAppleChartFeed(t *testing.T) {
	songs, err := parseAppleChart(strings.NewReader(`{"feed":{"results":[{"id":"1","name":"A song","artistName":"An artist","kind":"songs"},{"id":"1","name":"A song","artistName":"An artist","kind":"songs"},{"id":"2","name":"","artistName":"X","kind":"songs"}]}}`))
	if err != nil || len(songs) != 1 || songs[0].Rank != 1 || songs[0].Title != "A song" {
		t.Fatalf("bad feed: %+v %v", songs, err)
	}
	for _, body := range []string{`{}`, `oops`, `{"feed":{"results":[]}}`} {
		if _, err := parseAppleChart(strings.NewReader(body)); err == nil {
			t.Fatal("invalid feed accepted")
		}
	}
}
