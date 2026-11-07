package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestScrapeRejectsRemovedTestingTarget(t *testing.T) {
	h := NewScrapeHandler(nil, nil, nil)
	w := httptest.NewRecorder()
	// debug avoids existing data access until the retired target is rejected.
	h.Handle(w, httptest.NewRequest("POST", "/scrape?debug=true", strings.NewReader(`{"target":"testing"}`)))
	if w.Code != 400 || !json.Valid(w.Body.Bytes()) {
		t.Fatalf("response = %d %s", w.Code, w.Body.String())
	}
}

func TestAllSourcesHaveIndependentResponsesAndOneSummary(t *testing.T) {
	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(3)
	sources := []scrapeSource{
		{"good", func(w http.ResponseWriter, r *http.Request) {
			ready.Done()
			<-start
			w.Header().Set("X-Source", "good")
			io.WriteString(w, `[{"track":"first"}]`)
		}},
		{"current", func(w http.ResponseWriter, r *http.Request) {
			ready.Done()
			<-start
			http.Error(w, "already current", 409)
		}},
		{"failed", func(w http.ResponseWriter, r *http.Request) {
			ready.Done()
			<-start
			http.Error(w, "private provider details", 500)
		}},
	}
	go func() { ready.Wait(); close(start) }()
	w := httptest.NewRecorder()
	runScrapeSources(w, httptest.NewRequest("POST", "/scrape", strings.NewReader(`{}`)), sources)
	if w.Code != 502 {
		t.Errorf("status = %d, want 502", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" || w.Header().Get("X-Source") != "" {
		t.Errorf("source headers leaked into summary: %v", w.Header())
	}
	if strings.Contains(w.Body.String(), "private provider") || strings.Contains(w.Body.String(), "first") {
		t.Errorf("source bodies leaked into summary")
	}
	var response struct {
		Results []struct {
			Source  string `json:"source"`
			Status  int    `json:"status"`
			Outcome string `json:"outcome"`
		} `json:"results"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("response is not one JSON object: %s", w.Body.String())
	}
	if len(response.Results) != 3 || response.Error == "" {
		t.Fatalf("incomplete summary: %+v", response)
	}
	for i, want := range []struct {
		name    string
		status  int
		outcome string
	}{{"good", 200, "success"}, {"current", 409, "skipped"}, {"failed", 500, "failed"}} {
		got := response.Results[i]
		if got.Source != want.name || got.Status != want.status || got.Outcome != want.outcome {
			t.Errorf("result %d = %+v", i, got)
		}
	}
}

func TestAllSourcesReceiveIndependentBodiesAndRequestContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sources := make([]scrapeSource, 2)
	for i := range sources {
		sources[i] = scrapeSource{fmt.Sprint(i), func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "{}" {
				t.Errorf("source request body = %q, error %v", body, err)
			}
			if r.Context().Err() != context.Canceled {
				t.Error("source lost request cancellation")
			}
			io.WriteString(w, `[]`)
		}}
	}
	r := httptest.NewRequest("POST", "/scrape", http.NoBody).WithContext(ctx)
	w := httptest.NewRecorder()
	runScrapeSources(w, r, sources)
	if w.Code != 200 || !json.Valid(w.Body.Bytes()) {
		t.Errorf("successful sources returned %d %s", w.Code, w.Body.String())
	}
}

func TestCurrentSourcesDoNotFailScheduledRun(t *testing.T) {
	sources := []scrapeSource{{"current", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "data already exists", http.StatusConflict)
	}}}
	w := httptest.NewRecorder()
	runScrapeSources(w, httptest.NewRequest("POST", "/scrape", http.NoBody), sources)
	if w.Code != 200 || !json.Valid(w.Body.Bytes()) || !strings.Contains(w.Body.String(), `"outcome":"skipped"`) {
		t.Fatalf("response = %d %s", w.Code, w.Body.String())
	}
}
