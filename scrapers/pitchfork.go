package scrapers

import (
	"log"
	"net/http"
	"strings"

	fs "melodex/firestore"

	"github.com/gocolly/colly"
)

// ScrapePitchforkBestNewTracks scrapes Pitchfork's Best New Tracks page.
func ScrapePitchforkBestNewTracks(w http.ResponseWriter) ([]fs.Song, error) {
	c := colly.NewCollector()
	var songs []fs.Song
	rank := 1

	// Set user agent to avoid blocking
	c.UserAgent = "melodex/1.0 music discovery scraper"

	// Target the track listings on the Best New Tracks page
	c.OnHTML("div.review", func(e *colly.HTMLElement) {
		// Extract artist and title from the review
		artistElement := e.ChildText("ul.artist-list li")
		titleElement := e.ChildText("h2")
		
		// Clean up the extracted text
		artist := strings.TrimSpace(artistElement)
		title := strings.TrimSpace(titleElement)
		
		// Sometimes the title includes the artist, try to separate
		if strings.Contains(title, " – ") {
			parts := strings.Split(title, " – ")
			if len(parts) >= 2 && artist == "" {
				artist = strings.TrimSpace(parts[0])
				title = strings.TrimSpace(strings.Join(parts[1:], " – "))
			}
		}
		
		// Skip if we couldn't extract both artist and title
		if artist == "" || title == "" {
			log.Printf("Failed to parse Pitchfork track: artist='%s', title='%s'", artist, title)
			return
		}

		// Remove quotes if they wrap the title
		title = strings.Trim(title, `"`)
		
		songs = append(songs, fs.Song{
			Rank:   rank,
			Title:  title,
			Artist: artist,
		})
		rank++
	})

	// Alternative selector for newer Pitchfork layout
	c.OnHTML("div.track-collection-item", func(e *colly.HTMLElement) {
		artist := strings.TrimSpace(e.ChildText(".artist"))
		title := strings.TrimSpace(e.ChildText(".track-title"))
		
		if artist == "" || title == "" {
			return
		}

		title = strings.Trim(title, `"`)
		
		songs = append(songs, fs.Song{
			Rank:   rank,
			Title:  title,
			Artist: artist,
		})
		rank++
	})

	// Another possible selector structure
	c.OnHTML("article", func(e *colly.HTMLElement) {
		// Look for track information in article elements
		titleText := e.ChildText("h2, .title")
		artistText := e.ChildText(".artist, .artist-name")
		
		if titleText != "" && artistText != "" {
			artist := strings.TrimSpace(artistText)
			title := strings.TrimSpace(titleText)
			title = strings.Trim(title, `"`)
			
			songs = append(songs, fs.Song{
				Rank:   rank,
				Title:  title,
				Artist: artist,
			})
			rank++
		}
	})

	c.OnRequest(func(r *colly.Request) {
		log.Printf("Visiting Pitchfork: %s", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s \nError: %v", r.Request.URL, err)
		if w != nil {
			http.Error(w, "Error scraping Pitchfork", http.StatusInternalServerError)
		}
	})

	err := c.Visit("https://pitchfork.com/best/tracks/")
	if err != nil {
		log.Printf("Error visiting Pitchfork: %v", err)
		if w != nil {
			http.Error(w, "Error scraping Pitchfork", http.StatusInternalServerError)
		}
		return nil, err
	}

	log.Printf("Scraped %d tracks from Pitchfork Best New Tracks", len(songs))
	return songs, nil
}

type ScrapePitchforkBestNewTracksFunc func(http.ResponseWriter) ([]fs.Song, error)