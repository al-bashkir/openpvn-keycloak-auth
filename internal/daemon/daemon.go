// Package daemon orchestrates all the components of the OpenVPN SSO daemon.
package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"net/url"

	"github.com/al-bashkir/openvpn-keycloak/internal/config"
	"github.com/al-bashkir/openvpn-keycloak/internal/httpserver"
	"github.com/al-bashkir/openvpn-keycloak/internal/ipc"
	"github.com/al-bashkir/openvpn-keycloak/internal/oidc"
	"github.com/al-bashkir/openvpn-keycloak/internal/openvpn"
	"github.com/al-bashkir/openvpn-keycloak/internal/session"
)

// Daemon represents the main daemon process that coordinates all components.
type Daemon struct {
	cfg          *config.Config
	oidcProvider *oidc.Provider
	sessionMgr   *session.Manager
	httpServer   *httpserver.Server
	ipcServer    *ipc.Server
}

// New creates a new daemon with all components initialized.
func New(cfg *config.Config) (*Daemon, error) {
	// Initialize OIDC provider
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	oidcProvider, err := oidc.NewProvider(ctx, &cfg.OIDC)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	slog.Info("OIDC provider initialized",
		"issuer", cfg.OIDC.Issuer,
		"client_id", cfg.OIDC.ClientID,
	)

	// Initialize session manager
	sessionTimeout := time.Duration(cfg.Auth.SessionTimeout) * time.Second
	sessionMgr := session.NewManager(sessionTimeout)

	slog.Info("session manager initialized",
		"timeout", sessionTimeout,
	)

	// Initialize HTTP server
	httpServer, err := httpserver.NewServer(cfg, oidcProvider, sessionMgr)
	if err != nil {
		sessionMgr.Stop()
		return nil, fmt.Errorf("failed to initialize HTTP server: %w", err)
	}

	slog.Info("HTTP server initialized",
		"listen", cfg.Listen.HTTP,
		"tls", cfg.TLS.Enabled,
	)

	// Initialize IPC server with auth handler
	ipcServer := ipc.NewServer(cfg.Listen.Socket, func(ctx context.Context, req *ipc.AuthRequest) (*ipc.AuthResponse, error) {
		return handleAuthRequest(ctx, cfg, oidcProvider, sessionMgr, req)
	})

	slog.Info("IPC server initialized",
		"socket", cfg.Listen.Socket,
	)

	return &Daemon{
		cfg:          cfg,
		oidcProvider: oidcProvider,
		sessionMgr:   sessionMgr,
		httpServer:   httpServer,
		ipcServer:    ipcServer,
	}, nil
}

// Run starts all daemon components and blocks until shutdown signal is received.
func (d *Daemon) Run() error {
	slog.Info("starting OpenVPN Keycloak SSO daemon")

	// Start IPC server synchronously to catch startup errors
	ctx := context.Background()
	if err := d.ipcServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start IPC server: %w", err)
	}

	// Start HTTP server in a goroutine (it blocks on ListenAndServe)
	httpErrCh := make(chan error, 1)
	go func() {
		if err := d.httpServer.Start(); err != nil && err.Error() != "http: Server closed" {
			httpErrCh <- err
		}
		close(httpErrCh)
	}()

	// Wait for shutdown signal or startup error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("shutdown signal received", "signal", sig.String())
	case err := <-httpErrCh:
		if err != nil {
			slog.Error("HTTP server failed to start", "error", err)
			// Clean up IPC server before returning
			if stopErr := d.ipcServer.Stop(); stopErr != nil {
				slog.Error("error stopping IPC server after HTTP server startup failure", "error", stopErr)
			}
			d.sessionMgr.Stop()
			return fmt.Errorf("HTTP server failed: %w", err)
		}
	}

	// Shutdown gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop IPC server
	if err := d.ipcServer.Stop(); err != nil {
		slog.Error("error stopping IPC server", "error", err)
	}

	// Stop HTTP server
	if err := d.httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("error stopping HTTP server", "error", err)
	}

	// Stop session manager
	d.sessionMgr.Stop()

	slog.Info("daemon shutdown complete")
	return nil
}

