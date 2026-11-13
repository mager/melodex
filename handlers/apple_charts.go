package handlers

import (
	"context"
	spot "github.com/zmb3/spotify/v2"
	fs "melodex/firestore"
	"melodex/scrapers"
	"net/http"
	"strings"
)

func (h *ScrapeHandler) HandleAppleCharts(w http.ResponseWriter, r *http.Request) {
	h.handleMusicChart(w, r, "apple_music_charts", func(ctx context.Context, w http.ResponseWriter) ([]fs.Song, error) {
		return scrapers.FetchAppleChart(ctx)
	}, true)
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
