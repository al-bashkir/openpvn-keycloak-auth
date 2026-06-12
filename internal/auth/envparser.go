// Package auth implements OpenVPN auth-script mode and environment parsing.
package auth

import (
	"fmt"
	"os"
	"strings"
)

// OpenVPNEnv contains environment variables set by OpenVPN when calling the auth script.
// The OpenVPN password is intentionally never captured here: SSO ignores it and it
// must not reach IPC messages or logs.
type OpenVPNEnv struct {
	// Username may be empty when using via-file; the via-file value takes precedence.
	Username string

	// Client information
	CommonName    string
	UntrustedIP   string
	UntrustedPort string

	// Client SSO capabilities reported via IV_SSO peer info.
	// Comma-separated list, e.g. "webauth,crtext" or "openurl".
	SSOMethods []string

	// OpenVPN control files
	AuthControlFile      string
	AuthPendingFile      string
	AuthFailedReasonFile string
}

// ParseEnv reads and validates OpenVPN environment variables
func ParseEnv() (*OpenVPNEnv, error) {
	env := &OpenVPNEnv{
		Username:             os.Getenv("username"),
		CommonName:           os.Getenv("common_name"),
		UntrustedIP:          os.Getenv("untrusted_ip"),
		UntrustedPort:        os.Getenv("untrusted_port"),
		AuthControlFile:      os.Getenv("auth_control_file"),
		AuthPendingFile:      os.Getenv("auth_pending_file"),
		AuthFailedReasonFile: os.Getenv("auth_failed_reason_file"),
	}

	// Parse IV_SSO client capabilities (e.g. "webauth,crtext" or "openurl").
	// OpenVPN exports peer info IV_* variables to the auth script environment.
	if ivSSO := os.Getenv("IV_SSO"); ivSSO != "" {
		for _, m := range strings.Split(ivSSO, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				env.SSOMethods = append(env.SSOMethods, m)
			}
		}
	}

	// Validate required fields
	if env.AuthControlFile == "" {
		return nil, fmt.Errorf("auth_control_file environment variable not set")
	}

	if env.AuthPendingFile == "" {
		return nil, fmt.Errorf("auth_pending_file environment variable not set")
	}

	if env.AuthFailedReasonFile == "" {
		return nil, fmt.Errorf("auth_failed_reason_file environment variable not set")
	}

	return env, nil
}
