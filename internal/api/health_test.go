package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neeeb1/rate_birds/internal/database"
)

func TestHealthReadiness_DBAvailable(t *testing.T) {
	testDB, cleanup := database.SetupTestDB(t)
	defer cleanup()

	cfg := &ApiConfig{
		Db:        testDB.DB,
		DbQueries: testDB.Queries,
	}

	req := httptest.NewRequest("GET", "/health/ready", nil)
	rr := httptest.NewRecorder()

	cfg.HandleReadiness(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var status HealthStatus
	json.NewDecoder(rr.Body).Decode(&status)

	if status.Status != "ready" {
		t.Errorf("Expected 'ready', got '%s'", status.Status)
	}

}

func TestHealthReadiness_DBUnavailable(t *testing.T) {
	testDB, cleanup := database.SetupTestDB(t)
	defer cleanup()

	cfg := &ApiConfig{
		Db:        testDB.DB,
		DbQueries: testDB.Queries,
	}

	testDB.DB.Close()

	req := httptest.NewRequest("GET", "/health/ready", nil)
	rr := httptest.NewRecorder()

	cfg.HandleReadiness(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}

	var status HealthStatus
	json.NewDecoder(rr.Body).Decode(&status)

	if status.Status != "unavailable" {
		t.Errorf("Expected 'ready', got '%s'", status.Status)
	}
}
