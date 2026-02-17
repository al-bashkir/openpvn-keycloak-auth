package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/al-bashkir/openvpn-keycloak/internal/config"
	"github.com/al-bashkir/openvpn-keycloak/internal/session"
)

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{
			HTTP: ":9000",
		},
		TLS: config.TLSConfig{
			Enabled: false,
		},
	}

	server, err := NewServer(cfg, nil, nil)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	if server == nil {
		t.Fatal("expected server, got nil")
	}

	if server.templates == nil {
		t.Error("expected templates to be loaded")
	}
}

func TestHealthEndpoint(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
	}

	server, err := NewServer(cfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if healthResp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", healthResp.Status)
	}
}

func TestAuthRedirectEndpoint(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
	}

	sessionMgr := session.NewManager(5 * time.Minute)
	defer sessionMgr.Stop()

	server, err := NewServer(cfg, nil, sessionMgr)
	if err != nil {
		t.Fatal(err)
	}

	// Create a session with a known state and auth URL
	sess, err := sessionMgr.Create("testuser", "", "192.0.2.1", "12345",
		"/tmp/acf", "/tmp/apf", "/tmp/arf")
	if err != nil {
		t.Fatal(err)
	}

	testState := "abc123def456"
	testAuthURL := "https://keycloak.example.com/realms/test/protocol/openid-connect/auth?client_id=openvpn&very_long_param=value"
	err = sessionMgr.UpdateOIDCFlow(sess.ID, testState, "verifier", testAuthURL)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid state redirects to auth URL", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/"+testState, nil)
		w := httptest.NewRecorder()

		server.mux.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusFound {
			t.Errorf("expected status 302, got %d", resp.StatusCode)
		}

		location := resp.Header.Get("Location")
		if location != testAuthURL {
			t.Errorf("expected Location=%s, got %s", testAuthURL, location)
		}
	})

	t.Run("unknown state returns error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/unknownstate", nil)
		w := httptest.NewRecorder()

		server.mux.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "Session not found") {
			t.Error("expected session not found error message")
		}
	})

	t.Run("empty state returns error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/", nil)
		w := httptest.NewRecorder()

		server.mux.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "Invalid auth URL") {
			t.Error("expected invalid auth URL error message")
		}
	})
}

// TestCallbackEndpointValidParams is skipped because it requires a full OIDC setup.
// TODO: Create integration tests with mock OIDC provider and session manager.
func TestCallbackEndpointValidParams(t *testing.T) {
	t.Skip("Skipping: requires full OIDC and session manager setup")
}

func TestCallbackEndpointMissingCode(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
	}

	server, err := NewServer(cfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/callback?state=abc456", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Invalid callback parameters") {
		t.Error("expected error message in response")
	}
}

func TestCallbackEndpointOIDCError(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
	}

	server, err := NewServer(cfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/callback?error=access_denied&error_description=User+denied+access", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "User denied access") {
		t.Error("expected OIDC error description in response")
	}
}

func TestRenderSuccess(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
	}

	server, err := NewServer(cfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	server.renderSuccess(w, "Test success message")

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "Test success message") {
		t.Error("expected success message in rendered HTML")
	}

	if !strings.Contains(bodyStr, "Authentication Successful") {
		t.Error("expected success title in rendered HTML")
	}
}

func TestRenderError(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
	}

	server, err := NewServer(cfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	server.renderError(w, "Test error message")

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "Test error message") {
		t.Error("expected error message in rendered HTML")
	}

	if !strings.Contains(bodyStr, "Authentication Failed") {
		t.Error("expected error title in rendered HTML")
	}
}

func TestSecurityHeaders(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
	}

	server, err := NewServer(cfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(w, req)

	resp := w.Result()

	expectedHeaders := map[string]string{
		"X-Frame-Options":        "DENY",
		"X-Content-Type-Options": "nosniff",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "no-referrer",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(header)
		if actualValue != expectedValue {
			t.Errorf("expected %s='%s', got '%s'", header, expectedValue, actualValue)
		}
	}
}

func TestRateLimiting(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
	}

	server, err := NewServer(cfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Make many requests quickly
	successCount := 0
	rateLimitCount := 0

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "192.0.2.1:12345" // Same IP
		w := httptest.NewRecorder()

		server.httpServer.Handler.ServeHTTP(w, req)

		if w.Result().StatusCode == http.StatusOK {
			successCount++
		} else if w.Result().StatusCode == http.StatusTooManyRequests {
			rateLimitCount++
		}
	}

	// Should have some rate limited requests
	if rateLimitCount == 0 {
		t.Error("expected some requests to be rate limited")
	}

	// Should have some successful requests
	if successCount == 0 {
		t.Error("expected some requests to succeed")
	}
}

func TestGracefulShutdown(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: "127.0.0.1:0"}, // Random port
	}

	server, err := NewServer(cfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Start server in background
	startErrCh := make(chan error, 1)
	go func() {
		startErrCh <- server.Start()
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	select {
	case err := <-startErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Start failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for server to stop")
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "direct connection",
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "192.0.2.1",
		},
		{
			name:       "ignores X-Forwarded-For (anti-spoofing)",
			remoteAddr: "127.0.0.1:12345",
			expectedIP: "127.0.0.1",
		},
		{
			name:       "IPv6 address",
			remoteAddr: "[::1]:12345",
			expectedIP: "::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr

			// Set spoofable headers to verify they're ignored
			req.Header.Set("X-Forwarded-For", "203.0.113.42")
			req.Header.Set("X-Real-IP", "203.0.113.42")

			ip := extractIP(req)
			if ip != tt.expectedIP {
				t.Errorf("expected IP '%s', got '%s'", tt.expectedIP, ip)
			}
		})
	}
}
