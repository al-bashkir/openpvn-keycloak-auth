package httpserver

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/oidc"
	"github.com/al-bashkir/openvpn-keycloak-auth/internal/openvpn"
	"github.com/al-bashkir/openvpn-keycloak-auth/internal/session"
)

// handleAuthRedirect handles short auth redirect URLs.
// OpenVPN's auth_pending_file has a 256-char line limit (OPTION_LINE_SIZE).
// Full OIDC auth URLs with PKCE parameters exceed this limit.
// This endpoint serves a short URL (e.g., /auth/<state>) that 302-redirects
// to the full Keycloak authorization URL stored in the session.
func (s *Server) handleAuthRedirect(w http.ResponseWriter, r *http.Request) {
	// Extract state from URL path: /auth/{state}
	state := strings.TrimPrefix(r.URL.Path, "/auth/")
	if state == "" {
		s.renderError(w, "Invalid auth URL")
		return
	}

	// Look up session by state
	sess, err := s.sessionMgr.GetByState(state)
	if err != nil {
		slog.Error("auth redirect: session not found", // #nosec G706 -- values sanitized via sanitizeLog
			"state", sanitizeLog(state),
			"error", err,
		)
		s.renderError(w, "Session not found or expired. Please try connecting again.")
		return
	}

	if sess.AuthURL == "" {
		slog.Error("auth redirect: no auth URL in session", // #nosec G706 -- values sanitized via sanitizeLog
			"state", sanitizeLog(state),
			"session_id", sess.ID,
		)
		s.renderError(w, "Authentication flow not initialized. Please try connecting again.")
		return
	}

	slog.Debug("auth redirect", // #nosec G706 -- values sanitized via sanitizeLog
		"state", sanitizeLog(state),
		"session_id", sess.ID,
	)

	http.Redirect(w, r, sess.AuthURL, http.StatusFound)
}

// handleCallback handles OIDC callback requests.
// This completes the OAuth2 authorization code flow:
// 1. Extract code and state from query parameters
// 2. Look up session by state
// 3. Exchange code for tokens (with PKCE)
// 4. Verify ID token and extract claims
// 5. Validate username (if required)
// 6. Write success/failure to OpenVPN control file
func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Extract callback parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")
	errorDesc := r.URL.Query().Get("error_description")

	slog.Info("callback received", // #nosec G706 -- only boolean values logged, no injection risk
		"code_present", code != "",
		"state_present", state != "",
		"error_present", errorParam != "",
	)

	// Handle OIDC error responses
	if errorParam != "" {
		slog.Error("OIDC error in callback", // #nosec G706 -- values sanitized via sanitizeLog
			"error", sanitizeLog(errorParam),
			"description", sanitizeLog(errorDesc),
		)
		msg := fmt.Sprintf("Authentication failed: %s", errorDesc)
		if errorDesc == "" {
			msg = fmt.Sprintf("Authentication failed: %s", errorParam)
		}

		// Write auth failure immediately so OpenVPN doesn't hang until timeout
		if state != "" && s.sessionMgr != nil {
			if sess, err := s.sessionMgr.GetByState(state); err == nil {
				slog.Info("writing auth failure for OIDC error", // #nosec G706 -- values sanitized via sanitizeLog
					"session_id", sess.ID,
					"error", sanitizeLog(errorParam),
				)
				s.writeAuthFailure(sess, msg)
			}
		}

		s.renderError(w, msg)
		return
	}

	// Validate parameters
	if code == "" || state == "" {
		slog.Error("invalid callback parameters", // #nosec G706 -- only boolean values logged, no injection risk
			"code_present", code != "",
			"state_present", state != "",
		)
		s.renderError(w, "Invalid callback parameters")
		return
	}

	// Look up session by state
	session, err := s.sessionMgr.GetByState(state)
	if err != nil {
		slog.Error("session not found", // #nosec G706 -- values sanitized via sanitizeLog
			"state", sanitizeLog(state),
			"error", err,
		)
		s.renderError(w, "Session not found or expired. Please try connecting again.")
		return
	}

	// Ensure we always write a result (safety net).
	// Only deletes the session if the auth_control_file write succeeds.
	defer func() {
		if s.sessionMgr == nil {
			return
		}

		written, ok := s.sessionMgr.ResultWritten(session.ID)
		if !ok || written {
			return
		}

		slog.Error("callback completed without writing result, writing failure",
			"session_id", session.ID,
		)

		if err := openvpn.WriteAuthFailure(
			session.AuthControlFile,
			session.AuthFailedReasonFile,
			"Internal error",
		); err != nil {
			slog.Error("failed to write safety-net auth failure",
				"session_id", session.ID,
				"error", err,
			)
			// Keep session for cleanup/retry attempts.
			return
		}

		_ = s.sessionMgr.MarkResultWritten(session.ID)
		s.sessionMgr.Delete(session.ID)
	}()

	// Exchange code for tokens
	tokenData, err := s.oidcProvider.ExchangeCode(r.Context(), code, session.CodeVerifier)
	if err != nil {
		slog.Error("token exchange failed", // #nosec G706 -- session.ID is crypto/rand hex; err is from OIDC library
			"session_id", session.ID,
			"error", err,
		)
		s.writeAuthFailure(session, "Token exchange failed")
		s.renderError(w, "Authentication failed. Please try again.")
		return
	}

	// Validate token claims
	validator := oidc.NewValidator(&s.cfg.OIDC, &s.cfg.Auth)

	// Always validate roles (even when username mismatch is allowed)
	if err := validator.ValidateRoles(tokenData.Claims); err != nil {
		slog.Error("role validation failed", // #nosec G706 -- values sanitized via sanitizeLog
			"session_id", session.ID,
			"username", sanitizeLog(session.Username),
			"error", err,
		)
		s.writeAuthFailure(session, err.Error())
		s.renderError(w, "Authentication failed: "+err.Error())
		return
	}

	// Validate username match unless explicitly allowed to differ
	if !s.cfg.Auth.AllowUsernameMismatch {
		if err := validator.ValidateToken(tokenData.Claims, session.Username); err != nil {
			slog.Error("token validation failed", // #nosec G706 -- values sanitized via sanitizeLog
				"session_id", session.ID,
				"username", sanitizeLog(session.Username),
				"error", err,
			)
			s.writeAuthFailure(session, err.Error())
			s.renderError(w, "Authentication failed: "+err.Error())
			return
		}
	}

	// Extract username for logging (already validated by validator if AllowUsernameMismatch is false)
	username, _ := tokenData.Claims[s.cfg.Auth.UsernameClaim].(string)

	slog.Info("user authenticated successfully", // #nosec G706 -- values sanitized via sanitizeLog
		"session_id", session.ID,
		"username", sanitizeLog(username),
		"expected_username", sanitizeLog(session.Username),
		"ip", sanitizeLog(session.UntrustedIP),
	)

	// Authentication successful!
	if err := s.writeAuthSuccess(session); err != nil {
		s.renderError(w, "Authentication succeeded, but the VPN server could not be notified. Please try connecting again.")
		return
	}

	s.renderSuccess(w, "You are now connected to the VPN. You may close this window.")
}

