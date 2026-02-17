package ipc

import "github.com/al-bashkir/openvpn-keycloak-auth/internal/logsanitize"

// sanitizeIPCValue strips control characters from a string before it is
// written to structured log output, providing defense against log injection
// (CWE-117). Values logged by the IPC server (username, IP, CommonName)
// originate from external sources (VPN client, client certificates).
func sanitizeIPCValue(s string) string {
	return logsanitize.Sanitize(s)
}
