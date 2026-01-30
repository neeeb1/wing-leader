package api

import (
	"context"
	"sync"
	"testing"

	"github.com/neeeb1/rate_birds/internal/testdb"
)

func TestScoreMatch_Concurrent(t *testing.T) {
	testDB, cleanup := testdb.SetupTestDB(t)
	defer cleanup()

	birds := testDB.SeedTestData(t, 2)
	bird0, bird1 := birds[0], birds[1]

	testCfg := &ApiConfig{
		Db:        testDB.DB,
		DbQueries: testDB.Queries,
	}

	numVotes := 100
	var wg sync.WaitGroup
	errors := make(chan error, numVotes)

	for i := 0; i < numVotes; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := testCfg.ScoreMatch(bird0, bird1); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent vote error: %v", err)
		}
	}

	rating0, err := testDB.Queries.GetRatingByBirdID(context.Background(), bird0.ID)
	if err != nil {
		t.Fatalf("Failed to get bird0 rating: %v", err)
	}

	rating1, err := testDB.Queries.GetRatingByBirdID(context.Background(), bird1.ID)
	if err != nil {
		t.Fatalf("Failed to get bird1 rating: %v", err)
	}

	expectedWinner := int32(1000 + (numVotes * 16))
	expectedLoser := int32(1000 - (numVotes * 16))

	if rating0.Rating.Int32 != expectedWinner {
		t.Errorf("Expected bird0 rating %d, got %d", expectedWinner, rating0.Rating.Int32)
	}
	if rating1.Rating.Int32 != expectedLoser {
		t.Errorf("Expected bird1 rating %d, got %d", expectedLoser, rating1.Rating.Int32)
	}
	if rating0.Matches.Int32 != int32(numVotes) {
		t.Errorf("Expected bird0 matches %d, got %d", numVotes, rating0.Matches.Int32)
	}
	if rating1.Matches.Int32 != int32(numVotes) {
		t.Errorf("Expected bird1 matches %d, got %d", numVotes, rating1.Matches.Int32)
	}
}

func TestHandleScoreMatch_ExpiredSession(t *testing.T) {}

func TestHandleScoreMatch_AlreadyVoted(t *testing.T) {}

func TestHandleScoreMatch_InvalidToken(t *testing.T) {}
