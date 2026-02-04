package scrapers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	fs "melodex/firestore"
)

// ScrapeRedditFresh scrapes Reddit's public JSON API for r/listentothis and r/hiphopheads,
// filtering for posts with "[FRESH]" in the title.
func ScrapeRedditFresh(w http.ResponseWriter) ([]fs.Song, error) {
	var allSongs []fs.Song
	subreddits := []string{"listentothis", "hiphopheads"}

	for _, subreddit := range subreddits {
		songs, err := scrapeSubreddit(subreddit, w)
		if err != nil {
			log.Printf("Error scraping r/%s: %v", subreddit, err)
			continue
		}
		allSongs = append(allSongs, songs...)
	}

	log.Printf("Scraped %d FRESH tracks from Reddit", len(allSongs))
	return allSongs, nil
}

func scrapeSubreddit(subreddit string, w http.ResponseWriter) ([]fs.Song, error) {
	url := fmt.Sprintf("https://www.reddit.com/r/%s/search.json?q=[FRESH]&restrict_sr=1&sort=hot&limit=50", subreddit)
	
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Reddit requires a User-Agent header
	req.Header.Set("User-Agent", "melodex/1.0 music discovery scraper")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("Reddit API returned status %d", resp.StatusCode)
		log.Printf("Error: %v", err)
		if w != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return nil, err
	}

	var redditResp RedditResponse
	if err := json.NewDecoder(resp.Body).Decode(&redditResp); err != nil {
		log.Printf("Error decoding Reddit JSON: %v", err)
		if w != nil {
			http.Error(w, "Error decoding Reddit response", http.StatusInternalServerError)
		}
		return nil, err
	}

	var songs []fs.Song
	rank := 1

	for _, post := range redditResp.Data.Children {
		title := post.Data.Title
		
		// Check if title contains [FRESH] (case-insensitive)
		if !strings.Contains(strings.ToLower(title), "[fresh]") {
			continue
		}

		artist, trackTitle := parseRedditTitle(title)
		if artist == "" || trackTitle == "" {
			log.Printf("Failed to parse artist/title from: %s", title)
			continue
		}

		songs = append(songs, fs.Song{
			Rank:   rank,
			Title:  trackTitle,
			Artist: artist,
		})
		rank++
	}

	log.Printf("Found %d FRESH tracks from r/%s", len(songs), subreddit)
	return songs, nil
}

// parseRedditTitle extracts artist and title from Reddit post titles
// Common formats: "Artist - Title [FRESH]", "[FRESH] Artist - Title", etc.
func parseRedditTitle(title string) (string, string) {
	// Remove [FRESH] and similar tags (case-insensitive)
	re := regexp.MustCompile(`(?i)\[(fresh|new|premiere|video|audio)\]`)
	cleaned := re.ReplaceAllString(title, "")
	
	// Remove common prefixes/suffixes
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.Trim(cleaned, "()-[]")
	
	// Split by common separators
	separators := []string{" - ", " – ", " — ", ": "}
	
	for _, sep := range separators {
		parts := strings.Split(cleaned, sep)
		if len(parts) >= 2 {
			artist := strings.TrimSpace(parts[0])
			title := strings.TrimSpace(strings.Join(parts[1:], sep))
			
			// Basic validation
			if len(artist) > 0 && len(title) > 0 && len(artist) < 100 && len(title) < 100 {
				return artist, title
			}
		}
	}
	
	return "", ""
}

// RedditResponse represents the JSON response structure from Reddit API
type RedditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				Title string `json:"title"`
				URL   string `json:"url"`
				Score int    `json:"score"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type ScrapeRedditFreshFunc func(http.ResponseWriter) ([]fs.Song, error)