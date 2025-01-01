package scrapers

import (
	"log"
	"melodex/firestore"
	"net/http"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

// ScrapeBillboardHot100 scrapes the Billboard Hot 100 chart.
func ScrapeBillboardHot100(w http.ResponseWriter) ([]firestore.Song, error) {
	c := colly.NewCollector()
	var songs []firestore.Song

	c.OnHTML("ul.o-chart-results-list-row", func(e *colly.HTMLElement) {
		rankStr := e.ChildText("li.o-chart-results-list__item span.c-label.a-font-primary-bold-l")
		rankStr = strings.Split(rankStr, "\n")[0]
		rank, err := strconv.Atoi(strings.TrimSpace(rankStr))
		if err != nil {
			log.Printf("Error converting rank to integer: %v", err)
			rank = 0
		}
		title := strings.TrimSpace(e.ChildText("li.o-chart-results-list__item h3.c-title"))
		artist := e.ChildText("li.o-chart-results-list__item span.c-label:not(.a-font-primary-bold-l):not(.u-width-45):not(.u-width-40)")
		artist = strings.Split(artist, "\n")[0]
		artist = strings.TrimSpace(artist)

		songs = append(songs, firestore.Song{Rank: rank, Title: title, Artist: artist})
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

	return songs, nil
}

type ScrapeBillboardHot100Func func(http.ResponseWriter) ([]firestore.Song, error)
