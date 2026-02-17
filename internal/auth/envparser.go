// Package auth implements OpenVPN auth-script mode and environment parsing.
package auth

import (
	"fmt"
	"os"
	"strings"
)

// OpenVPNEnv contains environment variables set by OpenVPN when calling the auth script
type OpenVPNEnv struct {
	// User credentials (may be empty when using via-file)
	Username string
	// Password is intentionally excluded from all IPC messages and logs.
	// The json tag prevents accidental serialization should this struct ever
	// be marshalled in the future.
	Password string `json:"-"`

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

	// Script metadata
	ScriptType string

	// Additional useful fields
	Config               string
	IfconfigPoolRemoteIP string
	TimeASCII            string
	TimeUnix             string
}

// ParseEnv reads and validates OpenVPN environment variables
func ParseEnv() (*OpenVPNEnv, error) {
	env := &OpenVPNEnv{
		Username:             os.Getenv("username"),
		Password:             os.Getenv("password"),
		CommonName:           os.Getenv("common_name"),
		UntrustedIP:          os.Getenv("untrusted_ip"),
		UntrustedPort:        os.Getenv("untrusted_port"),
		AuthControlFile:      os.Getenv("auth_control_file"),
		AuthPendingFile:      os.Getenv("auth_pending_file"),
		AuthFailedReasonFile: os.Getenv("auth_failed_reason_file"),
		ScriptType:           os.Getenv("script_type"),
		Config:               os.Getenv("config"),
		IfconfigPoolRemoteIP: os.Getenv("ifconfig_pool_remote_ip"),
		TimeASCII:            os.Getenv("time_ascii"),
		TimeUnix:             os.Getenv("time_unix"),
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
