package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	fs "melodex/firestore"
	"melodex/scrapers"
)

func (h *ScrapeHandler) HandleSpotifyNewReleases(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	debugMode := r.URL.Query().Get("debug") == "true"

	today := time.Now().Format("2006-01-02")
	log.Printf("Checking if document for today (%s) exists in spotify_new_releases", today)

	// Skip DB check in debug mode
	if !debugMode {
		// Check if today's document exists
		doc, err := h.db.Collection("spotify_new_releases").Doc(today).Get(ctx)
		if err == nil && doc.Exists() {
			http.Error(w, "Data for today already exists", http.StatusConflict)
			log.Printf("Data for today (%s) already exists in spotify_new_releases", today)
			return
		} else if err != nil {
			log.Printf("Error checking today's document existence: %v", err)
		}
	} else {
		log.Printf("Debug mode: Skipping database existence check")
	}

	// Scrape Spotify new releases
	log.Printf("Fetching Spotify new releases")
	spotifyTracks, err := scrapers.ScrapeSpotifyNewReleases(w, h.sp)
	if err != nil {
		http.Error(w, "Failed to fetch Spotify new releases: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Spotify new releases fetch failed: %v", err)
		return
	}

	tracks := make([]fs.Track, 0, len(spotifyTracks))
	for _, track := range spotifyTracks {
		// Since we already have Spotify data, we just need to find MBID
		mbid := h.FindMBID(track.ISRC, track.Artist, track.Title)
		if mbid == "" {
			log.Printf("No MBID found for track: %s by %s", track.Title, track.Artist)
			// Still add the track without MBID
		}

		newTrack := fs.Track{
			Rank:      track.Rank,
			Artist:    track.Artist,
			Title:     track.Title,
			ISRC:      track.ISRC,
			SpotifyID: track.SpotifyID,
			MBID:      mbid,
			Thumb:     track.Thumb,
			Source:    "spotify_new_releases",
			CreatedAt: time.Now(),
		}

		tracks = append(tracks, newTrack)
		log.Printf("Added Spotify new release: %s by %s", track.Title, track.Artist)
		
		// Rate limit MusicBrainz calls
		if mbid != "" {
			time.Sleep(3 * time.Second)
		}
	}

	// Save today's data to Firestore
	if !debugMode {
		_, err = h.db.Collection("spotify_new_releases").Doc(today).Set(ctx, map[string]interface{}{
			"tracks": tracks,
		})
		if err != nil {
			http.Error(w, "Failed to update Firestore: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Failed to update Firestore: %v", err)
			return
		}
		log.Printf("Successfully created spotify_new_releases document for today (%s)", today)
	} else {
		log.Printf("Debug mode: Skipping database save")
	}

	json.NewEncoder(w).Encode(tracks)
}