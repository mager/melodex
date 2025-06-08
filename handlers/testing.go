package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	fs "melodex/firestore"
	"melodex/scrapers"

	mb "github.com/mager/musicbrainz-go/musicbrainz"

	spotify "github.com/zmb3/spotify/v2"
)

func (h *ScrapeHandler) HandleTesting(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	today := time.Now().Format("2006-01-02")

	// Scrape today's Billboard data
	log.Printf("Scraping Billboard Hot 100")
	songs, err := scrapers.ScrapeBillboardHot100(w)
	if err != nil {
		http.Error(w, "Failed to scrape Billboard Hot 100: "+err.Error(), http.StatusInternalServerError)
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
		t := fs.Track{
			Rank:   song.Rank,
			Artist: song.Artist,
			Title:  song.Title,
		}

		// Find MBID first by ISRC, then by title and artist
		if isrc != "" {
			searchRecsReq := mb.SearchRecordingsByISRCRequest{
				ISRC: isrc,
			}
			recs, err := h.mb.Client.SearchRecordingsByISRC(searchRecsReq)
			if err != nil {
				log.Printf("Error getting MBID by ISRC: %v", err)
			}
			if len(recs.Recordings) > 0 {
				t.MBID = recs.Recordings[0].ID
			}
		} else {
			searchRecsReq := mb.SearchRecordingsByArtistAndTrackRequest{
				Artist: song.Artist,
				Track:  song.Title,
			}
			recs, err := h.mb.Client.SearchRecordingsByArtistAndTrack(searchRecsReq)
			if err != nil {
				log.Printf("Error getting MBID by title and artist: %v", err)
			}
			if len(recs.Recordings) > 0 {
				t.MBID = recs.Recordings[0].ID
			} else {
				log.Printf("No MBID found for track: %s by %s", song.Title, song.Artist)
			}
		}

		if t.MBID == "" {
			log.Printf("No MBID found for track: %s by %s", song.Title, song.Artist)
			continue
		}

		tracks = append(tracks, t)
		log.Printf("Added new track: %s by %s", song.Title, song.Artist)
		time.Sleep(250 * time.Millisecond) // Rate limit
	}

	// Save today's data to Firestore
	_, err = h.db.Collection("testing").Doc(today).Set(ctx, map[string]interface{}{
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
