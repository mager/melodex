package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"slices"

	"cloud.google.com/go/firestore"

	fs "melodex/firestore"
	mb "melodex/musicbrainz"
	spot "melodex/spotify"
)

type ScrapeHandler struct {
	db *firestore.Client
	sp *spot.SpotifyClient
	mb *mb.MusicbrainzClient
}

func NewScrapeHandler(
	db *firestore.Client,
	sp *spot.SpotifyClient,
	mb *mb.MusicbrainzClient,
) *ScrapeHandler {
	return &ScrapeHandler{
		db: db,
		sp: sp,
		mb: mb,
	}
}

type ScrapeHandlerReq struct {
	Target string `json:"target"`
}

type ScrapeHandlerResp struct {
}

func (h *ScrapeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Parse request body
	var req ScrapeHandlerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("Error decoding request body: %v", err)
		return
	}
	target := req.Target

	// Handle default case where no target is provided — run ALL scrapers concurrently
	if target == "" {
		errChan := make(chan error, 5)

		go func() {
			h.HandleBillboard(w, r)
			errChan <- nil
		}()

		go func() {
			h.HandleHotNewHipHop(w, r)
			errChan <- nil
		}()

		go func() {
			h.HandleSpotifyNewReleases(w, r)
			errChan <- nil
		}()

		go func() {
			h.HandleReddit(w, r)
			errChan <- nil
		}()

		go func() {
			h.HandlePitchfork(w, r)
			errChan <- nil
		}()

		// Collect results from all goroutines
		for i := 0; i < 5; i++ {
			if err := <-errChan; err != nil {
				log.Printf("Error in scraping: %v", err)
			}
		}
		close(errChan)

		// Run Firestore TTL cleanup after all scrapes complete
		ctx := context.Background()
		if err := fs.RunCleanup(ctx, h.db); err != nil {
			log.Printf("Error during TTL cleanup: %v", err)
		}

		return
	}

	// Valid targets
	validTargets := []string{
		"testing",
		"billboard-hot-100",
		"hot-new-hip-hop",
		"spotify-new-releases",
		"reddit-fresh",
		"pitchfork-bnm",
	}

	if !slices.Contains(validTargets, target) {
		http.Error(w, "Invalid target", http.StatusBadRequest)
		log.Printf("Invalid target: %s", target)
		return
	}

	// Scrape based on target
	switch target {
	case "testing":
		h.HandleTesting(w, r)
	case "billboard-hot-100":
		h.HandleBillboard(w, r)
	case "hot-new-hip-hop":
		h.HandleHotNewHipHop(w, r)
	case "spotify-new-releases":
		h.HandleSpotifyNewReleases(w, r)
	case "reddit-fresh":
		h.HandleReddit(w, r)
	case "pitchfork-bnm":
		h.HandlePitchfork(w, r)
	default:
		log.Printf("Unhandled target: %s", target)
	}
}
