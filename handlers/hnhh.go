package handlers

import (
	"context"
	fs "melodex/firestore"
	"melodex/scrapers"
	"net/http"
)

func (h *ScrapeHandler) HandleHotNewHipHop(w http.ResponseWriter, r *http.Request) {
	h.handleMusicChart(w, r, "hnhh", func(ctx context.Context, w http.ResponseWriter) ([]fs.Song, error) {
		tracks, err := scrapers.ScrapeHotNewHipHop(w)
		songs := make([]fs.Song, 0, len(tracks))
		for _, track := range tracks {
			songs = append(songs, fs.Song{Rank: track.Rank, Artist: track.Artist, Title: track.Title})
		}
		return songs, err
	}, false)
}
