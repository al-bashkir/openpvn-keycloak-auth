package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Manager manages authentication sessions in-memory with TTL-based cleanup.
// It is thread-safe and supports concurrent access.
type Manager struct {
	mu             sync.RWMutex
	sessions       map[string]*Session // sessionID -> Session
	stateIndex     map[string]*Session // state -> Session
	sessionTimeout time.Duration
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
}

// NewManager creates a new session manager with the specified timeout.
// It automatically starts a background cleanup goroutine that runs every minute.
func NewManager(sessionTimeout time.Duration) *Manager {
	m := &Manager{
		sessions:       make(map[string]*Session),
		stateIndex:     make(map[string]*Session),
		sessionTimeout: sessionTimeout,
		cleanupTicker:  time.NewTicker(1 * time.Minute),
		stopCleanup:    make(chan struct{}),
	}

	// Start cleanup goroutine
	go m.cleanupLoop()

	return m
}

// Stop stops the session manager's cleanup goroutine.
// Call this when shutting down the daemon.
func (m *Manager) Stop() {
	m.cleanupTicker.Stop()
	close(m.stopCleanup)
}

// Create creates a new session with the given parameters.
// The session ID is generated using crypto/rand (64 hex characters).
// Returns the new session or an error.
func (m *Manager) Create(username, commonName, untrustedIP, untrustedPort string,
	authControlFile, authPendingFile, authFailedReasonFile string) (*Session, error) {

	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Create session
	session := &Session{
		ID:                   sessionID,
		Username:             username,
		CommonName:           commonName,
		UntrustedIP:          untrustedIP,
		UntrustedPort:        untrustedPort,
		AuthControlFile:      authControlFile,
		AuthPendingFile:      authPendingFile,
		AuthFailedReasonFile: authFailedReasonFile,
		CreatedAt:            time.Now(),
		ExpiresAt:            time.Now().Add(m.sessionTimeout),
	}

	// Store session
	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return session, nil
}

// UpdateOIDCFlow updates a session with OIDC flow data (state, code verifier, auth URL).
// This is called after starting the OIDC authorization flow.
// The state is indexed for fast lookup during the callback.
func (m *Manager) UpdateOIDCFlow(sessionID, state, codeVerifier, authURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.State = state
	session.CodeVerifier = codeVerifier
	session.AuthURL = authURL

	// Add to state index for callback lookup
	m.stateIndex[state] = session

	return nil
}

// Get retrieves a session by its ID.
// Returns an error if the session is not found or has expired.
func (m *Manager) Get(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	// Check expiry
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// GetByState retrieves a session by its OIDC state parameter.
// This is used during the OAuth2 callback to find the session.
// Returns an error if the session is not found or has expired.
func (m *Manager) GetByState(state string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.stateIndex[state]
	if !ok {
		return nil, fmt.Errorf("session not found for state")
	}

	// Check expiry
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// ResultWritten returns whether a session has written an auth result.
// The second return value is false if the session does not exist (deleted/expired).
func (m *Manager) ResultWritten(sessionID string) (bool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return false, false
	}

	return session.ResultWritten, true
}

// MarkResultWritten atomically sets ResultWritten on a session.
// Returns false if the session was not found or was already marked.
func (m *Manager) MarkResultWritten(sessionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok || session.ResultWritten {
		return false
	}

	session.ResultWritten = true
	return true
}

// Delete removes a session from the manager.
// This should be called after the authentication completes (success or failure).
func (m *Manager) Delete(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return
	}

	// Remove from both indexes
	delete(m.sessions, sessionID)
	if session.State != "" {
		delete(m.stateIndex, session.State)
	}
}

// Count returns the current number of active sessions.
// Useful for monitoring and testing.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// generateSessionID generates a cryptographically secure random session ID.
// The ID is 64 hex characters (32 random bytes).
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
