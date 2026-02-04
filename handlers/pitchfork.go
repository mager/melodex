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

func (h *ScrapeHandler) HandlePitchfork(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	debugMode := r.URL.Query().Get("debug") == "true"

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	log.Printf("Checking if document for today (%s) exists in pitchfork_bnm", today)

	// Skip DB check in debug mode
	if !debugMode {
		doc, err := h.db.Collection("pitchfork_bnm").Doc(today).Get(ctx)
		if err == nil && doc.Exists() {
			http.Error(w, "Data for today already exists", http.StatusConflict)
			log.Printf("Data for today (%s) already exists in pitchfork_bnm", today)
			return
		} else if err != nil {
			log.Printf("Error checking today's document existence: %v", err)
		}
	} else {
		log.Printf("Debug mode: Skipping database existence check")
	}

	// Fetch yesterday's data for reuse
	yesterdayData := make(map[string]fs.Track)
	if !debugMode {
		doc, err := h.db.Collection("pitchfork_bnm").Doc(yesterday).Get(ctx)
		if err == nil && doc.Exists() {
			var yesterdayTracks struct {
				Tracks []fs.Track `json:"tracks"`
			}
			if err := doc.DataTo(&yesterdayTracks); err == nil {
				for _, track := range yesterdayTracks.Tracks {
					key := track.Artist + " - " + track.Title
					yesterdayData[key] = track
				}
				log.Printf("Loaded %d tracks from yesterday's pitchfork_bnm data", len(yesterdayData))
			}
		} else if err != nil {
			log.Printf("No pitchfork_bnm data found for yesterday (%s): %v", yesterday, err)
		}
	}

	log.Printf("Scraping Pitchfork Best New Tracks")
	songs, err := scrapers.ScrapePitchforkBestNewTracks(w)
	if err != nil {
		http.Error(w, "Failed to scrape Pitchfork: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Pitchfork scraping failed: %v", err)
		return
	}

	tracks := make([]fs.Track, 0, len(songs))
	for _, song := range songs {
		key := song.Artist + " - " + song.Title
		if existingTrack, found := yesterdayData[key]; found {
			tracks = append(tracks, existingTrack)
			log.Printf("Reused metadata for Pitchfork track: %s by %s", song.Title, song.Artist)
			continue
		}

		// Fetch metadata from Spotify
		q := buildSpotifyQuery(song.Artist, song.Title)
		results, err := h.sp.Client.Search(ctx, q, spotify.SearchTypeTrack)
		if err != nil || results.Tracks == nil || len(results.Tracks.Tracks) == 0 {
			log.Printf("Spotify search failed for query: %s", q)
			tracks = append(tracks, fs.Track{
				Rank:      song.Rank,
				Artist:    song.Artist,
				Title:     song.Title,
				Source:    "pitchfork_bnm",
				CreatedAt: time.Now(),
			})
			continue
		}

		track := results.Tracks.Tracks[0]
		isrc := track.ExternalIDs["isrc"]
		if isrc == "" {
			log.Printf("No ISRC found for Pitchfork track: %s by %s", song.Title, song.Artist)
			continue
		}

		mbid := h.FindMBID(isrc, song.Artist, song.Title)
		if mbid == "" {
			log.Printf("No MBID found for Pitchfork track: %s by %s", song.Title, song.Artist)
		}

		newTrack := fs.Track{
			Rank:      song.Rank,
			Artist:    song.Artist,
			Title:     song.Title,
			ISRC:      isrc,
			SpotifyID: track.ID.String(),
			MBID:      mbid,
			Source:    "pitchfork_bnm",
			CreatedAt: time.Now(),
		}

		for _, image := range track.Album.Images {
			if image.Height == 300 && image.Width == 300 {
				newTrack.Thumb = strings.TrimPrefix(image.URL, "https://i.scdn.co/image/")
				break
			}
		}

		tracks = append(tracks, newTrack)
		log.Printf("Added Pitchfork track: %s by %s", track.Name, track.Artists[0].Name)
		time.Sleep(3 * time.Second) // Rate limit
	}

	// Save today's data to Firestore
	if !debugMode {
		_, err = h.db.Collection("pitchfork_bnm").Doc(today).Set(ctx, map[string]interface{}{
			"tracks": tracks,
		})
		if err != nil {
			http.Error(w, "Failed to update Firestore: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Failed to update Firestore: %v", err)
			return
		}
		log.Printf("Successfully created pitchfork_bnm document for today (%s)", today)
	} else {
		log.Printf("Debug mode: Skipping database save")
	}

	json.NewEncoder(w).Encode(tracks)
}
