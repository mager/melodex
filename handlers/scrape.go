package handlers

import (
	"log"
	"net/http"
	"slices"

	"cloud.google.com/go/firestore"

	"github.com/gorilla/mux"

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

func (h *ScrapeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	scrapeID := vars["scrapeId"]

	log.Printf("Received scrape ID: %s", scrapeID)

	if !slices.Contains([]string{"billboard-hot-100", "hype-machine"}, scrapeID) {
		http.Error(w, "Invalid scrape ID", http.StatusBadRequest)
		log.Printf("Invalid scrape ID: %s", scrapeID)
		return
	}

	switch scrapeID {
	case "billboard-hot-100":
		h.HandleBillboard(w, r)
		break
	case "hype-machine":
		h.HandleHypeMachine(w, r)

	}
}
