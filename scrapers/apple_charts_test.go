package scrapers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAppleChartFeed(t *testing.T) {
	songs, err := parseAppleChart(strings.NewReader(`{"feed":{"results":[{"id":"1","name":"A song","artistName":"An artist","kind":"songs"},{"id":"1","name":"A song","artistName":"An artist","kind":"songs"},{"id":"2","name":"","artistName":"X","kind":"songs"}]}}`))
	if err != nil || len(songs) != 1 || songs[0].Rank != 1 || songs[0].Title != "A song" {
		t.Fatalf("bad feed: %+v %v", songs, err)
	}
	for _, body := range []string{`{}`, `oops`, `{"feed":{"results":[]}}`} {
		if _, err := parseAppleChart(strings.NewReader(body)); err == nil {
			t.Fatal("invalid feed accepted")
		}
	}
}

type appleTransport func(*http.Request) (*http.Response, error)

func (f appleTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAppleChartRetriesTransientFailures(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: appleTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, context.DeadlineExceeded
		}
		status := 200
		body := `{"feed":{"results":[{"id":"1","name":"A song","artistName":"An artist","kind":"songs"}]}}`
		if calls == 2 {
			status = 503
			body = "unavailable"
		}
		return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	songs, err := fetchAppleChart(context.Background(), client)
	if err != nil || len(songs) != 1 || songs[0].Title != "A song" || calls != 3 {
		t.Fatalf("retry failed: songs=%v calls=%d err=%v", songs, calls, err)
	}
}
func TestAppleChartRetriesAreBoundedAndCancel(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: appleTransport(func(r *http.Request) (*http.Response, error) { calls++; return nil, context.DeadlineExceeded })}
	if _, err := fetchAppleChart(context.Background(), client); err == nil || calls != 3 {
		t.Fatalf("retry budget not bounded: calls=%d err=%v", calls, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := fetchAppleChart(ctx, client); err == nil || calls != 3 {
		t.Fatal("canceled request reached provider")
	}
}
func TestAppleChartDoesNotRetryPermanentFailure(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: appleTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 404, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("missing"))}, nil
	})}
	if _, err := fetchAppleChart(context.Background(), client); err == nil || calls != 1 {
		t.Fatal("permanent failure retried")
	}
}

type interruptedAppleBody struct{}

func (interruptedAppleBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (interruptedAppleBody) Close() error             { return nil }
func TestAppleChartRetriesInterruptedBody(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: appleTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var body io.ReadCloser = interruptedAppleBody{}
		if calls > 1 {
			body = io.NopCloser(strings.NewReader(`{"feed":{"results":[{"id":"1","name":"A song","artistName":"An artist","kind":"songs"}]}}`))
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: body}, nil
	})}
	songs, err := fetchAppleChart(context.Background(), client)
	if err != nil || len(songs) != 1 || calls != 2 {
		t.Fatalf("interrupted body did not recover: calls=%d err=%v", calls, err)
	}
}
