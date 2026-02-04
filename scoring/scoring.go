package scoring

import (
	"sort"
	"strings"
	"time"
)

type ScoredTrack struct {
	// Embed or reference the firestore.Track fields
	Artist    string    `json:"artist"`
	Title     string    `json:"title"`
	MBID      string    `json:"mbid,omitempty"`
	ISRC      string    `json:"isrc,omitempty"`
	SpotifyID string    `json:"spotifyID,omitempty"`
	Thumb     string    `json:"thumb,omitempty"`
	Source    string    `json:"source"`
	Rank      int       `json:"rank"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	
	// Scoring fields
	Score        float64 `json:"score"`
	WeeksOnChart int     `json:"weeksOnChart,omitempty"`
	SourceCount  int     `json:"sourceCount"`
}

// ScoreTrack computes a composite discovery score
// score = (source_weight * normalized_rank) + freshness_bonus + cross_source_bonus
func ScoreTrack(track ScoredTrack) float64 {
	sourceWeight := getSourceWeight(track.Source)
	
	// Normalize rank (lower rank = higher score)
	// For ranks 1-100, map to 1.0-0.0
	normalizedRank := 1.0 - (float64(track.Rank-1) / 100.0)
	if normalizedRank < 0 {
		normalizedRank = 0
	}
	
	// Base score from source weight and rank
	baseScore := sourceWeight * normalizedRank
	
	// Freshness bonus - newer content gets higher score
	freshnessBonusScore := calculateFreshnessBonus(track.CreatedAt)
	
	// Cross-source bonus - tracks appearing in multiple sources get bonus
	crossSourceBonus := float64(track.SourceCount-1) * 0.1
	if crossSourceBonus > 0.3 {
		crossSourceBonus = 0.3 // Cap at 30% bonus
	}
	
	finalScore := baseScore + freshnessBonusScore + crossSourceBonus
	return finalScore
}

// getSourceWeight returns the weight for each source
func getSourceWeight(source string) float64 {
	weights := map[string]float64{
		"spotify_new_releases": 1.0,
		"reddit_fresh":         0.9,
		"hnhh":                 0.7,
		"pitchfork_bnm":        0.6,
		"billboard":            0.5,
	}
	
	if weight, exists := weights[source]; exists {
		return weight
	}
	
	// Default weight for unknown sources
	return 0.3
}

// calculateFreshnessBonus gives bonus points for recently created tracks
func calculateFreshnessBonus(createdAt time.Time) float64 {
	if createdAt.IsZero() {
		return 0
	}
	
	hoursOld := time.Since(createdAt).Hours()
	
	// Fresh content (< 24 hours) gets maximum bonus
	if hoursOld < 24 {
		return 0.2
	}
	
	// Gradually decrease bonus over a week
	if hoursOld < 168 { // 7 days
		return 0.2 * (1.0 - (hoursOld-24)/144) // Linear decay from 0.2 to 0
	}
	
	return 0
}

// normalizeArtistTitle creates a normalized version for deduplication
func normalizeArtistTitle(artist, title string) string {
	normalizeStr := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "-", "")
		s = strings.ReplaceAll(s, "_", "")
		s = strings.ReplaceAll(s, "'", "")
		s = strings.ReplaceAll(s, "\"", "")
		return s
	}
	
	return normalizeStr(artist) + "|" + normalizeStr(title)
}

// RankAndDeduplicate takes tracks from all sources, scores them, 
// deduplicates by normalized artist+title, and returns sorted by score descending
func RankAndDeduplicate(tracks []ScoredTrack) []ScoredTrack {
	// Group tracks by normalized artist+title for deduplication
	trackMap := make(map[string][]ScoredTrack)
	
	for _, track := range tracks {
		key := normalizeArtistTitle(track.Artist, track.Title)
		trackMap[key] = append(trackMap[key], track)
	}
	
	var deduplicatedTracks []ScoredTrack
	
	// Process each group of duplicate tracks
	for _, trackGroup := range trackMap {
		if len(trackGroup) == 1 {
			// Single track, just calculate its score
			track := trackGroup[0]
			track.SourceCount = 1
			track.Score = ScoreTrack(track)
			deduplicatedTracks = append(deduplicatedTracks, track)
		} else {
			// Multiple tracks - merge them intelligently
			bestTrack := mergeDuplicateTracks(trackGroup)
			deduplicatedTracks = append(deduplicatedTracks, bestTrack)
		}
	}
	
	// Sort by score descending
	sort.Slice(deduplicatedTracks, func(i, j int) bool {
		return deduplicatedTracks[i].Score > deduplicatedTracks[j].Score
	})
	
	return deduplicatedTracks
}

// mergeDuplicateTracks takes duplicate tracks and merges them into the best version
func mergeDuplicateTracks(tracks []ScoredTrack) ScoredTrack {
	// Start with the track from the highest-weight source
	bestTrack := tracks[0]
	bestWeight := getSourceWeight(tracks[0].Source)
	
	for _, track := range tracks[1:] {
		weight := getSourceWeight(track.Source)
		if weight > bestWeight {
			bestTrack = track
			bestWeight = weight
		}
	}
	
	// Set source count for cross-source bonus
	bestTrack.SourceCount = len(tracks)
	
	// Combine sources for reference
	sources := make([]string, len(tracks))
	for i, track := range tracks {
		sources[i] = track.Source
	}
	bestTrack.Source = strings.Join(sources, ",")
	
	// Use the best available metadata
	for _, track := range tracks {
		if bestTrack.MBID == "" && track.MBID != "" {
			bestTrack.MBID = track.MBID
		}
		if bestTrack.ISRC == "" && track.ISRC != "" {
			bestTrack.ISRC = track.ISRC
		}
		if bestTrack.SpotifyID == "" && track.SpotifyID != "" {
			bestTrack.SpotifyID = track.SpotifyID
		}
		if bestTrack.Thumb == "" && track.Thumb != "" {
			bestTrack.Thumb = track.Thumb
		}
	}
	
	// Calculate final score
	bestTrack.Score = ScoreTrack(bestTrack)
	
	return bestTrack
}