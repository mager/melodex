package scrapers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	fs "melodex/firestore"
	"net/http"
	"strings"
	"time"
)

const AppleChartURL = "https://rss.marketingtools.apple.com/api/v2/us/music/most-played/50/songs.json"

func FetchAppleChart(ctx context.Context) ([]fs.Song, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, AppleChartURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Apple chart status %d", resp.StatusCode)
	}
	return parseAppleChart(io.LimitReader(resp.Body, 1024*1024))
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
