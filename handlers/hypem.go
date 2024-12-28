package handlers

import (
	"encoding/json"
	"log"
	"melodex/scrapers"
	"net/http"
)

func (h *ScrapeHandler) HandleHypeMachine(w http.ResponseWriter, r *http.Request) {
	log.Printf("Scraping Hype Machine")

	log.Printf("Scraping Billboard Hot 100")
	tracks, err := scrapers.ScrapeHypeMachine(w)
	if err != nil {
		http.Error(w, "Failed to scrape Billboard Hot 100: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Scraping failed: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}
