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
	"time"

	"cloud.google.com/go/firestore"

	fs "melodex/firestore"
	mb "melodex/musicbrainz"
	spot "melodex/spotify"
)

type ScrapeHandler struct {
	db    *firestore.Client
	store musicStore
	sp    *spot.SpotifyClient
	mb    *mb.MusicbrainzClient
}

func NewScrapeHandler(
	db *firestore.Client,
	sp *spot.SpotifyClient,
	mb *mb.MusicbrainzClient,
) *ScrapeHandler {
	return &ScrapeHandler{
		db:    db,
		store: firestoreMusicStore{db},
		sp:    sp,
		mb:    mb,
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
	if target == "" || target == "music" {
		sources := []scrapeSource{
			{"billboard-hot-100", h.HandleBillboard},
			{"hot-new-hip-hop", h.HandleHotNewHipHop},
			{"spotify-new-releases", h.HandleSpotifyNewReleases},
			{"apple-music-charts", h.HandleAppleCharts},
			{"pitchfork-bnm", h.HandlePitchfork},
		}
		if target == "" {
			sources = append(sources, scrapeSource{"spotify-podcasts", h.HandlePodcasts})
		}
		results := runScrapeSources(w, r, sources)
		h.recordSourceHealth(r, results)

		// Run Firestore TTL cleanup after all scrapes complete
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
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
		"apple-music-charts",
		"pitchfork-bnm",
		"spotify-podcasts",
	}

	if !slices.Contains(validTargets, target) {
		writeScrapeError(w, http.StatusBadRequest, "invalid target")
		log.Printf("Invalid target: %s", target)
		return
	}

	recorder := &statusWriter{ResponseWriter: w}
	defer func() { h.recordSourceHealth(r, []sourceResult{makeSourceResult(target, recorder.status)}) }()
	w = recorder
	// Scrape based on target
	switch target {
	case "billboard-hot-100":
		h.HandleBillboard(w, r)
	case "hot-new-hip-hop":
		h.HandleHotNewHipHop(w, r)
	case "spotify-new-releases":
		h.HandleSpotifyNewReleases(w, r)
	case "apple-music-charts":
		h.HandleAppleCharts(w, r)
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

func runScrapeSources(w http.ResponseWriter, r *http.Request, sources []scrapeSource) []sourceResult {
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
			results[index] = makeSourceResult(source.name, response.status)
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
	return results
}

func writeScrapeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Five small records retain the latest daily attempt without an event table.
func (h *ScrapeHandler) recordSourceHealth(r *http.Request, results []sourceResult) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
	defer cancel()
	collections := map[string]string{"billboard-hot-100": "billboard", "hot-new-hip-hop": "hnhh", "spotify-new-releases": "spotify_new_releases", "apple-music-charts": "apple_music_charts", "pitchfork-bnm": "pitchfork_bnm"}
	for _, result := range results {
		data := map[string]interface{}{"source": result.Source, "status": result.Status, "outcome": result.Outcome, "checkedAt": time.Now().UTC()}
		encoded, _ := json.Marshal(data)
		log.Print(string(encoded))
		collection := collections[result.Source]
		if h.db == nil || collection == "" || r.URL.Query().Get("debug") == "true" {
			continue
		}
		if _, err := h.db.Collection("discovery_health").Doc(collection).Set(ctx, data); err != nil {
			log.Printf("source=%s stage=health error=%v", collection, err)
		}
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}
func makeSourceResult(source string, status int) sourceResult {
	if status == 0 {
		status = http.StatusOK
	}
	outcome := "success"
	if status == http.StatusConflict {
		outcome = "skipped"
	} else if status < 200 || status >= 300 {
		outcome = "failed"
	}
	return sourceResult{Source: source, Status: status, Outcome: outcome}
}
