package spotify

import (
	"context"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"time"

	"melodex/config"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

type SpotifyClient struct {
	ID     string
	Secret string
	Client *spotify.Client
}

func ProvideSpotify(cfg config.Config) *SpotifyClient {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Timeout: 15 * time.Second})

	var c SpotifyClient

	c.ID = cfg.SpotifyID
	c.Secret = cfg.SpotifySecret

	config := &clientcredentials.Config{
		ClientID:     c.ID,
		ClientSecret: c.Secret,
		TokenURL:     spotifyauth.TokenURL,
	}
	tokenSource := config.TokenSource(ctx)
	_, err := tokenSource.Token()
	if err != nil {
		log.Fatalf("couldn't get token: %v", err)
	}

	httpClient := oauth2.NewClient(ctx, tokenSource)
	httpClient.Timeout = 20 * time.Second
	client := spotify.New(httpClient)
	c.Client = client

	return &c
}

var Options = ProvideSpotify
