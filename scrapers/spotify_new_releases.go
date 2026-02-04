package scrapers

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	fs "melodex/firestore"
	spot "melodex/spotify"

	"github.com/zmb3/spotify/v2"
)

// ScrapeSpotifyNewReleases fetches new releases from Spotify API.
func ScrapeSpotifyNewReleases(w http.ResponseWriter, sp *spot.SpotifyClient) ([]fs.Track, error) {
	ctx := context.Background()
	var tracks []fs.Track

	// Get new releases (albums)
	newReleases, err := sp.Client.NewReleases(ctx, spotify.Limit(50))
	if err != nil {
		log.Printf("Error getting new releases from Spotify: %v", err)
		if w != nil {
			http.Error(w, "Error getting new releases from Spotify", http.StatusInternalServerError)
		}
		return nil, err
	}

	rank := 1
	for _, album := range newReleases.Albums {
		// Get album tracks
		albumTracks, err := sp.Client.GetAlbumTracks(ctx, album.ID, spotify.Limit(50))
		if err != nil {
			log.Printf("Error getting tracks for album %s: %v", album.Name, err)
			continue
		}

		for _, track := range albumTracks.Tracks {
			// Get full track info to access external IDs
			fullTrack, err := sp.Client.GetTrack(ctx, track.ID)
			if err != nil {
				log.Printf("Error getting full track info for %s: %v", track.Name, err)
				continue
			}

			isrc := fullTrack.ExternalIDs["isrc"]
			var thumb string

			// Get thumbnail from album images — strip Spotify CDN prefix
			// to match existing convention (stored as just the image hash)
			for _, image := range album.Images {
				if image.Height == 300 && image.Width == 300 {
					thumb = strings.TrimPrefix(image.URL, "https://i.scdn.co/image/")
					break
				}
			}

			newTrack := fs.Track{
				Rank:      rank,
				Artist:    track.Artists[0].Name,
				Title:     track.Name,
				ISRC:      isrc,
				SpotifyID: track.ID.String(),
				Thumb:     thumb,
				Source:    "spotify_new_releases",
				CreatedAt: time.Now(),
			}

			tracks = append(tracks, newTrack)
			rank++

			// Limit to avoid too many tracks
			if rank > 100 {
				break
			}
		}

		if rank > 100 {
			break
		}
	}

	log.Printf("Scraped %d tracks from Spotify new releases", len(tracks))
	return tracks, nil
}

type ScrapeSpotifyNewReleasesFunc func(http.ResponseWriter, *spot.SpotifyClient) ([]fs.Track, error)