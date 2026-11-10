package handlers

import (
	"encoding/json"
	spot "github.com/zmb3/spotify/v2"
	"log"
	fs "melodex/firestore"
	"melodex/scrapers"
	"net/http"
	"strings"
	"time"
)

func (h *ScrapeHandler) HandleAppleCharts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	debug := r.URL.Query().Get("debug") == "true"
	today := time.Now().UTC().Format("2006-01-02")
	if !debug {
		doc, err := h.db.Collection("apple_music_charts").Doc(today).Get(ctx)
		if err == nil && doc.Exists() {
			writeScrapeError(w, 409, "data for today already exists")
			return
		}
	}
	songs, err := scrapers.FetchAppleChart(ctx)
	if err != nil {
		log.Printf("Apple chart failed: %v", err)
		writeScrapeError(w, 502, "Apple chart unavailable")
		return
	}
	tracks := make([]fs.Track, 0, len(songs))
	for _, song := range songs {
		if ctx.Err() != nil {
			writeScrapeError(w, 504, "chart request expired")
			return
		}
		results, err := h.sp.Client.Search(ctx, buildSpotifyQuery(song.Artist, song.Title), spot.SearchTypeTrack, spot.Limit(5))
		if err != nil {
			writeScrapeError(w, 502, "chart matching unavailable")
			return
		}
		if results.Tracks == nil {
			continue
		}
		match := matchChartTrack(song, results.Tracks.Tracks)
		if match == nil {
			continue
		}
		thumb := ""
		for _, im := range match.Album.Images {
			if strings.HasPrefix(im.URL, "https://i.scdn.co/image/") {
				thumb = strings.TrimPrefix(im.URL, "https://i.scdn.co/image/")
				if im.Width == 300 {
					break
				}
			}
		}
		if thumb == "" {
			continue
		}
		tracks = append(tracks, fs.Track{Rank: song.Rank, Artist: song.Artist, Title: song.Title, SpotifyID: match.ID.String(), ISRC: match.ExternalIDs["isrc"], Thumb: thumb, Source: "apple_music_charts", CreatedAt: time.Now().UTC()})
	}
	if len(tracks) == 0 {
		writeScrapeError(w, 502, "no chart tracks matched")
		return
	}
	if !debug {
		_, err = h.db.Collection("apple_music_charts").Doc(today).Set(ctx, map[string]interface{}{"tracks": tracks, "source_url": scrapers.AppleChartURL, "market": "US"})
		if err != nil {
			writeScrapeError(w, 500, "chart save failed")
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}
func chartName(value string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("&", " ", ",", " ", "’", "'").Replace(strings.ToLower(value))), " ")
}
func matchChartTrack(song fs.Song, tracks []spot.FullTrack) *spot.FullTrack {
	for i := range tracks {
		track := &tracks[i]
		names := []string{}
		for _, artist := range track.Artists {
			names = append(names, artist.Name)
		}
		if track.ID != "" && chartName(track.Name) == chartName(song.Title) && chartName(strings.Join(names, " ")) == chartName(song.Artist) {
			return track
		}
	}
	return nil
}
