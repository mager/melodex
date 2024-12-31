package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"

	"melodex/scrapers"
	spot "melodex/spotify"
)

type WhoSampledHandler struct {
	db *firestore.Client
	sp *spot.SpotifyClient
}

func NewWhoSampledHandler(
	db *firestore.Client,
	sp *spot.SpotifyClient,
) *WhoSampledHandler {
	return &WhoSampledHandler{
		db: db,
		sp: sp,
	}
}

type WhoSampledHandlerReq struct {
	Query string `json:"query"`
}

type WhoSampledHandlerResp struct {
}

func (h *WhoSampledHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Parse request body
	var req WhoSampledHandlerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("Error decoding request body: %v", err)
		return
	}
	q := req.Query
	log.Printf("Query: %v", q)

	_, err := scrapers.ScrapeWhoSampled(w, q)
	if err != nil {
		http.Error(w, "Failed to scrape Billboard Hot 100: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Scraping failed: %v", err)
		return
	}
}
