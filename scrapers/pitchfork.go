package scrapers

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	fs "melodex/firestore"
	"net/http"
	"strings"
	"time"
)

func ScrapePitchforkBestNewTracks(ctx context.Context, w http.ResponseWriter) ([]fs.Song, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://pitchfork.com/reviews/best/tracks/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Beatbrain-Melodex/1.0 (https://beatbrain.xyz)")
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("Pitchfork status %d", response.StatusCode)
	}
	return parsePitchfork(io.LimitReader(response.Body, 2*1024*1024))
}
func parsePitchfork(reader io.Reader) ([]fs.Song, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}
	songs := []fs.Song{}
	seen := map[string]bool{}
	doc.Find(".summary-item").Each(func(_ int, card *goquery.Selection) {
		link, ok := card.Find(".summary-item__hed-link").Attr("href")
		if !ok || !strings.HasPrefix(link, "/reviews/tracks/") || seen[link] {
			return
		}
		title := strings.Trim(strings.TrimSpace(card.Find(".summary-item__hed").Text()), "\"“”")
		artist := strings.TrimSpace(card.Find(".summary-item__sub-hed").Text())
		if title == "" || artist == "" {
			return
		}
		seen[link] = true
		songs = append(songs, fs.Song{Rank: len(songs) + 1, Title: title, Artist: artist})
	})
	if len(songs) == 0 {
		return nil, fmt.Errorf("Pitchfork returned no readable tracks")
	}
	return songs, nil
}

type ScrapePitchforkBestNewTracksFunc func(context.Context, http.ResponseWriter) ([]fs.Song, error)
