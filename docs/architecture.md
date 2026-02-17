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
10. [Design Decisions](#design-decisions)

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
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Script-based Deferred Authentication (2.6+)              │ │
│  │  - Calls auth script with via-file                        │ │
│  │  - Expects exit code 2 (deferred)                         │ │
│  │  - Reads auth_pending_file (WEB_AUTH:: URL)              │ │
│  │  - Sends AUTH_PENDING to client                           │ │
│  │  - Waits for auth_control_file (0 or 1)                   │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────┬───────────────────────────────────────────┘
                       │
                       │ 2. Calls /etc/openvpn/auth-keycloak.sh
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
                       │ 3. Unix socket: /run/openvpn-keycloak-sso/auth.sock
                       ▼
┌──────────────────────────────────────────────────────────────────┐
│            Daemon (Go binary, serve mode, systemd)               │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Session Manager  │  IPC Server  │  HTTP Server            │ │
│  │  - In-memory map  │  - Unix sock │  - OIDC callback        │ │
│  │  - TTL cleanup    │  - JSON API  │  - Success/error pages  │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  OIDC Provider                                              │ │
│  │  - Authorization URL builder                               │ │
│  │  - PKCE generator (S256)                                    │ │
│  │  - Token exchanger                                          │ │
│  │  - JWT validator (signature + claims)                       │ │
│  └────────────────────────────────────────────────────────────┘ │
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

### 1. Go Binary (`openvpn-keycloak-sso`)

**Single executable with 4 operating modes:**

#### Mode 1: `serve` (Daemon)

Runs as systemd service, handles:
- Unix socket IPC server
- HTTP server for OIDC callbacks
- Session management with TTL cleanup
- OIDC provider integration
- Token validation

**Entry point:** `cmd/openvpn-keycloak-sso/main.go` → `serveCmd`

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

**Entry point:** `cmd/openvpn-keycloak-sso/main.go` → `authCmd`

**Execution time:** <100ms (just IPC call)

#### Mode 3: `version`

Display version information.

#### Mode 4: `check-config`

Validate configuration file:
- YAML syntax
- Required fields
- Keycloak connectivity
- OIDC discovery

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

`/etc/openvpn/auth-keycloak.sh`:

```bash
#!/bin/bash
# Thin wrapper called by OpenVPN
# Just execs the Go binary in auth mode
exec /usr/local/bin/openvpn-keycloak-sso auth "$@"
```

**Why a wrapper?**
- OpenVPN `--auth-user-pass-verify` expects a shell script
- Easier to update just the binary without touching OpenVPN config
- Can add environment setup if needed

---

## Data Flow

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
│ Unix Socket  │ /run/openvpn-keycloak-sso/auth.sock
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
└────┬─────────┘    300\nopenurl\nWEB_AUTH::https://...
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

---

## IPC Protocol

### Transport

**Unix Domain Socket:**
- Path: `/run/openvpn-keycloak-sso/auth.sock`
- Permissions: `0660` (rw-rw----)
- Owner: `openvpn:openvpn`

**Protocol:** JSON over stream socket

### Message Types

#### Auth Request (Script → Daemon)

```json
{
  "type": "auth_request",
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
  "type": "auth_response",
  "status": "deferred",
  "session_id": "64-character-hex-string",
  "auth_url": "https://keycloak.example.com/realms/myrealm/protocol/openid-connect/auth?..."
}
```

**Error:**
```json
{
  "type": "auth_response",
  "status": "error",
  "error": "Failed to create session"
}
```

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
content := fmt.Sprintf("%d\nopenurl\nWEB_AUTH::%s\n", timeout, authURL)
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
// Ensure auth_control_file is ALWAYS written
defer func() {
    if !session.ResultWritten {
        // Safety net: write failure if nothing else did
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
- ❌ Limited to OpenVPN 2.6+

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

**Document Version:** 1.0  
**Last Updated:** 2026-02-15  
**Audience:** Developers, architects, security reviewers

For questions about architecture, see [CONTRIBUTING.md](../CONTRIBUTING.md) or open an issue.
