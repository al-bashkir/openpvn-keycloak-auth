# OpenVPN Keycloak SSO Authentication - Agent Instructions

## Project Overview

This project implements SSO authentication for OpenVPN Community Server 2.6.19 using Keycloak as the Identity Provider. The solution is written in Go and leverages OpenVPN 2.6's script-based deferred authentication capabilities to eliminate the need for C plugins entirely.

**Target Platform:** Rocky Linux 9
**OpenVPN Version:** Community Server 2.6.19
**IdP:** Keycloak (OpenID Connect / OAuth 2.0)
**Language:** Go 1.22+
**Architecture:** Single Go binary with dual modes (auth script + daemon)

## Why This Approach?

OpenVPN 2.6 introduced critical features that make pure script-based SSO possible:

1. **Deferred auth from scripts** - Scripts can return exit code `2` to defer authentication
2. **`auth_pending_file` support** - Scripts can trigger browser opening via `WEB_AUTH::` URLs
3. **`auth_failed_reason_file`** - Custom error messages for users
4. **Bug fixes** - AUTH_PENDING/INFO_PRE messages work reliably from 2.6.2+

**Result:** No C code, no C compiler, no CGO, no shared libraries. Just Go.

## Architecture Summary

### Components

1. **Go Binary: `openvpn-keycloak-sso`**
   - **Mode 1 (`auth`)**: Called by OpenVPN via `--auth-user-pass-verify`, sends auth request to daemon via Unix socket, returns exit code 2 (deferred)
   - **Mode 2 (`serve`)**: Runs as systemd service, handles OIDC flow, writes auth results to OpenVPN control files
   - **Mode 3 (`version`)**: Version information
   - **Mode 4 (`check-config`)**: Configuration validation

2. **Shell Wrapper: `/etc/openvpn/auth-keycloak.sh`**
   - Thin wrapper that OpenVPN calls directly
   - Simply execs the Go binary in `auth` mode

### Authentication Flow

```
User → Client → OpenVPN Server → Auth Script → Daemon → Keycloak
                                      ↓            ↓
                                 Exit code 2   OIDC Flow
                                      ↓            ↓
                                 auth_pending  Callback
                                      ↓            ↓
Client opens browser ←────────────────┘            ↓
      ↓                                            ↓
User authenticates in Keycloak ───────────────────→┘
      ↓
Daemon validates token, writes auth_control_file=1
      ↓
VPN connected
```

### Key Technical Decisions

1. **Script-based auth (NOT C plugin, NOT management interface)**
   - Simpler deployment: one binary, one config, one systemd unit
   - No CGO, easier cross-compilation
   - Better testability and debugging

2. **Single Go binary with subcommands**
   - `openvpn-keycloak-sso serve` - daemon
   - `openvpn-keycloak-sso auth` - auth script
   - Shared code between modes

3. **Authorization Code Flow with PKCE**
   - Standard browser redirect flow
   - Best security and UX
   - Compatible with all modern OpenVPN clients

4. **Unix socket IPC**
   - Fast, secure communication between script and daemon
   - No network exposure for internal communication
   - JSON protocol for simplicity

5. **Token validation on server**
   - Verify JWT signature via JWKS
   - Validate claims: iss, aud, exp, iat, nbf
   - Optional group/role enforcement

## Project Structure

```
openvpn-keycloak-sso/
├── AGENTS.md                    # This file
├── WORKLOG.md                   # Work log
├── README.md                    # User-facing documentation
├── LICENSE                      # MIT License
├── go.mod, go.sum               # Go dependencies
│
├── cmd/openvpn-keycloak-sso/
│   └── main.go                  # Entry point
│
├── internal/
│   ├── auth/                    # Auth script logic
│   ├── daemon/                  # Daemon server
│   ├── config/                  # Configuration
│   ├── httpserver/              # HTTP server (OIDC callback)
│   ├── ipc/                     # Unix socket IPC
│   ├── oidc/                    # OIDC flow implementation
│   ├── session/                 # Session management
│   └── openvpn/                 # OpenVPN file writing
│
├── config/                      # Configuration templates
├── scripts/                     # Shell wrapper
├── deploy/                      # systemd, install scripts
├── docs/                        # Documentation
├── web/templates/               # HTML pages
└── tasks/                       # Task breakdown
```

