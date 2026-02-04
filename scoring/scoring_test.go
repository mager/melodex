package scoring

import (
	"testing"
)

func TestScoreTrack_HighRankFreshSpotify(t *testing.T) {
	track := ScoredTrack{
		Artist:       "Kendrick Lamar",
		Title:        "Not Like Us",
		Source:       "spotify_new_releases",
		Rank:         1,
		WeeksOnChart: 0,
		SourceCount:  1,
	}
	score := ScoreTrack(track)
	if score < 1.5 {
		t.Errorf("Expected high score for #1 fresh Spotify track, got %f", score)
	}
}

func TestScoreTrack_StaleChartEntry(t *testing.T) {
	track := ScoredTrack{
		Artist:       "Some Artist",
		Title:        "Old Song",
		Source:       "billboard",
		Rank:         80,
		WeeksOnChart: 30,
		SourceCount:  1,
	}
	score := ScoreTrack(track)
	if score > 0.5 {
		t.Errorf("Expected low score for stale Billboard entry, got %f", score)
	}
}

func TestScoreTrack_CrossSourceBonus(t *testing.T) {
	base := ScoredTrack{
		Artist:       "Artist",
		Title:        "Track",
		Source:       "hnhh",
		Rank:         5,
		WeeksOnChart: 0,
		SourceCount:  1,
	}
	scoreSingle := ScoreTrack(base)

	base.SourceCount = 3
	scoreMulti := ScoreTrack(base)

	if scoreMulti-scoreSingle < 0.49 {
		t.Errorf("Cross-source bonus should add ~0.5, got diff %f", scoreMulti-scoreSingle)
	}
}

func TestRankAndDeduplicate_Basic(t *testing.T) {
	tracks := []ScoredTrack{
		{Artist: "Kendrick Lamar", Title: "Track A", Source: "billboard", Rank: 1, WeeksOnChart: 0},
		{Artist: "Kendrick Lamar", Title: "Track A", Source: "hnhh", Rank: 5, WeeksOnChart: 0},
		{Artist: "SZA", Title: "Track B", Source: "spotify_new_releases", Rank: 1, WeeksOnChart: 0},
	}

	result := RankAndDeduplicate(tracks)

	if len(result) != 2 {
		t.Fatalf("Expected 2 deduplicated tracks, got %d", len(result))
	}

	// The Kendrick track should have SourceCount=2
	for _, r := range result {
		if r.Artist == "Kendrick Lamar" && r.SourceCount != 2 {
			t.Errorf("Kendrick track should have SourceCount=2, got %d", r.SourceCount)
		}
	}
}

func TestRankAndDeduplicate_FeatNormalization(t *testing.T) {
	tracks := []ScoredTrack{
		{Artist: "Drake feat. Future", Title: "Big Song", Source: "billboard", Rank: 1},
		{Artist: "Drake", Title: "Big Song", Source: "hnhh", Rank: 3},
	}

	result := RankAndDeduplicate(tracks)

	if len(result) != 1 {
		t.Fatalf("Expected feat. normalization to deduplicate, got %d tracks", len(result))
	}
	if result[0].SourceCount != 2 {
		t.Errorf("Expected SourceCount=2 after feat. dedup, got %d", result[0].SourceCount)
	}
}

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		artist, title, expected string
	}{
		{"Drake", "God's Plan", "drake - god's plan"},
		{"Drake feat. Future", "Life Is Good", "drake - life is good"},
		{"  Kendrick Lamar  ", "  HUMBLE.  ", "kendrick lamar - humble."},
		{"SZA ft. Travis Scott", "Love Galore", "sza - love galore"},
	}

	for _, tt := range tests {
		got := normalizeKey(tt.artist, tt.title)
		if got != tt.expected {
			t.Errorf("normalizeKey(%q, %q) = %q, want %q", tt.artist, tt.title, got, tt.expected)
		}
	}
}
