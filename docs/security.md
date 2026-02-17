# Security Guide - OpenVPN Keycloak SSO

This document covers security considerations, best practices, and hardening recommendations for the OpenVPN Keycloak SSO authentication system.

## Table of Contents

1. [Security Architecture](#security-architecture)
2. [Authentication Flow Security](#authentication-flow-security)
3. [Token Security](#token-security)
4. [Network Security](#network-security)
5. [File Permissions](#file-permissions)
6. [Logging Security](#logging-security)
7. [systemd Hardening](#systemd-hardening)
8. [Keycloak Security](#keycloak-security)
9. [OpenVPN Security](#openvpn-security)
10. [Threat Model](#threat-model)
11. [Security Checklist](#security-checklist)
12. [Incident Response](#incident-response)

---

## Security Architecture

### Defense in Depth

The OpenVPN Keycloak SSO system implements multiple layers of security:

```
┌─────────────────────────────────────────────────────────────┐
│ Layer 1: Network Security (Firewall, TLS)                  │
├─────────────────────────────────────────────────────────────┤
│ Layer 2: Authentication (OIDC + PKCE + MFA)                │
├─────────────────────────────────────────────────────────────┤
│ Layer 3: Authorization (Token Validation + Role Checks)    │
├─────────────────────────────────────────────────────────────┤
│ Layer 4: Application Security (Rate Limiting, Input Valid) │
├─────────────────────────────────────────────────────────────┤
│ Layer 5: System Security (systemd sandboxing, SELinux)     │
├─────────────────────────────────────────────────────────────┤
│ Layer 6: Monitoring (Logging, Audit Trail)                 │
└─────────────────────────────────────────────────────────────┘
```

### Security Principles

1. **Least Privilege** - Components run with minimal required permissions
2. **Fail Secure** - Failures result in authentication denial
3. **Defense in Depth** - Multiple security controls at each layer
4. **Audit Logging** - All authentication attempts are logged
5. **Secure by Default** - Safe default configurations

---

## Authentication Flow Security

### PKCE (Proof Key for Code Exchange)

**Purpose:** Prevents authorization code interception attacks

**Implementation:**
```go
// Code verifier: 32 bytes from crypto/rand
verifier := generateCodeVerifier() // 32 random bytes, base64url encoded

// Code challenge: SHA256 hash of verifier
challenge := sha256.Sum256([]byte(verifier))
challengeB64 := base64.RawURLEncoding.EncodeToString(challenge[:])

// Authorization request includes challenge
authURL := buildAuthURL(challengeB64, "S256")

// Token exchange includes verifier
token := exchangeCode(code, verifier)
```

**Security Properties:**
- Code verifier is 32 bytes from `crypto/rand` (256 bits of entropy)
- Challenge method is S256 (SHA-256 hash)
- Verifier is never sent in authorization request (only challenge)
- Server validates verifier matches challenge when exchanging code
- Protects against authorization code interception even on insecure networks

**Reference:** [RFC 7636 - Proof Key for Code Exchange](https://datatracker.ietf.org/doc/html/rfc7636)

### CSRF Protection (State Parameter)

**Purpose:** Prevents cross-site request forgery attacks

**Implementation:**
```go
// State parameter: 16 bytes from crypto/rand
state := generateState() // 16 random bytes, hex encoded (32 characters)

// State is tied to session ID
session := Session{
    ID:    sessionID,
    State: state,
    // ...
}

// Callback validates state matches session
session, err := sessionMgr.GetByState(callbackState)
if err != nil {
    return ErrInvalidState
}
```

**Security Properties:**
- State is 16 bytes from `crypto/rand` (128 bits of entropy)
- Unique per authentication attempt
- Validated on callback before accepting code
- Tied to specific OpenVPN session
- Prevents attacker from injecting authorization code into victim's session

### Session IDs

**Purpose:** Unique identifier for each authentication attempt

**Implementation:**
```go
// Session ID: 32 bytes from crypto/rand
id := make([]byte, 32)
if _, err := rand.Read(id); err != nil {
    return err
}
sessionID := hex.EncodeToString(id) // 64 hex characters
```

**Security Properties:**
- 32 bytes from `crypto/rand` (256 bits of entropy)
- Hex encoded (64 characters)
- Unique per authentication attempt
- Cannot be predicted or guessed
- Short-lived (TTL: 5 minutes by default)

### Authentication Flow Diagram

```
┌────────┐         ┌─────────┐         ┌─────────┐         ┌──────────┐
│ Client │         │ OpenVPN │         │ Daemon  │         │ Keycloak │
└───┬────┘         └────┬────┘         └────┬────┘         └─────┬────┘
    │                   │                   │                    │
    │ 1. Connect        │                   │                    │
    ├──────────────────>│                   │                    │
    │                   │                   │                    │
    │                   │ 2. Call auth script                    │
    │                   ├──────────────────>│                    │
    │                   │                   │                    │
    │                   │                   │ 3. Generate:       │
    │                   │                   │    - Session ID    │
    │                   │                   │    - PKCE verifier │
    │                   │                   │    - State param   │
    │                   │                   │                    │
    │                   │ 4. Defer (exit 2) │                    │
    │                   │<──────────────────┤                    │
    │                   │    + WEB_AUTH URL │                    │
    │                   │                   │                    │
    │ 5. Open browser   │                   │                    │
    │<──────────────────┤                   │                    │
    │                   │                   │                    │
    │ 6. Redirect to Keycloak (with PKCE challenge + state)     │
    ├────────────────────────────────────────────────────────────>│
    │                   │                   │                    │
    │ 7. User login + MFA                   │                    │
    │<───────────────────────────────────────────────────────────>│
    │                   │                   │                    │
    │ 8. Redirect to callback (code + state)                     │
    ├─────────────────────────────────────>│                    │
    │                   │                   │                    │
    │                   │                   │ 9. Validate state  │
    │                   │                   │                    │
    │                   │                   │ 10. Exchange code  │
    │                   │                   │    (with verifier) │
    │                   │                   ├───────────────────>│
    │                   │                   │                    │
    │                   │                   │ 11. Token + ID     │
    │                   │                   │<───────────────────┤
    │                   │                   │                    │
    │                   │                   │ 12. Validate JWT   │
    │                   │                   │     signature      │
    │                   │                   │                    │
    │                   │                   │ 13. Validate       │
    │                   │                   │     claims         │
    │                   │                   │                    │
    │                   │ 14. Write success │                    │
    │                   │<──────────────────┤                    │
    │                   │                   │                    │
    │ 15. VPN connected │                   │                    │
    │<──────────────────┤                   │                    │
    │                   │                   │                    │
```

### Attack Mitigation

| Attack | Mitigation |
|--------|------------|
| **Authorization Code Interception** | PKCE prevents use of intercepted code |
| **CSRF** | State parameter validation |
| **Session Fixation** | New session ID per attempt, tied to OpenVPN session |
| **Man-in-the-Middle** | TLS for all communications, JWT signature validation |
| **Replay Attacks** | Short-lived tokens (exp claim), one-time authorization codes |
| **Token Injection** | State parameter ties token to specific session |

---

## Token Security

### JWT Signature Verification

**Implementation:**
```go
// Fetch JWKS from Keycloak
keySet := oidc.NewRemoteKeySet(ctx, jwksURL)

// Verify token signature
verifier := oidc.NewVerifier(issuer, keySet, &oidc.Config{
    ClientID: clientID,
})

idToken, err := verifier.Verify(ctx, rawToken)
```

**Security Properties:**
- Public keys fetched from Keycloak's JWKS endpoint
- Keys cached with TTL to reduce latency
- Signature verified using RS256 (RSA SHA-256)
- Only tokens signed by Keycloak are accepted

### Claim Validation

All JWT claims are validated before accepting a token:

| Claim | Validation | Purpose |
|-------|------------|---------|
| `iss` (Issuer) | Must match configured Keycloak URL | Prevents token from other issuers |
| `aud` (Audience) | Must match client ID | Prevents token for other applications |
| `exp` (Expiration) | Must be in the future | Prevents use of expired tokens |
| `iat` (Issued At) | Must be in the past | Prevents premature token use |
| `nbf` (Not Before) | Must be in the past or now | Prevents premature token use |
| `preferred_username` | Must match OpenVPN username | Ensures correct user |

**Implementation:**
```go
// Automatic validation by go-oidc library
idToken, err := verifier.Verify(ctx, rawToken)
if err != nil {
    return err // Token invalid
}

// Additional custom validation
claims := make(map[string]interface{})
if err := idToken.Claims(&claims); err != nil {
    return err
}

// Validate username
username, ok := claims["preferred_username"].(string)
if !ok || username == "" {
    return errors.New("username claim missing")
}

// Validate roles (if required)
if len(requiredRoles) > 0 {
    if !hasRequiredRole(claims, requiredRoles) {
        return errors.New("user missing required roles")
    }
}
```

### Token Lifetime

**Recommendations:**
- **Access Token:** 5 minutes (Keycloak default: 5 minutes)
- **ID Token:** 5 minutes (Keycloak default: 5 minutes)
- **Session TTL (daemon):** 5 minutes (configurable)

**Rationale:**
- Short lifetime reduces window for token theft
- Sessions are short-lived (just for authentication)
- VPN session lasts longer than SSO session
- Re-authentication required for new VPN session

### No Password Transmission

**Critical Security Property:**

The user's Keycloak password is **NEVER** transmitted to the VPN server.

```
Traditional VPN Auth (password sent to VPN server):
User → [username + password] → VPN Server → LDAP/AD

SSO Auth (password stays with Keycloak):
User → [username + "sso"] → VPN Server → Daemon → Redirect to Keycloak
User → [password] → Keycloak → [token] → Daemon → VPN Server
```

**Benefits:**
- VPN server never sees or stores passwords
- Password only entered on trusted Keycloak server
- Reduces attack surface significantly
- Enables passwordless auth (WebAuthn, TOTP, etc.)

---

## Network Security

### TLS Requirements

**Keycloak Connection:**
- **Minimum:** TLS 1.2
- **Recommended:** TLS 1.3
- **Cipher Suites:** Modern AEAD ciphers (AES-GCM)
- **Certificate Validation:** Required (system CA bundle)

**HTTP Callback Server:**
- **Default:** HTTP on port 9000
- **Recommended:** HTTPS with TLS termination at reverse proxy
- **Production:** Always use HTTPS for callback URL

**Example nginx reverse proxy:**
```nginx
server {
    listen 443 ssl http2;
    server_name vpn.example.com;

    ssl_certificate /etc/ssl/certs/vpn.example.com.crt;
    ssl_certificate_key /etc/ssl/private/vpn.example.com.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers 'ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_prefer_server_ciphers on;

    location /callback {
        proxy_pass http://127.0.0.1:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Firewall Configuration

**Inbound Rules:**
```bash
# Allow HTTPS for reverse proxy (if using nginx)
firewall-cmd --permanent --add-service=https

# Or allow direct HTTP callback (less secure)
firewall-cmd --permanent --add-port=9000/tcp

# Apply changes
firewall-cmd --reload
```

**Outbound Rules:**
```bash
# Allow HTTPS to Keycloak
# (usually default ALLOW on Rocky Linux)
firewall-cmd --permanent --add-service=https
firewall-cmd --reload
```

**Recommendation:** Use a reverse proxy (nginx, Apache, Caddy) for TLS termination rather than exposing the Go HTTP server directly.

### Rate Limiting

**Implementation:**

Per-IP rate limiting prevents brute force and DoS attacks:

```go
// Default: 10 requests per second per IP, burst of 50
var globalLimiter = newIPRateLimiter(10, 50)
```

**Configuration:**

Adjust in `internal/httpserver/middleware.go` if needed:

```go
// More restrictive (1 req/sec per IP, burst 10)
var globalLimiter = newIPRateLimiter(1, 10)

// Less restrictive (30 req/sec per IP, burst 100)
var globalLimiter = newIPRateLimiter(30, 100)
```

**IP Extraction:**

Rate limiting considers proxy headers:
1. `X-Forwarded-For` (first IP if behind proxy)
2. `X-Real-IP` (if set by proxy)
3. `RemoteAddr` (fallback)

**Important:** If behind a reverse proxy, ensure the proxy sets `X-Forwarded-For` correctly.

### Security Headers

All HTTP responses include security headers:

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Frame-Options` | `DENY` | Prevent clickjacking |
| `X-Content-Type-Options` | `nosniff` | Prevent MIME sniffing |
| `X-XSS-Protection` | `1; mode=block` | Enable XSS filter |
| `Referrer-Policy` | `no-referrer` | Don't leak referer |
| `Content-Security-Policy` | `default-src 'self'` | Restrict content sources |
| `Strict-Transport-Security` | `max-age=31536000` | Force HTTPS (if TLS enabled) |

**Implementation:** See `internal/httpserver/middleware.go` - `securityHeadersMiddleware()`

---

## File Permissions

### Critical Files

All sensitive files must have restrictive permissions:

| File/Directory | Permissions | Owner:Group | Contents |
|----------------|-------------|-------------|----------|
| `/etc/openvpn/keycloak-sso.yaml` | `0600` | `root:openvpn` | Keycloak client ID, issuer URL |
| `/usr/local/bin/openvpn-keycloak-auth` | `0755` | `root:root` | Binary (world-readable OK) |
| `/etc/openvpn/auth-keycloak.sh` | `0755` | `root:root` | Auth script (executable) |
| `/var/lib/openvpn-keycloak-auth/` | `0755` | `openvpn:openvpn` | Data directory |
| `/run/openvpn-keycloak-auth/` | `0770` | `openvpn:openvpn` | Socket directory (runtime) |
| `/run/openvpn-keycloak-auth/auth.sock` | `0660` | `openvpn:openvpn` | Unix socket |

### Verification Script

```bash
#!/bin/bash
# Verify file permissions

check_perms() {
    local file="$1"
    local expected_perms="$2"
    local expected_owner="$3"
    
    if [ ! -e "$file" ]; then
        echo "❌ $file does not exist"
        return 1
    fi
    
    actual_perms=$(stat -c '%a' "$file")
    actual_owner=$(stat -c '%U:%G' "$file")
    
    if [ "$actual_perms" != "$expected_perms" ]; then
        echo "❌ $file has wrong permissions: $actual_perms (expected $expected_perms)"
        return 1
    fi
    
    if [ "$actual_owner" != "$expected_owner" ]; then
        echo "⚠️  $file has wrong owner: $actual_owner (expected $expected_owner)"
    fi
    
    echo "✅ $file: $actual_perms $actual_owner"
}

echo "Checking file permissions..."
echo ""

check_perms "/etc/openvpn/keycloak-sso.yaml" "600" "root:openvpn"
check_perms "/usr/local/bin/openvpn-keycloak-auth" "755" "root:root"
check_perms "/etc/openvpn/auth-keycloak.sh" "755" "root:root"
check_perms "/var/lib/openvpn-keycloak-auth" "755" "openvpn:openvpn"

if [ -e "/run/openvpn-keycloak-auth/auth.sock" ]; then
    check_perms "/run/openvpn-keycloak-auth" "770" "openvpn:openvpn"
    check_perms "/run/openvpn-keycloak-auth/auth.sock" "660" "openvpn:openvpn"
else
    echo "⚠️  Socket not found (daemon not running?)"
fi

echo ""
echo "Checking configuration file readability..."
if [ -r "/etc/openvpn/keycloak-sso.yaml" ]; then
    echo "✅ Configuration readable by current user"
else
    echo "⚠️  Configuration not readable (run as root or openvpn group)"
fi
```

### SELinux Contexts

If SELinux is enabled, verify contexts:

```bash
# Check binary context
ls -Z /usr/local/bin/openvpn-keycloak-auth
# Should show: system_u:object_r:bin_t:s0

# Check config context
ls -Z /etc/openvpn/keycloak-sso.yaml
# Should show: system_u:object_r:etc_t:s0 or openvpn_etc_t:s0

# Restore contexts if needed
restorecon -v /usr/local/bin/openvpn-keycloak-auth
restorecon -Rv /etc/openvpn/
```

---

## Logging Security

### No Secrets in Logs

**Critical Rule:** Never log sensitive data

**Prohibited:**
- ❌ Keycloak client secrets
- ❌ Access tokens or ID tokens
- ❌ Authorization codes
- ❌ PKCE verifiers
- ❌ User passwords (shouldn't have them anyway)

**Code Review:**
```bash
# Check for potential secret leakage
grep -r "slog.*token\|slog.*secret\|slog.*password" --include="*.go" internal/

# Should find NO instances of logging actual values
# Only "token exchange failed", "token validation failed", etc.
```

**Implementation:**

```go
// ✅ GOOD - Log event without sensitive data
slog.Info("token received from Keycloak")
slog.Error("token validation failed", "error", err)

// ❌ BAD - Never do this!
slog.Debug("token", "value", tokenString) // NEVER LOG TOKENS!
```

### Audit Logging

All authentication attempts are logged:

**Successful Authentication:**
```
INFO user authenticated successfully session_id=abc123 username=john.doe ip=203.0.113.10
INFO auth success written session_id=abc123 username=john.doe ip=203.0.113.10
```

**Failed Authentication:**
```
ERROR token validation failed session_id=abc123 username=john.doe error="username mismatch"
INFO auth failure written session_id=abc123 reason="username mismatch"
```

**Rate Limiting:**
```
WARN rate limit exceeded ip=203.0.113.50 path=/callback
```

### Log Rotation

Configure logrotate for journal logs:

```bash
# /etc/systemd/journald.conf
[Journal]
SystemMaxUse=500M
SystemKeepFree=1G
SystemMaxFileSize=100M
MaxRetentionSec=2weeks
```

Restart journald:
```bash
systemctl restart systemd-journald
```

### PII Minimization

**Logged Data:**
- ✅ Usernames (required for audit trail)
- ✅ IP addresses (required for security monitoring)
- ✅ Session IDs (required for troubleshooting)
- ✅ Timestamps (automatic)
- ✅ Success/failure status

**Not Logged:**
- ❌ Full names or email addresses (unless part of username)
- ❌ Phone numbers
- ❌ Any data not required for security or troubleshooting

---

## systemd Hardening

The systemd service file includes extensive security hardening:

### Sandboxing

```ini
# Filesystem protection
ProtectSystem=strict              # /usr, /boot, /efi read-only
ProtectHome=true                  # /home inaccessible
ReadWritePaths=/var/lib/openvpn-keycloak-auth
PrivateTmp=true                   # Private /tmp namespace

# Kernel protection
ProtectKernelTunables=true        # /proc/sys, /sys read-only
ProtectKernelModules=true         # Can't load kernel modules
ProtectKernelLogs=true            # Can't access kernel logs
ProtectControlGroups=true         # /sys/fs/cgroup read-only
```

### Privilege Restrictions

```ini
# Prevent privilege escalation
NoNewPrivileges=true              # Can't gain new privileges
RestrictSUIDSGID=true             # SUID/SGID bits ignored
LockPersonality=true              # Can't change execution domain
PrivateUsers=true                 # Private user namespace

# Restrict namespaces
RestrictNamespaces=true           # Limit namespace creation
RestrictRealtime=true             # No real-time scheduling
```

### System Call Filtering

```ini
# Allow only system calls needed for typical services
SystemCallFilter=@system-service

# Block dangerous system calls
SystemCallFilter=~@privileged @resources @obsolete @debug @mount
SystemCallErrorNumber=EPERM       # Return permission denied
SystemCallArchitectures=native    # Block non-native architectures
```

### Capabilities

```ini
# Drop all capabilities (port 9000 doesn't require CAP_NET_BIND_SERVICE)
CapabilityBoundingSet=
AmbientCapabilities=

# If using port < 1024, uncomment:
# CapabilityBoundingSet=CAP_NET_BIND_SERVICE
# AmbientCapabilities=CAP_NET_BIND_SERVICE
```

### Resource Limits

```ini
LimitNOFILE=65536                 # Max open files
LimitNPROC=512                    # Max processes
TasksMax=512                      # Max threads
```

### Testing Hardening

Verify systemd security:

```bash
# Analyze service security
systemd-analyze security openvpn-keycloak-auth.service

# Should show low exposure score (< 3.0)

# Check running process restrictions
systemctl status openvpn-keycloak-auth
cat /proc/$(systemctl show -p MainPID --value openvpn-keycloak-auth)/status | grep Cap
```

---

## Keycloak Security

### Client Configuration

**Required Settings:**

1. **Client Type:** Public (no client secret)
2. **PKCE:** Required, S256 method
3. **Valid Redirect URIs:** Exact match only
   ```
   https://vpn.example.com:9000/callback
   ```
4. **Web Origins:** For CORS (if needed)
   ```
   https://vpn.example.com
   ```

**Recommended Settings:**

1. **Access Token Lifespan:** 5 minutes
2. **SSO Session Idle:** 30 minutes
3. **SSO Session Max:** 10 hours
4. **Client Session Idle:** 5 minutes
5. **Client Session Max:** 10 hours

### MFA (Multi-Factor Authentication)

**Highly Recommended:** Enable MFA for all VPN users

**Supported Methods:**
- TOTP (Time-based One-Time Password) - Google Authenticator, Authy
- WebAuthn - YubiKey, TouchID, Windows Hello
- SMS (less secure, but better than nothing)

**Configuration:**

1. Realm → Authentication → Required Actions
2. Enable "Configure OTP"
3. Authentication → Flows → Browser
4. Add "OTP Form" execution
5. Set to "Required"

### Brute Force Protection

Enable Keycloak's built-in brute force detection:

1. Realm → Security Defenses → Brute Force Detection
2. Enable "Permanently Lockout"
3. Set:
   - Max Login Failures: 5
   - Wait Increment: 60 seconds
   - Max Wait: 900 seconds (15 minutes)
   - Failure Reset Time: 720 seconds (12 minutes)

### Event Logging

Enable audit logging:

1. Realm → Events → Config
2. Save Events: ON
3. Expiration: 30 days (or longer for compliance)
4. Event Listeners: Add "jboss-logging"
5. Saved Types: Select all login/logout events

**Monitor for:**
- Failed login attempts
- Unexpected login locations
- Multiple rapid logins (potential token theft)

---

## OpenVPN Security

### Server Configuration

**Critical Security Settings:**

```conf
# Require minimum TLS 1.2
tls-version-min 1.2

# Use strong ciphers
data-ciphers AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305

# TLS authentication (extra security layer)
tls-auth ta.key 0

# Verify client certificates (if using mutual TLS)
verify-client-cert require  # or 'optional' for SSO-only
```

### Certificate Security

**Key Sizes:**
- CA: 4096-bit RSA or EC P-384
- Server: 2048-bit RSA or EC P-256
- Client: 2048-bit RSA or EC P-256 (if using)

**Lifetimes:**
- CA: 10 years maximum
- Server: 2-3 years
- Client: 1-2 years

**Revocation:**
- Maintain CRL (Certificate Revocation List)
- Or use OCSP stapling

### Network Security

**Routing:**
```conf
# Only push necessary routes (principle of least privilege)
push "route 10.0.0.0 255.0.0.0"  # Internal network only

# DNS
push "dhcp-option DNS 10.0.0.1"  # Internal DNS server

# Don't route all traffic unless necessary
# redirect-gateway def1  # Only if needed
```

**Firewall:**
```bash
# Allow only OpenVPN and SSH
firewall-cmd --permanent --add-service=openvpn
firewall-cmd --permanent --add-service=ssh
firewall-cmd --reload
```

---

## Threat Model

### In-Scope Threats

| Threat | Likelihood | Impact | Mitigation |
|--------|-----------|--------|------------|
| Phishing | High | High | User education, MFA |
| Credential stuffing | Medium | High | Unique passwords, MFA |
| Token theft | Low | High | Short token lifetime, TLS |
| Man-in-the-Middle | Low | High | TLS everywhere, cert pinning |
| Brute force | Medium | Medium | Rate limiting, account lockout |
| DoS | Medium | Medium | Rate limiting, firewall |
| Session hijacking | Low | High | CSRF protection, short sessions |

### Out-of-Scope Threats

- Physical server compromise (assumed trusted infrastructure)
- Keycloak compromise (assumed hardened separately)
- Client-side malware (user's responsibility)
- Social engineering (user education)

### Attack Scenarios

**Scenario 1: Stolen Authorization Code**

*Attack:* Attacker intercepts authorization code

*Mitigation:*
- PKCE prevents use of stolen code (attacker doesn't have verifier)
- Code is one-time use only
- Code expires in 60 seconds

**Scenario 2: Stolen Access Token**

*Attack:* Attacker steals access token from network or logs

*Mitigation:*
- Tokens never logged
- TLS encrypts network traffic
- Short token lifetime (5 minutes)
- Token only valid for specific client

**Scenario 3: CSRF Attack**

*Attack:* Attacker tricks user into using attacker's authorization code

*Mitigation:*
- State parameter ties callback to specific session
- State is random and unpredictable
- State validated before accepting code

**Scenario 4: Session Fixation**

*Attack:* Attacker forces user to use known session ID

*Mitigation:*
- Session ID generated server-side from crypto/rand
- New session ID for each authentication attempt
- Session ID tied to OpenVPN connection

**Scenario 5: DoS via Unlimited Requests**

*Attack:* Attacker floods callback endpoint

*Mitigation:*
- Rate limiting (10 req/s per IP)
- Firewall can block abusive IPs
- systemd resource limits prevent resource exhaustion

---

## Security Checklist

### Deployment Checklist

**Before Production:**

- [ ] TLS enabled for all connections (Keycloak, callback)
- [ ] Firewall configured (only necessary ports open)
- [ ] SELinux enabled and configured
- [ ] File permissions verified (run verification script)
- [ ] systemd service hardening enabled
- [ ] Keycloak MFA enabled for all users
- [ ] Keycloak brute force protection enabled
- [ ] Keycloak event logging enabled
- [ ] OpenVPN using strong ciphers (AES-256-GCM)
- [ ] OpenVPN TLS 1.2+ required
- [ ] Certificates have appropriate lifetimes
- [ ] CRL or OCSP configured
- [ ] Log rotation configured
- [ ] Monitoring/alerting set up
- [ ] Backup and recovery tested
- [ ] Incident response plan documented

### Regular Maintenance

**Weekly:**
- [ ] Review authentication logs for anomalies
- [ ] Check for failed login attempts
- [ ] Verify services are running

**Monthly:**
- [ ] Review and rotate logs
- [ ] Check for security updates (OpenVPN, Keycloak, daemon)
- [ ] Review Keycloak event logs
- [ ] Test backup recovery

**Quarterly:**
- [ ] Review user access (remove inactive users)
- [ ] Update dependencies (go.mod)
- [ ] Security audit with automated tools
- [ ] Review and update firewall rules
- [ ] Test incident response procedures

**Annually:**
- [ ] Renew certificates before expiration
- [ ] Full security penetration test
- [ ] Review and update security policies
- [ ] User security awareness training

### Automated Checks

```bash
#!/bin/bash
# Security check script

echo "=== Security Checks ==="
echo ""

# 1. Check service is running
if systemctl is-active --quiet openvpn-keycloak-auth; then
    echo "✅ Service is running"
else
    echo "❌ Service is NOT running"
fi

# 2. Check file permissions
echo ""
echo "Checking file permissions..."
if [ "$(stat -c '%a' /etc/openvpn/keycloak-sso.yaml 2>/dev/null)" = "600" ]; then
    echo "✅ Config permissions correct (600)"
else
    echo "❌ Config permissions WRONG (should be 600)"
fi

# 3. Check for secrets in logs
echo ""
echo "Checking logs for secrets..."
if journalctl -u openvpn-keycloak-auth --since "1 day ago" | grep -iE "token.*:[[:space:]]*ey|secret.*:[[:space:]]*[^[]" > /dev/null; then
    echo "❌ WARNING: Possible secrets in logs!"
else
    echo "✅ No secrets found in logs"
fi

# 4. Check firewall
echo ""
echo "Checking firewall..."
if firewall-cmd --list-ports 2>/dev/null | grep -q "9000/tcp"; then
    echo "✅ Firewall port 9000/tcp open"
else
    echo "⚠️  Firewall port 9000/tcp NOT open (may be intended)"
fi

# 5. Check SELinux
echo ""
echo "Checking SELinux..."
if command -v getenforce >/dev/null && [ "$(getenforce)" = "Enforcing" ]; then
    echo "✅ SELinux is Enforcing"
else
    echo "⚠️  SELinux is NOT enforcing"
fi

echo ""
echo "=== Checks Complete ==="
```

---

## Incident Response

### Detection

**Indicators of Compromise:**

1. Multiple failed authentication attempts from single IP
2. Successful authentication from unusual location
3. Rapid succession of authentications
4. Authentication outside normal hours
5. Service crashes or restarts
6. Unexpected network traffic patterns

**Monitoring:**

```bash
# Monitor failed authentications
journalctl -u openvpn-keycloak-auth -f | grep -i "failed\|error"

# Count failed attempts by IP
journalctl -u openvpn-keycloak-auth --since "1 hour ago" \
  | grep "token validation failed" \
  | grep -oP 'ip=\S+' \
  | sort | uniq -c | sort -rn

# Monitor rate limiting
journalctl -u openvpn-keycloak-auth --since "1 hour ago" \
  | grep "rate limit exceeded"
```

### Response Steps

**If compromise is suspected:**

1. **Immediate Actions:**
   ```bash
   # Stop the service
   sudo systemctl stop openvpn-keycloak-auth
   
   # Block suspicious IP in firewall
   sudo firewall-cmd --add-rich-rule="rule family='ipv4' source address='<IP>' reject"
   
   # Disable affected user in Keycloak
   # (via Keycloak admin console)
   ```

2. **Investigation:**
   ```bash
   # Collect logs
   journalctl -u openvpn-keycloak-auth --since "24 hours ago" > incident-$(date +%Y%m%d).log
   journalctl -u openvpn@server --since "24 hours ago" >> incident-$(date +%Y%m%d).log
   
   # Check active VPN connections
   sudo openvpn --status /var/log/openvpn/status.log
   
   # Review Keycloak event logs
   # (via Keycloak admin console → Events)
   ```

3. **Containment:**
   - Revoke all active VPN sessions
   - Force password reset for affected users
   - Rotate OpenVPN certificates if needed
   - Update firewall rules

4. **Recovery:**
   - Apply security patches
   - Update configurations
   - Restore from known-good backup if needed
   - Re-enable service

5. **Post-Incident:**
   - Document timeline and actions taken
   - Update security policies
   - Conduct lessons-learned review
   - Improve monitoring/detection

### Reporting

**Internal:**
- Document incident in security log
- Notify security team and management
- Update risk register

**External:**
- Notify users if credentials compromised
- Report to regulatory bodies if required (GDPR, etc.)

---

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [CIS Controls](https://www.cisecurity.org/controls)
- [OpenID Connect Core](https://openid.net/specs/openid-connect-core-1_0.html)
- [RFC 7636 - PKCE](https://datatracker.ietf.org/doc/html/rfc7636)
- [RFC 7519 - JWT](https://datatracker.ietf.org/doc/html/rfc7519)

---

**Document Version:** 1.0  
**Last Updated:** 2026-02-15  
**Maintained By:** OpenVPN SSO Security Team

For security issues, see [SECURITY.md](../SECURITY.md) for responsible disclosure information.
