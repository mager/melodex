package musicbrainz

import (
	"time"

	"github.com/mager/musicbrainz-go/musicbrainz"
)

type MusicbrainzClient struct {
	Client      *musicbrainz.MusicbrainzClient
	rateLimiter *RateLimiter
}

func ProvideMusicbrainz() *MusicbrainzClient {
	var c MusicbrainzClient
	c.Client = musicbrainz.NewMusicbrainzClient().
		WithUserAgent("beatbrain/melodex", "1.0.0", "https://github.com/mager/melodex")
	// MusicBrainz requires at least 3 seconds between requests
	c.rateLimiter = NewRateLimiter(3 * time.Second)
	return &c
}

// SearchRecordingsByISRC searches for recordings by ISRC with rate limiting
func (c *MusicbrainzClient) SearchRecordingsByISRC(req musicbrainz.SearchRecordingsByISRCRequest) (musicbrainz.SearchRecordingsByISRCResponse, error) {
	c.rateLimiter.Wait()
	return c.Client.SearchRecordingsByISRC(req)
}

// SearchRecordingsByArtistAndTrack searches for recordings by artist and track with rate limiting
func (c *MusicbrainzClient) SearchRecordingsByArtistAndTrack(req musicbrainz.SearchRecordingsByArtistAndTrackRequest) (musicbrainz.SearchRecordingsByArtistAndTrackResponse, error) {
	c.rateLimiter.Wait()
	return c.Client.SearchRecordingsByArtistAndTrack(req)
}

var Options = ProvideMusicbrainz
