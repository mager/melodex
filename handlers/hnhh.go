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

func (h *ScrapeHandler) HandleHotNewHipHop(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	log.Printf("Checking if document for today (%s) exists", today)

	// Check if today's document exists
	doc, err := h.db.Collection("hnhh").Doc(today).Get(ctx)
	if err == nil && doc.Exists() {
		http.Error(w, "Data for today already exists", http.StatusConflict)
		log.Printf("Data for today (%s) already exists", today)
		return
	} else if err != nil {
		log.Printf("Error checking today's document existence: %v", err)
	}

	// Fetch yesterday's data into a map
	yesterdayData := make(map[string]fs.Track)
	doc, err = h.db.Collection("hnhh").Doc(yesterday).Get(ctx)
	if err == nil && doc.Exists() {
		var yesterdayTracks struct {
			Tracks []fs.Track `json:"tracks"`
		}
		if err := doc.DataTo(&yesterdayTracks); err == nil {
			for _, track := range yesterdayTracks.Tracks {
				key := track.Artist + " - " + track.Title
				yesterdayData[key] = track
			}
			log.Printf("Loaded %d tracks from yesterday's data", len(yesterdayData))
		}
	} else if err != nil {
		log.Printf("No data found for yesterday (%s): %v", yesterday, err)
	}

	log.Printf("Scraping Hot New Hip Hop")
	songs, err := scrapers.ScrapeHotNewHipHop(w)
	if err != nil {
		http.Error(w, "Failed to scrape Billboard Hot 100: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Scraping failed: %v", err)
		return
	}

	tracks := make([]fs.Track, 0, len(songs))
	for _, song := range songs {
		key := song.Artist + " - " + song.Title
		if existingTrack, found := yesterdayData[key]; found {
			// Reuse yesterday's track metadata
			tracks = append(tracks, existingTrack)
			log.Printf("Reused metadata for track: %s by %s", song.Title, song.Artist)
			continue
		}

		// Fetch metadata from Spotify
		q := buildSpotifyQuery(song.Artist, song.Title)
		results, err := h.sp.Client.Search(ctx, q, spotify.SearchTypeTrack)
		if err != nil || results.Tracks == nil || len(results.Tracks.Tracks) == 0 {
			log.Printf("Spotify search failed or no results for query: %s", q)
			continue // Skip adding this track
		}

		track := results.Tracks.Tracks[0]
		newTrack := fs.Track{
			Rank:      song.Rank,
			Artist:    song.Artist,
			Title:     song.Title,
			SpotifyID: track.ID.String(),
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
		time.Sleep(250 * time.Millisecond) // Rate limit
	}

	// Save today's data to Firestore
	_, err = h.db.Collection("hnhh").Doc(today).Set(ctx, map[string]interface{}{
		"tracks": tracks,
	})
	if err != nil {
		http.Error(w, "Failed to update Firestore: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Failed to update Firestore: %v", err)
		return
	}

	log.Printf("Successfully created document for today (%s)", today)
	json.NewEncoder(w).Encode(tracks)
}
