package auth

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/al-bashkir/openvpn-keycloak/internal/ipc"
)

func TestParseEnv(t *testing.T) {
	envKeys := []string{
		"auth_control_file",
		"auth_pending_file",
		"auth_failed_reason_file",
		"username",
		"password",
		"common_name",
		"untrusted_ip",
		"untrusted_port",
		"IV_SSO",
	}

	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid environment",
			envVars: map[string]string{
				"auth_control_file":       "/tmp/acf",
				"auth_pending_file":       "/tmp/apf",
				"auth_failed_reason_file": "/tmp/arf",
				"username":                "testuser",
				"untrusted_ip":            "192.0.2.1",
			},
			wantErr: false,
		},
		{
			name: "missing auth_control_file",
			envVars: map[string]string{
				"auth_pending_file":       "/tmp/apf",
				"auth_failed_reason_file": "/tmp/arf",
			},
			wantErr: true,
			errMsg:  "auth_control_file",
		},
		{
			name: "missing auth_pending_file",
			envVars: map[string]string{
				"auth_control_file":       "/tmp/acf",
				"auth_failed_reason_file": "/tmp/arf",
			},
			wantErr: true,
			errMsg:  "auth_pending_file",
		},
		{
			name: "missing auth_failed_reason_file",
			envVars: map[string]string{
				"auth_control_file": "/tmp/acf",
				"auth_pending_file": "/tmp/apf",
			},
			wantErr: true,
			errMsg:  "auth_failed_reason_file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure a clean baseline per test case
			for _, k := range envKeys {
				t.Setenv(k, "")
			}
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			env, err := ParseEnv()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if env == nil {
					t.Error("expected env, got nil")
				} else {
					// Validate parsed values
					if env.Username != tt.envVars["username"] {
						t.Errorf("username = %v, want %v", env.Username, tt.envVars["username"])
					}
					if env.AuthControlFile != tt.envVars["auth_control_file"] {
						t.Errorf("auth_control_file = %v, want %v", env.AuthControlFile, tt.envVars["auth_control_file"])
					}
				}
			}
		})
	}
}

func TestReadCredentialsFile(t *testing.T) {
	tests := []struct {
		name         string
		fileContent  string
		wantUsername string
		wantPassword string
		wantErr      bool
	}{
		{
			name:         "valid credentials",
			fileContent:  "testuser\nmypassword\n",
			wantUsername: "testuser",
			wantPassword: "mypassword",
			wantErr:      false,
		},
		{
			name:         "valid credentials with extra newlines",
			fileContent:  "testuser\nmypassword\n\n\n",
			wantUsername: "testuser",
			wantPassword: "mypassword",
			wantErr:      false,
		},
		{
			name:         "credentials with whitespace",
			fileContent:  "  testuser  \n  mypassword  \n",
			wantUsername: "testuser",
			wantPassword: "mypassword",
			wantErr:      false,
		},
		{
			name:         "only username (SSO flow, no password)",
			fileContent:  "testuser\n",
			wantUsername: "testuser",
			wantPassword: "",
			wantErr:      false,
		},
		{
			name:         "username with empty password line (SSO flow)",
			fileContent:  "testuser\n\n",
			wantUsername: "testuser",
			wantPassword: "",
			wantErr:      false,
		},
		{
			name:        "empty file",
			fileContent: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpfile, err := os.CreateTemp(t.TempDir(), "creds-*")
			if err != nil {
				t.Fatal(err)
			}

			if _, err := tmpfile.Write([]byte(tt.fileContent)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			username, password, err := readCredentialsFile(tmpfile.Name())

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if username != tt.wantUsername {
					t.Errorf("username = %v, want %v", username, tt.wantUsername)
				}
				if password != tt.wantPassword {
					t.Errorf("password = %v, want %v", password, tt.wantPassword)
				}
			}
		})
	}
}

