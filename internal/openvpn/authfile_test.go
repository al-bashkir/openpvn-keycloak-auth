package openvpn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAuthPending(t *testing.T) {
	tmpDir := t.TempDir()
	pendingFile := filepath.Join(tmpDir, "auth_pending")

	tests := []struct {
		name            string
		filePath        string
		timeoutSeconds  int
		method          string
		authURL         string
		wantErr         bool
		wantErrContains string
	}{
		{
			name:           "valid auth pending with webauth",
			filePath:       pendingFile,
			timeoutSeconds: 300,
			method:         "webauth",
			authURL:        "https://keycloak.example.com/auth?client_id=test",
			wantErr:        false,
		},
		{
			name:           "valid auth pending with openurl",
			filePath:       pendingFile,
			timeoutSeconds: 300,
			method:         "openurl",
			authURL:        "https://keycloak.example.com/auth?client_id=test",
			wantErr:        false,
		},
		{
			name:            "empty file path",
			filePath:        "",
			timeoutSeconds:  300,
			method:          "webauth",
			authURL:         "https://example.com",
			wantErr:         true,
			wantErrContains: "path is empty",
		},
		{
			name:            "empty auth URL",
			filePath:        pendingFile,
			timeoutSeconds:  300,
			method:          "webauth",
			authURL:         "",
			wantErr:         true,
			wantErrContains: "auth URL is empty",
		},
		{
			name:            "empty method",
			filePath:        pendingFile,
			timeoutSeconds:  300,
			method:          "",
			authURL:         "https://example.com",
			wantErr:         true,
			wantErrContains: "method is empty",
		},
		{
			name:            "zero timeout",
			filePath:        pendingFile,
			timeoutSeconds:  0,
			method:          "webauth",
			authURL:         "https://example.com",
			wantErr:         true,
			wantErrContains: "timeout must be positive",
		},
		{
			name:            "negative timeout",
			filePath:        pendingFile,
			timeoutSeconds:  -1,
			method:          "webauth",
			authURL:         "https://example.com",
			wantErr:         true,
			wantErrContains: "timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up file before each test
			_ = os.Remove(tt.filePath)

			err := WriteAuthPending(tt.filePath, tt.timeoutSeconds, tt.method, tt.authURL)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error = %v, want error containing %q", err, tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify file contents
			content, err := os.ReadFile(tt.filePath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			lines := strings.Split(string(content), "\n")
			if len(lines) != 4 { // 3 lines + empty string after final newline
				t.Errorf("expected 4 elements (3 lines + trailing), got %d: %v", len(lines), lines)
			}

			// Verify line 1: timeout
			if lines[0] != "300" {
				t.Errorf("line 1 = %q, want %q", lines[0], "300")
			}

			// Verify line 2: method
			if lines[1] != tt.method {
				t.Errorf("line 2 = %q, want %q", lines[1], tt.method)
			}

			// Verify line 3: WEB_AUTH:: prefix
			if !strings.HasPrefix(lines[2], "WEB_AUTH::") {
				t.Errorf("line 3 does not start with WEB_AUTH::, got %q", lines[2])
			}

			expectedLine3 := "WEB_AUTH::" + tt.authURL
			if lines[2] != expectedLine3 {
				t.Errorf("line 3 = %q, want %q", lines[2], expectedLine3)
			}

			// Verify file permissions
			info, err := os.Stat(tt.filePath)
			if err != nil {
				t.Fatalf("failed to stat file: %v", err)
			}

			perm := info.Mode().Perm()
			if perm != 0600 {
				t.Errorf("file permissions = %o, want 0600", perm)
			}
		})
	}
}

func TestWriteAuthSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	controlFile := filepath.Join(tmpDir, "auth_control")

	tests := []struct {
		name            string
		filePath        string
		wantErr         bool
		wantErrContains string
	}{
		{
			name:     "valid auth success",
			filePath: controlFile,
			wantErr:  false,
		},
		{
			name:            "empty file path",
			filePath:        "",
			wantErr:         true,
			wantErrContains: "path is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Remove(tt.filePath)

			err := WriteAuthSuccess(tt.filePath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error = %v, want error containing %q", err, tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify file contents
			content, err := os.ReadFile(tt.filePath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			if string(content) != "1" {
				t.Errorf("content = %q, want %q", content, "1")
			}

			// Verify file permissions
			info, err := os.Stat(tt.filePath)
			if err != nil {
				t.Fatalf("failed to stat file: %v", err)
			}

			perm := info.Mode().Perm()
			if perm != 0600 {
				t.Errorf("file permissions = %o, want 0600", perm)
			}
		})
	}
}

func TestWriteAuthFailure(t *testing.T) {
	tmpDir := t.TempDir()
	controlFile := filepath.Join(tmpDir, "auth_control")
	reasonFile := filepath.Join(tmpDir, "auth_failed_reason")

	tests := []struct {
		name                 string
		authControlFile      string
		authFailedReasonFile string
		reason               string
		wantErr              bool
		wantErrContains      string
	}{
		{
			name:                 "valid auth failure with reason",
			authControlFile:      controlFile,
			authFailedReasonFile: reasonFile,
			reason:               "Invalid credentials",
			wantErr:              false,
		},
		{
			name:                 "valid auth failure without reason file",
			authControlFile:      controlFile,
			authFailedReasonFile: "",
			reason:               "Some error",
			wantErr:              false,
		},
		{
			name:                 "valid auth failure without reason text",
			authControlFile:      controlFile,
			authFailedReasonFile: reasonFile,
			reason:               "",
			wantErr:              false,
		},
		{
			name:                 "empty control file path",
			authControlFile:      "",
			authFailedReasonFile: reasonFile,
			reason:               "Some error",
			wantErr:              true,
			wantErrContains:      "path is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Remove(tt.authControlFile)
			_ = os.Remove(tt.authFailedReasonFile)

			err := WriteAuthFailure(tt.authControlFile, tt.authFailedReasonFile, tt.reason)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error = %v, want error containing %q", err, tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify control file contents
			content, err := os.ReadFile(tt.authControlFile)
			if err != nil {
				t.Fatalf("failed to read control file: %v", err)
			}

			if string(content) != "0" {
				t.Errorf("control file content = %q, want %q", content, "0")
			}

			// Verify control file permissions
			info, err := os.Stat(tt.authControlFile)
			if err != nil {
				t.Fatalf("failed to stat control file: %v", err)
			}

			perm := info.Mode().Perm()
			if perm != 0600 {
				t.Errorf("control file permissions = %o, want 0600", perm)
			}

			// Verify reason file if provided
			if tt.authFailedReasonFile != "" && tt.reason != "" {
				reasonContent, err := os.ReadFile(tt.authFailedReasonFile)
				if err != nil {
					t.Fatalf("failed to read reason file: %v", err)
				}

				if string(reasonContent) != tt.reason {
					t.Errorf("reason file content = %q, want %q", reasonContent, tt.reason)
				}
			}
		})
	}
}

func TestWriteOrder(t *testing.T) {
	// This test verifies that auth_failed_reason_file is written BEFORE auth_control_file
	// We can't easily test timing, but we can verify both files exist after WriteAuthFailure

	tmpDir := t.TempDir()
	controlFile := filepath.Join(tmpDir, "auth_control")
	reasonFile := filepath.Join(tmpDir, "auth_failed_reason")

	err := WriteAuthFailure(controlFile, reasonFile, "Test error")
	if err != nil {
		t.Fatalf("WriteAuthFailure failed: %v", err)
	}

	// Both files should exist
	if _, err := os.Stat(controlFile); err != nil {
		t.Errorf("control file does not exist: %v", err)
	}

	if _, err := os.Stat(reasonFile); err != nil {
		t.Errorf("reason file does not exist: %v", err)
	}

	// Verify contents
	controlContent, _ := os.ReadFile(controlFile)
	reasonContent, _ := os.ReadFile(reasonFile)

	if string(controlContent) != "0" {
		t.Errorf("control file = %q, want %q", controlContent, "0")
	}

	if string(reasonContent) != "Test error" {
		t.Errorf("reason file = %q, want %q", reasonContent, "Test error")
	}
}
