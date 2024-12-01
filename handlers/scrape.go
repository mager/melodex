package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"melodex/scrapers"
)

func HandleScrape(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scrapeID := vars["scrapeId"]

	if scrapeID != "billboard-hot-100" {
		http.Error(w, "Invalid scrape ID", http.StatusBadRequest)
		return
	}

	songs, err := scrapers.ScrapeBillboardHot100(w)
	if err != nil {
		http.Error(w, "Failed to scrape Billboard Hot 100: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(songs)

}