## Build and Run

### Development Build

```bash
go build -o openvpn-keycloak-sso ./cmd/openvpn-keycloak-sso
```

### Production Build

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o openvpn-keycloak-sso ./cmd/openvpn-keycloak-sso
```

### Run Daemon

```bash
./openvpn-keycloak-sso serve --config /etc/openvpn/keycloak-sso.yaml
```

### Test Auth Script Mode

```bash
# Simulate OpenVPN environment
export username="testuser"
export auth_control_file="/tmp/test_acf"
export auth_pending_file="/tmp/test_apf"
export auth_failed_reason_file="/tmp/test_arf"
export untrusted_ip="192.0.2.1"
export untrusted_port="12345"

echo -e "testuser\nsso" > /tmp/test_creds

./openvpn-keycloak-sso auth /tmp/test_creds
echo "Exit code: $?"
cat /tmp/test_apf
```

## Environment Setup

### Prerequisites

- Go 1.22 or later
- OpenVPN 2.6.19 (from EPEL 9 on Rocky Linux)
- Keycloak instance (any recent version supporting OIDC)
- Rocky Linux 9 or compatible

### Development Dependencies

```bash
# Install Go (if not already installed)
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install OpenVPN (for testing)
sudo dnf install epel-release
sudo dnf install openvpn

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Go Dependencies

```bash
go get github.com/coreos/go-oidc/v3/oidc
go get golang.org/x/oauth2
go get gopkg.in/yaml.v3
go get github.com/spf13/cobra  # or use stdlib flags
```

## Important Conventions and Patterns

### File Writing (Critical!)

All writes to OpenVPN control files MUST be atomic and reliable:

```go
// Use os.WriteFile with explicit permissions
err := os.WriteFile(path, data, 0600)
if err != nil {
    // Handle error - this is critical path
}
```

### Error Handling Pattern

Always write to `auth_control_file` in ALL code paths, including errors:

```go
func handleAuth(session *Session) {
    defer func() {
        // Ensure we ALWAYS write a result
        if session.Result == "" {
            writeAuthResult(session.AuthControlFile, false, "Internal error")
        }
    }()
    
    // ... auth logic
}
```

### Session ID Generation

Always use `crypto/rand`, never `math/rand`:

```go
import "crypto/rand"

func generateSessionID() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}
```

### Logging

Use structured logging with `log/slog`:

```go
import "log/slog"

slog.Info("auth request received",
    "username", username,
    "ip", untrustedIP,
    "session_id", sessionID,
)

// NEVER log tokens or passwords
slog.Debug("token received") // ← Good
slog.Debug("token", "token", tokenString) // ← NEVER DO THIS
```

### Context Usage

Use context.Context throughout for cancellation and timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

