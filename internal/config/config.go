// Package config provides loading, validation, and logging setup from config.
package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration
type Config struct {
	Listen ListenConfig `yaml:"listen"`
	OIDC   OIDCConfig   `yaml:"oidc"`
	Auth   AuthConfig   `yaml:"auth"`
	TLS    TLSConfig    `yaml:"tls"`
	Log    LogConfig    `yaml:"log"`
}

// ListenConfig defines where the daemon listens for requests
type ListenConfig struct {
	HTTP   string `yaml:"http"`   // HTTP server address (e.g., ":9000")
	Socket string `yaml:"socket"` // Unix socket path
}

// OIDCConfig defines OIDC/OAuth2 settings for Keycloak
type OIDCConfig struct {
	Issuer        string   `yaml:"issuer"`                 // Keycloak issuer URL
	ClientID      string   `yaml:"client_id"`              // OIDC client ID
	ClientSecret  string   `yaml:"client_secret" json:"-"` // OIDC client secret (empty for public clients)
	RedirectURI   string   `yaml:"redirect_uri"`           // Callback URL
	Scopes        []string `yaml:"scopes"`                 // OIDC scopes
	RequiredRoles []string `yaml:"required_roles"`         // Required roles for VPN access
	RoleClaim     string   `yaml:"role_claim"`             // JSON path to roles in token
}

// AuthConfig defines authentication behavior
type AuthConfig struct {
	SessionTimeout        int    `yaml:"session_timeout"`         // Session timeout in seconds
	UsernameClaim         string `yaml:"username_claim"`          // Claim to use as username
	AllowUsernameMismatch bool   `yaml:"allow_username_mismatch"` // Allow any authenticated user
}