// handleAuthRequest handles authentication requests from the IPC server.
// It creates a session, starts the OIDC flow, and writes the auth_pending_file.
func handleAuthRequest(ctx context.Context, cfg *config.Config, oidcProvider *oidc.Provider,
	sessionMgr *session.Manager, req *ipc.AuthRequest) (*ipc.AuthResponse, error) {

	slog.Info("auth request received",
		"username", req.Username,
		"ip", req.UntrustedIP,
		"port", req.UntrustedPort,
	)

	// Create session
	sess, err := sessionMgr.Create(
		req.Username,
		req.CommonName,
		req.UntrustedIP,
		req.UntrustedPort,
		req.AuthControlFile,
		req.AuthPendingFile,
		req.AuthFailedReasonFile,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	slog.Debug("session created", "session_id", sess.ID)

	// Start OIDC flow
	flowData, err := oidcProvider.StartAuthFlow(ctx)
	if err != nil {
		sessionMgr.Delete(sess.ID)
		return nil, fmt.Errorf("failed to start OIDC flow: %w", err)
	}

	// Update session with OIDC flow data
	err = sessionMgr.UpdateOIDCFlow(sess.ID, flowData.State, flowData.CodeVerifier, flowData.AuthURL)
	if err != nil {
		sessionMgr.Delete(sess.ID)
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	slog.Debug("OIDC flow started",
		"session_id", sess.ID,
		"state", flowData.State,
	)

	// Build a short redirect URL for the auth_pending_file.
	// OpenVPN's OPTION_LINE_SIZE is 256 chars, and full OIDC auth URLs with PKCE
	// parameters easily exceed this. We use /auth/<state> which 302-redirects to
	// the full Keycloak auth URL.
	shortAuthURL, err := buildShortAuthURL(cfg.OIDC.RedirectURI, flowData.State)
	if err != nil {
		sessionMgr.Delete(sess.ID)
		return nil, fmt.Errorf("failed to build short auth URL: %w", err)
	}

	slog.Debug("short auth URL built",
		"session_id", sess.ID,
		"short_url", shortAuthURL,
		"full_url_length", len(flowData.AuthURL),
	)

	// Write auth_pending_file to trigger browser opening.
	// The method must match the client's IV_SSO capability.
	err = openvpn.WriteAuthPending(
		req.AuthPendingFile,
		cfg.Auth.SessionTimeout,
		req.PendingAuthMethod,
		shortAuthURL,
	)
	if err != nil {
		sessionMgr.Delete(sess.ID)
		// Also write auth failure since we can't proceed
		if wErr := openvpn.WriteAuthFailure(
			req.AuthControlFile,
			req.AuthFailedReasonFile,
			"Failed to start authentication flow",
		); wErr != nil {
			slog.Error("failed to write auth failure after pending write failure", "error", wErr)
		}
		return nil, fmt.Errorf("failed to write auth_pending_file: %w", err)
	}

	slog.Info("auth flow initiated",
		"session_id", sess.ID,
		"username", req.Username,
		"ip", req.UntrustedIP,
	)

	// Return response to auth script
	return &ipc.AuthResponse{
		Type:      ipc.MessageTypeAuthResponse,
		Status:    "deferred",
		SessionID: sess.ID,
		AuthURL:   shortAuthURL,
	}, nil
}

// maxWebAuthLineLen is OpenVPN's OPTION_LINE_SIZE limit for a single line in
// the auth_pending_file. The third line is "WEB_AUTH::<url>\n".
const maxWebAuthLineLen = 256

// webAuthPrefix is the prefix OpenVPN expects on the auth URL line.
const webAuthPrefix = "WEB_AUTH::"

// buildShortAuthURL constructs a short auth redirect URL from the redirect URI config.
// Given a redirect_uri like "https://vpn.example.com:9000/callback" and a state,
// it returns "https://vpn.example.com:9000/auth/<state>".
// If redirect_uri contains a base path (e.g. "https://host/vpn/callback"), the base
// path is preserved and the short URL becomes "https://host/vpn/auth/<state>".
// This keeps the WEB_AUTH:: line well under OpenVPN's 256-char OPTION_LINE_SIZE limit.
//
// It validates that the resulting "WEB_AUTH::<url>\n" line does not exceed the limit
// to prevent truncated/invalid URLs from reaching the client.
func buildShortAuthURL(redirectURI, state string) (string, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", fmt.Errorf("failed to parse redirect_uri: %w", err)
	}

	redirectPath := path.Clean(u.Path)
	if redirectPath == "." {
		redirectPath = "/"
	}

	basePath := path.Dir(redirectPath)
	if basePath == "." {
		basePath = "/"
	}

	u.Path = path.Join(basePath, "auth", state)
	u.RawQuery = ""
	u.Fragment = ""

	shortURL := u.String()

	// Validate that "WEB_AUTH::<url>\n" fits within OPTION_LINE_SIZE.
	// len("WEB_AUTH::") + len(url) + len("\n") must be <= 256.
	lineLen := len(webAuthPrefix) + len(shortURL) + 1 // +1 for trailing newline
	if lineLen > maxWebAuthLineLen {
		return "", fmt.Errorf("short auth URL too long (%d chars); WEB_AUTH:: line would be %d bytes, exceeding OpenVPN's %d-byte OPTION_LINE_SIZE limit",
			len(shortURL), lineLen, maxWebAuthLineLen)
	}

	return shortURL, nil
}
