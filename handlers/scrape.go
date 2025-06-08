package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"

	"cloud.google.com/go/firestore"

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

	// Handle default case where no target is provided
	if target == "" {
		errChan := make(chan error, 3)

		go func() {
			h.HandleBillboard(w, r)
			errChan <- nil
		}()

		go func() {
			h.HandleHypeMachine(w, r)
			errChan <- nil
		}()

		go func() {
			h.HandleHotNewHipHop(w, r)
			errChan <- nil
		}()

		// Collect results from both goroutines
		for i := 0; i < 3; i++ {
			if err := <-errChan; err != nil {
				log.Printf("Error in scraping: %v", err)
			}
		}

		close(errChan)

		return
	}

	// Validate target
	if !slices.Contains([]string{"testing", "billboard-hot-100", "hype-machine", "hot-new-hip-hop"}, target) {
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
	case "hype-machine":
		h.HandleHypeMachine(w, r)
	case "hot-new-hip-hop":
		h.HandleHotNewHipHop(w, r)
	default:
		log.Printf("Unhandled target: %s", target)
	}
}
