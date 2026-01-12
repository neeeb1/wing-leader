package server

import (
	"net/http"

	"github.com/neeeb1/rate_birds/internal/birds"
	"github.com/rs/zerolog/log"
)

func StartServer(cfg *birds.ApiConfig) {
	mux := http.NewServeMux()
	birds.RegisterEndpoints(mux, cfg)

	server := http.Server{}

	server.Handler = mux
	server.Addr = ":8080"

	log.Info().Msgf("now serving on 127.0.0.1:%s", server.Addr)
	server.ListenAndServe()
}
