package ipc

// AuthRequest is sent from the auth script to the daemon when OpenVPN
// initiates an authentication request.
// Note: Password is intentionally excluded from IPC to avoid transmitting
// secrets unnecessarily. For SSO, the password field is not used.
type AuthRequest struct {
	Username             string `json:"username"`
	CommonName           string `json:"common_name"`
	UntrustedIP          string `json:"untrusted_ip"`
	UntrustedPort        string `json:"untrusted_port"`
	AuthControlFile      string `json:"auth_control_file"`
	AuthPendingFile      string `json:"auth_pending_file"`
	AuthFailedReasonFile string `json:"auth_failed_reason_file"`
	// PendingAuthMethod is the auth pending method the client supports
	// (e.g. "webauth" or "openurl"), selected from the client's IV_SSO capabilities.
	PendingAuthMethod string `json:"pending_auth_method"`
}

// AuthResponse is sent from the daemon back to the auth script.
// The authorization URL is not returned: the daemon writes it to OpenVPN's
// auth_pending_file itself, and the auth script has no use for it.
type AuthResponse struct {
	Status    string `json:"status"` // StatusDeferred or StatusError
	SessionID string `json:"session_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ResponseStatus constants
const (
	StatusDeferred = "deferred"
	StatusError    = "error"
)