// writeAuthSuccess writes success to the OpenVPN control file and deletes the session.
func (s *Server) writeAuthSuccess(sess *session.Session) error {
	if s.sessionMgr == nil {
		return fmt.Errorf("session manager is nil")
	}

	written, ok := s.sessionMgr.ResultWritten(sess.ID)
	if !ok {
		return fmt.Errorf("session not found")
	}
	if written {
		slog.Warn("session already completed, skipping auth success write",
			"session_id", sess.ID,
		)
		return nil
	}

	if err := openvpn.WriteAuthSuccess(sess.AuthControlFile); err != nil {
		slog.Error("failed to write auth success",
			"session_id", sess.ID,
			"error", err,
		)
		return err
	}

	slog.Info("auth success written",
		"session_id", sess.ID,
		"username", sanitizeLog(sess.Username),
		"ip", sanitizeLog(sess.UntrustedIP),
	)

	_ = s.sessionMgr.MarkResultWritten(sess.ID)
	s.sessionMgr.Delete(sess.ID)
	return nil
}

// writeAuthFailure writes failure to the OpenVPN control file and deletes the session.
func (s *Server) writeAuthFailure(sess *session.Session, reason string) {
	if s.sessionMgr == nil {
		slog.Error("session manager is nil, cannot write auth failure", // #nosec G706 -- values sanitized via sanitizeLog
			"session_id", sess.ID,
			"reason", sanitizeLog(reason),
		)
		return
	}

	written, ok := s.sessionMgr.ResultWritten(sess.ID)
	if !ok {
		slog.Error("session not found, cannot write auth failure", // #nosec G706 -- values sanitized via sanitizeLog
			"session_id", sess.ID,
			"reason", sanitizeLog(reason),
		)
		return
	}
	if written {
		slog.Warn("session already completed, skipping auth failure write", // #nosec G706 -- session.ID is crypto/rand hex
			"session_id", sess.ID,
		)
		return
	}

	if err := openvpn.WriteAuthFailure(
		sess.AuthControlFile,
		sess.AuthFailedReasonFile,
		reason,
	); err != nil {
		slog.Error("failed to write auth failure", // #nosec G706 -- session.ID is crypto/rand hex; err is internal
			"session_id", sess.ID,
			"error", err,
		)
		// Keep session for cleanup/retry attempts.
		return
	}

	slog.Info("auth failure written", // #nosec G706 -- values sanitized via sanitizeLog
		"session_id", sess.ID,
		"username", sanitizeLog(sess.Username),
		"reason", sanitizeLog(reason),
	)

	_ = s.sessionMgr.MarkResultWritten(sess.ID)
	s.sessionMgr.Delete(sess.ID)
}
