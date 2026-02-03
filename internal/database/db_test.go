package database

import (
	"database/sql"
	"testing"
)

type ApiConfig struct {
	Db        *sql.DB
	DbQueries *Queries
}

func TestGetTopRatings_Ordered(t *testing.T) {
	testdb, cleanup := SetupTestDB(t)
	defer cleanup()

	cfg := ApiConfig{
		Db:        testdb.DB,
		DbQueries: testdb.Queries,
	}

	birds := testdb.SeedTestData(t, 50)
	for i, bird := range birds {
		rating := int32(1000 + i*10)
		cfg.DbQueries.UpdateRatingByBirdID(t.Context(), UpdateRatingByBirdIDParams{
			Rating: sql.NullInt32{Int32: rating, Valid: true},
			BirdID: bird.ID,
		})
	}

	topBirds, err := cfg.DbQueries.GetTopRatings(t.Context(), 10)
	if err != nil {
		t.Fatalf("Failed to get top ratings: %v", err)
	}

	if len(topBirds) != 10 {
		t.Fatalf("Expected 10 top birds, got %d", len(topBirds))
	}

	for i := 0; i < len(topBirds)-1; i++ {
		if topBirds[i].Rating.Int32 < topBirds[i+1].Rating.Int32 {
			t.Errorf("Top birds not ordered correctly: bird %d has rating %d, bird %d has rating %d",
				i, topBirds[i].Rating.Int32, i+1, topBirds[i+1].Rating.Int32)
		}
	}

}

func TestUpdateRatingsByBirdID_IncrementMatches(t *testing.T) {
	testdb, cleanup := SetupTestDB(t)
	defer cleanup()

	cfg := ApiConfig{
		Db:        testdb.DB,
		DbQueries: testdb.Queries,
	}

	birds := testdb.SeedTestData(t, 2)

	rating, err := cfg.DbQueries.UpdateRatingByBirdID(t.Context(), UpdateRatingByBirdIDParams{
		Rating: sql.NullInt32{Int32: 1032, Valid: true},
		BirdID: birds[0].ID,
	})
	if err != nil {
		t.Errorf("Unable to update bird rating by ID: %v", err)
	}

	if rating.Matches.Int32 != 1 {
		t.Errorf("Failed to increment matches after voting, expected 1 but got %d", rating.Matches.Int32)
	}

}
