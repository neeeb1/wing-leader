package server

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/neeeb1/rate_birds/internal/api"
	"github.com/rs/zerolog/log"
)

const cleanUpInterval = 10 * time.Minute

func StartServer(cfg *api.ApiConfig) (*http.Server, error) {
	mux := http.NewServeMux()
	api.RegisterEndpoints(mux, cfg)
	cleanUpTokens(cfg)

	server := http.Server{}

	server.Handler = mux

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server.Addr = ":" + port

	log.Info().Msgf("Server configured to listen on port %s", port)

	return &server, nil
}

func cleanUpTokens(cfg *api.ApiConfig) {
	if cfg.DbQueries == nil {
		return
	}
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
