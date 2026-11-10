package scrapers

import (
	"fmt"
	"log"
	"melodex/firestore"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

// ScrapeBillboardHot100 scrapes the Billboard Hot 100 chart.
func ScrapeBillboardHot100(w http.ResponseWriter) ([]firestore.Song, error) {
	c := colly.NewCollector()
	c.SetRequestTimeout(15 * time.Second)
	var songs []firestore.Song

	c.OnHTML("ul.o-chart-results-list-row", func(e *colly.HTMLElement) {
		rankStr := e.ChildText("li.o-chart-results-list__item:first-child span.c-label")
		rankStr = strings.Split(rankStr, "\n")[0]
		rank, err := strconv.Atoi(strings.TrimSpace(rankStr))
		if err != nil {
			log.Printf("Error converting rank to integer: %v", err)
			rank = 0
		}

		title := strings.TrimSpace(e.ChildText("h3.c-title.a-font-basic"))
		artist := e.ChildText("span.c-label.a-font-secondary")
		artist = strings.Split(artist, "\n")[0]
		artist = strings.TrimSpace(artist)

		// Only add if we have valid data
		if rank > 0 && title != "" && artist != "" {
			songs = append(songs, firestore.Song{Rank: rank, Title: title, Artist: artist})
		}
	})

	c.OnRequest(func(r *colly.Request) {
		log.Printf("Visiting %s", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s \nError: %v", r.Request.URL, err)
		http.Error(w, "Error scraping Billboard chart", http.StatusInternalServerError)
	})

	err := c.Visit("https://www.billboard.com/charts/hot-100/")
	if err != nil {
		log.Printf("Error visiting Billboard: %v", err)
		http.Error(w, "Error scraping Billboard chart", http.StatusInternalServerError)
		return nil, err
	}

	if len(songs) == 0 {
		return nil, fmt.Errorf("Billboard returned no readable songs")
	}
	log.Printf("Scraped %d songs from Billboard", len(songs))
	return songs, nil
}

type ScrapeBillboardHot100Func func(http.ResponseWriter) ([]firestore.Song, error)
