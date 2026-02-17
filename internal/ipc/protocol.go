package ipc

// MessageType represents the type of IPC message
type MessageType string

const (
	// MessageTypeAuthRequest is sent from auth script to daemon
	MessageTypeAuthRequest MessageType = "auth_request"
	// MessageTypeAuthResponse is sent from daemon to auth script
	MessageTypeAuthResponse MessageType = "auth_response"
)

// AuthRequest is sent from the auth script to the daemon when OpenVPN
// initiates an authentication request.
// Note: Password is intentionally excluded from IPC to avoid transmitting
// secrets unnecessarily. For SSO, the password field is not used.
type AuthRequest struct {
	Type                 MessageType `json:"type"`
	Username             string      `json:"username"`
	CommonName           string      `json:"common_name"`
	UntrustedIP          string      `json:"untrusted_ip"`
	UntrustedPort        string      `json:"untrusted_port"`
	AuthControlFile      string      `json:"auth_control_file"`
	AuthPendingFile      string      `json:"auth_pending_file"`
	AuthFailedReasonFile string      `json:"auth_failed_reason_file"`
	// PendingAuthMethod is the auth pending method the client supports
	// (e.g. "webauth" or "openurl"), selected from the client's IV_SSO capabilities.
	PendingAuthMethod string `json:"pending_auth_method"`
}

// AuthResponse is sent from the daemon back to the auth script
type AuthResponse struct {
	Type      MessageType `json:"type"`
	Status    string      `json:"status"` // "deferred" or "error"
	SessionID string      `json:"session_id,omitempty"`
	AuthURL   string      `json:"auth_url,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// ResponseStatus constants
const (
	StatusDeferred = "deferred"
	StatusError    = "error"
)
