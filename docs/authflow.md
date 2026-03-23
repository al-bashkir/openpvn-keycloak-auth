# Complete Authentication Flow

## Overview

```
VPN Client <-> OpenVPN Server <-> Auth Script <-> Daemon <-> Keycloak
                                    (Unix socket)  (HTTPS)
                                       |              |
                                   File I/O       User's Browser
```

Single Go binary runs in two modes: **auth script** (short-lived, per-connection) and **daemon** (long-running service). They communicate via **Unix domain socket** (`/run/openvpn-keycloak-auth/auth.sock`) using a **JSON protocol**.

---

## Phase 1: Connection Initiation

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

## Phase 2: Auth Script Execution

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
  "type": "auth_request",
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

IPC client (`internal/ipc/client.go`): connects with `net.DialTimeout("unix", socketPath, 5s)`, writes JSON, reads JSON response.

---

## Phase 3: Daemon Processes Auth Request

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

5. **Returns IPC response** -- JSON over Unix socket:
   ```json
   {
     "type": "auth_response",
     "status": "deferred",
     "session_id": "f8a3b1c2d4...",
     "auth_url": "https://vpn.example.com:9000/auth/a1b2c3d4e5f6..."
   }
   ```

---

## Phase 4: Deferred Auth + Browser Opens

**Auth script** receives `"deferred"` response, **exits with code `2`** (tells OpenVPN: "auth is pending").

**OpenVPN server** reads `auth_pending_file`, sends to client via the OpenVPN control channel:
```
AUTH_PENDING,300,webauth,WEB_AUTH::https://vpn.example.com:9000/auth/a1b2c3d4e5f6...
```

**VPN client** (if it supports `IV_SSO=webauth`) opens the user's default browser to the short URL.

---

## Phase 5: Short URL Redirect

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

## Phase 6: Keycloak Authentication

**Browser** follows redirect to Keycloak (HTTPS). **User** authenticates (password, MFA, etc.).

**Keycloak** redirects browser back:
```
HTTP/1.1 302 Found
Location: https://vpn.example.com:9000/callback?code=AUTHORIZATION_CODE&state=a1b2c3d4e5f6...
```

---

## Phase 7: Token Exchange + Validation

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

5. **Claim merging** (`internal/oidc/flow.go`): Decodes Keycloak access token JWT (without signature check -- already trusted from token endpoint), merges `resource_access`, `realm_access`, and `groups` claims into ID token claims (ID token claims take precedence).

6. **Validation** (`internal/oidc/validator.go`):
   - Extracts username from `preferred_username` claim (configurable via `username_claim`)
   - Validates username matches OpenVPN username (unless `allow_username_mismatch: true`)
   - If `required_roles` configured, extracts roles from `realm_access.roles` claim path, checks user has at least one required role

---

## Phase 8: Result Written to OpenVPN

**On success** (`internal/httpserver/callback.go`):
- Writes `"1"` to `auth_control_file` (file I/O, mode 0600)
- Marks session `ResultWritten = true` (atomic, prevents double-write)
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

## Phase 9: Safety Nets

Three mechanisms guarantee `auth_control_file` is **always** written:

1. **Defer in `handleCallback`** (`internal/httpserver/callback.go`): If callback finishes without writing a result, writes `"0"` automatically
2. **Background cleanup** (`internal/session/cleanup.go`): Every 60 seconds, expired sessions get `"0"` written + "Authentication timeout" reason
3. **Error paths in daemon** (`internal/daemon/daemon.go`): If OIDC flow or pending file write fails, writes `"0"` immediately

---

## Protocol Summary

| Hop | Protocol | Data Format |
|-----|----------|-------------|
| User -> OpenVPN | OpenVPN (TLS over UDP/TCP :1194) | username + password |
| OpenVPN -> Auth Script | Process exec + env vars + temp file | Env vars + 2-line credentials file |
| Auth Script -> Daemon | Unix socket (`AF_UNIX SOCK_STREAM`) | JSON (`AuthRequest`) |
| Daemon -> Auth Script | Unix socket | JSON (`AuthResponse`) |
| Daemon -> OpenVPN | File I/O (`auth_pending_file`) | 3-line text: `timeout\nmethod\nWEB_AUTH::url\n` |
| OpenVPN -> Client | OpenVPN control channel | `AUTH_PENDING` message |
| Client -> Browser | OS URL open | HTTPS URL |
| Browser -> Daemon | HTTPS (`GET /auth/<state>`) | HTTP request |
| Daemon -> Browser | HTTPS (302 redirect) | `Location:` header to Keycloak |
| Browser -> Keycloak | HTTPS | OIDC Authorization Request |
| Keycloak -> Browser | HTTPS (302 redirect) | `Location:` header with auth code |
| Browser -> Daemon | HTTPS (`GET /callback`) | Query params: `code`, `state` |
| Daemon -> Keycloak | HTTPS (`POST` token endpoint) | `application/x-www-form-urlencoded` (code + PKCE verifier) |
| Keycloak -> Daemon | HTTPS | JSON (access_token, id_token, refresh_token) |
| Daemon -> Keycloak | HTTPS (`GET` JWKS) | JSON Web Key Set (for JWT verification) |
| Daemon -> OpenVPN | File I/O (`auth_control_file`) | Single char: `"1"` or `"0"` |
| Daemon -> Browser | HTTPS | HTML success/error page |

---

## Security Architecture

| Concern | Mitigation |
|---------|-----------|
| CSRF | OIDC `state` parameter (16 random bytes) |
| Code interception | PKCE S256 (32-byte verifier) |
| Token tampering | JWT signature verification via JWKS |
| Credential exposure | Password excluded from IPC; tokens tagged `json:"-"` |
| Log injection | Control characters stripped from all external inputs (CWE-117) |
| Rate limiting | Per-IP token bucket (10/s, burst 50) |
| Socket access | Unix socket mode 0660, group `openvpn` |
| Session IDs | 32 bytes from `crypto/rand` |
| Double-write | Atomic `MarkResultWritten` prevents duplicate auth results |
| Hanging connections | Safety-net defer + TTL cleanup guarantee `auth_control_file` is always written |
| Privilege escalation | systemd hardening: NoNewPrivileges, ProtectSystem=strict, syscall filtering |
| XSS/Clickjacking | Security headers: CSP, X-Frame-Options, HSTS |
