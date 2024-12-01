package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/fx"

	"melodex/handlers"
)

func main() {
	fx.New(
		fx.Provide(NewRouter),
		fx.Invoke(StartServer),
	).Run()
}

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/scrape/{scrapeId}", handlers.HandleScrape).Methods("POST")

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
