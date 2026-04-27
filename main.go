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

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	var err error
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true, TimeLocation: time.FixedZone("PST", -8*60*60)})

	if !isRunningInDockerContainer() && !isRunningInCloudRun() && !isRunningOnFly() {
		log.Info().Msg("Running locally, loading .env")
		if err := godotenv.Load(); err != nil {
			log.Fatal().Err(err).Msg("Failed to load .env")
			return
		}
	} else {
		log.Info().Msg("Running in container, skipping .env load")
	}

	apiCfg := api.ApiConfig{}
	apiCfg.NuthatcherApiKey = os.Getenv("NUTHATCH_KEY")

	// sql.Open is fast (no network) — safe to do before server start
	var db *sql.DB
	if isRunningInCloudRun() {
		db, err = connectUnixSocket()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to open db connection, starting without database")
		}
	} else {
		apiCfg.DbURL = os.Getenv("DB_URL")
		db, err = sql.Open("postgres", apiCfg.DbURL)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to open db connection, starting without database")
			db = nil
		}
	}

	if db != nil {
		defer db.Close()
		db.SetMaxOpenConns(50)
		db.SetMaxIdleConns(20)
		db.SetConnMaxLifetime(10 * time.Minute)
		apiCfg.Db = db
		apiCfg.DbQueries = database.New(db)
	}

	// S3 client init is fast (no network) — safe to do before server start
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	endpointURL := os.Getenv("AWS_ENDPOINT_URL_S3")
	bucketName := os.Getenv("BUCKET_NAME")

	if accessKeyID != "" && secretAccessKey != "" && bucketName != "" {
		s3Cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
			awsconfig.WithRegion("auto"),
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
			),
		)
		if err != nil {
			log.Warn().Err(err).Msg("failed to configure S3 client, image caching disabled")
		} else {
			apiCfg.S3Client = s3.NewFromConfig(s3Cfg, func(o *s3.Options) {
				o.BaseEndpoint = aws.String(endpointURL)
				o.UsePathStyle = true
			})
			apiCfg.BucketName = bucketName
			log.Info().Str("bucket", bucketName).Msg("S3 client initialized for Tigris")
		}
	} else {
		log.Info().Msg("Tigris env vars not set, image caching disabled")
	}

	// Start server immediately before slow network operations
	srv, err := server.StartServer(&apiCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
		return
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Slow startup tasks run in background after server is already listening
	go func() {
		if db != nil {
			if err := db.Ping(); err != nil {
				log.Warn().Err(err).Msg("Failed to ping db — data endpoints may fail until DB is reachable")
			}

			goose.SetBaseFS(embedMigrations)
			if err := goose.SetDialect("postgres"); err != nil {
				log.Fatal().Err(err).Msg("failed to set sql dialect")
				return
			}
			if err := goose.Up(db, "sql/schema"); err != nil {
				log.Fatal().Err(err).Msg("failed to run goose db migrations")
				return
			}

			count, err := apiCfg.DbQueries.GetTotalBirdCount(context.Background())
			if err != nil {
				log.Fatal().Err(err).Msg("failed to count db entries")
				return
			}
			if count == 0 || os.Getenv("REPOPULATE") == "true" {
				if err = apiCfg.PopulateBirdDB(); err != nil {
					log.Fatal().Err(err).Msg("failed to populate birds")
					return
				}
			} else {
				log.Info().Msg("Bird db already populated - skipping initial population...")
			}

			if err = apiCfg.PopulateRatingsDB(); err != nil {
				log.Fatal().Err(err).Msg("failed to populate ratings")
				return
			}
		} else {
			log.Warn().Msg("Starting without database — data endpoints will return 503")
		}

		if os.Getenv("RESET_IMAGES") == "true" && apiCfg.DbQueries != nil {
			log.Info().Msg("RESET_IMAGES=true — clearing all image_urls")
			if err := apiCfg.DbQueries.ClearAllImageUrls(context.Background()); err != nil {
				log.Error().Err(err).Msg("failed to clear image URLs")
			} else {
				log.Info().Msg("image_urls cleared")
			}
		}

		if err := apiCfg.CacheImages(); err != nil {
			log.Error().Err(err).Msg("image caching failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced shutdown")
	}

	log.Info().Msg("Server exited cleanly")
}

func isRunningInDockerContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

func isRunningInCloudRun() bool {
	return os.Getenv("K_SERVICE") != "" ||
		os.Getenv("K_REVISION") != "" ||
		os.Getenv("K_CONFIGURATION") != ""
}

func isRunningOnFly() bool {
	return os.Getenv("FLY_APP_NAME") != ""
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
		dbUser         = mustGetenv("DB_USER")
		dbPwd          = mustGetenv("DB_PASS")
		dbName         = mustGetenv("DB_NAME")
		unixSocketPath = mustGetenv("INSTANCE_UNIX_SOCKET")
	)

	dbURI := fmt.Sprintf("user=%s password=%s dbname=%s host=%s",
		dbUser, dbPwd, dbName, unixSocketPath)

	dbPool, err := sql.Open("postgres", dbURI)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	return dbPool, nil
}
