package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// HealthResponse is the JSON response for the health check endpoint
type HealthResponse struct {
	Status string `json:"status"`
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status: "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Best-effort: headers/status may already be written.
		slog.Error("failed to encode health response", "error", err)
	}
}
