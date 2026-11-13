package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	spot "github.com/zmb3/spotify/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	fs "melodex/firestore"
)

// Version 2 means all provider requests completed. Unmatched songs may still
// lack a Spotify link; they are retried on the next day's chart.
type musicSnapshot struct {
	Tracks  []fs.Track `firestore:"tracks"`
	Version int        `firestore:"version"`
}
type musicStore interface {
	load(context.Context, string, string) (musicSnapshot, error)
	save(context.Context, string, string, []fs.Track) error
}
type firestoreMusicStore struct{ db *firestore.Client }

func (s firestoreMusicStore) load(ctx context.Context, collection, date string) (musicSnapshot, error) {
	var data musicSnapshot
	doc, err := s.db.Collection(collection).Doc(date).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return data, nil
	}
	if err != nil {
		return data, err
	}
	err = doc.DataTo(&data)
	return data, err
}
func (s firestoreMusicStore) save(ctx context.Context, collection, date string, tracks []fs.Track) error {
	_, err := s.db.Collection(collection).Doc(date).Set(ctx, map[string]interface{}{"tracks": tracks, "version": 2, "fetchedAt": time.Now().UTC()}, firestore.MergeAll)
	return err
}

func reusableTrack(track fs.Track) bool {
	if len(track.SpotifyID) != 22 || track.Thumb == "" {
		return false
	}
	for _, c := range track.SpotifyID {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// Read only until a usable snapshot is found. A forced refresh keeps the
// current snapshot available as metadata, but never skips the source fetch.
func (h *ScrapeHandler) musicCache(ctx context.Context, collection string, now time.Time, refresh bool) (map[string]fs.Track, bool, error) {
	cache := map[string]fs.Track{}
	for day := 0; day <= 5; day++ {
		snapshot, err := h.store.load(ctx, collection, now.AddDate(0, 0, -day).Format("2006-01-02"))
		if err != nil {
			return nil, false, err
		}
		if day == 0 && snapshot.Version == 2 && len(snapshot.Tracks) > 0 && !refresh {
			return cache, true, nil
		}
		for _, track := range snapshot.Tracks {
			if reusableTrack(track) {
				cache[track.Artist+" - "+track.Title] = track
			}
		}
		if len(cache) > 0 {
			break
		}
	}
	return cache, false, nil
}

func (h *ScrapeHandler) handleMusicChart(w http.ResponseWriter, r *http.Request, collection string, fetch func(context.Context, http.ResponseWriter) ([]fs.Song, error), exact bool) {
	ctx := r.Context()
	now := time.Now().UTC()
	debug := r.URL.Query().Get("debug") == "true"
	cache := map[string]fs.Track{}
	if !debug {
		var skip bool
		var err error
		cache, skip, err = h.musicCache(ctx, collection, now, r.URL.Query().Get("refresh") == "true")
		if err != nil {
			log.Printf("source=%s stage=read error=%v", collection, err)
			writeScrapeError(w, 500, "source cache unavailable")
			return
		}
		if skip {
			writeScrapeError(w, 409, "data for today already exists")
			return
		}
	}
	// Legacy scrapers also write errors. Keep them out of the API response.
	songs, err := fetch(ctx, &sourceResponse{header: make(http.Header)})
	if err != nil || len(songs) == 0 {
		log.Printf("source=%s stage=fetch error=%v songs=%d", collection, err, len(songs))
		writeScrapeError(w, 502, "source unavailable")
		return
	}
	tracks := make([]fs.Track, 0, len(songs))
	matched := 0
	reused := 0
	for _, song := range songs {
		if ctx.Err() != nil {
			writeScrapeError(w, 504, "source request expired")
			return
		}
		if track, ok := cache[song.Artist+" - "+song.Title]; ok {
			track.Rank = song.Rank
			track.Source = collection
			tracks = append(tracks, track)
			matched++
			reused++
			continue
		}
		results, err := h.sp.Client.Search(ctx, buildSpotifyQuery(song.Artist, song.Title), spot.SearchTypeTrack, spot.Limit(5))
		if err != nil {
			log.Printf("source=%s stage=match error=%v", collection, err)
			writeScrapeError(w, 502, "source matching unavailable")
			return
		}
		var match *spot.FullTrack
		if results != nil && results.Tracks != nil && len(results.Tracks.Tracks) > 0 {
			if exact {
				match = matchChartTrack(song, results.Tracks.Tracks)
			} else {
				match = &results.Tracks.Tracks[0]
			}
		}
		track := fs.Track{Rank: song.Rank, Artist: song.Artist, Title: song.Title, Source: collection, CreatedAt: now}
		if match != nil {
			track.SpotifyID = match.ID.String()
			track.ISRC = match.ExternalIDs["isrc"]
			for _, im := range match.Album.Images {
				if strings.HasPrefix(im.URL, "https://i.scdn.co/image/") {
					track.Thumb = strings.TrimPrefix(im.URL, "https://i.scdn.co/image/")
					if im.Width == 300 {
						break
					}
				}
			}
		}
		if reusableTrack(track) {
			matched++
		}
		tracks = append(tracks, track)
	}
	if matched == 0 {
		writeScrapeError(w, 502, "no source tracks matched")
		return
	}
	if !debug {
		if err := h.store.save(ctx, collection, now.Format("2006-01-02"), tracks); err != nil {
			log.Printf("source=%s stage=save error=%v", collection, err)
			writeScrapeError(w, 500, "source save failed")
			return
		}
	}
	log.Printf("source=%s outcome=success tracks=%d matched=%d reused=%d", collection, len(tracks), matched, reused)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tracks)
}

func requireTracks(tracks []fs.Track) error {
	if len(tracks) == 0 {
		return fmt.Errorf("source returned no tracks")
	}
	return nil
}
