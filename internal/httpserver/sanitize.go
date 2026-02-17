package httpserver

import "github.com/al-bashkir/openvpn-keycloak-auth/internal/logsanitize"

// sanitizeLog sanitizes a string for safe inclusion in structured log output
// before logging external HTTP input.
func sanitizeLog(s string) string {
	return logsanitize.Sanitize(s)
}
