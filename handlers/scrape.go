package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync"

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
		writeScrapeError(w, http.StatusBadRequest, "invalid request body")
		log.Printf("Error decoding request body: %v", err)
		return
	}
	target := req.Target

	// Handle default case where no target is provided — run ALL scrapers concurrently
	if target == "" {
		runScrapeSources(w, r, []scrapeSource{
			{"billboard-hot-100", h.HandleBillboard},
			{"hot-new-hip-hop", h.HandleHotNewHipHop},
			{"spotify-new-releases", h.HandleSpotifyNewReleases},
			{"reddit-fresh", h.HandleReddit},
			{"pitchfork-bnm", h.HandlePitchfork},
			{"spotify-podcasts", h.HandlePodcasts},
		})

		// Run Firestore TTL cleanup after all scrapes complete
		ctx := context.Background()
		if err := fs.RunCleanup(ctx, h.db); err != nil {
			log.Printf("Error during TTL cleanup: %v", err)
		}

		return
	}

	// Valid targets
	validTargets := []string{
		"billboard-hot-100",
		"hot-new-hip-hop",
		"spotify-new-releases",
		"reddit-fresh",
		"pitchfork-bnm",
		"spotify-podcasts",
	}

	if !slices.Contains(validTargets, target) {
		writeScrapeError(w, http.StatusBadRequest, "invalid target")
		log.Printf("Invalid target: %s", target)
		return
	}

	// Scrape based on target
	switch target {
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
	case "spotify-podcasts":
		h.HandlePodcasts(w, r)
	default:
		log.Printf("Unhandled target: %s", target)
	}
}

type scrapeSource struct {
	name   string
	handle http.HandlerFunc
}

// sourceResponse records only the status. Source payloads can be large, and
// the scheduler needs a status summary rather than every scraped record.
type sourceResponse struct {
	header http.Header
	status int
}

func (w *sourceResponse) Header() http.Header { return w.header }
func (w *sourceResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *sourceResponse) Write(data []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	return len(data), nil
}

type sourceResult struct {
	Source  string `json:"source"`
	Status  int    `json:"status"`
	Outcome string `json:"outcome"`
}

func runScrapeSources(w http.ResponseWriter, r *http.Request, sources []scrapeSource) {
	results := make([]sourceResult, len(sources))
	var workers sync.WaitGroup
	for index, source := range sources {
		workers.Add(1)
		go func() {
			defer workers.Done()
			response := &sourceResponse{header: make(http.Header)}
			request := r.Clone(r.Context())
			// The top-level decoder consumed the body. Each source gets its own
			// default options, including the podcast handler that reads a JSON body.
			request.Body = io.NopCloser(strings.NewReader("{}"))
			request.ContentLength = 2
			defer request.Body.Close()
			source.handle(response, request)
			if response.status == 0 {
				response.status = http.StatusOK
			}
			outcome := "success"
			if response.status == http.StatusConflict {
				outcome = "skipped"
			} else if response.status < 200 || response.status >= 300 {
				outcome = "failed"
			}
			results[index] = sourceResult{Source: source.name, Status: response.status, Outcome: outcome}
		}()
	}
	workers.Wait()
	response := struct {
		Results []sourceResult `json:"results"`
		Error   string         `json:"error,omitempty"`
	}{Results: results}
	status := http.StatusOK
	for _, result := range results {
		if result.Outcome == "failed" {
			status = http.StatusBadGateway
			response.Error = "one or more sources failed"
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeScrapeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
