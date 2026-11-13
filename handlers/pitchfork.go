package handlers

import (
	"melodex/scrapers"
	"net/http"
)

func (h *ScrapeHandler) HandlePitchfork(w http.ResponseWriter, r *http.Request) {
	h.handleMusicChart(w, r, "pitchfork_bnm", scrapers.ScrapePitchforkBestNewTracks, false)
}
