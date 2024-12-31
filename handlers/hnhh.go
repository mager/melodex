package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"melodex/scrapers"
)

func (h *ScrapeHandler) HandleHotNewHipHop(w http.ResponseWriter, r *http.Request) {
	log.Printf("Scraping Hot New Hip Hop")
	songs, err := scrapers.ScrapeHotNewHipHop(w)
	if err != nil {
		http.Error(w, "Failed to scrape Billboard Hot 100: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Scraping failed: %v", err)
		return
	}
	json.NewEncoder(w).Encode(songs)
}
