package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/neeeb1/rate_birds/internal/birds"
	"github.com/neeeb1/rate_birds/internal/database"
	"github.com/neeeb1/rate_birds/internal/server"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

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

	apiCfg := birds.ApiConfig{}

	apiCfg.NuthatcherApiKey = os.Getenv("NUTHATCH_KEY")
	apiCfg.DbURL = os.Getenv("DB_URL")
	apiCfg.CacheHost = os.Getenv("CACHE_HOST")

	db, err := sql.Open("postgres", apiCfg.DbURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open db")
		return
	}
	apiCfg.Db = db
	apiCfg.DbQueries = database.New(db)

	log.Info().Msg("apicfg loaded")

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

	err = apiCfg.PopulateRatingsDB()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to populate ratings")
		return
	}

	/* 	err = apiCfg.CacheImages()
	   	if err != nil {
	   		log.Fatal().Err(err).Msg("failed to cache remote images")
	   		return
	   	} */

	server.StartServer(&apiCfg)
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
