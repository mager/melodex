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

	today := time.Now().Format("2006-01-02")
	log.Printf("Checking if document for today (%s) exists", today)

	// Check if the document for today already exists
	doc, err := h.db.Collection("billboard").Doc(today).Get(ctx)
	if err == nil && doc.Exists() {
		// Document exists, exit early
		http.Error(w, "Data for today already exists", http.StatusConflict)
		log.Printf("Data for today (%s) already exists", today)
		return
	} else if err != nil {
		log.Printf("Error checking document existence: %v", err)
	}

	log.Printf("Scraping Billboard Hot 100")
	songs, err := scrapers.ScrapeBillboardHot100(w)
	if err != nil {
		http.Error(w, "Failed to scrape Billboard Hot 100: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Scraping failed: %v", err)
		return
	}

	tracks := make([]fs.Track, 0, 100)
	// Add the songs to the tracks
	for _, song := range songs {
		tracks = append(tracks, fs.Track{
			Rank:   song.Rank,
			Artist: song.Artist,
			Title:  song.Title,
		})
	}

	for i, track := range tracks {
		q := buildSpotifyQuery(track.Artist, track.Title)
		results, err := h.sp.Client.Search(ctx, q, spotify.SearchTypeTrack)
		if err != nil {
			http.Error(w, "search error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if results.Tracks == nil || len(results.Tracks.Tracks) == 0 {
			log.Printf("No tracks found for query: %s", q)
			q = track.Artist + " " + track.Title
			results, err = h.sp.Client.Search(ctx, q, spotify.SearchTypeTrack)
			if err != nil {
				http.Error(w, "search error: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if results.Tracks != nil && len(results.Tracks.Tracks) > 0 {
			track := results.Tracks.Tracks[0]
			tracks[i].SpotifyID = track.ID.String()

			for _, image := range track.Album.Images {
				if image.Height == 300 && image.Width == 300 {
					tracks[i].Thumb = strings.TrimPrefix(image.URL, "https://i.scdn.co/image/")
					break
				}
			}

			log.Printf("Added track: %s by %s", track.Name, track.Artists[0].Name)
		} else {
			log.Printf("No tracks found for raw query: %s", q)
		}

		time.Sleep(250 * time.Millisecond)
	}

	_, err = h.db.Collection("billboard").Doc(today).Set(ctx, map[string]interface{}{
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

func buildSpotifyQuery(artist, title string) string {
	var q strings.Builder
	q.WriteString("artist:")
	q.WriteString(artist)
	q.WriteString(" ")
	q.WriteString("track:")
	q.WriteString(title)
	return q.String()
}
