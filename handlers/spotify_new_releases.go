package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"melodex/scrapers"
)

func (h *ScrapeHandler) HandleSpotifyNewReleases(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	debugMode := r.URL.Query().Get("debug") == "true"

	today := time.Now().UTC().Format("2006-01-02")
	log.Printf("Checking if document for today (%s) exists in spotify_new_releases", today)

	// Skip DB check in debug mode
	if !debugMode {
		_, skip, err := h.musicCache(ctx, "spotify_new_releases", time.Now().UTC(), r.URL.Query().Get("refresh") == "true")
		if err != nil {
			writeScrapeError(w, 500, "source cache unavailable")
			return
		}
		if skip {
			writeScrapeError(w, 409, "data for today already exists")
			return
		}
	} else {
		log.Printf("Debug mode: Skipping database existence check")
	}

	// Scrape Spotify new releases
	log.Printf("Fetching Spotify new releases")
	spotifyTracks, err := scrapers.ScrapeSpotifyNewReleases(ctx, w, h.sp)
	if err == nil {
		err = requireTracks(spotifyTracks)
	}
	if err != nil {
		writeScrapeError(w, 502, "new releases unavailable")
		log.Printf("Spotify new releases fetch failed: %v", err)
		return
	}

	tracks := spotifyTracks

	// Save today's data to Firestore
	if !debugMode {
		err = h.store.save(ctx, "spotify_new_releases", today, tracks)
		if err != nil {
			writeScrapeError(w, 500, "source save failed")
			log.Printf("Failed to update Firestore: %v", err)
			return
		}
		log.Printf("Successfully created spotify_new_releases document for today (%s)", today)
	} else {
		log.Printf("Debug mode: Skipping database save")
	}

	json.NewEncoder(w).Encode(tracks)
}
