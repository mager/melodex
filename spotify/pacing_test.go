package spotify

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPacingIsSharedAcrossConcurrentRequests(t *testing.T) {
	var mu sync.Mutex
	var starts []time.Time
	transport := newPacedTransport(tokenTransport(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		starts = append(starts, time.Now())
		mu.Unlock()
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: http.Header{}}, nil
	}), 20*time.Millisecond)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, _ := http.NewRequest("GET", "https://api.spotify.com/test", nil)
			resp, err := transport.RoundTrip(r)
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
		}()
	}
	wg.Wait()
	for i := 1; i < len(starts); i++ {
		if starts[i].Sub(starts[i-1]) < 19*time.Millisecond {
			t.Fatal("concurrent sources bypassed pacing")
		}
	}
}

func TestRetryAfterPausesAllSources(t *testing.T) {
	calls := 0
	transport := newPacedTransport(tokenTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": []string{"60"}}, Body: io.NopCloser(strings.NewReader("limited"))}, nil
	}), time.Millisecond)
	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		r, _ := http.NewRequestWithContext(ctx, "GET", "https://api.spotify.com/test", nil)
		_, err := transport.RoundTrip(r)
		cancel()
		if err == nil {
			t.Fatal("rate-limited request ignored deadline")
		}
	}
	if calls != 1 {
		t.Fatalf("another source ignored shared cooldown: %d calls", calls)
	}
}

func TestRateLimitRetriesAreBounded(t *testing.T) {
	calls := 0
	transport := newPacedTransport(tokenTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": []string{"0"}}, Body: io.NopCloser(strings.NewReader("limited"))}, nil
	}), time.Millisecond)
	r, _ := http.NewRequest("GET", "https://api.spotify.com/test", nil)
	resp, err := transport.RoundTrip(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if calls != 3 || resp.StatusCode != 429 {
		t.Fatalf("retry budget: calls=%d status=%d", calls, resp.StatusCode)
	}
}

func TestRateLimitRecoveryAndHTTPDate(t *testing.T) {
	calls := 0
	transport := newPacedTransport(tokenTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		status := 200
		if calls == 1 {
			status = 429
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": []string{time.Now().Add(-time.Second).UTC().Format(http.TimeFormat)}}, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	}), time.Millisecond)
	r, _ := http.NewRequest("GET", "https://api.spotify.com/test", nil)
	resp, err := transport.RoundTrip(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if calls != 2 || resp.StatusCode != 200 {
		t.Fatal("did not recover after rate limit")
	}
}
