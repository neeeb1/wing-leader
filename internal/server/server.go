package server

import (
	"net/http"

	"github.com/neeeb1/rate_birds/internal/api"
	"github.com/rs/zerolog/log"
)

func StartServer(cfg *api.ApiConfig) {
	mux := http.NewServeMux()
	api.RegisterEndpoints(mux, cfg)

	server := http.Server{}

	server.Handler = mux
	server.Addr = ":8080"

	log.Info().Msgf("now serving at https://localhost:%s", server.Addr)
	server.ListenAndServe()
}
