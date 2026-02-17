package session

import (
	"log/slog"
	"time"

	"github.com/al-bashkir/openvpn-keycloak/internal/openvpn"
)

// cleanupLoop runs in a background goroutine and periodically cleans up expired sessions.
// It runs every minute (configured by cleanupTicker) and stops when the stopCleanup channel is closed.
func (m *Manager) cleanupLoop() {
	for {
		select {
		case <-m.cleanupTicker.C:
			m.cleanup()
		case <-m.stopCleanup:
			return
		}
	}
}

// cleanup removes all expired sessions from the manager.
// For sessions that expired without completing authentication,
// it writes an auth failure to the OpenVPN control file.
// This method is called periodically by cleanupLoop.
func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for sessionID, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			// Write failure for expired sessions that haven't completed
			if !session.ResultWritten {
				slog.Warn("session expired, writing auth failure",
					"session_id", sessionID,
					"username", session.Username,
					"ip", session.UntrustedIP,
				)
				err := openvpn.WriteAuthFailure(
					session.AuthControlFile,
					session.AuthFailedReasonFile,
					"Authentication timeout - session expired",
				)
				if err != nil {
					slog.Error("failed to write auth failure for expired session",
						"session_id", sessionID,
						"error", err,
					)
				}
			}

			// Remove expired session
			delete(m.sessions, sessionID)
			if session.State != "" {
				delete(m.stateIndex, session.State)
			}
			expiredCount++
		}
	}

	if expiredCount > 0 {
		slog.Info("cleaned up expired sessions", "count", expiredCount)
	}
}
