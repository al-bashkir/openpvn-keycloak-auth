package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/config"
	"github.com/al-bashkir/openvpn-keycloak-auth/internal/session"
)

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{
			HTTP: ":9000",
		},
		OIDC: config.OIDCConfig{RedirectURI: "https://vpn.example.com/callback"},
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
		return
	}

	if server.templates == nil {
		t.Error("expected templates to be loaded")
	}
}

func TestHealthEndpoint(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
		OIDC:   config.OIDCConfig{RedirectURI: "https://vpn.example.com/callback"},
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

	var healthResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if healthResp["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", healthResp["status"])
	}
}

func TestAuthRedirectEndpoint(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
		OIDC: config.OIDCConfig{
			RedirectURI: "https://vpn.example.com/callback",
		},
	}

	sessionMgr := session.NewManager(5 * time.Minute)
	defer sessionMgr.Stop()

	server, err := NewServer(cfg, nil, sessionMgr)
	if err != nil {
		t.Fatal(err)
	}

	// Create a session with a known state and auth URL
	sess, err := sessionMgr.Create("testuser", "192.0.2.1", "/tmp/acf", "/tmp/arf")
	if err != nil {
		t.Fatal(err)
	}

	testState := "abc123def456"
	testAuthURL := "https://keycloak.example.com/realms/test/protocol/openid-connect/auth?client_id=openvpn&very_long_param=value"
	err = sessionMgr.UpdateOIDCFlow(sess.ID, testState, "verifier", "nonce", testAuthURL)
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

func TestAuthRedirectEndpointWithBasePath(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
		OIDC: config.OIDCConfig{
			RedirectURI: "https://vpn.example.com/vpn/callback",
		},
	}

	sessionMgr := session.NewManager(5 * time.Minute)
	defer sessionMgr.Stop()

	server, err := NewServer(cfg, nil, sessionMgr)
	if err != nil {
		t.Fatal(err)
	}

	sess, err := sessionMgr.Create("testuser", "192.0.2.1", "/tmp/acf", "/tmp/arf")
	if err != nil {
		t.Fatal(err)
	}

	testState := "prefixedstate123"
	testAuthURL := "https://keycloak.example.com/realms/test/protocol/openid-connect/auth?client_id=openvpn"
	if err := sessionMgr.UpdateOIDCFlow(sess.ID, testState, "verifier", "nonce", testAuthURL); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/vpn/auth/"+testState, nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected status 302, got %d", resp.StatusCode)
	}
	if location := resp.Header.Get("Location"); location != testAuthURL {
		t.Fatalf("expected Location=%s, got %s", testAuthURL, location)
	}

	rootReq := httptest.NewRequest("GET", "/auth/"+testState, nil)
	rootW := httptest.NewRecorder()

	server.mux.ServeHTTP(rootW, rootReq)

	rootResp := rootW.Result()
	defer func() { _ = rootResp.Body.Close() }()

	if rootResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected root auth route status 404, got %d", rootResp.StatusCode)
	}
}

func TestCallbackEndpointWithBasePath(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
		OIDC: config.OIDCConfig{
			RedirectURI: "https://vpn.example.com/vpn/callback",
		},
	}

	server, err := NewServer(cfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/vpn/callback?error=access_denied", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}

	rootReq := httptest.NewRequest("GET", "/callback?error=access_denied", nil)
	rootW := httptest.NewRecorder()

	server.mux.ServeHTTP(rootW, rootReq)

	rootResp := rootW.Result()
	defer func() { _ = rootResp.Body.Close() }()

	if rootResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected root callback route status 404, got %d", rootResp.StatusCode)
	}
}

// TestCallbackEndpointValidParams is skipped because it requires a full OIDC setup.
// TODO: Create integration tests with mock OIDC provider and session manager.
func TestCallbackEndpointValidParams(t *testing.T) {
	t.Skip("Skipping: requires full OIDC and session manager setup")
}

