# Architecture - OpenVPN Keycloak SSO

This document provides a technical deep dive into the architecture, design decisions, and implementation details of the OpenVPN Keycloak SSO authentication system.

## Table of Contents

1. [System Overview](#system-overview)
2. [Components](#components)
3. [Data Flow](#data-flow)
4. [IPC Protocol](#ipc-protocol)
5. [Session Management](#session-management)
6. [OIDC Implementation](#oidc-implementation)
7. [File Operations](#file-operations)
8. [Concurrency Model](#concurrency-model)
9. [Error Handling](#error-handling)
10. [Protocol Summary](#protocol-summary)
11. [Security Architecture Summary](#security-architecture-summary)
12. [Design Decisions](#design-decisions)

---

## System Overview

### High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        VPN Client                                │
│                   (OpenVPN Connect, CLI, etc.)                   │
└──────────────────────┬───────────────────────────────────────────┘
                       │
                       │ 1. TCP/UDP 1194
                       ▼
┌──────────────────────────────────────────────────────────────────┐
│                     OpenVPN Server                               │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  Script-based Deferred Authentication (2.6+)               │  │
│  │  - Calls auth script with via-file                         │  │
│  │  - Expects exit code 2 (deferred)                          │  │
│  │  - Reads auth_pending_file (WEB_AUTH:: URL)                │  │
│  │  - Sends AUTH_PENDING to client                            │  │
│  │  - Waits for auth_control_file (0 or 1)                    │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────┬───────────────────────────────────────────┘
                       │
                       │ 2. Calls /etc/openvpn/scripts/auth-keycloak.sh
                       ▼
┌──────────────────────────────────────────────────────────────────┐
│               Auth Script (Go binary, auth mode)                 │
│  - Parses OpenVPN environment variables                          │
│  - Reads credentials from via-file                               │
│  - Sends IPC request to daemon via Unix socket                   │
│  - Receives session ID and auth URL                              │
│  - Writes auth_pending_file                                      │
│  - Returns exit code 2                                           │
└──────────────────────┬───────────────────────────────────────────┘
                       │
                        │ 3. Unix socket: /run/openvpn-keycloak-auth/auth.sock
                       ▼
┌──────────────────────────────────────────────────────────────────┐
│            Daemon (Go binary, serve mode, systemd)               │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  Session Manager  │  IPC Server  │  HTTP Server            │  │
│  │  - In-memory map  │  - Unix sock │  - OIDC callback        │  │
│  │  - TTL cleanup    │  - JSON API  │  - Success/error pages  │  │
│  └────────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  OIDC Provider                                             │  │
│  │  - Authorization URL builder                               │  │
│  │  - PKCE generator (S256)                                   │  │
│  │  - Token exchanger                                         │  │
│  │  - JWT validator (signature + claims)                      │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────┬──────────┬────────────────────────────────┘
                       │          │
                       │          │ 4. HTTPS (OIDC flow)
                       │          ▼
                       │    ┌──────────────────┐
                       │    │    Keycloak      │
                       │    │  (OIDC Provider) │
                       │    └──────────────────┘
                       │
                       │ 5. Writes auth_control_file (0 or 1)
                       ▼
              ┌─────────────────┐
              │  OpenVPN temp   │
              │  control files  │
              └─────────────────┘
```

### Core Principle

**Script-based deferred authentication:** OpenVPN 2.6 allows auth scripts to:

1. Return exit code 2 (deferred)
2. Write a `auth_pending_file` with a URL
3. Later write `auth_control_file` (0=failure, 1=success)

This eliminates the need for C plugins entirely!

---

## Components

### 1. Go Binary (`openvpn-keycloak-auth`)

**Single executable with 4 operating modes:**

#### Mode 1: `serve` (Daemon)

Runs as systemd service, handles:

- Unix socket IPC server
- HTTP server for OIDC callbacks
- Session management with TTL cleanup
- OIDC provider integration
- Token validation

**Entry point:** `cmd/openvpn-keycloak-auth/main.go` → `serveCmd`

**Goroutines:**

- Main: HTTP server listener
- IPC server: Unix socket listener
- Session cleanup: Timer-based TTL expiration
- Per-request: HTTP handlers, IPC handlers

#### Mode 2: `auth` (Auth Script)

Called by OpenVPN for each authentication attempt:

- Parse environment variables
- Read credentials from via-file
- Send IPC request to daemon
- Write `auth_pending_file`
- Return exit code 2

**Entry point:** `cmd/openvpn-keycloak-auth/main.go` → `authCmd`

**Execution time:** <100ms (just IPC call)

#### Mode 3: `version`

Display version information.

#### Mode 4: `check-config`

Validate configuration file:

- YAML syntax
- Required fields
- Known YAML keys
- URL syntax and local TLS file paths
- Socket path and log settings

`check-config` does not contact Keycloak. OIDC discovery occurs when the daemon starts.

### 2. Internal Packages

**Package structure:**

```
internal/
├── auth/                    # Auth script mode
│   ├── envparser.go        # Parse OpenVPN env vars
│   └── handler.go          # Auth script orchestration
│
├── config/                  # Configuration
│   └── config.go           # YAML loading & validation
│
├── daemon/                  # Daemon orchestration
│   └── daemon.go           # Start all components
│
├── httpserver/              # HTTP server
│   ├── server.go           # Server setup
│   ├── callback.go         # OIDC callback handler
│   ├── health.go           # Health endpoint
│   ├── pages.go            # HTML rendering
│   └── middleware.go       # Logging, recovery, rate limiting, security headers
│
├── ipc/                     # Unix socket IPC
│   ├── protocol.go         # JSON message types
│   ├── client.go           # Client (auth script side)
│   └── server.go           # Server (daemon side)
│
├── oidc/                    # OIDC implementation
│   ├── provider.go         # Provider discovery
│   ├── flow.go             # Authorization Code Flow with PKCE
│   └── validator.go        # Token validation
│
├── openvpn/                 # OpenVPN file operations
│   └── authfile.go         # Write control files
│
└── session/                 # Session management
    ├── session.go          # Session struct
    ├── manager.go          # Thread-safe session manager
    └── cleanup.go          # TTL-based cleanup
```

### 3. Shell Wrapper

`/etc/openvpn/scripts/auth-keycloak.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
# Thin wrapper called by OpenVPN
# Validates wrapper inputs, adjusts process limits, and execs auth mode.
exec /usr/local/bin/openvpn-keycloak-auth --config /etc/openvpn/keycloak-sso.yaml auth "$@"
```

**Why a wrapper?**

- OpenVPN `--auth-user-pass-verify` expects a shell script
- Easier to update just the binary without touching OpenVPN config
- Can add environment setup if needed

---

## Data Flow

The two diagrams below are the shape of the flow; the phase-by-phase
walkthrough that follows names the code that implements each step.

### Authentication Initiation Flow

```
┌─────────┐
│ OpenVPN │ Calls: auth-keycloak.sh /tmp/creds_123 via-file
└────┬────┘
     │ ENV: username=john.doe, auth_control_file=/tmp/acf_123,
     │      auth_pending_file=/tmp/apf_123, untrusted_ip=...
     ▼
┌──────────────┐
│ Auth Script  │ 1. Parse ENV vars
│ (Go: auth)   │ 2. Read username from /tmp/creds_123
└────┬─────────┘ 3. Create IPC request
     │ {
     │   "username": "john.doe",
     │   "auth_control_file": "/tmp/acf_123",
     │   ...
     │ }
     ▼
┌──────────────┐
│ Unix Socket  │ /run/openvpn-keycloak-auth/auth.sock
└────┬─────────┘
     │ JSON over Unix socket
     ▼
┌──────────────┐
│   Daemon     │ 4. Receive IPC request
│ (Go: serve)  │ 5. Create session (ID, state, PKCE verifier)
└────┬─────────┘ 6. Generate authorization URL
     │ {
     │   "auth_url": "https://keycloak.../auth?...",
     │   "session_id": "abc123..."
     │ }
     ▼
┌──────────────┐
│ Auth Script  │ 7. Receive IPC response
│              │ 8. Write auth_pending_file:
└────┬─────────┘    300\nwebauth\nWEB_AUTH::https://...
     │
     │ 9. Return exit code 2
     ▼
┌──────────────┐
│   OpenVPN    │ 10. Read auth_pending_file
│              │ 11. Send AUTH_PENDING to client
└──────────────┘ 12. Client opens browser
```

### OIDC Callback Flow

```
┌─────────┐
│ Browser │ User logs in to Keycloak
└────┬────┘
     │ Keycloak redirects to:
     │ https://vpn.example.com:9000/callback?code=xyz&state=abc
     ▼
┌──────────────┐
│ HTTP Server  │ 1. Receive callback request
│ (Daemon)     │ 2. Extract code and state
└────┬─────────┘ 3. Look up session by state
     │
     ▼
┌──────────────┐
│ Session Mgr  │ 4. Retrieve session
│              │    session = sessions[state]
└────┬─────────┘    session.CodeVerifier = "..."
     │
     ▼
┌──────────────┐
│ OIDC Provider│ 5. Exchange code for token
│              │    POST /token with:
└────┬─────────┘    - code=xyz
     │              - code_verifier=...
     │              - client_id=openvpn
     ▼
┌──────────────┐
│  Keycloak    │ 6. Validate PKCE
│              │ 7. Return ID token + access token
└────┬─────────┘
     │ {
     │   "id_token": "eyJ...",
     │   "access_token": "...",
     │   "token_type": "Bearer"
     │ }
     ▼
┌──────────────┐
│  Validator   │ 8. Verify JWT signature (via JWKS)
│              │ 9. Validate claims:
└────┬─────────┘    - iss (issuer)
     │              - aud (audience = client_id)
     │              - exp (not expired)
     │              - iat, nbf (time checks)
     │          10. Validate username matches
     │          11. Validate roles (if required)
     ▼
┌──────────────┐
│ Auth Writer  │ 12. Write auth_control_file:
│              │     echo "1" > /tmp/acf_123
└────┬─────────┘ 13. Delete session
     │
     ▼
┌──────────────┐
│   OpenVPN    │ 14. Read auth_control_file
│              │ 15. Complete VPN connection
└──────────────┘
```

### Phase-by-Phase Walkthrough

### Phase 1: Connection Initiation

**User** starts VPN client, enters Keycloak username + any password (e.g., `"sso"`).

**OpenVPN server** (protocol: OpenVPN UDP/TCP on port 1194):

- Receives TLS handshake + auth credentials
- Creates 3 temporary files on disk:
  - `auth_control_file` -- will receive `"1"` (success) or `"0"` (failure)
  - `auth_pending_file` -- script writes pending auth info here
  - `auth_failed_reason_file` -- optional failure reason text
- Sets environment variables and calls the shell wrapper:

```bash
# OpenVPN calls:
/etc/openvpn/scripts/auth-keycloak.sh /tmp/openvpn_creds_XXXXX
```

**Env vars set by OpenVPN:**

```
username=jdoe
common_name=jdoe
untrusted_ip=192.0.2.1
untrusted_port=12345
IV_SSO=webauth,crtext
auth_control_file=/tmp/openvpn_acf_XXXXX
auth_pending_file=/tmp/openvpn_apf_XXXXX
auth_failed_reason_file=/tmp/openvpn_arf_XXXXX
```

**Credentials file** (2 lines):

```
jdoe
sso
```

---

### Phase 2: Auth Script Execution

`scripts/auth-keycloak.sh` fixes `RLIMIT_NPROC` (OpenVPN's systemd unit sets it too low for Go's runtime), then execs:

```bash
exec /usr/local/bin/openvpn-keycloak-auth --config /etc/openvpn/keycloak-sso.yaml auth "$1"
```

**`internal/auth/handler.go:Run()`**:

1. Reads env vars via `ParseEnv()` (`internal/auth/envparser.go`)
2. Reads username/password from credentials file (password is **discarded** -- never sent over IPC)
3. Selects SSO method from `IV_SSO`: prefers `"webauth"`, falls back to `"openurl"`
4. **Sends over Unix socket** -- protocol: `AF_UNIX SOCK_STREAM`, JSON encoding:

```json
{
  "username": "jdoe",
  "common_name": "jdoe",
  "untrusted_ip": "192.0.2.1",
  "untrusted_port": "12345",
  "auth_control_file": "/tmp/openvpn_acf_XXXXX",
  "auth_pending_file": "/tmp/openvpn_apf_XXXXX",
  "auth_failed_reason_file": "/tmp/openvpn_arf_XXXXX",
  "pending_auth_method": "webauth"
}
```

IPC client (`internal/ipc/client.go:SendAuthRequest`): connects with `net.DialTimeout("unix", socketPath, 5s)`, writes JSON, reads JSON response.

---

### Phase 3: Daemon Processes Auth Request

**IPC server** (`internal/ipc/server.go`) receives connection on Unix socket (mode `0660`, group `openvpn`), decodes JSON, sanitizes all string inputs (strips control chars via `internal/logsanitize/sanitize.go` to prevent CWE-117 log injection).

**`internal/daemon/daemon.go:handleAuthRequest()`**:

1. **Creates session** (`internal/session/manager.go`):
   - ID: 32 bytes from `crypto/rand` -> 64 hex chars
   - Stores username, IP, file paths, expiry (default 300s)

2. **Starts OIDC flow** (`internal/oidc/flow.go`):
   - Generates **PKCE code verifier**: 32 bytes `crypto/rand` -> base64url (43 chars)
   - Generates **code challenge**: `SHA256(verifier)` -> base64url (S256 method)
   - Generates **state** (CSRF token): 16 bytes `crypto/rand` -> 32 hex chars
   - Constructs full Keycloak auth URL:
   ```
   https://keycloak.example.com/realms/myrealm/protocol/openid-connect/auth
     ?client_id=openvpn
     &redirect_uri=https://vpn.example.com:9000/callback
     &response_type=code
     &scope=openid+profile+email
     &state=a1b2c3d4e5f6...
     &code_challenge=E9Melhoa2OwvFrEMTJguCH...
     &code_challenge_method=S256
   ```

3. **Builds short URL** -- the full Keycloak URL is too long for OpenVPN's 256-byte `OPTION_LINE_SIZE` limit:
   ```
   https://vpn.example.com:9000/auth/a1b2c3d4e5f6...
   ```
   Validates that `WEB_AUTH::<url>\n` fits within 256 chars.

4. **Writes `auth_pending_file`** (`internal/openvpn/authfile.go`) -- file I/O, mode `0600`, exactly 3 lines:
   ```
   300
   webauth
   WEB_AUTH::https://vpn.example.com:9000/auth/a1b2c3d4e5f6...
   ```

5. **Returns IPC response** -- JSON over Unix socket. The auth script never
   sees the authorization URL; the daemon has already written it to
   `auth_pending_file` itself:
   ```json
   {
     "status": "deferred",
     "session_id": "f8a3b1c2d4..."
   }
   ```

---

### Phase 4: Deferred Auth + Browser Opens

**Auth script** receives `"deferred"` response, **exits with code `2`** (tells OpenVPN: "auth is pending").

**OpenVPN server** reads `auth_pending_file`, sends to client via the OpenVPN control channel:

```
AUTH_PENDING,300,webauth,WEB_AUTH::https://vpn.example.com:9000/auth/a1b2c3d4e5f6...
```

**VPN client** (if it supports `IV_SSO=webauth`) opens the user's default browser to the short URL.

---

### Phase 5: Short URL Redirect

**Browser** -> `GET https://vpn.example.com:9000/auth/a1b2c3d4e5f6...` (HTTPS)

**HTTP server** (`internal/httpserver/callback.go`) -- `handleAuthRedirect`:

1. Extracts state from URL path
2. Looks up session via `sessionMgr.GetByState(state)`
3. **Returns HTTP 302 redirect** to the full Keycloak authorization URL:
   ```
   HTTP/1.1 302 Found
   Location: https://keycloak.example.com/realms/myrealm/protocol/openid-connect/auth?client_id=openvpn&...
   ```

Middleware applied to all requests (`internal/httpserver/middleware.go`):

- Rate limiting: per-IP token bucket, 10 req/s burst 50
- Security headers: `X-Frame-Options: DENY`, CSP, `X-Content-Type-Options: nosniff`, HSTS
- Request logging (all inputs sanitized)
- Panic recovery

---

### Phase 6: Keycloak Authentication

**Browser** follows redirect to Keycloak (HTTPS). **User** authenticates (password, MFA, etc.).

**Keycloak** redirects browser back:

```
HTTP/1.1 302 Found
Location: https://vpn.example.com:9000/callback?code=AUTHORIZATION_CODE&state=a1b2c3d4e5f6...
```

---

### Phase 7: Token Exchange + Validation

**Browser** -> `GET https://vpn.example.com:9000/callback?code=...&state=...` (HTTPS)

**HTTP server** (`internal/httpserver/callback.go`) -- `handleCallback`:

1. **CSRF check**: Looks up session by `state` parameter -> validates it matches a known session

2. **Token exchange** (`internal/oidc/flow.go`) -- the daemon calls Keycloak's token endpoint:
   ```
   POST https://keycloak.example.com/realms/myrealm/protocol/openid-connect/token
   Content-Type: application/x-www-form-urlencoded

   grant_type=authorization_code
   &code=AUTHORIZATION_CODE
   &redirect_uri=https://vpn.example.com:9000/callback
   &client_id=openvpn
   &client_secret=SECRET
   &code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk
   ```

3. **Keycloak responds** with JSON:
   ```json
   {
     "access_token": "eyJhbGciOiJSUzI1NiIs...",
     "id_token": "eyJhbGciOiJSUzI1NiIs...",
     "refresh_token": "eyJhbGciOiJSUzI1NiIs...",
     "token_type": "Bearer",
     "expires_in": 300
   }
   ```

4. **ID token verification** (via `coreos/go-oidc` library):
   - Fetches JWKS from `https://keycloak.example.com/realms/myrealm/protocol/openid-connect/certs`
   - Validates JWT signature (RS256)
   - Validates claims: `iss`, `aud`, `exp`, `iat`, `nbf`

5. **Claim merging** (`internal/oidc/flow.go`): Verifies the Keycloak access token JWT signature, issuer, expiry, and client binding before merging `resource_access`, `realm_access`, and `groups` claims into ID token claims. ID token claims take precedence. A non-empty access token that cannot be verified is rejected.

6. **Validation** (`internal/oidc/validator.go`):
   - Extracts username from `preferred_username` claim (configurable via `username_claim`)
   - Validates username matches OpenVPN username (unless `allow_username_mismatch: true`)
   - If `required_roles` configured, extracts roles from `realm_access.roles` claim path, checks user has at least one required role

---

### Phase 8: Result Written to OpenVPN

**On success** (`internal/httpserver/callback.go`):

- Claims the final result write for this session so duplicate callbacks cannot race to write twice
- Writes `"1"` to `auth_control_file` (file I/O, mode 0600)
- Marks session `ResultWritten = true` after the write succeeds
- Deletes session from memory
- Renders `success.html` in user's browser (embedded template)

**OpenVPN** reads `"1"` -> **VPN tunnel established**

**On failure** (`internal/httpserver/callback.go`):

- Writes reason to `auth_failed_reason_file` **first** (critical ordering -- `internal/openvpn/authfile.go`)
- Writes `"0"` to `auth_control_file`
- Marks session, deletes it
- Renders `error.html` in user's browser

**OpenVPN** reads `"0"` -> **connection rejected**, shows reason to user

---

### Phase 9: Safety Nets

The daemon attempts to write a final result for every known session that has an OpenVPN `auth_control_file`:

1. **Defer in `handleCallback`** (`internal/httpserver/callback.go`): If callback finishes without writing a result, it claims the result write and writes `"0"`
2. **Background cleanup** (`internal/session/cleanup.go`): Every 60 seconds, expired unclaimed sessions get `"0"` written + "Authentication timeout" reason
3. **Error paths in daemon** (`internal/daemon/daemon.go`): If OIDC flow or pending file write fails after result paths are known, writes `"0"` immediately

Missing or unknown callback states have no associated OpenVPN result path, and filesystem write failures may leave OpenVPN to timeout or retry. Those failures are logged.

---

## IPC Protocol

### Transport

**Unix Domain Socket:**

- Path: `/run/openvpn-keycloak-auth/auth.sock`
- Permissions: `0660` (rw-rw----)
- Owner: `openvpn:openvpn`

**Protocol:** JSON over stream socket

### Message Types

#### Auth Request (Script → Daemon)

```json
{
  "username": "john.doe",
  "common_name": "john.doe",
  "untrusted_ip": "192.0.2.100",
  "untrusted_port": "54321",
  "auth_control_file": "/tmp/openvpn_acf_abc123.tmp",
  "auth_pending_file": "/tmp/openvpn_apf_abc123.tmp",
  "auth_failed_reason_file": "/tmp/openvpn_arf_abc123.tmp"
}
```

#### Auth Response (Daemon → Script)

**Success (deferred):**

```json
{
  "status": "deferred",
  "session_id": "64-character-hex-string"
}
```

**Error:**

```json
{
  "status": "error",
  "error": "Failed to create session"
}
```

There is no message-type envelope: the socket carries exactly one request
shape and one response shape, and the peer is already constrained by
`SO_PEERCRED` (see [Concurrency Model](#concurrency-model)).

### Connection Flow

```go
// Client (auth script)
conn, err := net.Dial("unix", socketPath)
defer conn.Close()

// Send request
encoder := json.NewEncoder(conn)
encoder.Encode(request)

// Receive response
decoder := json.NewDecoder(conn)
var response AuthResponse
decoder.Decode(&response)
```

### Error Handling

**IPC errors result in authentication failure:**

- Socket not accessible → Auth failure
- Timeout (5 seconds) → Auth failure
- Malformed response → Auth failure
- Daemon not running → Auth failure

**Fail-secure:** Any IPC error prevents VPN connection.

---

## Session Management

### Session Structure

```go
type Session struct {
    // Identifiers
    ID                   string    // 64-char hex (32 bytes crypto/rand)
    State                string    // 32-char hex (16 bytes crypto/rand)
    
    // User info
    Username             string    // Keycloak username
    CommonName           string    // OpenVPN common name
    UntrustedIP          string    // Client IP
    UntrustedPort        string    // Client port
    
    // PKCE
    CodeVerifier         string    // 43-char base64url (32 bytes crypto/rand)
    CodeChallenge        string    // SHA256(verifier), base64url
    
    // OpenVPN files
    AuthControlFile      string    // /tmp/openvpn_acf_*.tmp
    AuthPendingFile      string    // /tmp/openvpn_apf_*.tmp
    AuthFailedReasonFile string    // /tmp/openvpn_arf_*.tmp
    
    // Lifecycle
    CreatedAt            time.Time
    ExpiresAt            time.Time // CreatedAt + TTL
    ResultWritten        bool      // Has auth_control_file been written?
}
```

### Session Storage

**In-memory map:**

```go
type Manager struct {
    mu       sync.RWMutex
    sessions map[string]*Session  // Key: session ID
    byState  map[string]*Session  // Key: state parameter
    ttl      time.Duration
    cleanup  *time.Ticker
}
```

**Thread-safety:** All operations protected by `sync.RWMutex`

### Session Lifecycle

```
Create → [Active] → Callback → [Completed] → Delete
   ↓         ↓                                    ↑
   │         │                                    │
   │         └─→ [Expired] ─────────────────────→ │
   │                (TTL cleanup)                 │
   └──────────────────────────────────────────────┘
```

**States:**

1. **Created:** Session exists, waiting for callback
2. **Active:** Before TTL expiration
3. **Expired:** TTL passed, cleanup will delete
4. **Completed:** Callback received, result written

### TTL Cleanup

**Goroutine runs every 60 seconds:**

```go
func (m *Manager) startCleanup() {
    m.cleanup = time.NewTicker(60 * time.Second)
    go func() {
        for range m.cleanup.C {
            m.cleanupExpired()
        }
    }()
}
```

**Cleanup process:**

1. Lock sessions map
2. Iterate all sessions
3. If `time.Now().After(session.ExpiresAt)`:
   - Write auth failure to `auth_control_file`
   - Delete from map
4. Unlock

**Why write auth failure on expiry?**

- OpenVPN is waiting for `auth_control_file`
- Without it, connection hangs until hand-window timeout
- Writing "0" immediately fails the authentication

---

## OIDC Implementation

### Authorization Code Flow with PKCE

**Standard:** OpenID Connect Core 1.0, RFC 7636 (PKCE)

#### Step 1: Generate PKCE Verifier and Challenge

```go
// Verifier: 32 bytes from crypto/rand, base64url encoded
verifier := make([]byte, 32)
rand.Read(verifier)
verifierB64 := base64.RawURLEncoding.EncodeToString(verifier)

// Challenge: SHA256 hash of verifier, base64url encoded
hash := sha256.Sum256([]byte(verifierB64))
challenge := base64.RawURLEncoding.EncodeToString(hash[:])
```

**Verifier example:** `dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk`
**Challenge example:** `E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM`

#### Step 2: Build Authorization URL

```
https://keycloak.example.com/realms/myrealm/protocol/openid-connect/auth?
  response_type=code&
  client_id=openvpn&
  redirect_uri=https://vpn.example.com:9000/callback&
  scope=openid+profile+email&
  state=abc123...&
  code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&
  code_challenge_method=S256
```

#### Step 3: Exchange Authorization Code for Token

**POST** `https://keycloak.example.com/realms/myrealm/protocol/openid-connect/token`

**Body (form-encoded):**

```
grant_type=authorization_code&
code=xyz123...&
redirect_uri=https://vpn.example.com:9000/callback&
client_id=openvpn&
code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk
```

**Response:**

```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 300,
  "id_token": "eyJhbG...",
  "refresh_token": "eyJhbG..."
}
```

### Token Validation

#### JWT Structure

```
Header:
{
  "alg": "RS256",
  "typ": "JWT",
  "kid": "abc123"
}

Payload (claims):
{
  "iss": "https://keycloak.example.com/realms/myrealm",
  "aud": "openvpn",
  "exp": 1709215200,
  "iat": 1709214900,
  "nbf": 1709214900,
  "sub": "user-uuid",
  "preferred_username": "john.doe",
  "email": "john.doe@example.com",
  "realm_access": {
    "roles": ["vpn-user", "admin"]
  }
}

Signature: <RS256 signature>
```

#### Validation Steps

1. **Fetch JWKS** (JSON Web Key Set) from Keycloak:
   ```
   GET https://keycloak.example.com/realms/myrealm/protocol/openid-connect/certs
   ```

2. **Verify signature:**
   - Extract `kid` from JWT header
   - Find matching key in JWKS
   - Verify RS256 signature

3. **Validate claims:**
   ```go
   // Issuer
   if claims["iss"] != config.Issuer {
       return errors.New("invalid issuer")
   }

   // Audience
   if claims["aud"] != config.ClientID {
       return errors.New("invalid audience")
   }

   // Expiration
   exp := claims["exp"].(float64)
   if time.Now().Unix() > int64(exp) {
       return errors.New("token expired")
   }

   // Username
   username := claims["preferred_username"].(string)
   if username != expectedUsername {
       return errors.New("username mismatch")
   }

   // Roles (if required)
   if len(requiredRoles) > 0 {
       userRoles := extractRoles(claims, roleClaim)
       if !hasAnyRole(userRoles, requiredRoles) {
           return errors.New("insufficient roles")
       }
   }
   ```

---

## File Operations

### OpenVPN Control Files

OpenVPN creates temporary files for each authentication attempt:

```
/tmp/openvpn_acf_abc123.tmp    # Auth control file
/tmp/openvpn_apf_abc123.tmp    # Auth pending file
/tmp/openvpn_arf_abc123.tmp    # Auth failed reason file
```

### Writing Order (Critical!)

**For success:**

```go
// 1. Write auth_control_file with "1"
os.WriteFile(authControlFile, []byte("1\n"), 0600)
```

**For failure:**

```go
// 1. Write reason to auth_failed_reason_file
os.WriteFile(authFailedReasonFile, []byte(reason+"\n"), 0600)

// 2. Write "0" to auth_control_file
os.WriteFile(authControlFile, []byte("0\n"), 0600)
```

**For deferral:**

```go
// Write auth_pending_file (exactly 3 lines!)
content := fmt.Sprintf("%d\n%s\nWEB_AUTH::%s\n", timeout, method, authURL)
os.WriteFile(authPendingFile, []byte(content), 0600)
```

### File Permissions

All temporary files created with `0600` (owner read/write only).

---

## Concurrency Model

### Goroutines

**Daemon mode:**

```
Main goroutine
├─ HTTP server (blocking Listen)
├─ Unix socket server (blocking Accept)
│  └─ Per-connection handler (goroutine)
└─ Session cleanup ticker (goroutine)
   └─ Runs every 60 seconds
```

**Per-request goroutines:**

- HTTP request handlers (net/http automatically creates goroutines)
- IPC connection handlers (one goroutine per connection)

### Synchronization

**Session map:** Protected by `sync.RWMutex`

```go
// Read lock (multiple concurrent readers OK)
m.mu.RLock()
session := m.sessions[id]
m.mu.RUnlock()

// Write lock (exclusive access)
m.mu.Lock()
m.sessions[id] = session
m.mu.Unlock()
```

**Rate limiter:** Each IP has its own `rate.Limiter`

```go
type IPRateLimiter struct {
    mu       sync.Mutex
    limiters map[string]*rate.Limiter
}
```

---

## Error Handling

### Error Handling Philosophy

**Fail secure:** Errors always result in authentication denial.

### Critical Paths

**Auth script path:**

```go
// If ANY error occurs:
// 1. Log error
// 2. Return exit code 1 (failure)
// 3. OpenVPN denies connection
```

**Callback path:**

```go
// Best-effort final result for known sessions with OpenVPN result paths.
defer func() {
    if sessionMgr.ClaimResultWrite(session.ID) {
        // Safety net: write failure if nothing else claimed the result
        writeAuthFailure(session, "Internal error")
    }
}()
```

### Error Types

1. **Configuration errors** - Fail at startup (check-config)
2. **OIDC errors** - Write auth failure, log details
3. **IPC errors** - Auth script returns exit 1
4. **Session errors** - Write auth failure (session not found, expired)
5. **Token validation errors** - Write auth failure with reason
6. **File write errors** - Log, but can't recover (system issue)

---

## Protocol Summary

| Hop                    | Protocol                            | Data Format                                                |
| ---------------------- | ----------------------------------- | ---------------------------------------------------------- |
| User -> OpenVPN        | OpenVPN (TLS over UDP/TCP :1194)    | username + password                                        |
| OpenVPN -> Auth Script | Process exec + env vars + temp file | Env vars + 2-line credentials file                         |
| Auth Script -> Daemon  | Unix socket (`AF_UNIX SOCK_STREAM`) | JSON (`AuthRequest`)                                       |
| Daemon -> Auth Script  | Unix socket                         | JSON (`AuthResponse`)                                      |
| Daemon -> OpenVPN      | File I/O (`auth_pending_file`)      | 3-line text: `timeout\nmethod\nWEB_AUTH::url\n`            |
| OpenVPN -> Client      | OpenVPN control channel             | `AUTH_PENDING` message                                     |
| Client -> Browser      | OS URL open                         | HTTPS URL                                                  |
| Browser -> Daemon      | HTTPS (`GET /auth/<state>`)         | HTTP request                                               |
| Daemon -> Browser      | HTTPS (302 redirect)                | `Location:` header to Keycloak                             |
| Browser -> Keycloak    | HTTPS                               | OIDC Authorization Request                                 |
| Keycloak -> Browser    | HTTPS (302 redirect)                | `Location:` header with auth code                          |
| Browser -> Daemon      | HTTPS (`GET /callback`)             | Query params: `code`, `state`                              |
| Daemon -> Keycloak     | HTTPS (`POST` token endpoint)       | `application/x-www-form-urlencoded` (code + PKCE verifier) |
| Keycloak -> Daemon     | HTTPS                               | JSON (access_token, id_token, refresh_token)               |
| Daemon -> Keycloak     | HTTPS (`GET` JWKS)                  | JSON Web Key Set (for JWT verification)                    |
| Daemon -> OpenVPN      | File I/O (`auth_control_file`)      | Single char: `"1"` or `"0"`                                |
| Daemon -> Browser      | HTTPS                               | HTML success/error page                                    |

---

## Security Architecture Summary

Mechanism-level view; the reasoning, threat model, and hardening steps live in
[security.md](security.md).

| Concern              | Mitigation                                                                     |
| -------------------- | ------------------------------------------------------------------------------ |
| CSRF                 | OIDC `state` parameter (16 random bytes)                                       |
| Code interception    | PKCE S256 (32-byte verifier)                                                   |
| Token tampering      | JWT signature verification via JWKS                                            |
| Credential exposure  | Password excluded from IPC; tokens tagged `json:"-"`                           |
| Log injection        | Control characters stripped from all external inputs (CWE-117)                 |
| Rate limiting        | Per-IP token bucket (10/s, burst 50)                                           |
| Socket access        | Unix socket mode 0660, group `openvpn`                                         |
| Session IDs          | 32 bytes from `crypto/rand`                                                    |
| Double-write         | Atomic `MarkResultWritten` prevents duplicate auth results                     |
| Hanging connections  | Safety-net defer + TTL cleanup guarantee `auth_control_file` is always written |
| Privilege escalation | systemd hardening: NoNewPrivileges, ProtectSystem=strict, syscall filtering    |
| XSS/Clickjacking     | Security headers: CSP, X-Frame-Options, HSTS                                   |

---

## Design Decisions

### Why Script-Based Auth (Not C Plugin)?

**Pros:**

- ✅ No C code (easier to maintain)
- ✅ No CGO (easier to build)
- ✅ No shared libraries (simpler deployment)
- ✅ Better testability
- ✅ Easier debugging

**Cons:**

- ❌ Slightly higher overhead (exec process)
- ❌ Limited to OpenVPN 2.6.2+

**Decision:** Script-based is better for maintainability and simplicity.

### Why Unix Socket IPC (Not HTTP)?

**Pros:**

- ✅ No network exposure
- ✅ File permissions for security
- ✅ Lower overhead
- ✅ Simpler than HTTP

**Cons:**

- ❌ Local only (can't distribute across hosts)

**Decision:** Unix socket is more secure and simpler.

### Why In-Memory Sessions (Not Redis)?

**Pros:**

- ✅ Simpler deployment (no external dependencies)
- ✅ Faster access
- ✅ Lower latency

**Cons:**

- ❌ Sessions lost on restart
- ❌ Can't run multiple daemon instances
- ❌ Memory usage grows with sessions

**Decision:** In-memory is acceptable for single-instance deployment. Future v1.1 can add Redis support.

### Why Single Binary with Modes (Not Separate Binaries)?

**Pros:**

- ✅ Single file to distribute
- ✅ Shared code (no duplication)
- ✅ Consistent versioning

**Cons:**

- ❌ Slightly larger binary

**Decision:** Single binary is simpler to manage.

### Why JSON Over Unix Socket (Not Protocol Buffers)?

**Pros:**

- ✅ Human-readable (easier debugging)
- ✅ No schema compilation
- ✅ Simpler implementation

**Cons:**

- ❌ Slightly larger messages
- ❌ Slower serialization

**Decision:** JSON is fast enough for this use case.

---

## Performance Characteristics

### Latency Breakdown

**Typical authentication flow:**

```
Auth script execution:        50-100ms
  └─ IPC round-trip:           10-20ms
  └─ Session creation:         5-10ms
  └─ PKCE generation:          5-10ms
  └─ Auth URL construction:    1-2ms
  └─ File writes:              5-10ms

User browser authentication:  5-30 seconds (user-dependent)

OIDC callback processing:     100-300ms
  └─ Token exchange:           50-150ms (network to Keycloak)
  └─ JWT verification:         20-50ms (JWKS fetch cached)
  └─ Claim validation:         5-10ms
  └─ File write:               5-10ms

Total: ~5-30 seconds (mostly user interaction)
```

### Memory Usage

**Typical:**

- Base daemon: ~10MB
- Per session: ~2KB
- 1000 concurrent sessions: ~12MB total

### Throughput

**Tested:**

- 50 concurrent authentications: ✅ No issues
- Rate limit: 10 req/s per IP (configurable)

---

**Document Version:** 1.0\
**Last Updated:** 2026-02-15\
**Audience:** Developers, architects, security reviewers

For questions about architecture, see [CONTRIBUTING.md](../CONTRIBUTING.md) or open an issue.
