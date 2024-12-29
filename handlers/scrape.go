package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"

	"cloud.google.com/go/firestore"

	spot "melodex/spotify"
)

type ScrapeHandler struct {
	db *firestore.Client
	sp *spot.SpotifyClient
}

func NewScrapeHandler(
	db *firestore.Client,
	sp *spot.SpotifyClient,
) *ScrapeHandler {
	return &ScrapeHandler{
		db: db,
		sp: sp,
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
		errChan := make(chan error, 2)

		go func() {
			h.HandleBillboard(w, r)
			errChan <- nil
		}()

		go func() {
			h.HandleHypeMachine(w, r)
			errChan <- nil
		}()

		// Collect results from both goroutines
		for i := 0; i < 2; i++ {
			if err := <-errChan; err != nil {
				log.Printf("Error in scraping: %v", err)
			}
		}

		close(errChan)

		return
	}

	// Validate target
	if !slices.Contains([]string{"billboard-hot-100", "hype-machine"}, target) {
		http.Error(w, "Invalid target", http.StatusBadRequest)
		log.Printf("Invalid target: %s", target)
		return
	}

	// Scrape based on target
	switch target {
	case "billboard-hot-100":
		h.HandleBillboard(w, r)
	case "hype-machine":
		h.HandleHypeMachine(w, r)
	default:
		log.Printf("Unhandled target: %s", target)
	}
}
