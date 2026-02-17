package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/auth"
	"github.com/al-bashkir/openvpn-keycloak-auth/internal/config"
	"github.com/al-bashkir/openvpn-keycloak-auth/internal/daemon"
	"github.com/spf13/cobra"
)

// Version information (set via ldflags at build time)
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// Global flags
var (
	configFile string
	logLevel   string
	logFormat  string
)

// Exit codes
const (
	ExitSuccess  = 0
	ExitError    = 1
	ExitDeferred = 2 // Special: auth deferred (only for auth command)
	ExitConfig   = 3
)

var rootCmd = &cobra.Command{
	Use:   "openvpn-keycloak-auth",
	Short: "OpenVPN Keycloak SSO Authentication",
	Long: `Script-based SSO authentication for OpenVPN 2.6+ using Keycloak OIDC.

This binary operates in two modes:
  - serve: Run as daemon to handle OIDC authentication flows
  - auth:  Run as OpenVPN auth script (called via --auth-user-pass-verify)

OpenVPN 2.6+ supports deferred authentication from scripts, eliminating
the need for C plugins. This solution is pure Go with no CGO dependencies.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the SSO authentication daemon",
	Long: `Start the daemon that handles OIDC authentication flows.

The daemon:
  - Listens on a Unix socket for auth requests from the auth script
  - Runs an HTTP server for OIDC callbacks
  - Manages authentication sessions
  - Validates tokens and writes results to OpenVPN control files

This mode is typically run as a systemd service.`,
	RunE: runServe,
}

// overrideExitCode is set by subcommands (auth, check-config) so main() can
// call os.Exit() after cobra finishes.  This avoids calling os.Exit() inside
// RunE which would bypass deferred functions.  -1 means "use default".
var overrideExitCode = -1

var authCmd = &cobra.Command{
	Use:   "auth <credentials-file>",
	Short: "Auth script mode (called by OpenVPN)",
	Long: `OpenVPN auth script mode - handles single authentication request.

This command is called by OpenVPN via --auth-user-pass-verify with the
'via-file' option. The credentials file contains:
  Line 1: Username
  Line 2: Password (ignored for SSO)

The script:
  1. Reads OpenVPN environment variables
  2. Sends auth request to daemon via Unix socket
  3. Returns exit code 2 (deferred) on success
  4. Returns exit code 1 on error

Exit codes:
  0 = Authentication success (immediate, not used for SSO)
  1 = Authentication failure
  2 = Authentication deferred (daemon will complete it)`,
	Args: cobra.ExactArgs(1),
	RunE: runAuth,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  `Display version, commit hash, and build date.`,
	Run:   runVersion,
}

var checkConfigCmd = &cobra.Command{
	Use:   "check-config",
	Short: "Validate configuration file",
	Long: `Load and validate the configuration file without starting the daemon.

Checks for:
  - Valid YAML syntax
  - Required fields present
  - Valid URLs and paths
  - Logical consistency

Exit codes:
  0 = Configuration is valid
  3 = Configuration error`,
	RunE: runCheckConfig,
}

func init() {
	// Global flags (available to all commands)
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "/etc/openvpn/keycloak-sso.yaml",
		"Path to configuration file")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "",
		"Log level (debug, info, warn, error) - overrides config file")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "",
		"Log format (json, text) - overrides config file")

	// Add subcommands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(checkConfigCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitError)
	}

	// If a subcommand set a specific exit code, use it.
	// This is done outside RunE so deferred functions run properly.
	if overrideExitCode >= 0 {
		os.Exit(overrideExitCode)
	}
}

// runServe starts the daemon
func runServe(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override log settings from flags if provided
	if logLevel != "" {
		cfg.Log.Level = logLevel
	}
	if logFormat != "" {
		cfg.Log.Format = logFormat
	}

	// Initialize structured logging based on config
	config.SetupLogging(&cfg.Log)

	slog.Info("starting OpenVPN Keycloak SSO daemon",
		"version", version,
		"commit", commit,
		"build_date", buildDate,
		"config", configFile,
	)

	// Create and run daemon
	d, err := daemon.New(cfg)
	if err != nil {
		slog.Error("failed to create daemon", "error", err)
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	return d.Run()
}

// runAuth handles single auth request from OpenVPN
func runAuth(cmd *cobra.Command, args []string) error {
	credentialsFile := args[0]

	// Load config to get socket path
	// If config file doesn't exist, use default socket path
	socketPath := "/run/openvpn-keycloak-auth/auth.sock"

	cfg, err := config.Load(configFile)
	if err == nil {
		socketPath = cfg.Listen.Socket
	}
	// If config load fails, we still try with the default socket path

	// Create auth handler
	handler := auth.NewHandler(socketPath)

	// Run auth -- exit code is applied in main() after cobra finishes
	overrideExitCode = handler.Run(context.Background(), credentialsFile)
	return nil
}

// runVersion displays version information
func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("openvpn-keycloak-auth version %s\n", version)
	fmt.Printf("  Commit:     %s\n", commit)
	fmt.Printf("  Build date: %s\n", buildDate)
	fmt.Printf("  Go version: %s\n", getGoVersion())
}

// runCheckConfig validates the configuration
func runCheckConfig(cmd *cobra.Command, args []string) error {
	fmt.Printf("Checking configuration: %s\n\n", configFile)

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Configuration validation failed:\n")
		fmt.Fprintf(os.Stderr, "   %v\n", err)
		overrideExitCode = ExitConfig
		return nil // exit code handled via overrideExitCode
	}

	// Print configuration summary (with secrets redacted)
	fmt.Println("✅ Configuration is valid")
	fmt.Println()
	fmt.Println("Configuration summary:")
	fmt.Printf("  OIDC Issuer:     %s\n", cfg.OIDC.Issuer)
	fmt.Printf("  Client ID:       %s\n", cfg.OIDC.ClientID)
	fmt.Printf("  Redirect URI:    %s\n", cfg.OIDC.RedirectURI)
	fmt.Printf("  Scopes:          %v\n", cfg.OIDC.Scopes)
	fmt.Printf("  Required Roles:  %v\n", cfg.OIDC.RequiredRoles)
	fmt.Printf("  HTTP Listen:     %s\n", cfg.Listen.HTTP)
	fmt.Printf("  Unix Socket:     %s\n", cfg.Listen.Socket)
	fmt.Printf("  Session Timeout: %d seconds\n", cfg.Auth.SessionTimeout)
	fmt.Printf("  Log Level:       %s\n", cfg.Log.Level)
	fmt.Printf("  Log Format:      %s\n", cfg.Log.Format)
	fmt.Printf("  TLS Enabled:     %v\n", cfg.TLS.Enabled)

	if cfg.OIDC.ClientSecret != "" {
		fmt.Println("\n  Client Secret:   [SET]")
	} else {
		fmt.Println("\n  Client Secret:   [NOT SET] (using public client with PKCE)")
	}

	fmt.Println("\n✅ Ready to start daemon")

	return nil
}

// getGoVersion returns the Go version used to build the binary
func getGoVersion() string {
	return runtime.Version()
}
