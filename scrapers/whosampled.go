package scrapers

import (
	"context"
	"fmt"
	"log"
	fs "melodex/firestore"
	"net/http"
	"time"

	"github.com/chromedp/chromedp"
)

// ScrapeWhoSampled scrapes the WhoSampled.
func ScrapeWhoSampled(w http.ResponseWriter, q string) ([]fs.Track, error) {
	var songs []fs.Track
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set a timeout
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var result string

	// Run Chromedp tasks
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.whosampled.com/ajax/search/?q=please%20please%20please"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.InnerHTML(`body`, &result, chromedp.ByQuery),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Print the result
	fmt.Println("Page Content:")
	fmt.Println(result)

	return songs, nil
}
