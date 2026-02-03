package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/neeeb1/rate_birds/internal/database"
)

func TestScoreMatch_Concurrent(t *testing.T) {
	testDB, cleanup := database.SetupTestDB(t)
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

	rating0, err := testDB.Queries.GetRatingByBirdID(t.Context(), bird0.ID)
	if err != nil {
		t.Fatalf("Failed to get bird0 rating: %v", err)
	}

	rating1, err := testDB.Queries.GetRatingByBirdID(t.Context(), bird1.ID)
	if err != nil {
		t.Fatalf("Failed to get bird1 rating: %v", err)
	}

	if rating0.Rating.Int32+rating1.Rating.Int32 != 2000 {
		t.Errorf("Expected total rating 2000, got %d", rating0.Rating.Int32+rating1.Rating.Int32)
	}
	if rating0.Rating.Int32 <= 1000 {
		t.Errorf("Expected bird0 rating > 1000, got %d", rating0.Rating.Int32)
	}
	if rating1.Rating.Int32 >= 1000 {
		t.Errorf("Expected bird1 rating < 1000, got %d", rating1.Rating.Int32)
	}
	if rating0.Matches.Int32 != int32(numVotes) {
		t.Errorf("Expected bird0 matches %d, got %d", numVotes, rating0.Matches.Int32)
	}
	if rating1.Matches.Int32 != int32(numVotes) {
		t.Errorf("Expected bird1 matches %d, got %d", numVotes, rating1.Matches.Int32)
	}
}

func TestHandleScoreMatch_ExpiredSession(t *testing.T) {
	testDB, cleanup := database.SetupTestDB(t)
	defer cleanup()

	birds := testDB.SeedTestData(t, 2)
	cfg := &ApiConfig{
		Db:        testDB.DB,
		DbQueries: testDB.Queries,
	}

	session, err := testDB.Queries.CreateMatchSession(t.Context(), database.CreateMatchSessionParams{
		LeftbirdID:   birds[0].ID,
		RightbirdID:  birds[1].ID,
		SessionToken: "expired-token",
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
		UserIp:       sql.NullString{String: "127.0.0.1", Valid: true},
		UserAgent:    sql.NullString{String: "test-agent", Valid: true},
	})
	if err != nil {
		t.Fatalf("Failed to create expired match session: %v", err)
	}

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/scorematch/?winner=left&leftBirdID=%s&rightBirdID=%s", birds[0].ID, birds[1].ID), nil)
	req.AddCookie(&http.Cookie{
		Name:  "sessionToken",
		Value: session.SessionToken,
	})

	rr := httptest.NewRecorder()
	cfg.handleScoreMatch(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	updatedSession, err := testDB.Queries.GetMatchSessionByToken(t.Context(), session.SessionToken)
	if err != nil {
		t.Fatalf("Failed to get updated match session: %v", err)
	}
	if updatedSession.Voted.Bool {
		t.Errorf("Expected session voted to be false, got true")
	}
}

func TestHandleScoreMatch_AlreadyVoted(t *testing.T) {
	testDB, cleanup := database.SetupTestDB(t)
	defer cleanup()

	birds := testDB.SeedTestData(t, 2)
	cfg := &ApiConfig{
		Db:        testDB.DB,
		DbQueries: testDB.Queries,
	}

	session, err := testDB.Queries.CreateMatchSession(t.Context(), database.CreateMatchSessionParams{
		LeftbirdID:   birds[0].ID,
		RightbirdID:  birds[1].ID,
		SessionToken: "already-voted-token",
		ExpiresAt:    time.Now().Add(10 * time.Minute),
		UserIp:       sql.NullString{String: "127.0.0.1", Valid: true},
		UserAgent:    sql.NullString{String: "test-agent", Valid: true},
	})
	if err != nil {
		t.Fatalf("Failed to create already-voted match session: %v", err)
	}
	err = cfg.ScoreMatch(birds[0], birds[1])
	if err != nil {
		t.Fatalf("Failed to mark session as voted: %v", err)
	}

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/scorematch/?winner=left&leftBirdID=%s&rightBirdID=%s", birds[0].ID, birds[1].ID), nil)
	req.AddCookie(&http.Cookie{
		Name:  "sessionToken",
		Value: session.SessionToken,
	})

	rr := httptest.NewRecorder()
	cfg.handleScoreMatch(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	rating0, err := testDB.Queries.GetRatingByBirdID(t.Context(), birds[0].ID)
	if err != nil {
		t.Fatalf("Failed to get bird0 rating: %v", err)
	}
	if rating0.Matches.Int32 != 1 {
		t.Errorf("Expected bird0 matches to be 1, got %d", rating0.Matches.Int32)
	}

	rating1, err := testDB.Queries.GetRatingByBirdID(t.Context(), birds[1].ID)
	if err != nil {
		t.Fatalf("Failed to get bird1 rating: %v", err)
	}
	if rating1.Matches.Int32 != 1 {
		t.Errorf("Expected bird1 matches to be 1, got %d", rating1.Matches.Int32)
	}
}

func TestHandleScoreMatch_InvalidToken(t *testing.T) {
	testDB, cleanup := database.SetupTestDB(t)
	defer cleanup()

	birds := testDB.SeedTestData(t, 2)

	cfg := &ApiConfig{
		Db:        testDB.DB,
		DbQueries: testDB.Queries,
	}

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/scorematch/?winner=left&leftBirdID=%s&rightBirdID=%s",
			birds[0].ID, birds[1].ID), nil)
	req.AddCookie(&http.Cookie{
		Name:  "sessionToken",
		Value: "non-existent-token-12345",
	})

	rr := httptest.NewRecorder()
	cfg.handleScoreMatch(rr, req)

	// Should return 400 Bad Request
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	// Verify no votes were recorded
	rating0, err := testDB.Queries.GetRatingByBirdID(t.Context(), birds[0].ID)
	if err != nil {
		t.Fatalf("Failed to get bird0 rating: %v", err)
	}
	rating1, err := testDB.Queries.GetRatingByBirdID(t.Context(), birds[1].ID)
	if err != nil {
		t.Fatalf("Failed to get bird1 rating: %v", err)
	}

	// Ratings should remain at initial 1000
	if rating0.Rating.Int32 != 1000 {
		t.Errorf("Expected bird0 rating = 1000, got %d", rating0.Rating.Int32)
	}
	if rating1.Rating.Int32 != 1000 {
		t.Errorf("Expected bird1 rating = 1000, got %d", rating1.Rating.Int32)
	}

	// No matches should be recorded
	if rating0.Matches.Int32 != 0 {
		t.Errorf("Expected bird0 matches = 0, got %d", rating0.Matches.Int32)
	}
	if rating1.Matches.Int32 != 0 {
		t.Errorf("Expected bird1 matches = 0, got %d", rating1.Matches.Int32)
	}
}
