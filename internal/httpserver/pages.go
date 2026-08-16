package httpserver

import (
	"log/slog"
	"net/http"
)

// render writes an embedded HTML template with the given status.
func (s *Server) render(w http.ResponseWriter, tmpl string, status int, data map[string]string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := s.templates.ExecuteTemplate(w, tmpl, data); err != nil {
		slog.Error("failed to render template", "template", tmpl, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (s *Server) renderSuccess(w http.ResponseWriter, message string) {
	s.render(w, "success.html", http.StatusOK, map[string]string{"Message": message})
}

func (s *Server) renderError(w http.ResponseWriter, errMsg string) {
	s.render(w, "error.html", http.StatusBadRequest, map[string]string{"Error": errMsg})
}
