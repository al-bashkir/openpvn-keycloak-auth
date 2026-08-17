// Package session provides session management for OpenVPN SSO authentication flows.
package session

import (
	"time"
)

// Session represents an active authentication session.
// It stores all the information needed to track a user's authentication flow
// from the initial auth request through OIDC authentication to completion.
type Session struct {
	// ID is a unique identifier for this session (64-char hex string)
	ID string

	// State is the OIDC state parameter for CSRF protection
	State string

	// CodeVerifier is the PKCE code verifier (stored to verify the code later)
	CodeVerifier string

	// Nonce is the OIDC nonce sent on the authorization request; the id_token
	// returned by the provider must echo it back.
	Nonce string

	// Username is the username from the OpenVPN auth request
	Username string

	// UntrustedIP is the client's IP address (from untrusted_ip env var)
	UntrustedIP string

	// AuthControlFile is the path to OpenVPN's auth_control_file
	// Write "1" for success or "0" for failure
	AuthControlFile string

	// AuthFailedReasonFile is the path to OpenVPN's auth_failed_reason_file
	// Write error message before writing "0" to auth_control_file
	AuthFailedReasonFile string

	// AuthURL is the OIDC authorization URL the short /auth/<state> link
	// redirects to.
	AuthURL string

	// ExpiresAt is when this session will expire
	ExpiresAt time.Time

	// ResultWritten indicates whether a result has been written to auth_control_file
	// This is used to ensure we don't write multiple results for the same session
	// and to identify expired sessions that need failure results written
	ResultWritten bool

	// ResultWriteClaimed indicates that a callback is actively writing a final
	// result. It prevents concurrent callbacks from racing to write twice.
	ResultWriteClaimed bool
}
