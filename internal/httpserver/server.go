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

	// Register routes
	s.mux.HandleFunc("/callback", s.handleCallback)
	s.mux.HandleFunc("/auth/", s.handleAuthRedirect)
	s.mux.HandleFunc("/health", s.handleHealth)

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

	// Configure TLS if enabled
	if cfg.TLS.Enabled {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
			// Note: PreferServerCipherSuites is deprecated since Go 1.21.
			// The Go TLS stack handles cipher suite ordering automatically.
		}
		s.httpServer.TLSConfig = tlsConfig
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
