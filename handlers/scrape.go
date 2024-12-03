package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/gorilla/mux"
	spotify "github.com/zmb3/spotify/v2"

	fs "melodex/firestore"
	"melodex/scrapers"
	spot "melodex/spotify"
)

type ScrapeHandler struct {
	db *firestore.Client
	sp *spot.SpotifyClient
}

func NewScrapeHandler(
	db *firestore.Client,
	sp *spot.SpotifyClient,
) *ScrapeHandler {
	return &ScrapeHandler{
		db: db,
		sp: sp,
	}
}

func (h *ScrapeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	vars := mux.Vars(r)
	scrapeID := vars["scrapeId"]

	if scrapeID != "billboard-hot-100" {
		http.Error(w, "Invalid scrape ID", http.StatusBadRequest)
		return
	}

	songs, err := scrapers.ScrapeBillboardHot100(w)
	if err != nil {
		http.Error(w, "Failed to scrape Billboard Hot 100: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tracks := make([]fs.Track, 20)

	for i, song := range songs {
		if i >= 20 {
			break
		}

		q := buildSpotifyQuery(song.Artist, song.Title)
		results, err := h.sp.Client.Search(ctx, q, spotify.SearchTypeTrack)
		if err != nil {
			http.Error(w, "search error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if results.Tracks == nil || len(results.Tracks.Tracks) == 0 {
			log.Printf("No tracks found for query: %s", q)
			q = song.Artist + " " + song.Title
			results, err = h.sp.Client.Search(ctx, q, spotify.SearchTypeTrack)
			if err != nil {
				http.Error(w, "search error: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if results.Tracks != nil && len(results.Tracks.Tracks) > 0 {
			track := results.Tracks.Tracks[0]
			tracks[i] = fs.Track{
				SpotifyID: track.ID.String(),
				Title:     track.Name,
				Artist:    track.Artists[0].Name,
			}
			log.Printf("Added track: %s by %s", track.Name, track.Artists[0].Name)
		} else {
			log.Printf("No tracks found for raw query: %s", q)
		}

		time.Sleep(100 * time.Millisecond)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

func buildSpotifyQuery(artist, title string) string {
	var q strings.Builder
	q.WriteString("artist:")
	q.WriteString(artist)
	q.WriteString(" ")
	q.WriteString("track:")
	q.WriteString(title)
	return q.String()
}
