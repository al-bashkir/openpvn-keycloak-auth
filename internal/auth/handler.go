package auth

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/ipc"
)

// Exit codes for the auth script
const (
	ExitSuccess  = 0 // Auth success (immediate, not used for SSO)
	ExitFailure  = 1 // Auth failure
	ExitDeferred = 2 // Auth deferred (SSO flow initiated)
)

// Handler handles authentication requests from OpenVPN
type Handler struct {
	socketPath string
}

// NewHandler creates a new auth handler
func NewHandler(socketPath string) *Handler {
	return &Handler{
		socketPath: socketPath,
	}
}

// Run executes the auth script logic
// It reads OpenVPN environment, parses credentials, sends request to daemon,
// and returns the appropriate exit code
func (h *Handler) Run(ctx context.Context, credentialsFile string) int {
	// Parse OpenVPN environment variables
	env, err := ParseEnv()
	if err != nil {
		slog.Error("failed to parse OpenVPN environment", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return ExitFailure
	}

	// Read credentials from via-file
	username, password, err := readCredentialsFile(credentialsFile)
	if err != nil {
		slog.Error("failed to read credentials file", "error", err, "file", credentialsFile)
		fmt.Fprintf(os.Stderr, "Error reading credentials: %v\n", err)
		return ExitFailure
	}

	// Override env username/password if present in file
	if username != "" {
		env.Username = username
	}
	if password != "" {
		env.Password = password
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
			"username", env.Username,
			"iv_sso", env.SSOMethods,
		)
		fmt.Fprintf(os.Stderr, "Error: client does not support webauth or openurl (IV_SSO=%v)\n", env.SSOMethods)
		return ExitFailure
	}

	slog.Info("auth request",
		"username", env.Username,
		"ip", env.UntrustedIP,
		"port", env.UntrustedPort,
		"common_name", env.CommonName,
		"pending_method", pendingMethod,
	)

	// Create IPC client
	client := ipc.NewClient(h.socketPath)

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
	resp, err := client.SendAuthRequest(ctx, req)
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
			"username", env.Username,
		)
		slog.Debug("auth URL generated", "url", resp.AuthURL)

		// Auth is deferred - daemon will handle the SSO flow
		// and write to auth_control_file when complete
		return ExitDeferred
	}

	// Unknown status
	slog.Error("unknown response status", "status", resp.Status)
	fmt.Fprintf(os.Stderr, "Error: unexpected response from daemon\n")
	return ExitFailure
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

	// Split into lines
	lines := strings.Split(string(data), "\n")

	// Need at least 1 line (username); password line may be empty for SSO
	if len(lines) < 1 {
		return "", "", fmt.Errorf("invalid credentials file format: file is empty")
	}

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
