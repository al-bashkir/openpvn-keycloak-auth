package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/config"
	"github.com/al-bashkir/openvpn-keycloak-auth/internal/ipc"
)

func newTestOIDCIssuer(t *testing.T) string {
	t.Helper()

	var baseURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issuer := baseURL + "/realms/test"

		switch r.URL.Path {
		case "/realms/test/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"issuer":                 issuer,
				"authorization_endpoint": issuer + "/auth",
				"token_endpoint":         issuer + "/token",
				"jwks_uri":               issuer + "/keys",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	baseURL = ts.URL
	t.Cleanup(ts.Close)

	return baseURL + "/realms/test"
}

func TestBuildShortAuthURL(t *testing.T) {
	tests := []struct {
		name        string
		redirectURI string
		state       string
		want        string
		wantErr     bool
	}{
		{
			name:        "https with port",
			redirectURI: "https://vpn.example.com:9000/callback",
			state:       "abc123",
			want:        "https://vpn.example.com:9000/auth/abc123",
		},
		{
			name:        "https without port",
			redirectURI: "https://vpn.example.com/callback",
			state:       "def456",
			want:        "https://vpn.example.com/auth/def456",
		},
		{
			name:        "preserves base path prefix",
			redirectURI: "https://vpn.example.com/vpn/callback",
			state:       "pref123",
			want:        "https://vpn.example.com/vpn/auth/pref123",
		},
		{
			name:        "rejects base path with trailing slash",
			redirectURI: "https://vpn.example.com/vpn/callback/",
			state:       "pref456",
			wantErr:     true,
		},
		{
			name:        "http localhost",
			redirectURI: "http://localhost:9000/callback",
			state:       "state789",
			want:        "http://localhost:9000/auth/state789",
		},
		{
			name:        "redirect URI with query params stripped",
			redirectURI: "https://vpn.example.com:9000/callback?foo=bar",
			state:       "aaa",
			want:        "https://vpn.example.com:9000/auth/aaa",
		},
		{
			name:        "hex state (realistic)",
			redirectURI: "https://vpn.example.com:9000/callback",
			state:       "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
			want:        "https://vpn.example.com:9000/auth/a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
		},
		{
			name:        "invalid redirect URI",
			redirectURI: "://invalid",
			state:       "abc",
			wantErr:     true,
		},
		{
			name:        "redirect URI without path",
			redirectURI: "https://vpn.example.com",
			state:       "abc",
			wantErr:     true,
		},
		{
			name:        "redirect URI with root path",
			redirectURI: "https://vpn.example.com/",
			state:       "abc",
			wantErr:     true,
		},
		{
			name:        "redirect URI with duplicate slash path",
			redirectURI: "https://vpn.example.com/vpn//callback",
			state:       "abc",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildShortAuthURL(tt.redirectURI, tt.state)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}

			// Verify the short URL is under 256 chars (OPTION_LINE_SIZE)
			// WEB_AUTH:: prefix is 10 chars, so URL must be < 246
			webAuthLine := "WEB_AUTH::" + got
			if len(webAuthLine) >= 256 {
				t.Errorf("WEB_AUTH:: line is %d chars, must be < 256", len(webAuthLine))
			}
		})
	}
}