token, err := oauth2Config.Exchange(ctx, code, oauth2.VerifyCodeVerifier(verifier))
```

## OpenVPN 2.6 Script Auth Specifics

### Environment Variables (via-file mode)

When OpenVPN calls the auth script with `via-file`, it:
1. Creates a temporary file with username (line 1) and password (line 2)
2. Passes the file path as `$1` argument
3. Sets environment variables: `auth_control_file`, `auth_pending_file`, `auth_failed_reason_file`, `untrusted_ip`, etc.
4. Deletes the temp file after script exits

### Exit Codes

- `0` - Auth success (immediate)
- `1` - Auth failure (immediate)
- `2` - Auth deferred (pending)

### auth_pending_file Format

MUST be exactly 3 lines:

```
<timeout_in_seconds>
openurl
WEB_AUTH::<url>
```

Example:
```
300
openurl
WEB_AUTH::https://keycloak.example.com/realms/myrealm/protocol/openid-connect/auth?client_id=openvpn&...
```

Note: Use `WEB_AUTH::` (with underscore), not `WEBAUTH::`

### auth_control_file Format

Single character: `1` (success) or `0` (failure)

### auth_failed_reason_file Format

Plain text error message. Write this BEFORE writing `0` to auth_control_file.

## Current Project State

**Status:** Initial setup in progress

**Completed:**
- Go module initialized
- Directory structure created
- AGENTS.md created

**In Progress:**
- Task 001: Project setup

**Next Steps:**
- Complete task file creation
- Create WORKLOG.md
- Create LICENSE
- Create .gitignore

## Important Links

- **OpenVPN 2.6 script options:** https://github.com/OpenVPN/openvpn/blob/release/2.6/doc/man-sections/script-options.rst
- **OpenVPN 2.6 Changes:** https://github.com/OpenVPN/openvpn/blob/release/2.6/Changes.rst
- **OIDC spec:** https://openid.net/specs/openid-connect-core-1_0.html
- **PKCE RFC:** https://datatracker.ietf.org/doc/html/rfc7636
- **go-oidc library:** https://github.com/coreos/go-oidc
- **Reference project (architecture only):** https://github.com/jkroepke/openvpn-auth-oauth2

## Testing Strategy

### Unit Tests

- All packages in `internal/` should have `_test.go` files
- Table-driven tests for token validation logic
- Mock Unix socket for IPC tests
- Mock HTTP server for OIDC flow tests

### Integration Tests

- End-to-end flow with test Keycloak instance
- Docker Compose setup for local testing
- Simulated OpenVPN environment

### Manual Testing

- Test with actual OpenVPN server
- Test with multiple client types (CLI, Tunnelblick, NetworkManager)
- Test error paths (timeout, invalid token, network failures)

## Security Checklist

- [ ] All secrets in config file with 0600 permissions
- [ ] No tokens or passwords in logs
- [ ] PKCE implemented correctly
- [ ] JWT signature validation via JWKS
- [ ] All claims validated (iss, aud, exp, iat, nbf)
- [ ] CSRF protection via state parameter
- [ ] Rate limiting on auth endpoints
- [ ] Session IDs from crypto/rand
- [ ] Unix socket with 0660 permissions, group openvpn
- [ ] auth_control_file written in all code paths
- [ ] Graceful shutdown on SIGTERM/SIGINT
- [ ] No race conditions (verified with -race flag)

## Troubleshooting

### Common Issues

1. **Client hangs during auth**
   - Check that daemon is running
   - Verify Unix socket exists and is accessible
   - Ensure auth_control_file is written in all code paths

2. **Browser doesn't open**
   - Check client supports WEB_AUTH:: (IV_SSO capability)
   - Verify auth_pending_file format (must be exactly 3 lines)
   - Use `WEB_AUTH::` not `WEBAUTH::`

3. **Token validation fails**
   - Check Keycloak issuer URL matches exactly
   - Verify client_id matches
   - Check system time (JWT exp validation requires accurate time)

4. **Permission denied on Unix socket**
   - Socket should be 0660, group openvpn
   - OpenVPN runs as user openvpn (typically)
   - Daemon should create socket with correct permissions

## Development Workflow

1. Pick next task from tasks/ directory
2. Update task status to IN_PROGRESS
3. Implement the task
4. Write tests
5. Run linter: `golangci-lint run`
6. Run tests: `go test -race ./...`
7. Update task status to DONE
8. Update WORKLOG.md
9. Commit changes
10. Move to next task

## Contact and Support

For questions or issues during development, refer to:
- OpenVPN mailing list
- Keycloak documentation
- go-oidc GitHub issues

## License

MIT License - see LICENSE file
