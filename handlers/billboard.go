package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	spotify "github.com/zmb3/spotify/v2"

	fs "melodex/firestore"
	"melodex/scrapers"
)

func (h *ScrapeHandler) HandleBillboard(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	debugMode := r.URL.Query().Get("debug") == "true"

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	log.Printf("Checking if document for today (%s) exists", today)

	// Skip DB check in debug mode
	if !debugMode {
		// Check if today's document exists
		doc, err := h.db.Collection("billboard").Doc(today).Get(ctx)
		if err == nil && doc.Exists() {
			http.Error(w, "Data for today already exists", http.StatusConflict)
			log.Printf("Data for today (%s) already exists", today)
			return
		} else if err != nil {
			log.Printf("Error checking today's document existence: %v", err)
		}
	} else {
		log.Printf("Debug mode: Skipping database existence check")
	}

	// Fetch yesterday's data into a map
	yesterdayData := make(map[string]fs.Track)
	if !debugMode {
		doc, err := h.db.Collection("billboard").Doc(yesterday).Get(ctx)
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
	} else {
		log.Printf("Debug mode: Skipping yesterday's data fetch")
	}

	// Scrape today's Billboard data
	log.Printf("Scraping Billboard Hot 100")
	songs, err := scrapers.ScrapeBillboardHot100(w)
	if err != nil {
		http.Error(w, "Failed to scrape Billboard Hot 100: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Scraping failed: %v", err)
		return
	}

	tracks := make([]fs.Track, 0, len(songs))
	for i, song := range songs {
		if i >= 1 {
			break
		}

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
			log.Printf("Spotify search failed for query: %s", q)
			tracks = append(tracks, fs.Track{Rank: song.Rank, Artist: song.Artist, Title: song.Title})
			continue
		}

		track := results.Tracks.Tracks[0]
		isrc := track.ExternalIDs["isrc"]
		if isrc == "" {
			log.Printf("No ISRC found for track: %s by %s", song.Title, song.Artist)
			continue
		}
		newTrack := fs.Track{
			Rank:   song.Rank,
			Artist: song.Artist,
			Title:  song.Title,
			// TODO: Deprecate this field
			ISRC:      isrc,
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
	if !debugMode {
		_, err = h.db.Collection("billboard").Doc(today).Set(ctx, map[string]interface{}{
			"tracks": tracks,
		})
		if err != nil {
			http.Error(w, "Failed to update Firestore: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Failed to update Firestore: %v", err)
			return
		}
		log.Printf("Successfully created document for today (%s)", today)
	} else {
		log.Printf("Debug mode: Skipping database save")
	}

	json.NewEncoder(w).Encode(tracks)
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
