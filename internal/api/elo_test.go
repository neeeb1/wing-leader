package api

import (
	"testing"
)

func TestCalculateExpected(t *testing.T) {
	cases := []struct {
		ratingA, ratingB int
	}{
		{1000, 1000}, // equal ratings
		{1200, 1000}, // A higher
		{1000, 1200}, // B higher
		{800, 1500},  // large difference
		{1500, 800},  // large difference
		{1000, 1005}, // close ratings
	}
	for _, c := range cases {
		expectedA, expectedB := calculateExpected(c.ratingA, c.ratingB)
		if expectedA < 0 || expectedA > 1 {
			t.Errorf("expectedA out of bounds: got %f", expectedA)
		}
		if expectedB < 0 || expectedB > 1 {
			t.Errorf("expectedB out of bounds: got %f", expectedB)
		}
		sum := expectedA + expectedB
		if sum >= 1.0001 || sum <= 0.9999 {
			t.Errorf("expectedA + expectedB != 1: got %f", sum)
		}
	}
}

func TestCalculateDelta(t *testing.T) {
	cases := []struct {
		ratingA, ratingB int
		wantDelta        int32
	}{
		{1000, 1000, 16}, // equal ratings
		{1200, 1000, 7},  // A higher
		{1000, 1200, 24}, // B higher
		{800, 1500, 31},  // large difference
		{1500, 800, 0},   // large difference
		{1000, 1005, 16}, // close ratings
	}
	for _, c := range cases {
		expectedA, _ := calculateExpected(c.ratingA, c.ratingB)
		delta := calculateDelta(expectedA, 1.0)
		if delta != c.wantDelta {
			t.Errorf("calculateDelta(%f, 1.0) = %d, want %d", expectedA, delta, c.wantDelta)
		}
	}
}
