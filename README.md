# Melodex v2 - Music & Podcast Discovery Scraper

Melodex is a comprehensive discovery service that scrapes multiple music sources and podcast shows, matches tracks via Spotify, and stores them in Firestore for consumption by the BeatBrain discovery frontend.

## Architecture Overview

The system consists of several components:

- **Scrapers**: Collect track data from various music sources
- **Enrichment Pipeline**: Uses Spotify to add ISRCs and artwork. Occipital loads detailed MusicBrainz and sourced credits when a song page opens.
- **Firestore Storage**: Stores enriched track data with TTL cleanup
- **REST API**: Provides endpoints for triggering scrapes and data access

## Music Sources

| Source | Description | Weight | Collection |
|--------|-------------|---------|------------|
| **Spotify New Releases** | Latest album releases via Spotify API | 1.0 | `spotify_new_releases` |
| **Apple Music US Top 50** | Public [most-played song feed](https://rss.marketingtools.apple.com/api/v2/us/music/most-played/50/songs.json), exact title/artist matching | 0.9 | `apple_music_charts` |
| **Hot New Hip Hop** | HNHH Top 100 chart scrape | 0.7 | `hnhh` |
| **Pitchfork Best New Music** | Pitchfork Best New Tracks page | 0.6 | `pitchfork_bnm` |
| **Billboard Hot 100** | Billboard Hot 100 chart scrape | 0.5 | `billboard` |

## Podcast Discovery

Melodex now includes podcast show discovery across **100+ categories**, helping users find niche podcasts in specialized fields.

### Categories Include:
- Arts, Books, Business, Comedy, Education, Fiction, Government
- History, Health, Kids, Linguistics, Music, News, Religion
- Science, Society, Sports, Technology, True Crime, TV & Film
- **Niche specialties**: Astronomy, Blockchain, Cybersecurity, Linguistics, Stoicism, DJing, Firefighting, and 80+ more

### API Endpoints

#### POST /podcasts/scrape
Discover podcast shows across categories.

**Request Body:**
```json
{
  "category": "Linguistics",    // Optional: specific category
  "queries": ["syntax", "grammar"], // Optional: custom search queries
  "maxShows": 20                // Optional: max shows per query (default: 20)
}
```

**Response:**
```json
[
  {
    "category": "Linguistics",
    "showsFound": 45,
    "newShows": 12,
    "message": "Success"
  }
]
```

**To scrape all categories:** Send empty body `{}` or omit body entirely.

#### GET /podcasts/categories
Returns all 100+ available podcast categories.

**Response:**
```json
{
  "count": 100,
  "categories": ["Arts", "Books", "Business", ...]
}
```

### Firestore Schema

Podcast shows are stored in the `podcast_shows` collection:

```json
{
  "id": "spotify-show-id",
  "name": "Show Name",
  "publisher": "Publisher Name",
  "description": "Show description...",
  "categories": ["Linguistics", "Education"],
  "languages": ["en"],
  "imageURL": "https://...",
  "episodeCount": 150,
  "explicit": false,
  "externalURL": "https://open.spotify.com/show/...",
  "mediaType": "audio",
  "discoveredIn": "Linguistics",
  "firstSeenAt": "2024-03-12T20:00:00Z",
  "lastUpdated": "2024-03-12T20:00:00Z"
}
```

Note: Podcast shows are **not** TTL-cleared — they persist as a catalog for discovery.

## Discovery ranking

Melodex stores each source's rank and metadata. Occipital ranks and deduplicates the tracks when it serves discovery results. Melodex has no separate scoring package.

## Firestore Schema

### Collections Structure

Each collection stores daily documents with date keys (YYYY-MM-DD):

```
melodex/
├── billboard/2024-02-04
├── spotify_new_releases/2024-02-04
├── apple_music_charts/2026-11-09
├── pitchfork_bnm/2024-02-04
└── hnhh/2024-02-04
```

### Document Structure

```json
{
  "tracks": [
    {
      "artist": "Artist Name",
      "title": "Track Title",
      "rank": 1,
      "mbid": "musicbrainz-recording-id",
      "isrc": "ISRC-CODE",
      "spotifyID": "spotify-track-id",
      "thumb": "album-thumbnail-url",
      "source": "spotify_new_releases",
      "createdAt": "2024-02-04T16:23:00Z"
    }
  ]
}
```

### TTL Policy

- **Retention**: 7 days by default
- **Cleanup**: Runs automatically after each full scrape cycle
- **Collections**: All source collections are cleaned up together

## API Endpoints

### POST /scrape

Triggers scraping for specific sources or all sources.

**Request Body:**
```json
{
  "target": "spotify-new-releases"  // optional
}
```

**Valid Targets:**
- `billboard-hot-100`
- `spotify-new-releases`
- `apple-music-charts`
- `pitchfork-bnm`
- `hot-new-hip-hop`
- `spotify-podcasts` - Podcast show discovery
- Empty/null = all sources

**Query Parameters:**
- `?debug=true` - Skip database checks and saves

### GET /

Health check endpoint - returns "API is running"

## Environment Variables

| Variable | Description | Required |
|----------|-------------|-----------|
| `SPOTIFY_CLIENT_ID` | Spotify API client ID | Yes |
| `SPOTIFY_CLIENT_SECRET` | Spotify API client secret | Yes |
| `FIRESTORE_PROJECT_ID` | Google Cloud project ID | Yes (defaults to "beatbrain-dev") |

## Running Locally

### Prerequisites

1. **Go 1.23+** installed
2. **Google Cloud credentials** configured for Firestore access
3. **Spotify API credentials** from [Spotify Developer Dashboard](https://developer.spotify.com/)

### Setup

1. Clone and install dependencies:
```bash
git clone <repo-url> melodex
cd melodex
go mod download
```

2. Set environment variables:
```bash
export SPOTIFY_CLIENT_ID="your-client-id"
export SPOTIFY_CLIENT_SECRET="your-client-secret"
export FIRESTORE_PROJECT_ID="beatbrain-dev"
```

3. Authenticate with Google Cloud:
```bash
gcloud auth application-default login
```

4. Run the service:
```bash
go run main.go
```

The API will be available at `http://localhost:8080`

### Testing Individual Sources

Test specific scrapers with debug mode:

```bash
# Test Spotify new releases
curl -X POST http://localhost:8080/scrape?debug=true \
  -H "Content-Type: application/json" \
  -d '{"target": "spotify-new-releases"}'

# Test Apple Music US Top 50
curl -X POST http://localhost:8080/scrape?debug=true \
  -H "Content-Type: application/json" \
  -d '{"target": "apple-music-charts"}'
```

## Adding New Sources

To add a new music source:

1. **Create scraper** in `scrapers/new_source.go`:
   - Return `[]fs.Song` for basic data or `[]fs.Track` for enriched data
   - Follow existing patterns for error handling and logging

2. **Create handler** in `handlers/new_source.go`:
   - Implement enrichment pipeline (Spotify search → ISRC → MBID)
   - Save to new Firestore collection with date-keyed documents
   - Add Source and CreatedAt fields

3. **Update routing** in `handlers/scrape.go`:
   - Add new target to valid targets list
   - Add case to switch statement
   - Add goroutine to default concurrent execution

4. **Update cleanup** in `firestore/cleanup.go`:
   - Add new collection to `AllCollections()` function

## Development Notes

### Rate Limiting

- **MusicBrainz**: 3-second delay between requests (per their terms)
- **Spotify**: One shared queue, at most two requests per second, with a shared `Retry-After` cooldown and at most two GET retries. Requests cancel while queued.
- **Apple Music**: Public US chart feed; exact title and artist matching before Spotify links.

### Error Handling

- Individual scraper failures don't stop other scrapers
- Missing metadata (MBID, ISRC) doesn't prevent track storage
- Graceful degradation with logging at each step

### Performance

- Concurrent scraping of all sources by default
- Yesterday's data reuse to avoid redundant API calls
- Batch Firestore operations for efficient storage

### Dependencies

- **Colly**: Web scraping framework
- **Spotify SDK**: Official Spotify API client
- **MusicBrainz Go**: Custom MusicBrainz API client
- **Firestore**: Google Cloud document database
- **Uber FX**: Dependency injection framework

## License

[Add your license here]

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add comprehensive tests for new sources
4. Update documentation
5. Submit a pull request

---

Built with ❤️ for music discovery at BeatBrain.

## Deployment notes: November 2026

- The scheduled `POST /scrape` request uses `{"target":"music"}`. The five music sources run; podcast ingestion remains an explicit separate request.
- A full scrape now returns one JSON object with a `results` array. Each result has `source`, `status`, and `outcome` (`success`, `skipped`, or `failed`). HTTP 409 from a source means its daily data already exists and is reported as `skipped`.
- The full request returns HTTP 200 when every source succeeds or is skipped. It returns HTTP 502 and the partial results when a source fails. This prevents an early source response from hiding another source's failure.
- Each source has a separate response writer and request body. Large source response bodies are discarded from the summary. Direct single-source and podcast routes keep their existing response formats.
- Removed the unused `/whosampled` endpoint, `testing` scrape target, browser automation dependency, and unused scoring package. No stored data is removed by these code changes.
- Require Cloud Run IAM authentication for this ingestion service and configure the existing scheduler with an OIDC service account that has permission to invoke it. Do not expose scrape routes to the browser.
- Existing seven-day discovery cleanup still runs after a full scrape. Podcast records are excluded. No cleanup is run by the tests.
- Verify with `go test ./...`, `go test -race ./...`, and `go vet ./...`. Check the scheduler response and per-source statuses after deployment; upstream source failures are still possible.

## Daily music job

Cloud Scheduler `update-discover-index` in `beatbrain-dev/us-central1` runs at `0 5 * * *` UTC. It posts `{"target":"music"}` to `/scrape`. Podcasts are separate from this schedule. Each source reports success, skipped (already saved today), or failed; failed sources do not replace good cached data. Reddit was removed because approved API access was unavailable.

## Discovery repair: November 13, 2026

- Chart sources share one ingestion path. Only tracks with a valid Spotify ID and artwork are reused; missing matches are searched again on the next daily run. Reused tracks receive the current chart rank and retain their metadata timestamp.
- Version 2 snapshots mean all provider requests completed. A provider failure aborts the source save, leaving the last good snapshot intact. A no-match response is not a provider failure. Old snapshots are refreshed once after this change.
- `?refresh=true` refreshes a completed day without deleting the saved data. This remains an IAM-protected operator action. `?debug=true` skips database reads, saves, and source-health writes.
- The five small `discovery_health` records hold each source's latest attempt. Both full and targeted runs record their outcomes. Structured logs include source, HTTP status, outcome, and time; matching logs include total, matched, and reused counts.
- Cloud Run uses request-based billing, zero minimum instances, one maximum instance, and concurrency one for ingestion. The daily scheduler has two bounded retries with a 15-minute minimum backoff. Completed sources skip on a retry. No new cloud service or paid cache is needed.
- Spotify pacing applies to this Melodex process. Spotify limits are shared at the app level, so other services can still cause a cooldown. Respect `Retry-After`; do not assume two requests per second is a provider quota guarantee.
