package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	fs "melodex/firestore"
	"melodex/scrapers"

	spotify "github.com/zmb3/spotify/v2"
)

func (h *ScrapeHandler) HandleHypeMachine(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	today := time.Now().Format("2006-01-02")
	log.Printf("Checking if document for today (%s) exists", today)

	// Check if the document for today already exists
	doc, err := h.db.Collection("hypem").Doc(today).Get(ctx)
	if err == nil && doc.Exists() {
		// Document exists, exit early
		http.Error(w, "Data for today already exists", http.StatusConflict)
		log.Printf("Data for today (%s) already exists", today)
		return
	} else if err != nil {
		log.Printf("Error checking document existence: %v", err)
	}

	log.Printf("Scraping Hype Machine")
	songs, err := scrapers.ScrapeHypeMachine(w)
	if err != nil {
		http.Error(w, "Failed to scrape Hype Machine: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Scraping failed: %v", err)
		return
	}

	tracks := make([]fs.Track, 0, len(songs))
	for _, song := range songs {
		// Fetch metadata from Spotify
		q := buildSpotifyQuery(song.Artist, song.Title)
		results, err := h.sp.Client.Search(ctx, q, spotify.SearchTypeTrack)
		if err != nil || results.Tracks == nil || len(results.Tracks.Tracks) == 0 {
			log.Printf("Spotify search failed for query: %s", q)
			continue
		}

		track := results.Tracks.Tracks[0]
		isrc := track.ExternalIDs["isrc"]
		if isrc == "" {
			log.Printf("No ISRC found for track: %s by %s", song.Title, song.Artist)
			continue
		}

		// Find MBID using the helper function
		mbid := h.FindMBID(isrc, song.Artist, song.Title)
		if mbid == "" {
			log.Printf("No MBID found for track: %s by %s", song.Title, song.Artist)
			continue
		}

		newTrack := fs.Track{
			Rank:      song.Rank,
			Artist:    song.Artist,
			Title:     song.Title,
			ISRC:      isrc,
			SpotifyID: track.ID.String(),
			MBID:      mbid,
		}

		// Find and save thumbnail
		for _, image := range track.Album.Images {
			if image.Height == 300 && image.Width == 300 {
				newTrack.Thumb = strings.TrimPrefix(image.URL, "https://i.scdn.co/image/")
				break
			}
		}

		tracks = append(tracks, newTrack)
		log.Printf("Added new track: %s by %s", track.Name, track.Artists[0].Name)
		time.Sleep(3 * time.Second) // Rate limit
	}

	// Save today's data to Firestore
	_, err = h.db.Collection("hypem").Doc(today).Set(ctx, map[string]interface{}{
		"tracks": tracks,
	})
	if err != nil {
		http.Error(w, "Failed to update Firestore: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Failed to update Firestore: %v", err)
		return
	}

	log.Printf("Successfully created document for today (%s)", today)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}
