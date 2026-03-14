package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	fs "melodex/firestore"
	"melodex/scrapers"
)

// PodcastScrapeRequest represents the request body for podcast scraping
type PodcastScrapeRequest struct {
	Category string   `json:"category"`           // Specific category to scrape (optional)
	Queries  []string `json:"queries,omitempty"`  // Custom queries (optional)
	MaxShows int      `json:"maxShows,omitempty"` // Max shows per query (default: 15)
	Fresh    bool     `json:"fresh,omitempty"`    // If true, delete all existing shows first
}

// PodcastScrapeResponse represents the response
type PodcastScrapeResponse struct {
	Category   string `json:"category"`
	ShowsFound int    `json:"showsFound"`
	NewShows   int    `json:"newShows"`
	Message    string `json:"message"`
}

// HandlePodcasts handles podcast show discovery requests
func (h *ScrapeHandler) HandlePodcasts(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	w.Header().Set("Content-Type", "application/json")

	// Parse request
	var req PodcastScrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If no body, use default behavior
		req = PodcastScrapeRequest{}
	}

	// Set defaults
	if req.MaxShows == 0 {
		req.MaxShows = 15
	}

	// If fresh mode, nuke the collection first
	if req.Fresh {
		log.Println("Fresh mode: deleting all existing podcast_shows...")
		iter := h.db.Collection("podcast_shows").Documents(ctx)
		batch := h.db.Batch()
		batchCount := 0
		for {
			doc, err := iter.Next()
			if err != nil {
				break
			}
			batch.Delete(doc.Ref)
			batchCount++
			if batchCount >= 500 {
				_, _ = batch.Commit(ctx)
				batch = h.db.Batch()
				batchCount = 0
			}
		}
		if batchCount > 0 {
			_, _ = batch.Commit(ctx)
		}
		log.Printf("Deleted existing podcast_shows for fresh scrape")
	}

	// Determine which categories to scrape
	var categoriesToScrape []scrapers.CategorySearchQueries
	if req.Category != "" {
		// Find the requested category
		for _, cat := range scrapers.GetCategories() {
			if cat.Category == req.Category {
				if len(req.Queries) > 0 {
					cat.Queries = req.Queries
				}
				categoriesToScrape = append(categoriesToScrape, cat)
				break
			}
		}
		if len(categoriesToScrape) == 0 {
			http.Error(w, "Unknown category: "+req.Category, http.StatusBadRequest)
			return
		}
	} else {
		// Scrape all default categories
		categoriesToScrape = scrapers.GetCategories()
	}

	results := make([]PodcastScrapeResponse, 0, len(categoriesToScrape))

	for _, cat := range categoriesToScrape {
		log.Printf("Scraping podcasts for category: %s", cat.Category)

		// Scrape shows for this category
		shows, err := scrapers.ScrapePodcastShows(*h.sp.Client, cat.Queries, req.MaxShows)
		if err != nil {
			log.Printf("Error scraping category %s: %v", cat.Category, err)
			results = append(results, PodcastScrapeResponse{
				Category: cat.Category,
				Message:  "Error: " + err.Error(),
			})
			continue
		}

		// Save shows to Firestore
		newShows := 0
		for _, show := range shows {
			// Check if show already exists
			docRef := h.db.Collection("podcast_shows").Doc(show.ID)
			doc, err := docRef.Get(ctx)

			if err != nil || !doc.Exists() {
				// New show - create it
				podcastShow := fs.PodcastShow{
					ID:           show.ID,
					Name:         show.Name,
					Publisher:    show.Publisher,
					Description:  show.Description,
					Categories:   []string{cat.Category},
					Languages:    show.Languages,
					ImageURL:     show.ImageURL,
					EpisodeCount: show.EpisodeCount,
					Explicit:     show.Explicit,
					ExternalURL:  show.ExternalURL,
					MediaType:    show.MediaType,
					DiscoveredIn: cat.Category,
					FirstSeenAt:  time.Now(),
					LastUpdated:  time.Now(),
				}

				_, err = docRef.Set(ctx, podcastShow)
				if err != nil {
					log.Printf("Error saving show %s: %v", show.ID, err)
					continue
				}
				newShows++
			} else {
				// Existing show - update categories if needed
				var existingShow fs.PodcastShow
				if err := doc.DataTo(&existingShow); err == nil {
					// Add new category if not already present
					categoryExists := false
					for _, c := range existingShow.Categories {
						if c == cat.Category {
							categoryExists = true
							break
						}
					}
					if !categoryExists {
						existingShow.Categories = append(existingShow.Categories, cat.Category)
						existingShow.LastUpdated = time.Now()
						_, err = docRef.Set(ctx, existingShow)
						if err != nil {
							log.Printf("Error updating show %s: %v", show.ID, err)
						}
					}
				}
			}
		}

		results = append(results, PodcastScrapeResponse{
			Category:   cat.Category,
			ShowsFound: len(shows),
			NewShows:   newShows,
			Message:    "Success",
		})

		log.Printf("Category %s: found %d shows, %d new", cat.Category, len(shows), newShows)
	}

	json.NewEncoder(w).Encode(results)
}

// HandlePodcastCategories returns all available podcast categories
func (h *ScrapeHandler) HandlePodcastCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	categories := scrapers.GetCategories()
	simpleCategories := make([]string, len(categories))
	for i, cat := range categories {
		simpleCategories[i] = cat.Category
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":      len(simpleCategories),
		"categories": simpleCategories,
	})
}
