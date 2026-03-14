package scrapers

import (
	"context"
	"log"
	"strings"

	"github.com/zmb3/spotify/v2"
)

// PodcastShow represents a podcast show from Spotify
type PodcastShow struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Publisher       string   `json:"publisher"`
	Description     string   `json:"description"`
	HTMLDescription string   `json:"htmlDescription,omitempty"`
	Categories      []string `json:"categories"` // Our mapped categories
	Languages       []string `json:"languages"`
	ImageURL        string   `json:"imageURL"`
	EpisodeCount    int      `json:"episodeCount"`
	Explicit        bool     `json:"explicit"`
	ExternalURL     string   `json:"externalURL"`
	MediaType       string   `json:"mediaType"`
}

// CategorySearchQueries maps our custom categories to Spotify search queries
type CategorySearchQueries struct {
	Category string   `json:"category"`
	Queries  []string `json:"queries"`
}

// DefaultPodcastCategories - optimized queries for popular podcast discovery.
//
// Strategy: Use "best X podcast" and "[topic] podcast" phrasing.
// Spotify search heavily weights title matches and popularity for these
// intent-based queries, surfacing the shows people actually listen to
// instead of random keyword matches in descriptions.
//
// Each category uses 2-4 high-signal queries rather than 4-6 generic keywords.
var DefaultPodcastCategories = []CategorySearchQueries{
	// ─── CORE CATEGORIES ───
	{"Arts", []string{"best arts podcast", "arts and culture podcast", "creative podcast"}},
	{"Books", []string{"best book podcast", "book review podcast", "book club podcast"}},
	{"Business", []string{"best business podcast", "business news podcast", "how I built this"}},
	{"Comedy", []string{"best comedy podcast", "funny podcast", "stand up comedy podcast"}},
	{"Education", []string{"best education podcast", "learning podcast", "educational podcast"}},
	{"Fiction", []string{"best fiction podcast", "audio drama podcast", "storytelling podcast"}},
	{"Government", []string{"politics podcast", "government podcast", "political analysis podcast"}},
	{"History", []string{"best history podcast", "history podcast", "historical podcast"}},
	{"Health", []string{"best health podcast", "health and wellness podcast", "mental health podcast"}},
	{"Kids", []string{"best kids podcast", "children podcast", "family podcast", "stories for kids"}},
	{"Linguistics", []string{"linguistics podcast", "language podcast", "etymology podcast"}},
	{"Music", []string{"best music podcast", "music interview podcast", "music history podcast", "song exploder"}},
	{"News", []string{"best news podcast", "daily news podcast", "news analysis podcast"}},
	{"Religion", []string{"best religion podcast", "faith podcast", "spirituality podcast", "theology podcast"}},
	{"Science", []string{"best science podcast", "science explained podcast", "popular science podcast"}},
	{"Society", []string{"society and culture podcast", "social issues podcast", "culture podcast"}},
	{"Sports", []string{"best sports podcast", "sports news podcast", "sports analysis podcast"}},
	{"Technology", []string{"best tech podcast", "technology news podcast", "tech podcast", "silicon valley podcast"}},
	{"True Crime", []string{"best true crime podcast", "true crime podcast", "crime podcast", "murder mystery podcast"}},
	{"TV & Film", []string{"best tv podcast", "movie podcast", "film review podcast", "pop culture podcast"}},

	// ─── SCIENCE & ACADEMIC ───
	{"Astronomy", []string{"best astronomy podcast", "space podcast", "astrophysics podcast", "NASA podcast"}},
	{"Biology", []string{"biology podcast", "life science podcast", "evolution podcast"}},
	{"Chemistry", []string{"chemistry podcast", "chemical science podcast"}},
	{"Philosophy", []string{"best philosophy podcast", "philosophy podcast", "ethics podcast", "stoicism podcast"}},
	{"Psychology", []string{"best psychology podcast", "psychology today podcast", "behavioral science podcast"}},
	{"Neuroscience", []string{"neuroscience podcast", "brain science podcast", "cognitive science podcast"}},
	{"Physics", []string{"physics podcast", "quantum physics podcast", "theoretical physics podcast"}},
	{"Anthropology", []string{"anthropology podcast", "human culture podcast"}},
	{"Archaeology", []string{"archaeology podcast", "ancient history podcast"}},
	{"Paleontology", []string{"paleontology podcast", "dinosaur podcast", "fossil podcast"}},
	{"Sociology", []string{"sociology podcast", "social science podcast"}},

	// ─── BUSINESS & FINANCE ───
	{"Economics", []string{"best economics podcast", "economics explained podcast", "freakonomics"}},
	{"Entrepreneurship", []string{"best entrepreneurship podcast", "startup podcast", "founder podcast", "how I built this"}},
	{"Investing", []string{"best investing podcast", "stock market podcast", "investing for beginners podcast"}},
	{"Personal Finance", []string{"best personal finance podcast", "money podcast", "financial independence podcast", "all the hacks podcast"}},
	{"Real Estate", []string{"best real estate podcast", "real estate investing podcast", "property podcast"}},

	// ─── TECH & DEV ───
	{"AI & Machine Learning", []string{"best AI podcast", "artificial intelligence podcast", "machine learning podcast", "AI news podcast"}},
	{"Blockchain", []string{"best crypto podcast", "blockchain podcast", "bitcoin podcast", "web3 podcast"}},
	{"Cybersecurity", []string{"best cybersecurity podcast", "hacking podcast", "infosec podcast", "security podcast"}},
	{"Data Science", []string{"data science podcast", "data analytics podcast", "statistics podcast"}},
	{"Frontend Development", []string{"frontend podcast", "javascript podcast", "web development podcast", "react podcast"}},
	{"Backend Development", []string{"backend development podcast", "software engineering podcast", "API podcast"}},
	{"Mobile Development", []string{"mobile development podcast", "iOS podcast", "Android podcast", "app development podcast"}},
	{"Open Source", []string{"open source podcast", "github podcast", "developer podcast"}},
	{"DevOps", []string{"devops podcast", "cloud computing podcast", "SRE podcast", "infrastructure podcast"}},
	{"UX Design", []string{"best UX podcast", "UX design podcast", "product design podcast", "design podcast"}},

	// ─── LIFESTYLE ───
	{"Cooking", []string{"best cooking podcast", "food podcast", "recipe podcast", "chef podcast"}},
	{"Crafts", []string{"crafts podcast", "DIY podcast", "making podcast", "handmade podcast"}},
	{"Dating", []string{"best dating podcast", "relationship podcast", "love podcast", "dating advice podcast"}},
	{"Fashion", []string{"best fashion podcast", "style podcast", "fashion industry podcast"}},
	{"Gardening", []string{"gardening podcast", "plant podcast", "garden podcast"}},
	{"Meditation", []string{"best meditation podcast", "mindfulness podcast", "guided meditation podcast"}},
	{"Minimalism", []string{"minimalism podcast", "simple living podcast", "intentional living podcast"}},
	{"Nature", []string{"nature podcast", "wildlife podcast", "outdoors podcast", "animals podcast"}},
	{"Nutrition", []string{"best nutrition podcast", "diet podcast", "healthy eating podcast"}},
	{"Parenting", []string{"best parenting podcast", "mom podcast", "dad podcast", "raising kids podcast"}},
	{"Pets", []string{"best pets podcast", "dog podcast", "cat podcast", "animal podcast"}},
	{"Photography", []string{"photography podcast", "camera podcast", "photo podcast"}},
	{"Productivity", []string{"best productivity podcast", "time management podcast", "habits podcast", "getting things done podcast"}},
	{"Self Help", []string{"best self improvement podcast", "personal development podcast", "self help podcast", "motivation podcast"}},
	{"Travel", []string{"best travel podcast", "travel tips podcast", "adventure podcast", "backpacking podcast"}},
	{"Writing", []string{"best writing podcast", "author podcast", "creative writing podcast", "publishing podcast"}},

	// ─── SPORTS ───
	{"Basketball", []string{"best basketball podcast", "NBA podcast", "basketball analysis podcast"}},
	{"Football", []string{"best football podcast", "NFL podcast", "fantasy football podcast"}},
	{"Soccer", []string{"best soccer podcast", "football podcast premier league", "MLS podcast", "champions league podcast"}},
	{"Baseball", []string{"best baseball podcast", "MLB podcast", "baseball analysis podcast"}},
	{"Hockey", []string{"best hockey podcast", "NHL podcast", "hockey podcast"}},
	{"Tennis", []string{"tennis podcast", "ATP podcast", "grand slam podcast"}},
	{"Golf", []string{"best golf podcast", "PGA podcast", "golf tips podcast"}},
	{"MMA", []string{"best MMA podcast", "UFC podcast", "martial arts podcast", "combat sports podcast"}},
	{"Running", []string{"best running podcast", "marathon podcast", "trail running podcast"}},
	{"Cycling", []string{"cycling podcast", "bike podcast", "tour de france podcast"}},
	{"CrossFit", []string{"crossfit podcast", "functional fitness podcast"}},
	{"Weightlifting", []string{"weightlifting podcast", "strength training podcast", "powerlifting podcast"}},
	{"Yoga", []string{"yoga podcast", "yoga practice podcast"}},

	// ─── MUSIC GENRES ───
	{"DJing", []string{"DJ podcast", "DJing podcast", "electronic music podcast", "mixing podcast"}},
	{"Electronic Music", []string{"best electronic music podcast", "EDM podcast", "techno podcast", "house music podcast"}},
	{"Jazz", []string{"jazz podcast", "jazz music podcast", "jazz history podcast"}},
	{"Classical Music", []string{"classical music podcast", "orchestra podcast", "opera podcast"}},
	{"Hip Hop", []string{"best hip hop podcast", "rap podcast", "hip hop culture podcast"}},
	{"Rock Music", []string{"rock podcast", "rock music podcast", "indie rock podcast", "alternative podcast"}},
	{"Country Music", []string{"country music podcast", "country podcast", "americana podcast"}},
	{"R&B", []string{"R&B podcast", "soul music podcast", "funk podcast"}},

	// ─── NICHE / HOBBY ───
	{"Gaming", []string{"best gaming podcast", "video game podcast", "game review podcast", "esports podcast"}},
	{"Architecture", []string{"architecture podcast", "design podcast buildings", "urban design podcast"}},
	{"Aviation", []string{"aviation podcast", "pilot podcast", "flying podcast", "aircraft podcast"}},
	{"Cars", []string{"best car podcast", "automotive podcast", "car review podcast"}},
	{"Chess", []string{"chess podcast", "chess strategy podcast"}},
	{"Climate", []string{"best climate podcast", "climate change podcast", "environment podcast", "sustainability podcast"}},
	{"Coffee", []string{"coffee podcast", "barista podcast", "coffee brewing podcast"}},
	{"Documentary", []string{"documentary podcast", "investigative journalism podcast", "long form podcast"}},
	{"Law", []string{"best law podcast", "legal podcast", "court podcast", "justice podcast"}},
	{"Medicine", []string{"best medicine podcast", "medical podcast", "doctor podcast", "healthcare podcast"}},
	{"Military", []string{"military podcast", "war podcast", "military history podcast", "veterans podcast"}},
	{"Mythology", []string{"mythology podcast", "myths and legends podcast", "folklore podcast"}},
	{"Stoicism", []string{"stoicism podcast", "stoic philosophy podcast", "marcus aurelius podcast", "daily stoic"}},
	{"Theater", []string{"theater podcast", "broadway podcast", "acting podcast"}},
	{"Visual Arts", []string{"art podcast", "art history podcast", "contemporary art podcast"}},
	{"Wine", []string{"wine podcast", "wine tasting podcast", "sommelier podcast"}},
	{"Beer", []string{"beer podcast", "craft beer podcast", "brewing podcast"}},
	{"Camping", []string{"camping podcast", "outdoors podcast", "wilderness podcast", "survival podcast"}},
	{"Fishing", []string{"fishing podcast", "fly fishing podcast", "angling podcast"}},
	{"Hiking", []string{"hiking podcast", "backpacking podcast", "trails podcast"}},
	{"Language Learning", []string{"best language learning podcast", "learn language podcast", "polyglot podcast"}},
	{"Skepticism", []string{"skeptic podcast", "critical thinking podcast", "debunking podcast", "science vs podcast"}},
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

			// Skip shows with no images (usually garbage/test shows)
			if len(show.Images) == 0 {
				continue
			}

			// Skip shows with very short descriptions (low quality signal)
			if len(show.Description) < 20 {
				continue
			}

			podcastShow := PodcastShow{
				ID:           show.ID.String(),
				Name:         show.Name,
				Publisher:    show.Publisher,
				Description:  stripHTML(show.Description),
				Languages:    show.Languages,
				EpisodeCount: 0,
				Explicit:     show.Explicit,
				ExternalURL:  show.ExternalURLs["spotify"],
				MediaType:    show.MediaType,
			}

			// Get best image
			podcastShow.ImageURL = show.Images[0].URL

			allShows = append(allShows, podcastShow)
			count++
		}

		log.Printf("Found %d shows for query '%s'", count, query)
	}

	return allShows, nil
}

// stripHTML removes HTML tags from text
func stripHTML(s string) string {
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
