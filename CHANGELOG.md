# Changelog

All notable changes to OpenVPN Keycloak SSO will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-02-15

### Initial Release

The first production-ready release of OpenVPN Keycloak SSO - a complete Single Sign-On authentication solution for OpenVPN Community Server 2.6+ using Keycloak as the Identity Provider.

### Features

#### Core Authentication
- **Script-Based Deferred Authentication** - Leverages OpenVPN 2.6's native deferred auth capabilities (exit code 2)
- **Authorization Code Flow with PKCE** - Implements OAuth 2.0 / OIDC Authorization Code Flow with SHA-256 PKCE for maximum security
- **Browser-Based Login** - Automatic browser opening via `WEB_AUTH::` URLs for seamless user experience
- **Token Validation** - Complete JWT validation including signature verification via JWKS, issuer, audience, expiration, and custom claims
- **Role-Based Access Control** - Optional enforcement of Keycloak roles/groups for fine-grained authorization
- **Session Management** - In-memory session storage with automatic cleanup and configurable TTLs

#### Architecture
- **Single Go Binary** - No C code, no CGO, no shared libraries - pure Go implementation
- **Dual Operating Modes**:
  - `auth` mode: Called by OpenVPN as auth script, communicates with daemon
  - `serve` mode: Runs as systemd service, handles OIDC flow and token validation
- **Unix Socket IPC** - Fast, secure JSON-based communication between auth script and daemon
- **HTTP Callback Server** - Handles OIDC redirect callbacks with comprehensive middleware stack
- **Atomic File Operations** - Reliable OpenVPN control file writes with proper error handling

#### Security
- **PKCE Implementation** - 32-byte crypto/rand verifier with S256 challenge method
- **CSRF Protection** - 16-byte state parameter validation
- **Cryptographically Secure Random** - All session IDs and security parameters use crypto/rand
- **Rate Limiting** - 10 requests/second per IP address on HTTP endpoints
- **Security Headers** - X-Frame-Options, X-Content-Type-Options, CSP, HSTS, X-XSS-Protection, Referrer-Policy
- **No Secrets in Logs** - Verified exclusion of tokens, passwords, and sensitive data from all log output
- **Systemd Hardening** - 20+ security directives including sandboxing, capability restrictions, filesystem isolation
- **Input Validation** - Comprehensive validation of all user inputs and configuration values

#### Client Support
- **OpenVPN Community** - Full support for official OpenVPN clients
- **OpenVPN Connect** - Optimized profiles for mobile and desktop Connect clients
- **Tunnelblick** - macOS-specific configuration and compatibility
- **NetworkManager** - Linux desktop integration via NetworkManager plugin
- **Command-Line Clients** - Support for headless environments with manual URL flow
- **Windows, macOS, Linux, iOS, Android** - Cross-platform client support

#### Configuration
- **YAML Configuration** - Human-readable, well-documented configuration format
- **Validation** - Built-in config validation with `check-config` command
- **Examples** - Production-ready configuration templates for all components
- **Client Profiles** - 5 pre-configured client profiles for different platforms and use cases

#### Deployment
- **Automated Installation** - Interactive install script with comprehensive validation
- **Systemd Integration** - Production-ready systemd service unit with security hardening
- **Clean Uninstallation** - Interactive uninstall script with backup and rollback capabilities
- **Makefile** - Comprehensive build system with 25+ targets for development and deployment
- **No RPM Required** - Manual installation via Makefile for maximum flexibility

#### Documentation
- **200+ Pages** - Comprehensive documentation across 15+ files
- **Quick Start Guide** - 5-minute deployment guide (QUICKSTART.md)
- **Architecture Documentation** - 35KB technical deep dive (docs/architecture.md)
- **Keycloak Setup** - Complete Keycloak 25.0.6 configuration guide (16KB)
- **OpenVPN Server Setup** - Detailed server configuration guide (15KB)
- **Client Setup** - Platform-specific client guides for 6 operating systems (38KB)
- **Security Guide** - Comprehensive security documentation (34KB)
- **Security Checklist** - 100-point security assessment checklist (16KB)
- **Testing Guide** - 10 documented manual test cases (49KB)
- **Troubleshooting** - Keycloak-specific troubleshooting guide (16KB)
- **Deployment Guide** - Complete production deployment instructions (21KB)
- **Contributing Guide** - Developer contribution guidelines (10KB+)

#### Testing
- **56 Unit Tests** - Comprehensive test coverage across all packages
- **76% Code Coverage** - Average coverage across the codebase
- **Race Detector Clean** - All tests pass with `-race` flag
- **Table-Driven Tests** - Systematic testing of multiple scenarios
- **Mock Dependencies** - Isolated testing of components
- **Integration Test Support** - Docker Compose setup for end-to-end testing

#### CI/CD
- **GitHub Actions Pipeline** - 5 automated jobs:
  - Unit tests on Go 1.22 and 1.23
  - golangci-lint static analysis
  - Build verification for linux/amd64
  - Security scanning with govulncheck
  - Integration testing setup
- **Automated Releases** - Tagged releases trigger automated builds
- **Multi-Go Version Testing** - Ensures compatibility across Go versions

#### Developer Experience
- **CLI Commands**:
  - `serve` - Run daemon server
  - `auth` - Auth script mode (called by OpenVPN)
  - `version` - Display version information
  - `check-config` - Validate configuration file
- **Comprehensive Logging** - Structured logging with slog (debug, info, warn, error levels)
- **Development Tooling** - Makefile targets for build, test, lint, coverage, clean
- **Code Organization** - Clean internal package structure with clear separation of concerns

### Technical Implementation

#### Task Breakdown (17 Tasks Completed)