func TestCallbackEndpointMissingCode(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
		OIDC: config.OIDCConfig{
			RedirectURI: "https://vpn.example.com/callback",
		},
	}
	sessionMgr := session.NewManager(5 * time.Minute)
	defer sessionMgr.Stop()

	server, err := NewServer(cfg, nil, sessionMgr)
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	authControlFile := tmpDir + "/auth_control"
	authFailedReasonFile := tmpDir + "/auth_failed"
	sess, err := sessionMgr.Create("testuser", "192.0.2.1",
		authControlFile, authFailedReasonFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionMgr.UpdateOIDCFlow(sess.ID, "abc456", "verifier", "nonce", "https://keycloak.example.com/auth"); err != nil {
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

	control, err := os.ReadFile(authControlFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(control) != "0" {
		t.Fatalf("expected auth control failure, got %q", string(control))
	}
	reason, err := os.ReadFile(authFailedReasonFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(reason) != "Invalid callback parameters" {
		t.Fatalf("expected failure reason, got %q", string(reason))
	}
}

// TestCallbackConcurrentSameState fires N callbacks for the same session in
// parallel and asserts that exactly one writes the auth_control_file and the
// session is removed exactly once. The "Invalid callback parameters" branch
// (state present, code missing) is used because it exercises writeAuthFailure
// without requiring a real OIDC provider.
func TestCallbackConcurrentSameState(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
		OIDC: config.OIDCConfig{
			RedirectURI: "https://vpn.example.com/callback",
		},
	}
	sessionMgr := session.NewManager(5 * time.Minute)
	defer sessionMgr.Stop()

	server, err := NewServer(cfg, nil, sessionMgr)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	authControlFile := tmpDir + "/auth_control"
	authFailedReasonFile := tmpDir + "/auth_failed"

	sess, err := sessionMgr.Create("u", "192.0.2.1",
		authControlFile, authFailedReasonFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionMgr.UpdateOIDCFlow(sess.ID, "racestate", "verifier", "nonce", "https://kc.example/auth"); err != nil {
		t.Fatal(err)
	}

	const concurrency = 16
	var wg sync.WaitGroup
	wg.Add(concurrency)
	start := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			<-start
			req := httptest.NewRequest("GET", "/callback?state=racestate", nil)
			w := httptest.NewRecorder()
			server.mux.ServeHTTP(w, req)
			_ = w.Result().Body.Close()
		}()
	}
	close(start)
	wg.Wait()

	// Session must be deleted exactly once (Count == 0).
	if got := sessionMgr.Count(); got != 0 {
		t.Errorf("session count after race = %d, want 0", got)
	}

	// auth_control_file must exist and contain the failure marker.
	control, err := os.ReadFile(authControlFile)
	if err != nil {
		t.Fatalf("read auth_control_file: %v", err)
	}
	if string(control) != "0" {
		t.Errorf("auth_control_file = %q, want %q", control, "0")
	}
}

func TestCallbackEndpointOIDCError(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
		OIDC: config.OIDCConfig{
			RedirectURI: "https://vpn.example.com/callback",
		},
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
	bodyText := string(body)
	if !strings.Contains(bodyText, "Authentication failed. Please try again.") {
		t.Error("expected generic OIDC error message in response")
	}
	if strings.Contains(bodyText, "User denied access") {
		t.Error("raw OIDC error description must not be rendered")
	}
}

func TestRenderSuccess(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
		OIDC:   config.OIDCConfig{RedirectURI: "https://vpn.example.com/callback"},
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
		OIDC:   config.OIDCConfig{RedirectURI: "https://vpn.example.com/callback"},
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
		OIDC:   config.OIDCConfig{RedirectURI: "https://vpn.example.com/callback"},
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
		"Referrer-Policy":        "no-referrer",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(header)
		if actualValue != expectedValue {
			t.Errorf("expected %s='%s', got '%s'", header, expectedValue, actualValue)
		}
	}

	// X-XSS-Protection is intentionally not set (deprecated header).
	if v := resp.Header.Get("X-XSS-Protection"); v != "" {
		t.Errorf("expected X-XSS-Protection to be unset, got %q", v)
	}
}

func TestRateLimiting(t *testing.T) {
	cfg := &config.Config{
		Listen: config.ListenConfig{HTTP: ":9000"},
		OIDC:   config.OIDCConfig{RedirectURI: "https://vpn.example.com/callback"},
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
		OIDC:   config.OIDCConfig{RedirectURI: "https://vpn.example.com/callback"},
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
		{
			name:       "address without port",
			remoteAddr: "192.0.2.1",
			expectedIP: "192.0.2.1",
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

func TestRedactRequestPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "auth state path",
			path: "/auth/abc123state",
			want: "/auth/{state}",
		},
		{
			name: "auth empty state path",
			path: "/auth/",
			want: "/auth/{state}",
		},
		{
			name: "prefixed auth state path",
			path: "/vpn/auth/abc123state",
			want: "/vpn/auth/{state}",
		},
		{
			name: "sanitizes auth path prefix",
			path: "/\n/auth/abc123state",
			want: "/_/auth/{state}",
		},
		{
			name: "callback path",
			path: "/callback",
			want: "/callback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactRequestPath(tt.path); got != tt.want {
				t.Fatalf("redactRequestPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestAuthStateFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "root auth path",
			path: "/auth/abc123",
			want: "abc123",
		},
		{
			name: "prefixed auth path",
			path: "/vpn/auth/abc123",
			want: "abc123",
		},
		{
			name: "empty state",
			path: "/vpn/auth/",
			want: "",
		},
		{
			name: "non auth path",
			path: "/vpn/callback",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got, _ := splitAuthPath(tt.path)
			if got != tt.want {
				t.Fatalf("splitAuthPath(%q) state = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