func TestHandlerRun(t *testing.T) {
	// Create temp directory for socket
	tmpDir := t.TempDir()

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Set up OpenVPN environment
	t.Setenv("auth_control_file", "/tmp/test_acf")
	t.Setenv("auth_pending_file", "/tmp/test_apf")
	t.Setenv("auth_failed_reason_file", "/tmp/test_arf")
	t.Setenv("untrusted_ip", "192.0.2.1")
	t.Setenv("untrusted_port", "12345")
	t.Setenv("IV_SSO", "webauth,crtext")

	// Create IPC server
	handler := func(ctx context.Context, req *ipc.AuthRequest) (*ipc.AuthResponse, error) {
		return &ipc.AuthResponse{
			Status:    ipc.StatusDeferred,
			SessionID: "test-session-123",
			AuthURL:   "https://keycloak.example.com/auth",
		}, nil
	}

	server := ipc.NewServer(socketPath, handler)
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("server.Stop failed: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Create credentials file
	credsFile, err := os.CreateTemp(tmpDir, "creds-*")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := credsFile.WriteString("testuser\nsso\n"); err != nil {
		t.Fatal(err)
	}
	if err := credsFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Create handler
	authHandler := NewHandler(socketPath)

	// Run auth
	exitCode := authHandler.Run(context.Background(), credsFile.Name())

	if exitCode != ExitDeferred {
		t.Errorf("expected exit code %d (deferred), got %d", ExitDeferred, exitCode)
	}
}

func TestHandlerRunDaemonError(t *testing.T) {
	tmpDir := t.TempDir()

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Set up environment
	t.Setenv("auth_control_file", "/tmp/test_acf")
	t.Setenv("auth_pending_file", "/tmp/test_apf")
	t.Setenv("auth_failed_reason_file", "/tmp/test_arf")
	t.Setenv("IV_SSO", "webauth,crtext")

	// Server returns error
	handler := func(ctx context.Context, req *ipc.AuthRequest) (*ipc.AuthResponse, error) {
		return &ipc.AuthResponse{
			Status: ipc.StatusError,
			Error:  "daemon not initialized",
		}, nil
	}

	server := ipc.NewServer(socketPath, handler)
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("server.Stop failed: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Create credentials file
	credsFile, err := os.CreateTemp(tmpDir, "creds-*")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := credsFile.WriteString("testuser\nsso\n"); err != nil {
		t.Fatal(err)
	}
	if err := credsFile.Close(); err != nil {
		t.Fatal(err)
	}

	authHandler := NewHandler(socketPath)
	exitCode := authHandler.Run(context.Background(), credsFile.Name())

	if exitCode != ExitFailure {
		t.Errorf("expected exit code %d (failure), got %d", ExitFailure, exitCode)
	}
}

func TestHandlerRunNoDaemon(t *testing.T) {
	// Set up environment
	t.Setenv("auth_control_file", "/tmp/test_acf")
	t.Setenv("auth_pending_file", "/tmp/test_apf")
	t.Setenv("auth_failed_reason_file", "/tmp/test_arf")
	t.Setenv("IV_SSO", "webauth,crtext")

	// Create credentials file
	credsFile, err := os.CreateTemp(t.TempDir(), "creds-*")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := credsFile.WriteString("testuser\nsso\n"); err != nil {
		t.Fatal(err)
	}
	if err := credsFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Try to connect to non-existent daemon
	authHandler := NewHandler("/nonexistent/socket.sock")
	exitCode := authHandler.Run(context.Background(), credsFile.Name())

	if exitCode != ExitFailure {
		t.Errorf("expected exit code %d (failure), got %d", ExitFailure, exitCode)
	}
}

func TestSelectPendingMethod(t *testing.T) {
	tests := []struct {
		name    string
		methods []string
		want    string
	}{
		{
			name:    "webauth only",
			methods: []string{"webauth"},
			want:    "webauth",
		},
		{
			name:    "openurl only",
			methods: []string{"openurl"},
			want:    "openurl",
		},
		{
			name:    "webauth and crtext (Tunnelblick)",
			methods: []string{"webauth", "crtext"},
			want:    "webauth",
		},
		{
			name:    "openurl and webauth - prefers webauth",
			methods: []string{"openurl", "webauth"},
			want:    "webauth",
		},
		{
			name:    "crtext only - not supported",
			methods: []string{"crtext"},
			want:    "",
		},
		{
			name:    "empty list",
			methods: []string{},
			want:    "",
		},
		{
			name:    "nil list",
			methods: nil,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectPendingMethod(tt.methods)
			if got != tt.want {
				t.Errorf("selectPendingMethod(%v) = %q, want %q", tt.methods, got, tt.want)
			}
		})
	}
}

func TestParseEnvWithIVSSO(t *testing.T) {
	// Set required fields
	t.Setenv("auth_control_file", "/tmp/acf")
	t.Setenv("auth_pending_file", "/tmp/apf")
	t.Setenv("auth_failed_reason_file", "/tmp/arf")

	tests := []struct {
		name        string
		ivSSO       string
		wantMethods []string
	}{
		{
			name:        "webauth,crtext",
			ivSSO:       "webauth,crtext",
			wantMethods: []string{"webauth", "crtext"},
		},
		{
			name:        "openurl",
			ivSSO:       "openurl",
			wantMethods: []string{"openurl"},
		},
		{
			name:        "empty IV_SSO",
			ivSSO:       "",
			wantMethods: nil,
		},
		{
			name:        "spaces in IV_SSO",
			ivSSO:       "webauth, crtext, openurl",
			wantMethods: []string{"webauth", "crtext", "openurl"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("IV_SSO", tt.ivSSO)

			env, err := ParseEnv()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(env.SSOMethods) != len(tt.wantMethods) {
				t.Errorf("SSOMethods = %v, want %v", env.SSOMethods, tt.wantMethods)
				return
			}

			for i, m := range env.SSOMethods {
				if m != tt.wantMethods[i] {
					t.Errorf("SSOMethods[%d] = %q, want %q", i, m, tt.wantMethods[i])
				}
			}
		})
	}
}
