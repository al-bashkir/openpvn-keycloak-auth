package config

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Listen.HTTP != ":9000" {
		t.Errorf("expected HTTP listen :9000, got %s", cfg.Listen.HTTP)
	}

	if cfg.Auth.SessionTimeout != 300 {
		t.Errorf("expected session timeout 300, got %d", cfg.Auth.SessionTimeout)
	}

	if cfg.Log.Level != "info" {
		t.Errorf("expected log level info, got %s", cfg.Log.Level)
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			configYAML: `
listen:
  http: ":9000"
  socket: "/tmp/test.sock"
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000/callback"
  scopes:
    - openid
    - profile
auth:
  session_timeout: 300
  username_claim: "preferred_username"
log:
  level: "info"
  format: "json"
`,
			wantErr: false,
		},
		{
			name: "missing issuer",
			configYAML: `
listen:
  http: ":9000"
oidc:
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000/callback"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "issuer is required",
		},
		{
			name: "missing client_id",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  redirect_uri: "http://localhost:9000/callback"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing redirect_uri",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "redirect_uri is required",
		},
		{
			name: "scopes missing openid",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000/callback"
  scopes:
    - profile
    - email
`,
			wantErr:     true,
			errContains: "must include 'openid'",
		},
		{
			name: "invalid issuer URL",
			configYAML: `
oidc:
  issuer: "https://"
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000/callback"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "oidc.issuer must be a valid HTTP(S) URL",
		},
		{
			name: "issuer URL with query string",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test?foo=bar"
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000/callback"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "oidc.issuer must not contain a query string",
		},
		{
			name: "issuer URL with userinfo",
			configYAML: `
oidc:
  issuer: "https://user:pass@keycloak.example.com/realms/test"
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000/callback"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "oidc.issuer must not contain URL userinfo",
		},
		{
			name: "invalid redirect URL",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  redirect_uri: "javascript:alert(1)"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "oidc.redirect_uri must be a valid HTTP(S) URL",
		},
		{
			name: "redirect URL without path",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "oidc.redirect_uri must include a non-root callback path",
		},
		{
			name: "redirect URL with root path",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000/"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "oidc.redirect_uri must include a non-root callback path",
		},
		{
			name: "redirect URL with trailing slash path",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000/vpn/callback/"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "oidc.redirect_uri path must be canonical",
		},
		{
			name: "redirect URL with duplicate slash path",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000/vpn//callback"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "oidc.redirect_uri path must be canonical",
		},
		{
			name: "redirect URL with userinfo",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  redirect_uri: "http://user:pass@localhost:9000/callback"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "oidc.redirect_uri must not contain URL userinfo",
		},
		{
			name: "redirect URL with reserved path",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000/health"
  scopes:
    - openid
`,
			wantErr:     true,
			errContains: "oidc.redirect_uri path is reserved",
		},
		{
			name: "invalid log level",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000/callback"
  scopes:
    - openid
log:
  level: "verbose"
`,
			wantErr:     true,
			errContains: "log.level must be one of",
		},
		{
			name: "invalid yaml",
			configYAML: `
this is not: valid: yaml:
  bad: [syntax
`,
			wantErr:     true,
			errContains: "failed to parse",
		},
		{
			name: "unknown yaml key",
			configYAML: `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  redirect_uri: "http://localhost:9000/callback"
  scopes:
    - openid
  required_role:
    - vpn-users
`,
			wantErr:     true,
			errContains: "field required_role not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp config file
			tmpfile, err := os.CreateTemp("", "config-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Remove(tmpfile.Name()) }()

			if _, err := tmpfile.Write([]byte(tt.configYAML)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Load config
			cfg, err := Load(tmpfile.Name())

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errContains)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if cfg == nil {
					t.Error("expected config, got nil")
				}
			}
		})
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	// Set environment variables
	t.Setenv("OVPN_SSO_OIDC_CLIENT_SECRET", "env-secret")
	t.Setenv("OVPN_SSO_LOG_LEVEL", "debug")

	configYAML := `
oidc:
  issuer: "https://keycloak.example.com/realms/test"
  client_id: "openvpn"
  client_secret: "yaml-secret"
  redirect_uri: "http://localhost:9000/callback"
  scopes:
    - openid
log:
  level: "info"
`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	if _, err := tmpfile.Write([]byte(configYAML)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if cfg.OIDC.ClientSecret != "env-secret" {
		t.Errorf("expected client_secret='env-secret', got '%s'", cfg.OIDC.ClientSecret)
	}

	if cfg.Log.Level != "debug" {
		t.Errorf("expected log level 'debug', got '%s'", cfg.Log.Level)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "session timeout too high",
			modify: func(c *Config) {
				c.Auth.SessionTimeout = 7200
			},
			wantErr: true,
			errMsg:  "should not exceed 3600",
		},
		{
			name: "session timeout zero",
			modify: func(c *Config) {
				c.Auth.SessionTimeout = 0
			},
			wantErr: true,
			errMsg:  "must be positive",
		},
		{
			name: "TLS enabled without cert",
			modify: func(c *Config) {
				c.TLS.Enabled = true
				c.TLS.CertFile = ""
			},
			wantErr: true,
			errMsg:  "are required when TLS is enabled",
		},
		{
			name: "relative socket path",
			modify: func(c *Config) {
				c.Listen.Socket = "auth.sock"
			},
			wantErr: true,
			errMsg:  "listen.socket must be an absolute path",
		},
		{
			name: "redirect query string allowed",
			modify: func(c *Config) {
				c.OIDC.RedirectURI = "http://localhost:9000/callback?tenant=vpn"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Listen: ListenConfig{
					HTTP:   ":9000",
					Socket: "/tmp/test.sock",
				},
				OIDC: OIDCConfig{
					Issuer:      "https://keycloak.example.com/realms/test",
					ClientID:    "openvpn",
					RedirectURI: "http://localhost:9000/callback",
					Scopes:      []string{"openid"},
				},
				Auth: AuthConfig{
					SessionTimeout: 300,
					UsernameClaim:  "preferred_username",
				},
				Log: LogConfig{
					Level:  "info",
					Format: "json",
				},
			}

			tt.modify(cfg)

			err := cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRedact(t *testing.T) {
	cfg := &Config{
		OIDC: OIDCConfig{
			ClientSecret: "super-secret",
		},
	}

	redacted := cfg.Redact()

	if redacted.OIDC.ClientSecret != "[REDACTED]" {
		t.Errorf("expected [REDACTED], got %s", redacted.OIDC.ClientSecret)
	}

	// Original should be unchanged
	if cfg.OIDC.ClientSecret != "super-secret" {
		t.Errorf("original was modified")
	}
}

func TestSetupLogging(t *testing.T) {
	old := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(old)
	})

	SetupLogging(&LogConfig{Level: "debug", Format: "json"})
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected debug logs to be enabled")
	}

	SetupLogging(&LogConfig{Level: "error", Format: "text"})
	if slog.Default().Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected info logs to be disabled at error level")
	}
	if !slog.Default().Enabled(context.Background(), slog.LevelError) {
		t.Error("expected error logs to be enabled")
	}
}
