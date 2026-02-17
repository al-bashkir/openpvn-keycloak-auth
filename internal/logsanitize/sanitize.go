// Package logsanitize provides helpers for sanitizing untrusted values before logging.
package logsanitize

import "strings"

// Sanitize removes control characters from log field values to reduce
// the risk of log injection (CWE-117).
//
// Stripped ranges:
//   - C0 controls 0x00-0x1F (except horizontal tab 0x09)
//   - DEL 0x7F and C1 controls 0x80-0x9F
func Sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' {
			return '_'
		}
		if r >= 0x7f && r <= 0x9f {
			return '_'
		}
		return r
	}, s)
}
