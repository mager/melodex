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

	c.OnHTML("#track-list .section-player", func(e *colly.HTMLElement) {
		rankStr := e.ChildText("span.rank")
		title := strings.TrimSpace(e.ChildText("h3.track_name a.track span.base-title"))
		artist := strings.TrimSpace(e.ChildText("h3.track_name a.artist"))
		spotifyLink := e.ChildAttr(".meta .download a[href^='/go/spotify_track/']", "href")
		thumb := e.ChildAttr("a.thumb", "style")

		// Extract the Spotify ID from the link
		spotifyID := ""
		if spotifyLink != "" {
			parts := strings.Split(spotifyLink, "/")
			spotifyID = parts[len(parts)-1]
		}

		// Extract the thumbnail URL from the inline style
		thumbURL := ""
		if thumb != "" {
			start := strings.Index(thumb, "url(") + len("url(")
			end := strings.Index(thumb[start:], ")")
			if start > -1 && end > -1 {
				fullURL := thumb[start : start+end]
				thumbURL = strings.TrimPrefix(fullURL, "https://static.hypem.com/items_images/")
			}
		}

		// Only add if we have all required fields
		if rankStr != "" && title != "" && artist != "" {
			rank, err := strconv.Atoi(rankStr)
			if err != nil {
				log.Printf("Error converting rank to integer: %v", err)
				rank = 0 // Default rank in case of an error
			}
			songs = append(songs, fs.Track{
				Rank:      rank,
				Title:     title,
				Artist:    artist,
				SpotifyID: spotifyID,
				Thumb:     thumbURL,
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
