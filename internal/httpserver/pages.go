package httpserver

import (
	"log/slog"
	"net/http"
)

// renderSuccess renders the success page
func (s *Server) renderSuccess(w http.ResponseWriter, message string) {
	data := map[string]string{
		"Message": message,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if err := s.templates.ExecuteTemplate(w, "success.html", data); err != nil {
		slog.Error("failed to render success template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// renderError renders the error page
func (s *Server) renderError(w http.ResponseWriter, errMsg string) {
	data := map[string]string{
		"Error": errMsg,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)

	if err := s.templates.ExecuteTemplate(w, "error.html", data); err != nil {
		slog.Error("failed to render error template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