// TLSConfig defines TLS settings for the HTTP server
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// LogConfig defines logging settings
type LogConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	// Canonicalize the path (resolves ".." components and redundant separators).
	// The path originates from a trusted source (CLI flag / systemd unit), so
	// G304 (file inclusion via variable) is not a concern here.
	cleanPath := filepath.Clean(path)

	// Read file
	data, err := os.ReadFile(cleanPath) // #nosec G304 -- path from trusted CLI argument
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	cfg := DefaultConfig()
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Listen: ListenConfig{
			HTTP:   ":9000",
			Socket: "/run/openvpn-keycloak-auth/auth.sock",
		},
		OIDC: OIDCConfig{
			Scopes:    []string{"openid", "profile", "email"},
			RoleClaim: "realm_access.roles",
		},
		Auth: AuthConfig{
			SessionTimeout:        300, // 5 minutes
			UsernameClaim:         "preferred_username",
			AllowUsernameMismatch: false,
		},
		TLS: TLSConfig{
			Enabled: false,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// applyEnvOverrides applies environment variable overrides
func (c *Config) applyEnvOverrides() {
	// OIDC overrides
	if v := os.Getenv("OVPN_SSO_OIDC_ISSUER"); v != "" {
		c.OIDC.Issuer = v
	}
	if v := os.Getenv("OVPN_SSO_OIDC_CLIENT_ID"); v != "" {
		c.OIDC.ClientID = v
	}
	if v := os.Getenv("OVPN_SSO_OIDC_CLIENT_SECRET"); v != "" {
		c.OIDC.ClientSecret = v
	}
	if v := os.Getenv("OVPN_SSO_OIDC_REDIRECT_URI"); v != "" {
		c.OIDC.RedirectURI = v
	}

	// Log overrides
	if v := os.Getenv("OVPN_SSO_LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
	if v := os.Getenv("OVPN_SSO_LOG_FORMAT"); v != "" {
		c.Log.Format = v
	}

	// Listen overrides
	if v := os.Getenv("OVPN_SSO_LISTEN_HTTP"); v != "" {
		c.Listen.HTTP = v
	}
	if v := os.Getenv("OVPN_SSO_LISTEN_SOCKET"); v != "" {
		c.Listen.Socket = v
	}
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	// Validate OIDC config
	if c.OIDC.Issuer == "" {
		return fmt.Errorf("oidc.issuer is required")
	}
	if err := validateIssuerURL(c.OIDC.Issuer); err != nil {
		return err
	}

	if c.OIDC.ClientID == "" {
		return fmt.Errorf("oidc.client_id is required")
	}

	if c.OIDC.RedirectURI == "" {
		return fmt.Errorf("oidc.redirect_uri is required")
	}
	if err := validateRedirectURI(c.OIDC.RedirectURI); err != nil {
		return err
	}

	if len(c.OIDC.Scopes) == 0 {
		return fmt.Errorf("oidc.scopes must contain at least 'openid'")
	}
	hasOpenID := false
	for _, scope := range c.OIDC.Scopes {
		if scope == "openid" {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID {
		return fmt.Errorf("oidc.scopes must include 'openid'")
	}

	// Validate auth config
	if c.Auth.SessionTimeout <= 0 {
		return fmt.Errorf("auth.session_timeout must be positive")
	}
	if c.Auth.SessionTimeout > 3600 {
		return fmt.Errorf("auth.session_timeout should not exceed 3600 seconds (1 hour)")
	}

	if c.Auth.UsernameClaim == "" {
		return fmt.Errorf("auth.username_claim is required")
	}

	// Validate TLS config
	if c.TLS.Enabled {
		if c.TLS.CertFile == "" || c.TLS.KeyFile == "" {
			return fmt.Errorf("tls.cert_file and tls.key_file are required when TLS is enabled")
		}

		// Check if files exist
		if _, err := os.Stat(c.TLS.CertFile); err != nil {
			return fmt.Errorf("tls.cert_file not found: %w", err)
		}
		if _, err := os.Stat(c.TLS.KeyFile); err != nil {
			return fmt.Errorf("tls.key_file not found: %w", err)
		}
	}

	// Validate log config
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.Log.Level] {
		return fmt.Errorf("log.level must be one of: debug, info, warn, error")
	}

	validFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validFormats[c.Log.Format] {
		return fmt.Errorf("log.format must be one of: json, text")
	}

	// Validate listen config
	if c.Listen.HTTP == "" {
		return fmt.Errorf("listen.http is required")
	}
	if c.Listen.Socket == "" {
		return fmt.Errorf("listen.socket is required")
	}
	if !filepath.IsAbs(c.Listen.Socket) {
		return fmt.Errorf("listen.socket must be an absolute path")
	}

	return nil
}

func validateHTTPURL(name, rawURL string) (*url.URL, error) {
	if strings.TrimSpace(rawURL) != rawURL {
		return nil, fmt.Errorf("%s must not contain leading or trailing whitespace", name)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%s must be a valid HTTP(S) URL", name)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%s must be a valid HTTP(S) URL", name)
	}

	if parsed.Fragment != "" {
		return nil, fmt.Errorf("%s must not contain a URL fragment", name)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("%s must not contain URL userinfo", name)
	}

	return parsed, nil
}

func validateIssuerURL(rawURL string) error {
	parsed, err := validateHTTPURL("oidc.issuer", rawURL)
	if err != nil {
		return err
	}
	if parsed.RawQuery != "" {
		return fmt.Errorf("oidc.issuer must not contain a query string")
	}
	return nil
}

func validateRedirectURI(rawURL string) error {
	parsed, err := validateHTTPURL("oidc.redirect_uri", rawURL)
	if err != nil {
		return err
	}

	if parsed.Path == "" || parsed.Path == "/" {
		return fmt.Errorf("oidc.redirect_uri must include a non-root callback path")
	}
	if path.Clean(parsed.Path) != parsed.Path {
		return fmt.Errorf("oidc.redirect_uri path must be canonical")
	}
	if isReservedCallbackPath(parsed.Path) {
		return fmt.Errorf("oidc.redirect_uri path is reserved")
	}

	return nil
}

func isReservedCallbackPath(callbackPath string) bool {
	return callbackPath == "/health"
}

// SetupLogging configures the global slog logger based on the LogConfig.
func SetupLogging(cfg *LogConfig) {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch cfg.Format {
	case "text":
		handler = slog.NewTextHandler(os.Stderr, opts)
	default:
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler))
}

// Redact returns a deep-enough copy of the config with secrets redacted for safe logging
func (c *Config) Redact() *Config {
	redacted := *c
	// Deep copy slices to avoid sharing underlying arrays with the original
	if c.OIDC.Scopes != nil {
		redacted.OIDC.Scopes = make([]string, len(c.OIDC.Scopes))
		copy(redacted.OIDC.Scopes, c.OIDC.Scopes)
	}
	if c.OIDC.RequiredRoles != nil {
		redacted.OIDC.RequiredRoles = make([]string, len(c.OIDC.RequiredRoles))
		copy(redacted.OIDC.RequiredRoles, c.OIDC.RequiredRoles)
	}
	if redacted.OIDC.ClientSecret != "" {
		redacted.OIDC.ClientSecret = "[REDACTED]"
	}
	return &redacted
}
