package logsanitize

import (
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain ASCII", "hello world", "hello world"},
		{"keeps tab", "a\tb", "a\tb"},
		{"strips newline", "a\nb", "a_b"},
		{"strips carriage return", "a\rb", "a_b"},
		{"strips NUL", "a\x00b", "a_b"},
		{"strips DEL", "a\x7fb", "a_b"},
		{"strips C1 control (UTF-8)", "a\u009fb", "a_b"},
		{"keeps NBSP just past C1 range", "a b", "a b"},
		{"preserves multi-byte UTF-8", "héllo é 中文", "héllo é 中文"},
		{"replaces invalid UTF-8 byte with U+FFFD (kept)", "a\xffb", "a�b"},
		{"strips ANSI escape sequence intro", "\x1b[31mred\x1b[0m", "_[31mred_[0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sanitize(tt.in); got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeLargeInput(t *testing.T) {
	in := strings.Repeat("a\nb", 10_000)
	out := Sanitize(in)
	if len(out) != len(in) {
		t.Fatalf("length changed: got %d, want %d", len(out), len(in))
	}
	if strings.ContainsRune(out, '\n') {
		t.Error("expected no newlines after sanitize")
	}
}
