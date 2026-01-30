package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/neeeb1/rate_birds/internal/database"
	"github.com/pressly/goose"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestDB struct {
	Container *postgres.PostgresContainer
	DB        *sql.DB
	Queries   *database.Queries
	ConnStr   string
}

func SetupTestDB(t *testing.T) (*TestDB, func()) {
	t.Helper()

	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx,
		"postgres:17-bookworm",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)

	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	if err := runMigrations(t, db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	queries := database.New(db)

	TestDB := &TestDB{
		Container: pgContainer,
		DB:        db,
		Queries:   queries,
		ConnStr:   connStr,
	}

	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database connection: %v", err)
		}
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Errorf("Failed to terminate postgres container: %v", err)
		}
		if err := goose.Reset(db, "../../sql/schema"); err != nil {
			t.Errorf("Failed to reset database schema: %v", err)
		}
	}

	return TestDB, cleanup
}

func runMigrations(t *testing.T, db *sql.DB) error {
	t.Helper()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.Up(db, "../../sql/schema"); err != nil {
		return err
	}

	return nil
}

func (db *TestDB) SeedTestData(t *testing.T, count int) []database.Bird {
	t.Helper()

	birds := make([]database.Bird, 0, count)

	for i := 0; i < count; i++ {
		bird, err := db.Queries.CreateBird(context.Background(), database.CreateBirdParams{
			CommonName:     sql.NullString{String: fmt.Sprintf("Test Bird%d", i), Valid: true},
			ScientificName: sql.NullString{String: fmt.Sprintf("Tesus birdus%d", i), Valid: true},
			Family:         sql.NullString{String: "Testidae", Valid: true},
			Order:          sql.NullString{String: "Testiformes", Valid: true},
			Status:         sql.NullString{String: "alive and well", Valid: true},
			ImageUrls: []string{
				"http://example.com/image1.jpg",
				"http://example.com/image2.jpg",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create test bird: %v", err)
		}

		err = db.Queries.PopulateRating(context.Background(), database.PopulateRatingParams{
			BirdID:  bird.ID,
			Rating:  sql.NullInt32{Int32: 1000, Valid: true},
			Matches: sql.NullInt32{Int32: 0, Valid: true},
		})
		if err != nil {
			t.Fatalf("Failed to populate rating for test bird: %v", err)
		}

		birds = append(birds, bird)
	}

	return birds
}
