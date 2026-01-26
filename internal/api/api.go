package api

import (
	"database/sql"

	"net/http"

	"github.com/neeeb1/rate_birds/internal/database"
)

type ApiConfig struct {
	NuthatcherApiKey string
	DbURL            string
	DbQueries        *database.Queries
	Db               *sql.DB
	CacheHost        string
}

func RegisterEndpoints(mux *http.ServeMux, cfg *ApiConfig) {
	mux.Handle("/", http.FileServer(http.Dir("./web")))
	mux.HandleFunc("GET /api/scorematch/", cfg.handleScoreMatch)
	mux.HandleFunc("GET /api/leaderboard/", cfg.handleLoadLeaderboard)
	mux.HandleFunc("GET /api/image/", cfg.handleCachedImage)
	mux.HandleFunc("GET /api/loadbirds/", cfg.handleLoadBirds)
	//mux.Handle("/matches", http.HandlerFunc(cfg.handleLoadMatches))
	mux.HandleFunc("GET /health/live", HandleLiveness)
	mux.HandleFunc("GET /health/ready", cfg.HandleReadiness)
}