**Tasks 001-010: Core Implementation**
- ✅ Task 001: Project setup and scaffolding
- ✅ Task 002: Configuration management with YAML
- ✅ Task 003: CLI framework with Cobra
- ✅ Task 004: Unix socket IPC implementation
- ✅ Task 005: Auth script mode with environment parsing
- ✅ Task 006: HTTP server with callback handler
- ✅ Task 007: OIDC provider integration
- ✅ Task 008: Token validation with JWT verification
- ✅ Task 009: Session management with TTL cleanup
- ✅ Task 010: Daemon orchestration

**Tasks 011-016: Documentation & Deployment**
- ✅ Task 011: Keycloak setup documentation (16KB)
- ✅ Task 012: OpenVPN server configuration (15KB + 5 client profiles)
- ✅ Task 013: Client setup guides (38KB, 6 platforms)
- ✅ Task 014: Deployment automation (install/uninstall scripts, systemd, Makefile)
- ✅ Task 015: Security hardening (34KB guide, 16KB checklist, SECURITY.md)
- ✅ Task 016: Testing infrastructure (49KB guide, GitHub Actions, 56 tests)

**Task 017: Final Documentation**
- ✅ README.md - Complete project overview with architecture, features, roadmap
- ✅ QUICKSTART.md - 5-minute deployment guide
- ✅ docs/architecture.md - 35KB technical deep dive
- ✅ CONTRIBUTING.md - Contribution guidelines and coding standards
- ✅ CHANGELOG.md - This file

### Package Coverage

- `internal/auth` - 76.7% coverage
- `internal/config` - 75.7% coverage
- `internal/daemon` - Core orchestration
- `internal/httpserver` - 64.2% coverage
- `internal/ipc` - 74.7% coverage
- `internal/oidc` - 67.0% coverage
- `internal/openvpn` - 85.2% coverage
- `internal/session` - 90.7% coverage

### Dependencies

**Direct Dependencies:**
- `github.com/coreos/go-oidc/v3` v3.11.0 - OIDC client library
- `github.com/spf13/cobra` v1.8.1 - CLI framework
- `golang.org/x/oauth2` v0.23.0 - OAuth 2.0 client
- `golang.org/x/time` v0.8.0 - Rate limiting
- `gopkg.in/yaml.v3` v3.0.1 - YAML parsing

**Standard Library:**
- `crypto/rand`, `crypto/sha256` - Cryptographic operations
- `encoding/json`, `encoding/hex` - Data encoding
- `log/slog` - Structured logging
- `net/http` - HTTP server
- `context` - Cancellation and timeouts
- `os`, `io`, `path/filepath` - File operations

### Platform Support

**Server:**
- Rocky Linux 9 (primary target)
- CentOS Stream 9
- RHEL 9
- Any Linux distribution with OpenVPN 2.6.19+ and systemd

**Clients:**
- Windows 10/11 (OpenVPN Community, OpenVPN Connect)
- macOS 12+ (Tunnelblick, OpenVPN Connect)
- Linux (NetworkManager, openvpn CLI)
- iOS 14+ (OpenVPN Connect)
- Android 8+ (OpenVPN Connect)

### Known Limitations

1. **OpenVPN Version** - Requires OpenVPN 2.6.2+ for reliable AUTH_PENDING support
2. **Client Capability** - Clients must support IV_SSO capability for browser opening
3. **Session Storage** - In-memory only (sessions lost on daemon restart)
4. **Keycloak Version** - Tested with Keycloak 25.0.6 specifically
5. **Token Refresh** - No automatic token refresh (users must reauthenticate)

### Migration Notes

This is the initial release. No migration required.

### Security Considerations

**IMPORTANT:** Before deploying to production:

1. Review and complete the [Security Checklist](docs/security-checklist.md)
2. Follow all recommendations in the [Security Guide](docs/security.md)
3. Ensure proper TLS configuration for all components
4. Review and harden systemd service configuration
5. Implement monitoring and alerting
6. Establish vulnerability scanning and update processes

See [SECURITY.md](SECURITY.md) for vulnerability reporting procedures.

### Upgrade Instructions

Not applicable for initial release.

### Roadmap

See [README.md](README.md#roadmap) for planned features in versions 1.1 and 2.0.

### Contributors

**Project Author:**
- Initial implementation and documentation
- All 17 tasks completed
- 200+ pages of documentation
- 56 unit tests with 76% coverage

**Special Thanks:**
- OpenVPN Community for the amazing 2.6 script auth features
- go-oidc maintainers for the excellent OIDC library
- Keycloak team for the powerful IdP platform

### References

- **OpenVPN 2.6 Documentation:** https://github.com/OpenVPN/openvpn/blob/release/2.6/doc/man-sections/script-options.rst
- **OIDC Specification:** https://openid.net/specs/openid-connect-core-1_0.html
- **PKCE RFC 7636:** https://datatracker.ietf.org/doc/html/rfc7636
- **Keycloak Documentation:** https://www.keycloak.org/documentation

---

## [Unreleased]

### Planned for v1.1

- Token caching to reduce Keycloak load
- Multi-tenancy support (multiple Keycloak realms)
- Prometheus metrics for monitoring
- Session persistence (Redis/file-based)
- Admin API for session management
- Enhanced logging with structured fields

### Planned for v2.0

- Management interface support (alternative to script mode)
- MFA enforcement options
- Rate limiting per user
- Session sharing across multiple OpenVPN servers
- IPv6 support validation

---

**Note:** This changelog follows [Keep a Changelog](https://keepachangelog.com/) principles. For detailed development history, see [WORKLOG.md](WORKLOG.md).

**Last Updated:** 2026-02-15  
**Version:** 1.0.0
