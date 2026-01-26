package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type HealthStatus struct {
	Status string `json:"status"`
}

func HandleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthStatus{Status: "alive"})
}

func (cfg *ApiConfig) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Check database connectivity
	if err := cfg.Db.PingContext(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HealthStatus{Status: "unavailable"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthStatus{Status: "ready"})
}
