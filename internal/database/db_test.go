package database

import (
	"testing"
)

func TestGetTopRatings_Ordered(t *testing.T) {
	testdb, cleanup := SetupTestDB(t)
	defer cleanup()
}

func TestUpdateRatingsByBirdID_IncrementMatches(t *testing.T) {
	testdb, cleanup := SetupTestDB(t)
	defer cleanup()
}
