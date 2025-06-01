package firestore

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
)

type Track struct {
	Rank   int    `json:"rank" firestore:"rank"`
	Artist string `json:"artist" firestore:"artist"`
	Title  string `json:"title" firestore:"title"`
	ISRC   string `json:"isrc" firestore:"isrc"`
	Thumb  string `json:"thumb" firestore:"thumb"`
	// TODO: Deprecate this field
	SpotifyID string `json:"spotifyID" firestore:"spotifyID"`
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
