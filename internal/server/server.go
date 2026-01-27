package server

import (
	"context"
	"net/http"
	"time"

	"github.com/neeeb1/rate_birds/internal/api"
	"github.com/rs/zerolog/log"
)

const cleanUpInterval = 10 * time.Minute

func StartServer(cfg *api.ApiConfig) {
	mux := http.NewServeMux()
	api.RegisterEndpoints(mux, cfg)
	cleanUpTokens(cfg)

	server := http.Server{}

	server.Handler = mux
	server.Addr = ":8080"

	log.Info().Msgf("now serving at https://localhost:%s", server.Addr)
	server.ListenAndServe()
}

func cleanUpTokens(cfg *api.ApiConfig) {
	log.Info().Msg("Started token cleanup routine")
	ticker := time.NewTicker(cleanUpInterval)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				err := cfg.DbQueries.CleanUpSessions(context.Background())
				if err != nil {
					log.Error().Err(err).Msg("Failed to clean up expired tokens")
				}
				log.Info().Msgf("Cleaned up expired tokens at %v", t)
			}
		}
	}()
}
