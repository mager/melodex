package scrapers

import (
	"context"
	"log"
	"strings"

	"github.com/zmb3/spotify/v2"
)

// PodcastShow represents a podcast show from Spotify
type PodcastShow struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Publisher    string   `json:"publisher"`
	Description  string   `json:"description"`
	HTMLDescription string `json:"htmlDescription,omitempty"`
	Categories   []string `json:"categories"` // Our mapped categories
	Languages    []string `json:"languages"`
	ImageURL     string   `json:"imageURL"`
	EpisodeCount int      `json:"episodeCount"`
	Explicit     bool     `json:"explicit"`
	ExternalURL  string   `json:"externalURL"`
	MediaType    string   `json:"mediaType"`
}

// CategorySearchQueries maps our custom categories to Spotify search queries
type CategorySearchQueries struct {
	Category string   `json:"category"`
	Queries  []string `json:"queries"`
}

// DefaultPodcastCategories - 100+ categories for discovery
var DefaultPodcastCategories = []CategorySearchQueries{
	{"Arts", []string{"arts", "design", "photography"}},
	{"Books", []string{"books", "literature", "reading", "book review"}},
	{"Business", []string{"business", "entrepreneurship", "startups", "marketing", "leadership"}},
	{"Comedy", []string{"comedy", "standup", "humor", "funny"}},
	{"Education", []string{"education", "learning", "teaching", "study"}},
	{"Fiction", []string{"fiction", "storytelling", "drama", "audio drama"}},
	{"Government", []string{"government", "politics", "policy", "civics"}},
	{"History", []string{"history", "historical", "past", "archives"}},
	{"Health", []string{"health", "fitness", "wellness", "mental health"}},
	{"Kids", []string{"kids", "children", "family", "parenting"}},
	{"Linguistics", []string{"linguistics", "language", "etymology", "syntax", "grammar", "philology"}},
	{"Music", []string{"music", "musician", "audio production", "music history"}},
	{"News", []string{"news", "journalism", "current events", "headlines"}},
	{"Religion", []string{"religion", "spirituality", "faith", "theology"}},
	{"Science", []string{"science", "research", "discovery", "scientific"}},
	{"Society", []string{"society", "culture", "social issues", "community"}},
	{"Sports", []string{"sports", "athletics", "fitness", "games"}},
	{"Technology", []string{"technology", "tech", "software", "computing", "AI", "programming"}},
	{"True Crime", []string{"true crime", "crime", "investigation", "criminal"}},
	{"TV & Film", []string{"tv", "film", "movies", "cinema", "television"}},
	// Niche categories for deep discovery
	{"Astronomy", []string{"astronomy", "space", "cosmos", "astrophysics"}},
	{"Biology", []string{"biology", "life science", "genetics", "evolution"}},
	{"Chemistry", []string{"chemistry", "chemical", "molecules"}},
	{"Philosophy", []string{"philosophy", "philosophical", "ethics", "metaphysics"}},
	{"Psychology", []string{"psychology", "mental", "behavior", "cognitive"}},
	{"Economics", []string{"economics", "finance", "markets", "trading"}},
	{"Law", []string{"law", "legal", "court", "justice"}},
	{"Medicine", []string{"medicine", "medical", "doctor", "healthcare"}},
	{"Physics", []string{"physics", "physical science", "quantum", "mechanics"}},
	{"Anthropology", []string{"anthropology", "anthropological", "culture", "human"}},
	{"Archaeology", []string{"archaeology", "archaeological", "ancient", "digs"}},
	{"Architecture", []string{"architecture", "design", "buildings", "urban"}},
	{"Astronomy", []string{"astronomy", "space", "cosmos", "stars"}},
	{"Blockchain", []string{"blockchain", "crypto", "bitcoin", "web3"}},
	{"Climate", []string{"climate", "environment", "ecology", "green"}},
	{"Cooking", []string{"cooking", "food", "culinary", "recipes", "chef"}},
	{"Crafts", []string{"crafts", "diy", "making", "handmade"}},
	{"Cybersecurity", []string{"cybersecurity", "hacking", "security", "infosec"}},
	{"Dating", []string{"dating", "relationships", "romance", "love"}},
	{"Documentary", []string{"documentary", "non-fiction", "real stories"}},
	{"Entrepreneurship", []string{"entrepreneurship", "founder", "startup", "venture"}},
	{"Fashion", []string{"fashion", "style", "clothing", "trends"}},
	{"Gaming", []string{"gaming", "video games", "esports", "game design"}},
	{"Gardening", []string{"gardening", "plants", "horticulture", "growing"}},
	{"Investing", []string{"investing", "stocks", "portfolio", "wealth"}},
	{"Language Learning", []string{"language learning", "learn language", "polyglot", "bilingual"}},
	{"Meditation", []string{"meditation", "mindfulness", "zen", "relaxation"}},
	{"Military", []string{"military", "war", "defense", "veterans"}},
	{"Mythology", []string{"mythology", "myths", "legends", "folklore"}},
	{"Nature", []string{"nature", "wildlife", "outdoors", "animals"}},
	{"Nutrition", []string{"nutrition", "diet", "healthy eating", "supplements"}},
	{"Parenting", []string{"parenting", "motherhood", "fatherhood", "raising kids"}},
	{"Personal Finance", []string{"personal finance", "budget", "saving", "money"}},
	{"Pets", []string{"pets", "dogs", "cats", "animals"}},
	{"Photography", []string{"photography", "photos", "camera", "shooting"}},
	{"Productivity", []string{"productivity", "time management", "efficiency", "habits"}},
	{"Real Estate", []string{"real estate", "housing", "property", "investment"}},
	{"Self Help", []string{"self help", "self improvement", "personal growth", "development"}},
	{"Skepticism", []string{"skepticism", "critical thinking", "debunking", "science"}},
	{"Space", []string{"space", "NASA", "rockets", "exploration"}},
	{"Travel", []string{"travel", "adventure", "backpacking", "destinations"}},
	{"Writing", []string{"writing", "author", "storytelling", "publishing"}},
	{"AI & Machine Learning", []string{"artificial intelligence", "machine learning", "AI", "deep learning"}},
	{"Data Science", []string{"data science", "data analytics", "big data", "statistics"}},
	{"DevOps", []string{"devops", "SRE", "infrastructure", "cloud"}},
	{"Frontend Development", []string{"frontend", "javascript", "react", "web development"}},
	{"Backend Development", []string{"backend", "API", "server", "database"}},
	{"Mobile Development", []string{"mobile development", "iOS", "Android", "app development"}},
	{"Open Source", []string{"open source", "github", "contribution", "community"}},
	{"UX Design", []string{"UX design", "user experience", "product design", "HCI"}},
	{"Neuroscience", []string{"neuroscience", "brain", "neurobiology", "cognition"}},
	{"Paleontology", []string{"paleontology", "dinosaurs", "fossils", "prehistoric"}},
	{"Sociology", []string{"sociology", "social science", "culture", "behavior"}},
	{"Theater", []string{"theater", "acting", "drama", "performance"}},
	{"Visual Arts", []string{"visual arts", "art history", "painting", "sculpture"}},
	{"Wine", []string{"wine", "wine tasting", "sommelier", "vineyard"}},
	{"Beer", []string{"beer", "brewing", "craft beer", "homebrew"}},
	{"Coffee", []string{"coffee", "barista", "roasting", "brewing"}},
	{"Chess", []string{"chess", "strategy", "tactics", "grandmaster"}},
	{"Poker", []string{"poker", "gambling", "strategy", "cards"}},
	{"MMA", []string{"MMA", "martial arts", "UFC", "fighting"}},
	{"Basketball", []string{"basketball", "NBA", "hoops", "ball"}},
	{"Football", []string{"football", "NFL", "gridiron"}},
	{"Soccer", []string{"soccer", "football", "premier league", "futbol"}},
	{"Baseball", []string{"baseball", "MLB", "diamond"}},
	{"Hockey", []string{"hockey", "NHL", "ice"}},
	{"Tennis", []string{"tennis", "ATP", "WTA", "grand slam"}},
	{"Golf", []string{"golf", "PGA", "links", "tour"}},
	{"Running", []string{"running", "marathon", "trail running", "jogging"}},
	{"Cycling", []string{"cycling", "biking", "tour de france", "peloton"}},
	{"Swimming", []string{"swimming", "aquatics", "pool", "open water"}},
	{"Yoga", []string{"yoga", "asana", "practice", "mindfulness"}},
	{"Pilates", []string{"pilates", "core", "fitness", "exercise"}},
	{"Weightlifting", []string{"weightlifting", "strength", "powerlifting", "gym"}},
	{"CrossFit", []string{"crossfit", "functional fitness", "WOD", "metcon"}},
	{"Hiking", []string{"hiking", "trekking", "backpacking", "trails"}},
	{"Camping", []string{"camping", "outdoors", "wilderness", "survival"}},
	{"Fishing", []string{"fishing", "angling", "fly fishing", "bass"}},
	{"Hunting", []string{"hunting", "outdoors", "conservation", "wild game"}},
	{"Birding", []string{"birding", "bird watching", "ornithology", "species"}},
	{"Stoicism", []string{"stoicism", "stoic", "marcus aurelius", "philosophy"}},
	{"Existentialism", []string{"existentialism", "sartre", "camus", "meaning"}},
	{"Minimalism", []string{"minimalism", "simple living", "declutter", "intentional"}},
	{"Firefighting", []string{"firefighting", "fire service", "emergency", "first responder"}},
	{"Police", []string{"police", "law enforcement", "officer", "patrol"}},
	{"EMS", []string{"EMS", "paramedic", "EMT", "emergency medical"}},
	{"Nursing", []string{"nursing", "nurse", "healthcare", "RN"}},
	{"Teaching", []string{"teaching", "teacher", "education", "classroom"}},
	{"Aviation", []string{"aviation", "flying", "pilot", "aircraft"}},
	{"Boating", []string{"boating", "sailing", "yacht", "maritime"}},
	{"Motorcycles", []string{"motorcycles", "riding", "motorbike", "chopper"}},
	{"Cars", []string{"cars", "automotive", "racing", "mechanic"}},
	{"Trains", []string{"trains", "railways", "locomotive", "model trains"}},
	{"DJing", []string{"DJ", "DJing", "turntablism", "mixing"}},
	{"Electronic Music", []string{"electronic music", "EDM", "synthesizer", "production"}},
	{"Jazz", []string{"jazz", "improvisation", "bebop", "standards"}},
	{"Classical Music", []string{"classical music", "orchestra", "symphony", "opera"}},
	{"Hip Hop", []string{"hip hop", "rap", "MC", "urban"}},
	{"Rock Music", []string{"rock", "alternative", "indie", "metal"}},
	{"Country Music", []string{"country", "western", "honky tonk", "americana"}},
	{"Blues", []string{"blues", "delta blues", "Chicago blues", "guitar"}},
	{"Folk", []string{"folk", "traditional", "acoustic", "roots"}},
	{"R&B", []string{"R&B", "soul", "funk", "motown"}},
	{"Reggae", []string{"reggae", "dub", "dancehall", "roots"}},
	{"Latin Music", []string{"latin music", "salsa", "bachata", "reggaeton"}},
	{"World Music", []string{"world music", "global", "international", "ethnic"}},
}

