package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
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
	"github.com/pressly/goose/v3"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//go:embed sql/schema/*.sql
var embedMigrations embed.FS

func main() {
	// Intialize zerolog and pretty console output
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true, TimeLocation: time.FixedZone("PST", -8*60*60)})

	// Perform a check to see if we are running in a docker container
	// If not, load the .env file for local development
	if !isRunningInDockerContainer() && !isRunningInCloudRun() {
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
	// apiCfg.CacheHost = os.Getenv("CACHE_HOST")

	// Intialize database connection
	// defer closing til program end
	db, err := connectUnixSocket()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to db via Unix Socket")
	}
	defer db.Close()

	// Set database connection pool settings
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(10 * time.Minute)

	// Add database to api config
	apiCfg.Db = db
	apiCfg.DbQueries = database.New(db)

	log.Info().Msg("apicfg loaded")

	// Run database migrations
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal().Err(err).Msg("failed to set sql dialect")
		return
	}

	if err := goose.Up(db, "sql/schema"); err != nil {
		log.Fatal().Err(err).Msg("failed to run goose db migrations")
		return
	}

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
	/* 	go func() {
	   		err = apiCfg.CacheImages()
	   	}()
	   	if err != nil {
	   		log.Fatal().Err(err).Msg("failed to cache remote images")
	   		return
	   	} */

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

func isRunningInCloudRun() bool {
	// Cloud Run sets these environment variables
	return os.Getenv("K_SERVICE") != "" ||
		os.Getenv("K_REVISION") != "" ||
		os.Getenv("K_CONFIGURATION") != ""
}

func connectUnixSocket() (*sql.DB, error) {
	mustGetenv := func(key string) string {
		env := os.Getenv(key)
		if env == "" {
			log.Fatal().Msgf("Fatal Error in connect_unix.go: %s environment variable not set.", key)
		}
		return env
	}

	var (
		dbUser         = mustGetenv("DB_USER")              // e.g. 'my-db-user'
		dbPwd          = mustGetenv("DB_PASS")              // e.g. 'my-db-password'
		dbName         = mustGetenv("DB_NAME")              // e.g. 'my-database'
		unixSocketPath = mustGetenv("INSTANCE_UNIX_SOCKET") // e.g. '/cloudsql/project:region:instance'
	)

	dbURI := fmt.Sprintf("user=%s password=%s database=%s host=%s",
		dbUser, dbPwd, dbName, unixSocketPath)

	dbPool, err := sql.Open("pgx", dbURI)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	return dbPool, nil
}
