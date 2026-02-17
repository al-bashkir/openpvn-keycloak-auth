package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/ipc"
)

func writeTestConfig(t *testing.T, path string, socketPath string) {
	t.Helper()

	data := fmt.Sprintf(`listen:
  http: "127.0.0.1:0"
  socket: %q
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "test-client"
  redirect_uri: "http://localhost:9000/callback"
  scopes:
    - openid
auth:
  session_timeout: 300
  username_claim: "preferred_username"
log:
  level: "info"
  format: "json"
`, socketPath)

	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
}

func TestRunCheckConfig_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	writeTestConfig(t, cfgPath, filepath.Join(tmpDir, "auth.sock"))

	oldCfg := configFile
	oldExit := overrideExitCode
	t.Cleanup(func() {
		configFile = oldCfg
		overrideExitCode = oldExit
	})
	configFile = cfgPath
	overrideExitCode = -1

	if err := runCheckConfig(nil, nil); err != nil {
		t.Fatalf("runCheckConfig failed: %v", err)
	}
	if overrideExitCode != -1 {
		t.Fatalf("overrideExitCode = %d, want -1 (unset)", overrideExitCode)
	}
}

func TestRunCheckConfig_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	// Missing required oidc.issuer
	data := `listen:
  http: "127.0.0.1:0"
  socket: "/tmp/test.sock"
oidc:
  client_id: "test-client"
  redirect_uri: "http://localhost:9000/callback"
  scopes:
    - openid
auth:
  session_timeout: 300
  username_claim: "preferred_username"
log:
  level: "info"
  format: "json"
`
	if err := os.WriteFile(cfgPath, []byte(data), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	oldCfg := configFile
	oldExit := overrideExitCode
	t.Cleanup(func() {
		configFile = oldCfg
		overrideExitCode = oldExit
	})
	configFile = cfgPath
	overrideExitCode = -1

	if err := runCheckConfig(nil, nil); err != nil {
		t.Fatalf("runCheckConfig returned unexpected error: %v", err)
	}
	if overrideExitCode != ExitConfig {
		t.Fatalf("overrideExitCode = %d, want %d (ExitConfig)", overrideExitCode, ExitConfig)
	}
}

func TestRunServe_ConfigLoadFailure(t *testing.T) {
	old := configFile
	t.Cleanup(func() { configFile = old })
	configFile = filepath.Join(t.TempDir(), "does-not-exist.yaml")

	if err := runServe(nil, nil); err == nil {
		t.Fatal("expected runServe to fail, got nil")
	}
}

func TestRunVersion(t *testing.T) {
	oldVersion, oldCommit, oldBuildDate := version, commit, buildDate
	t.Cleanup(func() {
		version, commit, buildDate = oldVersion, oldCommit, oldBuildDate
	})

	version = "1.2.3"
	commit = "deadbeef"
	buildDate = "2026-02-17"

	runVersion(nil, nil)
}

func TestRunAuth_Deferred(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "auth.sock")

	// Start a minimal IPC server (daemon side)
	server := ipc.NewServer(socketPath, func(ctx context.Context, req *ipc.AuthRequest) (*ipc.AuthResponse, error) {
		return &ipc.AuthResponse{
			Status:    ipc.StatusDeferred,
			SessionID: "test-session-123",
			AuthURL:   "http://localhost/auth",
		}, nil
	})
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("failed to start IPC server: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("server.Stop failed: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Write config pointing to the test socket
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	writeTestConfig(t, cfgPath, socketPath)

	oldConfigFile := configFile
	oldOverrideExitCode := overrideExitCode
	t.Cleanup(func() {
		configFile = oldConfigFile
		overrideExitCode = oldOverrideExitCode
	})
	configFile = cfgPath
	overrideExitCode = -1

	// OpenVPN environment (via-file mode)
	t.Setenv("auth_control_file", filepath.Join(tmpDir, "auth_control"))
	t.Setenv("auth_pending_file", filepath.Join(tmpDir, "auth_pending"))
	t.Setenv("auth_failed_reason_file", filepath.Join(tmpDir, "auth_failed"))
	t.Setenv("untrusted_ip", "192.0.2.1")
	t.Setenv("untrusted_port", "12345")
	t.Setenv("IV_SSO", "webauth")

	// Credentials file: username + password (ignored)
	credsPath := filepath.Join(tmpDir, "creds")
	if err := os.WriteFile(credsPath, []byte("testuser\nsso\n"), 0600); err != nil {
		t.Fatalf("failed to write credentials: %v", err)
	}

	if err := runAuth(nil, []string{credsPath}); err != nil {
		t.Fatalf("runAuth failed: %v", err)
	}
	if overrideExitCode != ExitDeferred {
		t.Fatalf("overrideExitCode = %d, want %d", overrideExitCode, ExitDeferred)
	}
}
