package handlers

import (
	"context"
	fs "melodex/firestore"
	"melodex/scrapers"
	"net/http"
	"strings"
)

func (h *ScrapeHandler) HandleBillboard(w http.ResponseWriter, r *http.Request) {
	h.handleMusicChart(w, r, "billboard", func(ctx context.Context, w http.ResponseWriter) ([]fs.Song, error) {
		return scrapers.ScrapeBillboardHot100(w)
	}, false)
}

func buildSpotifyQuery(artist, title string) string {
	// Clean up artist name by removing "Featuring" and similar words
	patterns := []string{
		" Featuring ", " featuring ",
		" feat. ", " feat ",
		" ft. ", " ft ",
	}

	cleanArtist := artist
	for _, pattern := range patterns {
		cleanArtist = strings.ReplaceAll(cleanArtist, pattern, " ")
	}

	var q strings.Builder
	q.WriteString("artist:")
	q.WriteString(cleanArtist)
	q.WriteString(" ")
	q.WriteString("track:")
	q.WriteString(title)
	return q.String()
}
