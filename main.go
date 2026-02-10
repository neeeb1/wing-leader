package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/neeeb1/rate_birds/internal/api"
	"github.com/neeeb1/rate_birds/internal/database"
	"github.com/neeeb1/rate_birds/internal/server"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Intialize zerolog and pretty console output
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Perform a check to see if we are running in a docker container
	// If not, load the .env file for local development
	if !isRunningInDockerContainer() {
		log.Info().Msg("Running locally, loading .env")
		err := godotenv.Load()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to load .env")
			return
		}
	} else {
		log.Info().Msg("Running in container, skipping .env load")
	}

	// Configure api config
	apiCfg := api.ApiConfig{}
	apiCfg.NuthatcherApiKey = os.Getenv("NUTHATCH_KEY")
	apiCfg.DbURL = os.Getenv("DB_URL")
	apiCfg.CacheHost = os.Getenv("CACHE_HOST")

	// Intialize database connection
	// defer closing til program end
	db, err := sql.Open("postgres", apiCfg.DbURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open db")
		return
	}
	defer db.Close()
	// Confirm database is reachable
	err = db.Ping()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to ping db")
		return
	}
	// Set database connection pool settings
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(10 * time.Minute)

	// Add database to api config
	apiCfg.Db = db
	apiCfg.DbQueries = database.New(db)

	log.Info().Msg("apicfg loaded")

	// Populate bird database if unintilaized
	count, err := apiCfg.DbQueries.GetTotalBirdCount(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to count db entries")
		return
	}
	if count == 0 {
		err = apiCfg.PopulateBirdDB()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to populate birds")
			return
		}
	} else {
		log.Info().Msg("Bird db already populated - skipping initial population...")
	}

	// Populate ratings database
	err = apiCfg.PopulateRatingsDB()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to populate ratings")
		return
	}

	// Start async caching of remote images
	go func() {
		err = apiCfg.CacheImages()
	}()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to cache remote images")
		return
	}

	// Start the server
	server, err := server.StartServer(&apiCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
		return
	}

	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced shutdown")
	}

	log.Info().Msg("Server exited cleanly")
}

func isRunningInDockerContainer() bool {
	// docker creates a .dockerenv file at the root
	// of the directory tree inside the container.
	// if this file exists then the viewer is running
	// from inside a container so return true

	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	return false
}
