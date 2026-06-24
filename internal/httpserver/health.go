package httpserver

import (
	"log/slog"
	"net/http"
)

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}` + "\n")); err != nil {
		// Best-effort: headers/status may already be written.
		slog.Error("failed to write health response", "error", err)
	}
}
