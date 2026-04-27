package api

import (
	"database/sql"
	"encoding/json"

	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/neeeb1/rate_birds/internal/database"
	"github.com/neeeb1/rate_birds/internal/middleware"
	"github.com/rs/zerolog/log"
)

type ApiConfig struct {
	NuthatcherApiKey string
	DbURL            string
	DbQueries        *database.Queries
	Db               *sql.DB
	S3Client         *s3.Client
	BucketName       string
}

func RegisterEndpoints(mux *http.ServeMux, cfg *ApiConfig) {
	voteLimiter := middleware.NewIPRateLimiter(5, 10)
	leaderboardLimiter := middleware.NewIPRateLimiter(5, 10)

	mux.Handle("/", http.FileServer(http.Dir("./web")))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))

	//mux.Handle("/matches", http.HandlerFunc(cfg.handleLoadMatches))
	mux.Handle("GET /api/scorematch/",
		voteLimiter.Limit(http.HandlerFunc(cfg.handleScoreMatch)))

	mux.Handle("GET /api/leaderboard/",
		leaderboardLimiter.Limit(http.HandlerFunc(cfg.handleLoadLeaderboard)))

	mux.HandleFunc("GET /api/loadbirds/", cfg.handleLoadBirds)
	mux.HandleFunc("GET /bird/{id}", cfg.handleBirdDetail)

	mux.HandleFunc("GET /health/live", HandleLiveness)
	mux.HandleFunc("GET /health/ready", cfg.HandleReadiness)

	//mux.Handle("/metrics", promhttp.Handler())
}

func RespondWithError(w http.ResponseWriter, code int, msg string) {
	type errorResponse struct {
		Error string
	}

	respBody := errorResponse{
		Error: msg,
	}

	data, err := json.Marshal(respBody)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal error response to json")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}
