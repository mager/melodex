package handlers

import (
	"context"
	"encoding/json"
	"log"
	"melodex/scrapers"
	"net/http"
	"time"
)

func (h *ScrapeHandler) HandleHypeMachine(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	today := time.Now().Format("2006-01-02")
	log.Printf("Checking if document for today (%s) exists", today)

	// Check if the document for today already exists
	doc, err := h.db.Collection("hypem").Doc(today).Get(ctx)
	if err == nil && doc.Exists() {
		// Document exists, exit early
		http.Error(w, "Data for today already exists", http.StatusConflict)
		log.Printf("Data for today (%s) already exists", today)
		return
	} else if err != nil {
		log.Printf("Error checking document existence: %v", err)
	}

	log.Printf("Scraping Hype Machine")

	log.Printf("Scraping Billboard Hot 100")
	tracks, err := scrapers.ScrapeHypeMachine(w)
	if err != nil {
		http.Error(w, "Failed to scrape Billboard Hot 100: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Scraping failed: %v", err)
		return
	}

	_, err = h.db.Collection("hypem").Doc(today).Set(ctx, map[string]interface{}{
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
