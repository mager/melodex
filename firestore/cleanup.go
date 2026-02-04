package firestore

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// DefaultTTL is the default max age for documents (7 days).
const DefaultTTL = 7 * 24 * time.Hour

// AllCollections returns the list of all melodex Firestore collections.
func AllCollections() []string {
	return []string{
		"billboard",
		"hnhh",
		"spotify_new_releases",
		"reddit_fresh",
		"pitchfork_bnm",
	}
}

// CleanupOldDocuments deletes documents older than maxAge from the given collections.
// Document IDs are dates in YYYY-MM-DD format, so we parse the ID to determine age.
func CleanupOldDocuments(ctx context.Context, client *firestore.Client, collections []string, maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	cutoffStr := cutoff.Format("2006-01-02")
	totalDeleted := 0

	for _, collName := range collections {
		col := client.Collection(collName)
		iter := col.Documents(ctx)
		defer iter.Stop()

		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Error iterating %s: %v", collName, err)
				break
			}

			// Document ID is a date string (YYYY-MM-DD)
			docDate := doc.Ref.ID

			// If the document date is before our cutoff, delete it
			if docDate < cutoffStr {
				_, err := doc.Ref.Delete(ctx)
				if err != nil {
					log.Printf("Error deleting %s/%s: %v", collName, docDate, err)
					continue
				}
				log.Printf("Deleted old document: %s/%s", collName, docDate)
				totalDeleted++
			}
		}
	}

	log.Printf("Cleanup complete: deleted %d documents older than %s", totalDeleted, cutoffStr)
	return nil
}

// RunCleanup is a convenience wrapper that cleans all known collections with the default TTL.
func RunCleanup(ctx context.Context, client *firestore.Client) error {
	return CleanupOldDocuments(ctx, client, AllCollections(), DefaultTTL)
}
