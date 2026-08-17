// Package httpserver serves OIDC callback, redirect, and health endpoints.
package httpserver

import (
	"context"
	"crypto/tls"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/config"
	"github.com/al-bashkir/openvpn-keycloak-auth/internal/oidc"
	"github.com/al-bashkir/openvpn-keycloak-auth/internal/session"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Server is the HTTP server for handling OIDC callbacks and health checks
type Server struct {
	cfg          *config.Config
	httpServer   *http.Server
	mux          *http.ServeMux
	templates    *template.Template
	oidcProvider *oidc.Provider
	sessionMgr   *session.Manager
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, oidcProvider *oidc.Provider, sessionMgr *session.Manager) (*Server, error) {
	// Parse templates
	templates, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:          cfg,
		mux:          http.NewServeMux(),
		templates:    templates,
		oidcProvider: oidcProvider,
		sessionMgr:   sessionMgr,
	}

	// Register routes. The redirect_uri was validated at config load, so a
	// parse failure here means the config was built by hand in a test.
	redirect, err := config.ParseRedirectURI(cfg.OIDC.RedirectURI)
	if err != nil {
		return nil, err
	}
	s.mux.HandleFunc(redirect.Path, s.handleCallback)
	s.mux.HandleFunc(config.AuthPrefix(redirect), s.handleAuthRedirect)
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"status":"ok"}` + "\n")); err != nil {
			slog.Error("failed to write health response", "error", err)
		}
	})

	// Wrap with middleware
	handler := loggingMiddleware(s.mux)
	handler = recoveryMiddleware(handler)
	handler = rateLimitMiddleware(handler)
	handler = securityHeadersMiddleware(handler)

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         cfg.Listen.HTTP,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Configure TLS if enabled. TLS 1.3 only — every browser and OpenVPN
	// client used to render the callback page supports it, and TLS 1.3 has
	// no negotiable cipher suites that could be downgraded.
	if cfg.TLS.Enabled {
		s.httpServer.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS13}
	}

	return s, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	slog.Info("starting HTTP server",
		"addr", s.cfg.Listen.HTTP,
		"tls", s.cfg.TLS.Enabled,
	)

	if s.cfg.TLS.Enabled {
		return s.httpServer.ListenAndServeTLS(s.cfg.TLS.CertFile, s.cfg.TLS.KeyFile)
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}
