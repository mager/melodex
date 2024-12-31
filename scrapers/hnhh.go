package scrapers

import (
	"log"
	fs "melodex/firestore"
	"net/http"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

// ScrapeHotNewHipHop scrapes the HNHH main page.
func ScrapeHotNewHipHop(w http.ResponseWriter) ([]fs.Track, error) {
	c := colly.NewCollector()
	var songs []fs.Track

	c.OnHTML("div.top100-content article", func(e *colly.HTMLElement) {
		rankStr := e.ChildText("div.flex span.font-bold")
		rankInt, _ := strconv.Atoi(rankStr)
		title := e.ChildText("h2 a span")
		if strings.HasPrefix(title, "\"") && strings.HasSuffix(title, "\"") {
			title = strings.TrimPrefix(title, "\"")
			title = strings.TrimSuffix(title, "\"")
		}
		artist := e.ChildText("a.text-sm span")

		if rankStr != "" && title != "" && artist != "" {
			songs = append(songs, fs.Track{
				Rank:   rankInt,
				Title:  title,
				Artist: artist,
			})
		}
	})

	c.OnRequest(func(r *colly.Request) {
		log.Printf("Visiting %s", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s \nError: %v", r.Request.URL, err)
		http.Error(w, "Error scraping HNHH", http.StatusInternalServerError)
		return
	})

	err := c.Visit("https://www.hotnewhiphop.com/top100")
	if err != nil {
		log.Printf("Error visiting HNHH: %v", err)
		http.Error(w, "Error scraping HNHH chart", http.StatusInternalServerError)
		return nil, err
	}

	return songs, nil
}
