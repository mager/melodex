package main

import (
	"context"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"

	"github.com/gorilla/mux"
	"go.uber.org/fx"

	cfg "melodex/config"
	fs "melodex/firestore"
	h "melodex/handlers"
	mb "melodex/musicbrainz"
	spot "melodex/spotify"
)

func main() {
	fx.New(
		fx.Provide(
			NewRouter,
			fs.Options,
			cfg.Options,
			spot.Options,
			mb.Options,
		),
		fx.Invoke(StartServer),
	).Run()
}

func NewRouter(
	db *firestore.Client,
	sp *spot.SpotifyClient,
	mb *mb.MusicbrainzClient,
) *mux.Router {
	r := mux.NewRouter()

	scrapeHandler := h.NewScrapeHandler(db, sp, mb)
	r.HandleFunc("/scrape", scrapeHandler.Handle).Methods("POST")

	whosampledHandler := h.NewWhoSampledHandler(db, sp)
	r.HandleFunc("/whosampled", whosampledHandler.Handle).Methods("POST")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("API is running"))
	})
	return r
}

func StartServer(lifecycle fx.Lifecycle, router *mux.Router) {
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				log.Printf("Starting server on :8080")
				if err := server.ListenAndServe(); err != nil {
					log.Printf("Server failed: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})
}
