package handlers

import (
	"log"

	mb "github.com/mager/musicbrainz-go/musicbrainz"
)

// FindMBID attempts to find a MusicBrainz ID for a track using ISRC first,
// then falling back to artist and title search if ISRC is not available.
// Returns the MBID if found, empty string if not found.
func (h *ScrapeHandler) FindMBID(isrc, artist, title string) string {
	if isrc != "" {
		searchRecsReq := mb.SearchRecordingsByISRCRequest{
			ISRC: isrc,
		}
		recs, err := h.mb.Client.SearchRecordingsByISRC(searchRecsReq)
		if err != nil {
			log.Printf("Error getting MBID by ISRC: %v", err)
		}
		if len(recs.Recordings) > 0 {
			return recs.Recordings[0].ID
		}
	}

	// Fall back to artist and title search
	searchRecsReq := mb.SearchRecordingsByArtistAndTrackRequest{
		Artist: artist,
		Track:  title,
	}
	recs, err := h.mb.Client.SearchRecordingsByArtistAndTrack(searchRecsReq)
	if err != nil {
		log.Printf("Error getting MBID by title and artist: %v", err)
	}
	if len(recs.Recordings) > 0 {
		return recs.Recordings[0].ID
	}

	log.Printf("No MBID found for track: %s by %s", title, artist)
	return ""
}