// ScrapePodcastShows searches Spotify for podcast shows by category queries
func ScrapePodcastShows(client spotify.Client, queries []string, maxPerQuery int) ([]PodcastShow, error) {
	ctx := context.Background()
	var allShows []PodcastShow
	seenIDs := make(map[string]bool)

	for _, query := range queries {
		log.Printf("Searching podcasts for query: %s", query)

		// Search for shows
		results, err := client.Search(ctx, query, spotify.SearchTypeShow)
		if err != nil {
			log.Printf("Error searching for '%s': %v", query, err)
			continue
		}

		if results.Shows == nil {
			continue
		}

		count := 0
		for _, show := range results.Shows.Shows {
			if count >= maxPerQuery {
				break
			}

			// Skip duplicates
			if seenIDs[show.ID.String()] {
				continue
			}
			seenIDs[show.ID.String()] = true

			podcastShow := PodcastShow{
				ID:          show.ID.String(),
				Name:        show.Name,
				Publisher:   show.Publisher,
				Description: stripHTML(show.Description),
				Languages:   show.Languages,
				EpisodeCount: 0,
				Explicit:    show.Explicit,
				ExternalURL: show.ExternalURLs["spotify"],
				MediaType:   show.MediaType,
			}

			// Get best image
			if len(show.Images) > 0 {
				podcastShow.ImageURL = show.Images[0].URL
			}

			allShows = append(allShows, podcastShow)
			count++
		}

		log.Printf("Found %d shows for query '%s'", count, query)
	}

	return allShows, nil
}

// stripHTML removes HTML tags from text
func stripHTML(s string) string {
	// Simple HTML stripping - remove anything between < and >
	var result strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				result.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(result.String())
}

// GetCategories returns all podcast categories
func GetCategories() []CategorySearchQueries {
	return DefaultPodcastCategories
}