func TestValidateOpenVPNResultPath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{name: "absolute canonical path", filePath: "/tmp/openvpn-auth/control", wantErr: false},
		{name: "empty path", filePath: "", wantErr: true},
		{name: "relative path", filePath: "tmp/control", wantErr: true},
		{name: "non-canonical path", filePath: "/tmp/../etc/passwd", wantErr: true},
		{name: "root path", filePath: "/", wantErr: true},
		{name: "nul byte", filePath: "/tmp/control\x00bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOpenVPNResultPath("test_path", tt.filePath)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewAndHandleAuthRequest_Success(t *testing.T) {
	issuer := newTestOIDCIssuer(t)
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Listen: config.ListenConfig{
			HTTP:   "127.0.0.1:0",
			Socket: filepath.Join(tmpDir, "auth.sock"),
		},
		OIDC: config.OIDCConfig{
			Issuer:      issuer,
			ClientID:    "test-client",
			RedirectURI: "http://127.0.0.1:9000/callback",
			Scopes:      []string{"openid"},
		},
		Auth: config.AuthConfig{
			SessionTimeout: 300,
			UsernameClaim:  "preferred_username",
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer d.sessionMgr.Stop()

	req := &ipc.AuthRequest{
		Username:             "testuser",
		CommonName:           "",
		UntrustedIP:          "192.0.2.1",
		UntrustedPort:        "12345",
		AuthControlFile:      filepath.Join(tmpDir, "auth_control"),
		AuthPendingFile:      filepath.Join(tmpDir, "auth_pending"),
		AuthFailedReasonFile: filepath.Join(tmpDir, "auth_failed"),
		PendingAuthMethod:    "webauth",
	}

	resp, err := handleAuthRequest(context.Background(), cfg, d.oidcProvider, d.sessionMgr, req)
	if err != nil {
		t.Fatalf("handleAuthRequest failed: %v", err)
	}
	if resp.Status != ipc.StatusDeferred {
		t.Fatalf("expected status %q, got %q", ipc.StatusDeferred, resp.Status)
	}
	if resp.SessionID == "" {
		t.Fatal("expected session ID to be set")
	}

	sess, err := d.sessionMgr.Get(resp.SessionID)
	if err != nil {
		t.Fatalf("failed to retrieve session: %v", err)
	}
	if sess.State == "" {
		t.Fatal("expected session state to be set")
	}
	if sess.CodeVerifier == "" {
		t.Fatal("expected code verifier to be set")
	}
	if sess.AuthURL == "" {
		t.Fatal("expected full auth URL to be set")
	}
	if !strings.Contains(sess.AuthURL, "code_challenge") {
		t.Fatalf("expected full auth URL to contain PKCE params, got: %s", sess.AuthURL)
	}

	// The short auth URL only reaches the client through auth_pending_file.
	content, err := os.ReadFile(req.AuthPendingFile)
	if err != nil {
		t.Fatalf("failed to read auth_pending_file: %v", err)
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 elements (3 lines + trailing), got %d: %v", len(lines), lines)
	}
	if lines[0] != "300" {
		t.Fatalf("timeout line = %q, want %q", lines[0], "300")
	}
	if lines[1] != "webauth" {
		t.Fatalf("method line = %q, want %q", lines[1], "webauth")
	}
	shortAuthURL := strings.TrimPrefix(lines[2], "WEB_AUTH::")
	if shortAuthURL == lines[2] {
		t.Fatalf("WEB_AUTH line = %q, want a WEB_AUTH:: prefix", lines[2])
	}
	if !strings.HasSuffix(shortAuthURL, "/auth/"+sess.State) {
		t.Fatalf("expected short auth URL to end with /auth/<state>, got: %s", shortAuthURL)
	}
	if len("WEB_AUTH::")+len(shortAuthURL)+1 > maxWebAuthLineLen {
		t.Fatalf("WEB_AUTH line exceeds OpenVPN's %d-byte limit: %q", maxWebAuthLineLen, lines[2])
	}
}

func TestHandleAuthRequest_PendingWriteFailureWritesAuthFailure(t *testing.T) {
	issuer := newTestOIDCIssuer(t)
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Listen: config.ListenConfig{
			HTTP:   "127.0.0.1:0",
			Socket: filepath.Join(tmpDir, "auth.sock"),
		},
		OIDC: config.OIDCConfig{
			Issuer:      issuer,
			ClientID:    "test-client",
			RedirectURI: "http://127.0.0.1:9000/callback",
			Scopes:      []string{"openid"},
		},
		Auth: config.AuthConfig{
			SessionTimeout: 300,
			UsernameClaim:  "preferred_username",
		},
		Log: config.LogConfig{Level: "info", Format: "json"},
	}

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer d.sessionMgr.Stop()

	authControl := filepath.Join(tmpDir, "auth_control")
	authReason := filepath.Join(tmpDir, "auth_failed")

	req := &ipc.AuthRequest{
		Username:             "testuser",
		AuthControlFile:      authControl,
		AuthPendingFile:      "", // trigger WriteAuthPending failure
		AuthFailedReasonFile: authReason,
		PendingAuthMethod:    "webauth",
	}

	resp, err := handleAuthRequest(context.Background(), cfg, d.oidcProvider, d.sessionMgr, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil response on error, got: %#v", resp)
	}

	// Should write auth failure even if pending write failed.
	controlContent, err := os.ReadFile(authControl)
	if err != nil {
		t.Fatalf("failed to read auth_control_file: %v", err)
	}
	if string(controlContent) != "0" {
		t.Fatalf("auth_control_file = %q, want %q", string(controlContent), "0")
	}

	reasonContent, err := os.ReadFile(authReason)
	if err != nil {
		t.Fatalf("failed to read auth_failed_reason_file: %v", err)
	}
	if string(reasonContent) != "Failed to start authentication flow" {
		t.Fatalf("auth_failed_reason_file = %q, want %q", string(reasonContent), "Failed to start authentication flow")
	}
}

func TestRun_HTTPServerStartFailureStopsAndReturnsError(t *testing.T) {
	issuer := newTestOIDCIssuer(t)
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Listen: config.ListenConfig{
			HTTP:   "127.0.0.1:-1", // invalid port -> ListenAndServe fails immediately
			Socket: filepath.Join(tmpDir, "auth.sock"),
		},
		OIDC: config.OIDCConfig{
			Issuer:      issuer,
			ClientID:    "test-client",
			RedirectURI: "http://127.0.0.1:9000/callback",
			Scopes:      []string{"openid"},
		},
		Auth: config.AuthConfig{
			SessionTimeout: 300,
			UsernameClaim:  "preferred_username",
		},
		Log: config.LogConfig{Level: "info", Format: "json"},
	}

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- d.Run()
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected Run to fail, got nil")
		}
	case <-time.After(5 * time.Second):
		// Best-effort cleanup to avoid leaking goroutines/sockets on failure.
		_ = d.ipcServer.Stop()
		d.sessionMgr.Stop()
		t.Fatal("timeout waiting for Run to return")
	}
}
