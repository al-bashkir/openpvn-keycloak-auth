# OpenVPN Keycloak SSO Authentication

**Single Sign-On authentication for OpenVPN Community Server 2.6+ using Keycloak as the Identity Provider.**

[![Go Version](https://img.shields.io/github/go-mod/go-version/al-bashkir/openpvn-keycloak-auth)](https://go.dev/)
[![License](https://img.shields.io/github/license/al-bashkir/openpvn-keycloak-auth)](LICENSE)
[![Tests](https://github.com/al-bashkir/openvpn-keycloak-auth/workflows/Test%20and%20Build/badge.svg)](https://github.com/al-bashkir/openvpn-keycloak-auth/actions)

## Why This Project?

Traditional VPN authentication requires managing passwords, LDAP integration, or complex PAM configurations. This project brings **modern SSO authentication** to OpenVPN using:

- ✅ **Your existing identity provider** (Keycloak, any OIDC provider)
- ✅ **Multi-factor authentication** (TOTP, WebAuthn, SMS)
- ✅ **Centralized access control** (roles, groups, policies)
- ✅ **No password exposure** to VPN server
- ✅ **Audit trail** via Keycloak event logging

## Features

### 🔒 Security First

- **PKCE (RFC 7636)** - Proof Key for Code Exchange prevents authorization code interception
- **CSRF Protection** - State parameter validation
- **JWT Validation** - Signature verification via JWKS, claim validation
- **No Password Transmission** - VPN server never sees user passwords
- **Role-Based Access** - Enforce Keycloak roles/groups
- **Rate Limiting** - Per-IP request throttling
- **Security Headers** - CSP, X-Frame-Options, HSTS, etc.
- **systemd Hardening** - 20+ security directives (NoNewPrivileges, ProtectSystem, etc.)

### 🚀 Simple Deployment

- **No C Code** - Pure Go implementation using OpenVPN 2.6 script-based deferred auth
- **Single Binary** - One executable, no dependencies
- **Single Config File** - YAML configuration with validation
- **One systemd Service** - Install and forget
- **No CGO** - Easy cross-compilation, portable
- **Static Binary** - No runtime dependencies

### 🌐 Great User Experience

- **Browser-based Auth** - Familiar login experience
- **Automatic Browser Opening** - On compatible clients (OpenVPN Connect, Tunnelblick)
- **Multi-Platform Support** - Windows, macOS, Linux, iOS, Android
- **Session Management** - Short-lived sessions with TTL cleanup
- **Structured Logging** - JSON logs with slog

### 📊 Production Ready

- **76% Test Coverage** - 56 unit tests across all packages
- **Race Detector Clean** - No data races
- **CI/CD Pipeline** - Automated testing, linting, security scanning
- **Comprehensive Docs** - 200+ pages of documentation
- **Security Audited** - Gosec, Trivy scans
- **Performance Tested** - Supports concurrent authentication

## How It Works

OpenVPN 2.6 introduced **script-based deferred authentication** (exit code 2) and `auth_pending_file` support, making SSO possible without C plugins!

```
┌────────┐         ┌─────────┐         ┌─────────┐         ┌──────────┐
│ User   │────────▶│ OpenVPN │────────▶│ Daemon  │────────▶│ Keycloak │
│ Client │         │ Server  │         │         │         │          │
└────────┘         └─────────┘         └─────────┘         └──────────┘
    │                   │                   │                    │
    │ 1. Connect        │                   │                    │
    │──────────────────▶│                   │                    │
    │                   │ 2. Call auth script                    │
    │                   │──────────────────▶│                    │
    │                   │                   │ 3. Generate PKCE   │
    │                   │ 4. Exit code 2    │                    │
    │                   │◀──────────────────│                    │
    │ 5. Open browser   │                   │                    │
    │◀──────────────────│                   │                    │
    │                   │                   │                    │
    │ 6. Redirect to Keycloak (with PKCE challenge)              │
    │───────────────────────────────────────────────────────────▶│
    │ 7. User login + MFA                                        │
    │◀──────────────────────────────────────────────────────────▶│
    │                   │                   │                    │
    │ 8. Callback (with authorization code) │                    │
    │──────────────────────────────────────▶│                    │
    │                   │                   │ 9. Exchange code   │
    │                   │                   │───────────────────▶│
    │                   │                   │ 10. Validate token │
    │                   │ 11. Write success │                    │
    │                   │◀──────────────────│                    │
    │ 12. VPN connected │                   │                    │
    │◀──────────────────│                   │                    │
```

## Quick Start

**Prerequisites:** Rocky Linux 9, OpenVPN 2.6.19+, Keycloak instance

```bash
# 1. Build and install
git clone https://github.com/al-bashkir/openvpn-keycloak-auth
cd openvpn-keycloak-auth
make build
sudo make install

# 2. Configure Keycloak (see docs/keycloak-setup.md)
# - Create realm "vpn"
# - Create public client "openvpn" with PKCE (S256)
# - Create test user

# 3. Edit configuration
sudo vi /etc/openvpn/keycloak-sso.yaml
# Set issuer_url, client_id, callback_url

# 4. Start daemon
sudo systemctl enable --now openvpn-keycloak-auth

# 5. Configure OpenVPN server
sudo cp config/openvpn-server.conf.example /etc/openvpn/server/server.conf
# Add: script-security 3, auth-user-pass-verify, etc.

# 6. Start OpenVPN
sudo systemctl enable --now openvpn-server@server

# 7. Connect from client
openvpn --config client.ovpn
# Username: your-keycloak-username
# Password: sso (any value)
# Browser opens → Log in to Keycloak → VPN connects!
```

**See [QUICKSTART.md](QUICKSTART.md) for a complete 5-minute guide.**

## Installation

### From Source

**Requirements:**
- Go 1.22+
- OpenVPN 2.6.19+ (from EPEL on Rocky Linux 9)

```bash
# Clone repository
git clone https://github.com/al-bashkir/openvpn-keycloak-auth
cd openvpn-keycloak-auth

# Build binary
make build

# Install (requires root)
sudo make install
```

### Pre-built Binaries

Download from [Releases](https://github.com/al-bashkir/openvpn-keycloak-auth/releases):

```bash
# Download latest release
wget https://github.com/al-bashkir/openvpn-keycloak-auth/releases/download/v1.0.0/openvpn-keycloak-auth-linux-amd64

# Install
sudo install -m 755 openvpn-keycloak-auth-linux-amd64 /usr/local/bin/openvpn-keycloak-auth

# Run installation script
sudo ./deploy/install.sh
```

## Configuration

### Minimal Configuration

`/etc/openvpn/keycloak-sso.yaml`:

```yaml
keycloak:
  issuer_url: "https://keycloak.example.com/realms/myrealm"
  client_id: "openvpn"

http:
  listen_addr: "0.0.0.0:9000"
  callback_url: "https://vpn.example.com:9000/callback"

socket:
  path: "/run/openvpn-keycloak-auth/auth.sock"

session:
  ttl: "5m"

auth:
  username_claim: "preferred_username"
```

### Full Configuration

See [`config/openvpn-keycloak-auth.yaml.example`](config/openvpn-keycloak-auth.yaml.example) for all options including:

- Role enforcement (`required_roles`)
- Custom claims (`username_claim`, `role_claim`)
- TLS configuration
- Rate limiting
- Logging levels

## Documentation

### Getting Started

- **[QUICKSTART.md](QUICKSTART.md)** - 5-minute deployment guide
- **[docs/deployment.md](docs/deployment.md)** - Complete deployment guide for Rocky Linux 9
- **[docs/keycloak-setup.md](docs/keycloak-setup.md)** - Configure Keycloak realm and client (25.0.6)
- **[docs/openvpn-server-setup.md](docs/openvpn-server-setup.md)** - OpenVPN server configuration
- **[docs/client-setup.md](docs/client-setup.md)** - Client setup for all platforms

### Technical Documentation

- **[docs/architecture.md](docs/architecture.md)** - System architecture and design
- **[docs/security.md](docs/security.md)** - Security model, threat analysis, best practices
- **[docs/testing.md](docs/testing.md)** - Testing guide, manual test cases
- **[AGENTS.md](AGENTS.md)** - Developer guide, project conventions

### Troubleshooting

- **[docs/keycloak-troubleshooting.md](docs/keycloak-troubleshooting.md)** - Keycloak-specific issues
- **[docs/security-checklist.md](docs/security-checklist.md)** - Security assessment checklist

## Supported Clients

| Client | Platform | SSO Support | Notes |
|--------|----------|-------------|-------|
| **OpenVPN Connect 3.x** | Windows, macOS, iOS, Android, Linux | ✅ Excellent | Built-in webview, best experience |
| **Tunnelblick 3.8.7+** | macOS | ✅ Excellent | Opens Safari automatically |
| **OpenVPN CLI 2.6+** | Linux, Unix, macOS | ⚠️ Manual | Displays URL to copy/paste |
| **NetworkManager** | Linux (GNOME, KDE) | ⚠️ Limited | May require manual browser opening |
| **OpenVPN GUI 2.6+** | Windows | ⚠️ Manual | Displays URL to copy/paste |

**Recommendation:** Use **OpenVPN Connect 3.x** for the best experience on all platforms.

## Development

### Building

```bash
# Development build (fast)
make build-dev

# Production build (optimized, static)
make build

# Multi-arch build
GOARCH=arm64 make build
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Run specific test
make test-one TEST=TestCreateSession

# Run tests with race detector
go test -race ./...
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Run vet
make vet

# All checks
make check
```

## Project Structure

```
openvpn-keycloak-auth/
├── cmd/openvpn-keycloak-auth/    # Main entry point
├── internal/                     # Internal packages
│   ├── auth/                    # Auth script mode
│   ├── config/                  # Configuration loading
│   ├── daemon/                  # Daemon orchestration
│   ├── httpserver/              # HTTP server & callback
│   ├── ipc/                     # Unix socket IPC
│   ├── oidc/                    # OIDC flow & validation
│   ├── openvpn/                 # OpenVPN file writing
│   └── session/                 # Session management
├── config/                      # Configuration templates
│   ├── openvpn-keycloak-auth.yaml.example
│   ├── openvpn-server.conf.example
│   └── client*.ovpn.example
├── scripts/                     # Shell scripts
│   ├── auth-keycloak.sh         # Auth wrapper
│   └── generate-client-profile.sh
├── deploy/                      # Deployment files
│   ├── openvpn-keycloak-auth.service  # systemd unit
│   ├── install.sh               # Installation script
│   └── uninstall.sh             # Uninstallation script
├── docs/                        # Documentation
├── .github/workflows/           # CI/CD
├── Makefile                     # Build system
├── go.mod, go.sum               # Go modules
├── README.md                    # This file
├── QUICKSTART.md                # 5-minute guide
├── CONTRIBUTING.md              # Contribution guidelines
├── CHANGELOG.md                 # Version history
├── LICENSE                      # Elastic License 2.0 (ELv2)
└── SECURITY.md                  # Security policy
```

## Architecture

**Components:**

1. **openvpn-keycloak-auth binary** - Single Go binary with 4 modes:
   - `serve` - Daemon mode (runs as systemd service)
   - `auth` - Auth script mode (called by OpenVPN)
   - `version` - Version information
   - `check-config` - Configuration validation

2. **Unix Socket IPC** - Communication between auth script and daemon
3. **HTTP Server** - OIDC callback endpoint
4. **Session Manager** - In-memory session storage with TTL cleanup
5. **OIDC Provider** - Integration with Keycloak

**See [docs/architecture.md](docs/architecture.md) for detailed architecture.**

## Security

This project implements multiple layers of security:

- **Authentication Flow Security:**
  - PKCE (Proof Key for Code Exchange) - RFC 7636
  - CSRF protection via state parameter
  - Short-lived sessions (5-minute TTL)
  - No password transmission to VPN server

- **Token Security:**
  - JWT signature verification via JWKS
  - Complete claim validation (iss, aud, exp, iat, nbf)
  - Username and role enforcement
  - No tokens logged

- **System Security:**
  - systemd sandboxing (20+ security directives)
  - File permissions (config 0600, socket 0660)
  - SELinux support
  - Rate limiting (10 req/s per IP)

- **Network Security:**
  - TLS for all external communication
  - Security headers (CSP, X-Frame-Options, etc.)
  - Firewall configuration

**See [docs/security.md](docs/security.md) for complete security documentation.**

**Report vulnerabilities:** See [SECURITY.md](SECURITY.md)

## Performance

**Tested with:**
- 50+ concurrent authentication requests
- No performance degradation
- Memory usage: <20MB under load
- Session cleanup: Every 60 seconds

**Scalability:**
- Current: Single daemon, in-memory sessions
- Future: Multi-instance with shared session store (Redis, etc.)

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Code of conduct
- How to submit pull requests
- Coding standards
- Testing requirements
- Documentation guidelines

## Roadmap

**v1.0 (Current):**
- ✅ OIDC Authorization Code Flow with PKCE
- ✅ JWT validation with role enforcement
- ✅ OpenVPN 2.6 script-based auth
- ✅ Comprehensive documentation
- ✅ Security hardening
- ✅ CI/CD pipeline

**v1.1 (Planned):**
- [ ] Prometheus metrics endpoint
- [ ] Grafana dashboard
- [ ] Shared session store (Redis)
- [ ] Docker/Podman container image
- [ ] Helm chart for Kubernetes

**v2.0 (Future):**
- [ ] Support for other OIDC providers (Azure AD, Okta, etc.)
- [ ] WebAuthn/FIDO2 support
- [ ] Advanced policy engine
- [ ] Web UI for administration

See [issues](https://github.com/al-bashkir/openvpn-keycloak-auth/issues) for details.

## Alternatives

**Why not use [openvpn-auth-oauth2](https://github.com/jkroepke/openvpn-auth-oauth2)?**

This project is inspired by openvpn-auth-oauth2 but redesigned for OpenVPN 2.6's script-based authentication:

- **No C plugin** - Uses OpenVPN 2.6 script-based deferred auth
- **Simpler deployment** - No openvpn-plugin-auth-pam dependency
- **Keycloak-specific** - Optimized for Keycloak (not generic OAuth2)
- **Role enforcement** - Built-in Keycloak role/group checking
- **Rocky Linux 9** - Tested and documented for RHEL 9 family

Both projects are excellent - choose based on your needs!

## FAQ

**Q: Does this work with OpenVPN Access Server?**  
A: No, this is for OpenVPN Community Server 2.6+. Access Server has its own authentication plugins.

**Q: Can I use this with Azure AD / Okta / Google?**  
A: Currently optimized for Keycloak. Other OIDC providers may work but are untested. Support planned for v2.0.

**Q: Does this support client certificates (mutual TLS)?**  
A: Yes! You can use client certificates AND SSO together. Just don't set `auth-user-pass-optional` in OpenVPN config.

**Q: What happens if Keycloak is down?**  
A: New authentications will fail. Existing VPN sessions continue working (they don't re-auth).

**Q: Can I run multiple daemon instances?**  
A: Not currently recommended (in-memory sessions). v1.1 will support shared session store for multi-instance deployments.

**Q: Is this production-ready?**  
A: Yes! The project has 76% test coverage, security hardening, and comprehensive documentation. However, it's recommended to test thoroughly in your environment first.

## License

This project is licensed under the [Elastic License 2.0 (ELv2)](LICENSE).

Free for personal use and internal business use. Redistribution, SaaS hosting,
and bundling in commercial products require a separate commercial license —
contact the author.

## Credits

- **Inspired by:** [openvpn-auth-oauth2](https://github.com/jkroepke/openvpn-auth-oauth2)
- **Built with:** [Go](https://go.dev/), [coreos/go-oidc](https://github.com/coreos/go-oidc), [spf13/cobra](https://github.com/spf13/cobra)
- **Tested on:** Rocky Linux 9, OpenVPN 2.6.19, Keycloak 25.0.6

## Support

- **Documentation:** See [docs/](docs/) directory
- **Issues:** [GitHub Issues](https://github.com/al-bashkir/openvpn-keycloak-auth/issues)
- **Discussions:** [GitHub Discussions](https://github.com/al-bashkir/openvpn-keycloak-auth/discussions)
- **Security:** See [SECURITY.md](SECURITY.md)

## Star History

If you find this project useful, please consider giving it a star ⭐

---

**Made with ❤️ for the OpenVPN and Keycloak communities**
