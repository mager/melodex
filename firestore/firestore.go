package firestore

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"
)

type Track struct {
	// MBID is the MusicBrainz recording ID
	MBID   string `json:"mbid" firestore:"mbid"`
	Artist string `json:"artist" firestore:"artist"`
	Title  string `json:"title" firestore:"title"`
	Rank   int    `json:"rank" firestore:"rank"`

	// TODO: Deprecate these field
	Thumb     string `json:"thumb,omitempty" firestore:"thumb,omitempty"`
	ISRC      string `json:"isrc,omitempty" firestore:"isrc,omitempty"`
	SpotifyID string `json:"spotifyID,omitempty" firestore:"spotifyID,omitempty"`
	
	// New fields for melodex v2
	Source    string    `json:"source,omitempty" firestore:"source,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty" firestore:"createdAt,omitempty"`
}

// ProvideDB provides a firestore client
func ProvideDB() *firestore.Client {
	projectID := "beatbrain-dev"

	client, err := firestore.NewClient(context.TODO(), projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return client
}

var Options = ProvideDB

type Song struct {
	Rank   int    `json:"rank"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

// PodcastShow represents a podcast show for Firestore storage
type PodcastShow struct {
	ID           string    `json:"id" firestore:"id"`
	Name         string    `json:"name" firestore:"name"`
	Publisher    string    `json:"publisher" firestore:"publisher"`
	Description  string    `json:"description" firestore:"description"`
	Categories   []string  `json:"categories" firestore:"categories"`
	Languages    []string  `json:"languages" firestore:"languages"`
	ImageURL     string    `json:"imageURL" firestore:"imageURL"`
	EpisodeCount int       `json:"episodeCount" firestore:"episodeCount"`
	Explicit     bool      `json:"explicit" firestore:"explicit"`
	ExternalURL  string    `json:"externalURL" firestore:"externalURL"`
	MediaType    string    `json:"mediaType" firestore:"mediaType"`
	DiscoveredIn string    `json:"discoveredIn" firestore:"discoveredIn"` // Category that found it
	FirstSeenAt  time.Time `json:"firstSeenAt" firestore:"firstSeenAt"`
	LastUpdated  time.Time `json:"lastUpdated" firestore:"lastUpdated"`
}
