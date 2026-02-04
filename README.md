# Melodex v2 - Music Discovery Scraper

Melodex is a comprehensive music discovery service that scrapes multiple music sources, enriches tracks via Spotify and MusicBrainz APIs, and stores them in Firestore for consumption by the BeatBrain discovery frontend.

## Architecture Overview

The system consists of several components:

- **Scrapers**: Collect track data from various music sources
- **Enrichment Pipeline**: Uses Spotify and MusicBrainz APIs to add metadata (ISRC, MBID, thumbnails)
- **Firestore Storage**: Stores enriched track data with TTL cleanup
- **Scoring Algorithm**: Ranks tracks by discovery potential across sources
- **REST API**: Provides endpoints for triggering scrapes and data access

## Music Sources

| Source | Description | Weight | Collection |
|--------|-------------|---------|------------|
| **Spotify New Releases** | Latest album releases via Spotify API | 1.0 | `spotify_new_releases` |
| **Reddit Fresh** | [FRESH] posts from r/listentothis and r/hiphopheads | 0.9 | `reddit_fresh` |
| **Hot New Hip Hop** | HNHH Top 100 chart scrape | 0.7 | `hnhh` |
| **Pitchfork Best New Music** | Pitchfork Best New Tracks page | 0.6 | `pitchfork_bnm` |
| **Billboard Hot 100** | Billboard Hot 100 chart scrape | 0.5 | `billboard` |

## Scoring Algorithm

The scoring system ranks tracks by discovery potential using:

```
score = (source_weight × normalized_rank) + freshness_bonus + cross_source_bonus
```

### Components:

- **Source Weight**: Higher weight sources contribute more to base score
- **Normalized Rank**: Lower chart positions get higher scores (rank 1 = 1.0, rank 100 = 0.0)
- **Freshness Bonus**: Recent tracks (<24h) get up to 0.2 bonus, decaying over a week
- **Cross-Source Bonus**: Tracks appearing in multiple sources get up to 0.3 bonus

### Deduplication:

Tracks are deduplicated by normalized artist+title matching, keeping the version from the highest-weight source while preserving all metadata and combining source information.

## Firestore Schema

### Collections Structure

Each collection stores daily documents with date keys (YYYY-MM-DD):

```
melodex/
├── billboard/2024-02-04
├── spotify_new_releases/2024-02-04  
├── reddit_fresh/2024-02-04
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
- `reddit-fresh`
- `pitchfork-bnm`
- `hot-new-hip-hop`
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

# Test Reddit fresh posts  
curl -X POST http://localhost:8080/scrape?debug=true \
  -H "Content-Type: application/json" \
  -d '{"target": "reddit-fresh"}'
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

4. **Update scoring** in `scoring/scoring.go`:
   - Add source weight to `getSourceWeight()` function
   - Update documentation

5. **Update cleanup** in `firestore/cleanup.go`:
   - Add new collection to `AllCollections()` function

## Development Notes

### Rate Limiting

- **MusicBrainz**: 3-second delay between requests (per their terms)
- **Spotify**: Handled by client library with automatic retries
- **Reddit**: Use proper User-Agent header to avoid blocking

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