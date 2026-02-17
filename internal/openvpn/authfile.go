// Package openvpn provides functions to write OpenVPN control files.
package openvpn

import (
	"fmt"
	"log/slog"
	"os"
)

const (
	// authPendingFormat is the exact 3-line format required by OpenVPN.
	// Line 1: timeout in seconds
	// Line 2: pending auth method (must match one of the client's IV_SSO values)
	// Line 3: "WEB_AUTH::" prefix followed by the authorization URL
	authPendingFormat = "%d\n%s\nWEB_AUTH::%s\n"
)

// WriteAuthPending writes the auth_pending_file to trigger browser opening.
// The file must be exactly 3 lines in the format:
//
//	<timeout_seconds>
//	<method>           (e.g. "webauth" or "openurl")
//	WEB_AUTH::<auth_url>
//
// The method must match one of the client's IV_SSO capabilities.
// Common values: "webauth" (Tunnelblick, OpenVPN Connect), "openurl" (newer clients).
//
// This triggers OpenVPN 2.6+ to send an INFO_PRE message to the client,
// which opens a browser to the authorization URL.
func WriteAuthPending(filePath string, timeoutSeconds int, method string, authURL string) error {
	if filePath == "" {
		return fmt.Errorf("auth_pending_file path is empty")
	}

	if authURL == "" {
		return fmt.Errorf("auth URL is empty")
	}

	if method == "" {
		return fmt.Errorf("pending auth method is empty")
	}

	if timeoutSeconds <= 0 {
		return fmt.Errorf("timeout must be positive, got %d", timeoutSeconds)
	}

	content := fmt.Sprintf(authPendingFormat, timeoutSeconds, method, authURL)

	// Write atomically with 0600 permissions
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write auth_pending_file: %w", err)
	}

	slog.Debug("wrote auth_pending_file", "path", filePath, "timeout", timeoutSeconds)
	return nil
}

// WriteAuthSuccess writes "1" to auth_control_file to indicate successful authentication.
// OpenVPN will then allow the client to connect.
func WriteAuthSuccess(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("auth_control_file path is empty")
	}

	if err := os.WriteFile(filePath, []byte("1"), 0600); err != nil {
		return fmt.Errorf("failed to write auth_control_file (success): %w", err)
	}

	slog.Debug("wrote auth_control_file (success)", "path", filePath)
	return nil
}

// WriteAuthFailure writes an error message to auth_failed_reason_file
// and "0" to auth_control_file to indicate authentication failure.
//
// IMPORTANT: The reason file MUST be written BEFORE the control file.
// This is because OpenVPN reads the reason file when it sees "0" in the control file.
//
// OpenVPN will reject the connection and show the reason to the user.
func WriteAuthFailure(authControlFile, authFailedReasonFile, reason string) error {
	if authControlFile == "" {
		return fmt.Errorf("auth_control_file path is empty")
	}

	// 1. Write error reason FIRST (if path provided)
	if authFailedReasonFile != "" && reason != "" {
		if err := os.WriteFile(authFailedReasonFile, []byte(reason), 0600); err != nil {
			// Log but don't fail - auth_control_file is more critical
			slog.Warn("failed to write auth_failed_reason_file",
				"path", authFailedReasonFile,
				"error", err,
			)
		} else {
			slog.Debug("wrote auth_failed_reason_file",
				"path", authFailedReasonFile,
				"reason", reason,
			)
		}
	}

	// 2. Write failure to auth_control_file
	if err := os.WriteFile(authControlFile, []byte("0"), 0600); err != nil {
		return fmt.Errorf("failed to write auth_control_file (failure): %w", err)
	}

	slog.Debug("wrote auth_control_file (failure)", "path", authControlFile)
	return nil
}
