package scrapers

import (
	"fmt"
	"log"
	fs "melodex/firestore" // Assuming melodex is your module name
	"net/http"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

// ScrapeHotNewHipHop scrapes the HNHH Top 100 page.
// The http.ResponseWriter (w) is not directly used in this scraping function
// but is kept if your design requires it for other reasons (e.g. direct error reporting).
// It's generally better to return errors and let the HTTP handler manage the response.
func ScrapeHotNewHipHop(w http.ResponseWriter) ([]fs.Track, error) {
	c := colly.NewCollector(
	// It's good practice to set a User-Agent
	// colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.51 Safari/537.36"),
	)
	var songs []fs.Track
	var scrapingError error

	// OnHTML callback for each song article
	c.OnHTML("div.top100-content article", func(e *colly.HTMLElement) {
		// --- Rank ---
		// The rank is in a specific span, e.g., <span class="... font-bold text-3xl ...">1</span>
		rankStr := strings.TrimSpace(e.ChildText("span.font-bold.text-3xl"))
		rankInt, err := strconv.Atoi(rankStr)
		if err != nil {
			log.Printf("Error converting rank '%s' to int for an item: %v. Skipping item.", rankStr, err)
			return // Skip this item if rank is not parsable
		}

		// --- Title ---
		// Title is the text of the <a> tag inside an <h2>, under a <div class="ml-4">
		// e.g. <div class="ml-4"> <h2> <a href="...">TITLE HERE</a> </h2> ... </div>
		title := strings.TrimSpace(e.ChildText("div.ml-4 h2 a"))
		// Remove surrounding quotes if present (as in your original code)
		if strings.HasPrefix(title, "\"") && strings.HasSuffix(title, "\"") {
			title = strings.TrimPrefix(title, "\"")
			title = strings.TrimSuffix(title, "\"")
		}

		// --- Artist(s) ---
		// Artists are in <a> tags (class="text-sm ...") inside a <div> that is a sibling to the <h2> for the title.
		// e.g. <div class="ml-4"> ... <h2>...</h2> <div> <a ...>Artist1, </a> <a ...>Artist2</a> ... </div> </div>
		var artists []string
		e.ForEach("div.ml-4 > div > a.text-sm", func(_ int, artistElement *colly.HTMLElement) {
			artistName := strings.TrimSpace(artistElement.Text)
			// Artist names in a list might end with a comma (e.g., "Artist Name,"), remove it.
			artistName = strings.TrimSuffix(artistName, ",")
			if artistName != "" {
				artists = append(artists, artistName)
			}
		})
		artist := strings.Join(artists, ", ") // Join multiple artists with ", "

		if title != "" && artist != "" {
			songs = append(songs, fs.Track{
				Rank:   rankInt,
				Title:  title,
				Artist: artist,
			})
			// For debugging: log.Printf("Scraped: Rank %d, Title '%s', Artist '%s'", rankInt, title, artist)
		} else {
			log.Printf("Skipping entry (Rank %d) due to missing title or artist: Title:'%s', Artist:'%s'", rankInt, title, artist)
		}
	})

	c.OnRequest(func(r *colly.Request) {
		log.Printf("Visiting %s", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s \nStatus: %d \nError: %v", r.Request.URL, r.StatusCode, err)
		scrapingError = fmt.Errorf("scraping error for %s: %w (status: %d)", r.Request.URL, err, r.StatusCode)
		// It's better to let the calling HTTP handler send the HTTP error response.
		// If w is not nil, you could use it, but it complicates error handling flow.
		// if w != nil {
		// 	http.Error(w, "Error scraping HNHH", http.StatusInternalServerError)
		// }
	})

	err := c.Visit("https://www.hotnewhiphop.com/top100")
	if err != nil { // Catches errors before any request is made (e.g., invalid URL)
		log.Printf("Error initiating visit to HNHH: %v", err)
		return nil, err
	}

	// If an error occurred in OnError callback
	if scrapingError != nil {
		return nil, scrapingError
	}

	if len(songs) == 0 && scrapingError == nil {
		log.Println("No songs were scraped. Check selectors and website structure.")
		// Optionally return an error here if no songs being scraped is considered an error condition.
		// return nil, fmt.Errorf("no songs scraped from HNHH")
	}

	return songs, nil
}
