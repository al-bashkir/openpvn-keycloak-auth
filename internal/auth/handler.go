package auth

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/ipc"
	"github.com/al-bashkir/openvpn-keycloak-auth/internal/logsanitize"
)

// Exit codes returned by Handler.Run.
// Immediate-success (exit 0) is unused because SSO is always deferred.
const (
	ExitFailure  = 1 // Auth failure
	ExitDeferred = 2 // Auth deferred (SSO flow initiated)
)

// Run executes the auth script logic: it reads the OpenVPN environment,
// parses credentials, sends the request to the daemon listening on socketPath,
// and returns the exit code OpenVPN expects.
func Run(ctx context.Context, socketPath, credentialsFile string) int {
	// Parse OpenVPN environment variables
	env, err := ParseEnv()
	if err != nil {
		slog.Error("failed to parse OpenVPN environment", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return ExitFailure
	}

	// Read credentials from via-file. The password is intentionally discarded:
	// SSO ignores it and it must never reach IPC messages or logs.
	username, _, err := readCredentialsFile(credentialsFile)
	if err != nil {
		slog.Error("failed to read credentials file", "error", err, "file", credentialsFile)
		fmt.Fprintf(os.Stderr, "Error reading credentials: %v\n", err)
		return ExitFailure
	}

	// Override env username if present in file
	if username != "" {
		env.Username = username
	}

	// Validate username
	if env.Username == "" {
		slog.Error("username is empty")
		fmt.Fprintf(os.Stderr, "Error: username is required\n")
		return ExitFailure
	}

	// Select the auth pending method from the client's SSO capabilities.
	// The method must match one of the values the client advertised in IV_SSO.
	pendingMethod := selectPendingMethod(env.SSOMethods)
	if pendingMethod == "" {
		slog.Error("client does not support any known SSO method",
			"username", logsanitize.Sanitize(env.Username),
			"iv_sso", sanitizeValues(env.SSOMethods),
		)
		fmt.Fprintf(os.Stderr, "Error: client does not support webauth or openurl (IV_SSO=%v)\n", env.SSOMethods)
		return ExitFailure
	}

	slog.Info("auth request",
		"username", logsanitize.Sanitize(env.Username),
		"ip", logsanitize.Sanitize(env.UntrustedIP),
		"port", logsanitize.Sanitize(env.UntrustedPort),
		"common_name", logsanitize.Sanitize(env.CommonName),
		"pending_method", pendingMethod,
	)

	// Build auth request (password intentionally excluded from IPC)
	req := &ipc.AuthRequest{
		Username:             env.Username,
		CommonName:           env.CommonName,
		UntrustedIP:          env.UntrustedIP,
		UntrustedPort:        env.UntrustedPort,
		AuthControlFile:      env.AuthControlFile,
		AuthPendingFile:      env.AuthPendingFile,
		AuthFailedReasonFile: env.AuthFailedReasonFile,
		PendingAuthMethod:    pendingMethod,
	}

	// Send request to daemon
	resp, err := ipc.SendAuthRequest(ctx, socketPath, req)
	if err != nil {
		slog.Error("failed to communicate with daemon", "error", err)
		fmt.Fprintf(os.Stderr, "Error: daemon communication failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "Is the daemon running? Check: systemctl status openvpn-keycloak-auth\n")
		return ExitFailure
	}

	// Handle response
	if resp.Status == ipc.StatusError {
		slog.Error("daemon returned error", "error", resp.Error)
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		return ExitFailure
	}

	if resp.Status == ipc.StatusDeferred {
		slog.Info("auth deferred",
			"session_id", resp.SessionID,
			"username", logsanitize.Sanitize(env.Username),
		)

		// Auth is deferred - daemon will handle the SSO flow
		// and write to auth_control_file when complete
		return ExitDeferred
	}

	// Unknown status
	slog.Error("unknown response status", "status", resp.Status)
	fmt.Fprintf(os.Stderr, "Error: unexpected response from daemon\n")
	return ExitFailure
}

func sanitizeValues(values []string) []string {
	sanitized := make([]string, 0, len(values))
	for _, value := range values {
		sanitized = append(sanitized, logsanitize.Sanitize(value))
	}
	return sanitized
}

// readCredentialsFile reads username and password from OpenVPN's via-file
// The file contains exactly two lines:
//
//	Line 1: username
//	Line 2: password (may be empty or "sso" for SSO flows)
func readCredentialsFile(path string) (username, password string, err error) {
	// Canonicalize the path (resolves ".." components and redundant separators).
	// The path originates from OpenVPN's via-file $1 argument, a trusted source,
	// so G304 (file inclusion via variable) is not a concern here.
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath) // #nosec G304 -- path from trusted OpenVPN via-file argument
	if err != nil {
		return "", "", fmt.Errorf("failed to read file: %w", err)
	}

	// Split into lines. strings.Split always returns at least one element,
	// so an empty file yields a single empty line that fails the username
	// check below.
	lines := strings.Split(string(data), "\n")

	username = strings.TrimSpace(lines[0])
	if len(lines) >= 2 {
		password = strings.TrimSpace(lines[1])
	}

	// Username is always required
	if username == "" {
		return "", "", fmt.Errorf("username is empty in credentials file")
	}

	// Password may be empty for SSO flows
	return username, password, nil
}

// selectPendingMethod picks the best auth pending method from the client's
// IV_SSO capabilities. Returns "" if the client supports none of the known
// methods. Preference order: webauth > openurl.
func selectPendingMethod(methods []string) string {
	has := make(map[string]bool, len(methods))
	for _, m := range methods {
		has[m] = true
	}

	// Prefer webauth (supported by Tunnelblick, OpenVPN Connect, etc.)
	if has["webauth"] {
		return "webauth"
	}
	// Fall back to openurl (some newer/other clients)
	if has["openurl"] {
		return "openurl"
	}
	return ""
}
