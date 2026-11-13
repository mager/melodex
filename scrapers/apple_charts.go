package scrapers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	fs "melodex/firestore"
	"net/http"
	"strings"
	"time"
)

const AppleChartURL = "https://rss.marketingtools.apple.com/api/v2/us/music/most-played/50/songs.json"

func FetchAppleChart(ctx context.Context) ([]fs.Song, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Apple intermittently accepts a connection but never sends headers. A retry
	// must make a new connection rather than reuse an unhealthy edge connection.
	transport.DisableKeepAlives = true
	defer transport.CloseIdleConnections()
	return fetchAppleChart(ctx, &http.Client{Timeout: 15 * time.Second, Transport: transport})
}

func fetchAppleChart(ctx context.Context, client *http.Client) ([]fs.Song, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if attempt > 0 {
			timer := time.NewTimer(time.Duration(attempt) * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, AppleChartURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("Apple chart attempt=%d failed: %v", attempt+1, err)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			songs, parseErr := parseAppleChart(io.LimitReader(resp.Body, 1024*1024))
			resp.Body.Close()
			return songs, parseErr
		}
		resp.Body.Close()
		lastErr = fmt.Errorf("Apple chart status %d", resp.StatusCode)
		if resp.StatusCode < 500 {
			return nil, lastErr
		}
		log.Printf("Apple chart attempt=%d failed: %v", attempt+1, lastErr)
	}
	return nil, lastErr
}

func parseAppleChart(reader io.Reader) ([]fs.Song, error) {
	var payload struct {
		Feed struct {
			Results []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Artist string `json:"artistName"`
				Kind   string `json:"kind"`
			} `json:"results"`
		} `json:"feed"`
	}
	if err := json.NewDecoder(reader).Decode(&payload); err != nil {
		return nil, err
	}
	songs := []fs.Song{}
	seen := map[string]bool{}
	for index, item := range payload.Feed.Results {
		if index >= 50 {
			break
		}
		if item.ID == "" || seen[item.ID] || item.Kind != "songs" || strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Artist) == "" {
			continue
		}
		seen[item.ID] = true
		songs = append(songs, fs.Song{Rank: index + 1, Title: item.Name, Artist: item.Artist})
	}
	if len(songs) == 0 {
		return nil, fmt.Errorf("Apple chart returned no readable songs")
	}
	return songs, nil
}
