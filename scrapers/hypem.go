package scrapers

import (
	"log"
	fs "melodex/firestore"
	"net/http"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

// ScrapeHypeMachine scrapes the Hype Machine popular pages.
func ScrapeHypeMachine(w http.ResponseWriter) ([]fs.Track, error) {
	c := colly.NewCollector()
	var songs []fs.Track

	c.OnHTML("#track-list .section-track", func(e *colly.HTMLElement) {
		// Rank is not directly available, so we'll infer it from the order on the page
		rank := len(songs) + 1

		title := strings.TrimSpace(e.ChildText("h3.track_name a.track span.base-title"))
		artist := strings.TrimSpace(e.ChildText("h3.track_name a.artist"))

		// The Spotify link is in a nested structure.
		spotifyLink := e.ChildAttr(".meta .download a[href*='spotify_track']", "href")

		// Get the thumbnail URL.
		thumbURL := e.ChildAttr("a.thumb", "style")

		// Extract the Spotify ID from the link
		spotifyID := ""
		if spotifyLink != "" {
			parts := strings.Split(spotifyLink, "/")
			if len(parts) > 0 {
				spotifyID = parts[len(parts)-1] // Get the last part of the URL
			}
		}

		// Extract the thumbnail URL from the inline style
		thumb := ""
		if thumbURL != "" {
			start := strings.Index(thumbURL, "url(") + len("url(")
			end := strings.Index(thumbURL[start:], ")")
			if start > -1 && end > -1 {
				fullURL := thumbURL[start : start+end]
				thumb = strings.TrimPrefix(fullURL, "https://static.hypem.com/items_images/")
			}
		}

		// Only add if we have all required fields (except rank)
		if title != "" && artist != "" && spotifyID != "" {
			songs = append(songs, fs.Track{
				Rank:      rank,
				Title:     title,
				Artist:    artist,
				SpotifyID: spotifyID,
				Thumb:     thumb,
			})
		}
	})

	c.OnRequest(func(r *colly.Request) {
		log.Printf("Visiting %s", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s \nError: %v", r.Request.URL, err)
		http.Error(w, "Error scraping Hype Machine", http.StatusInternalServerError)
	})

	// Visit multiple pages
	for i := 1; i <= 3; i++ {
		url := "https://hypem.com/popular/" + strconv.Itoa(i)
		err := c.Visit(url)
		if err != nil {
			log.Printf("Error visiting %s: %v", url, err)
			http.Error(w, "Error scraping Hype Machine chart", http.StatusInternalServerError)
			return nil, err
		}
	}

	return songs, nil
}
