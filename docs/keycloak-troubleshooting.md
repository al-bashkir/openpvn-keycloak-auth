# Keycloak Troubleshooting Guide

Common issues and solutions when using Keycloak with OpenVPN SSO.

## Table of Contents

- [Connection Issues](#connection-issues)
- [Authentication Failures](#authentication-failures)
- [Token Issues](#token-issues)
- [Role/Permission Issues](#rolepermission-issues)
- [PKCE Issues](#pkce-issues)
- [Redirect URI Issues](#redirect-uri-issues)
- [Performance Issues](#performance-issues)
- [Debugging Tools](#debugging-tools)

---

## Connection Issues

### Daemon Can't Connect to Keycloak

**Symptom:**
```
ERROR failed to create OIDC provider error="Get \"https://keycloak.example.com/realms/openvpn/.well-known/openid-configuration\": dial tcp: lookup keycloak.example.com: no such host"
```

**Diagnosis:**

```bash
# Test DNS resolution
nslookup keycloak.example.com

# Test HTTP connectivity
curl -v http://keycloak.example.com:8080/realms/openvpn/.well-known/openid-configuration

# Test from VPN server (if different from daemon server)
ssh vpn-server "curl -v http://keycloak.example.com:8080/realms/openvpn/.well-known/openid-configuration"
```

**Solutions:**

1. **DNS not resolving**:
   - Add entry to `/etc/hosts`: `192.168.1.100 keycloak.example.com`
   - Fix DNS server configuration
   - Use IP address instead of hostname (temporary)

2. **Firewall blocking**:
   ```bash
   # Check if port is open
   telnet keycloak.example.com 8080
   
   # Or use nc
   nc -zv keycloak.example.com 8080
   ```
   - Open port 8080 (or 443 for HTTPS) in firewall
   - On Rocky Linux 9:
     ```bash
     sudo firewall-cmd --add-port=8080/tcp --permanent
     sudo firewall-cmd --reload
     ```

3. **Keycloak not running**:
   ```bash
   # Check service status
   sudo systemctl status keycloak
   
   # Check container status (if using podman/docker)
   podman ps | grep keycloak
   ```

4. **Wrong URL**:
   - Verify issuer URL in config matches Keycloak
   - Format: `http://hostname:port/realms/realm-name`
   - No trailing slash!

### SSL/TLS Certificate Errors

**Symptom:**
```
ERROR x509: certificate signed by unknown authority
```

**Solutions:**

1. **Self-signed certificate**:
   ```bash
   # Add CA certificate to system trust store (Rocky Linux 9)
   sudo cp keycloak-ca.crt /etc/pki/ca-trust/source/anchors/
   sudo update-ca-trust
   ```

2. **Use HTTP for testing** (NOT for production):
   ```yaml
   # keycloak-sso.yaml
   oidc:
     issuer: "http://keycloak.example.com:8080/realms/openvpn"  # HTTP not HTTPS
   ```

3. **Skip TLS verification** (DANGEROUS - testing only):
   - Not recommended; fix certificates instead

---

## Authentication Failures

### Users Get "Invalid Credentials"

**Symptom:**
User enters correct username/password but gets error in Keycloak.

**Diagnosis:**

1. **Check user status in Keycloak**:
   - Go to **Users** → Find user
   - Verify **Enabled**: ON
   - Verify **Email verified**: ON (if email verification required)

2. **Check user password**:
   - Go to user → **Credentials** tab
   - Set a new password with **Temporary**: OFF

3. **Check realm status**:
   - Go to **Realm settings** → **General** tab
   - Verify **Enabled**: ON

**Solutions:**

- Reset user password in Keycloak
- Ensure user account is not locked (check **Events** in Keycloak)
- Check if email verification is required but not completed

### Browser Opens But Shows "Invalid Request"

**Symptom:**
Browser opens to Keycloak, but immediately shows error.

**Diagnosis:**

Check the browser URL bar. You'll see parameters like:
```
http://keycloak:8080/realms/openvpn/protocol/openid-connect/auth?
  client_id=openvpn&
  redirect_uri=http://vpn.example.com:9000/callback&
  ...
```

**Common Causes:**

1. **Client ID mismatch**:
   - Client ID in URL must match Keycloak client
   - Case-sensitive: `openvpn` ≠ `OpenVPN`

2. **Redirect URI not registered**:
   - The `redirect_uri` in URL must be in **Valid redirect URIs** list
   - Must match EXACTLY (including protocol, port, path)

3. **Client disabled**:
   - Go to **Clients** → `openvpn` → **Settings**
   - Verify **Enabled**: ON

**Solutions:**

- Update **Valid redirect URIs** in Keycloak client settings
- Verify client_id in `keycloak-sso.yaml` matches Keycloak
- Enable the client in Keycloak

---

## Token Issues

### "No id_token in token response"

**Symptom:**
```
ERROR token exchange failed error="no id_token in token response"
```

**Diagnosis:**

The token endpoint should return an `id_token`. Let's verify:

```bash
# Get an authorization code first (manual test)
# Then exchange it for tokens

curl -X POST http://keycloak:8080/realms/openvpn/protocol/openid-connect/token \
  -d "grant_type=authorization_code" \
  -d "client_id=openvpn" \
  -d "code=YOUR_CODE" \
  -d "redirect_uri=http://vpn.example.com:9000/callback" \
  -d "code_verifier=YOUR_VERIFIER"
```

**Causes:**

1. **Client scope configuration**:
   - `openid` scope not requested or assigned
   
2. **Client type wrong**:
   - Client authentication enabled (confidential client)
   - Should be public client for PKCE

**Solutions:**

1. **Verify client scopes**:
   - Go to **Clients** → `openvpn` → **Client scopes** tab
   - Ensure these are assigned:
     - `openid` (should be automatic)
     - `profile`
     - `email`

2. **Verify client type**:
   - Go to **Clients** → `openvpn` → **Settings**
   - **Client authentication**: OFF (public client)

### Token Contains Wrong Claims

**Symptom:**
```
ERROR username claim 'preferred_username' not found
```

**Diagnosis:**

Decode the ID token to see what claims it contains:

```bash
# Get ID token, then decode (without verification)
echo "PASTE_ID_TOKEN_HERE" | cut -d. -f2 | base64 -d 2>/dev/null | jq .
```

**Expected Claims:**
```json
{
  "exp": 1708022400,
  "iat": 1708022100,
  "iss": "http://keycloak:8080/realms/openvpn",
  "aud": "openvpn",
  "sub": "a1b2c3d4-...",
  "preferred_username": "testuser",
  "email": "testuser@example.com",
  "realm_access": {
    "roles": ["vpn-user", "offline_access", ...]
  }
}
```

**Solutions:**

1. **Missing `preferred_username`**:
   - Add `profile` scope to client
   - Or change `username_claim` in config to `email` or `sub`

2. **Missing `email`**:
   - Add `email` scope to client

3. **Missing roles**:
   - Add `roles` scope to client
   - See [Role/Permission Issues](#rolepermission-issues)

---

## Role/Permission Issues

### "User does not have required roles"

**Symptom:**
```
ERROR token validation failed error="user does not have required roles: [vpn-user] (user roles: [offline_access uma_authorization])"
```

**Diagnosis:**

The user's token doesn't contain the required VPN role.

**Steps to Verify:**

1. **Check user has role assigned**:
   - **Users** → Select user → **Role mapping** tab
   - Should see `vpn-user` in **Assigned roles**

2. **Check role exists**:
   - **Realm roles** → Look for `vpn-user`

3. **Check role is in token**:
   - Decode ID token (see above)
   - Look for `realm_access.roles` array
   - Should contain `"vpn-user"`

**Solutions:**

1. **Assign role to user**:
   - **Users** → Select user → **Role mapping** tab
   - Click **"Assign role"**
   - Check `vpn-user`
   - Click **"Assign"**

2. **Check role claim path**:
   ```yaml
   # keycloak-sso.yaml
   oidc:
     role_claim: "realm_access.roles"  # For realm roles
     # role_claim: "resource_access.openvpn.roles"  # For client roles
   ```

3. **Verify roles scope assigned**:
   - **Clients** → `openvpn` → **Client scopes** tab
   - Ensure `roles` is in **Assigned default client scopes**

### Roles Not Appearing in Token

**Symptom:**
ID token doesn't contain `realm_access.roles` at all.

**Solutions:**

1. **Add roles scope**:
   - **Clients** → `openvpn` → **Client scopes** tab
   - Click **"Add client scope"**
   - Select `roles`
   - Choose **"Default"**

2. **Check role mapper**:
   - **Client scopes** → `roles` → **Mappers** tab
   - Should see `realm roles` mapper
   - Verify it's enabled

3. **Request roles scope explicitly**:
   ```yaml
   # keycloak-sso.yaml
   oidc:
     scopes:
       - openid
       - profile
       - email
       - roles  # Add this
   ```

---

## PKCE Issues

### "PKCE verification failed"

**Symptom:**
```
ERROR token exchange failed error="failed to exchange code: PKCE verification failed"
```

**Diagnosis:**

This means the `code_verifier` sent during token exchange doesn't match the `code_challenge` sent during authorization.

**Causes:**

1. **PKCE not enabled in Keycloak**
2. **Wrong PKCE method configured**
3. **Bug in code (unlikely - our implementation is tested)**

**Solutions:**

1. **Enable PKCE in Keycloak**:
   - **Clients** → `openvpn` → **Advanced** tab
   - **Proof Key for Code Exchange Code Challenge Method**: `S256`
   - Click **"Save"**

2. **Verify client is public**:
   - **Clients** → `openvpn` → **Settings**
   - **Client authentication**: OFF

3. **Check Keycloak version**:
   - PKCE for public clients is standard in Keycloak 18+
   - If older version, upgrade Keycloak

### "Code challenge method not supported"

**Symptom:**
Error during authorization request.

**Solution:**

- Use Keycloak 18+ which supports S256 by default
- Verify `code_challenge_methods_supported` in well-known endpoint:
  ```bash
  curl -s http://keycloak:8080/realms/openvpn/.well-known/openid-configuration | jq .code_challenge_methods_supported
  ```
  Should include `["plain", "S256"]`

---

## Redirect URI Issues

### "Invalid redirect URI"

**Symptom:**
After successful login, Keycloak shows: "Invalid parameter: redirect_uri"

**Diagnosis:**

The redirect URI in the authorization request doesn't match any of the registered URIs in Keycloak.

**Steps:**

1. **Check authorization URL** (from browser):
   ```
   http://keycloak:8080/realms/openvpn/protocol/openid-connect/auth?
     redirect_uri=http://vpn.example.com:9000/callback  <-- This must match Keycloak
   ```

2. **Check Keycloak configuration**:
   - **Clients** → `openvpn` → **Settings**
   - **Valid redirect URIs**

**Common Mismatches:**

| Authorization Request | Keycloak Config | Match? |
|-----------------------|----------------|--------|
| `http://vpn.example.com:9000/callback` | `http://vpn.example.com:9000/callback` | ✅ Yes |
| `https://vpn.example.com:9000/callback` | `http://vpn.example.com:9000/callback` | ❌ No (HTTP vs HTTPS) |
| `http://vpn.example.com:9000/callback` | `http://vpn.example.com/callback` | ❌ No (missing port) |
| `http://10.0.0.1:9000/callback` | `http://vpn.example.com:9000/callback` | ❌ No (hostname vs IP) |

**Solutions:**

1. **Add missing redirect URI**:
   - **Clients** → `openvpn` → **Settings**
   - **Valid redirect URIs**: Add the exact URI from your config
   - Click **"+"** button
   - Click **"Save"**

2. **Use wildcards carefully** (only for development):
   - `http://localhost:*/callback` - Matches any port
   - `http://*:9000/callback` - Matches any hostname
   - `*` - Matches everything (insecure, don't use in production)

3. **Match your config exactly**:
   ```yaml
   # keycloak-sso.yaml
   oidc:
     redirect_uri: "http://vpn.example.com:9000/callback"
   ```
   Must match Keycloak **Valid redirect URIs** character-for-character.

---

## Performance Issues

### Token Validation Slow

**Symptom:**
Authentication takes longer than expected.

**Diagnosis:**

- Check JWKS cache settings
- Monitor network latency to Keycloak

**Solutions:**

1. **Increase JWKS cache duration**:
   ```yaml
   # keycloak-sso.yaml
   oidc:
     jwks_cache_duration: 3600  # 1 hour (default)
     # Increase to reduce JWKS fetches
   ```

2. **Use local Keycloak**:
   - Deploy Keycloak close to VPN server
   - Reduce network latency

3. **Check Keycloak performance**:
   - Monitor Keycloak database
   - Check Keycloak logs for slow queries
   - Tune Keycloak settings (connection pool, cache)

### Too Many Sessions in Keycloak

**Symptom:**
Keycloak admin shows thousands of sessions.

**Solutions:**

1. **Configure session timeout**:
   - **Realm settings** → **Sessions** tab
   - **SSO Session Idle**: `5 minutes`
   - **SSO Session Max**: `10 hours`
   - **Client Session Idle**: `5 minutes`

2. **Enable token revocation** (optional):
   - Configure `auth-gen-token` in OpenVPN to expire tokens

---

## Debugging Tools

### Check OIDC Discovery

```bash
curl -s http://keycloak:8080/realms/openvpn/.well-known/openid-configuration | jq .
```

Expected issuer:
```json
{
  "issuer": "http://keycloak:8080/realms/openvpn"
}
```

### Decode ID Token (Without Verification)

```bash
# Paste your ID token
TOKEN="eyJhbGc..."

# Decode header
echo $TOKEN | cut -d. -f1 | base64 -d 2>/dev/null | jq .

# Decode payload (claims)
echo $TOKEN | cut -d. -f2 | base64 -d 2>/dev/null | jq .
```

### Test Token Endpoint

```bash
# Exchange code for token (you need a valid authorization code)
curl -X POST http://keycloak:8080/realms/openvpn/protocol/openid-connect/token \
  -d "grant_type=authorization_code" \
  -d "client_id=openvpn" \
  -d "code=YOUR_AUTHORIZATION_CODE" \
  -d "redirect_uri=http://vpn.example.com:9000/callback" \
  -d "code_verifier=YOUR_PKCE_VERIFIER"
```

### Check Keycloak Logs

```bash
# If using systemd
sudo journalctl -u keycloak -f

# If using podman
podman logs -f keycloak

# Look for errors related to:
# - Invalid redirect URI
# - PKCE verification
# - Token exchange
```

### Monitor Daemon Logs

```bash
# Watch daemon logs
sudo journalctl -u openvpn-keycloak-auth -f

# Filter for errors
sudo journalctl -u openvpn-keycloak-auth | grep ERROR

# Show detailed OIDC flow
# Set log level to debug in config:
# log:
#   level: debug
```

### Test Authentication Flow Manually

1. **Get authorization URL**:
   ```bash
   # Generate PKCE verifier
   VERIFIER=$(openssl rand -base64 32 | tr -d '=' | tr '+/' '-_')
   
   # Generate challenge
   CHALLENGE=$(echo -n $VERIFIER | openssl dgst -sha256 -binary | base64 | tr -d '=' | tr '+/' '-_')
   
   # Build URL
   echo "http://keycloak:8080/realms/openvpn/protocol/openid-connect/auth?client_id=openvpn&redirect_uri=http://vpn.example.com:9000/callback&response_type=code&scope=openid%20profile%20email&code_challenge=$CHALLENGE&code_challenge_method=S256&state=test123"
   ```

2. **Open URL in browser**, log in, get code from redirect

3. **Exchange code for token**:
   ```bash
   CODE="paste_code_here"
   
   curl -X POST http://keycloak:8080/realms/openvpn/protocol/openid-connect/token \
     -d "grant_type=authorization_code" \
     -d "client_id=openvpn" \
     -d "code=$CODE" \
     -d "redirect_uri=http://vpn.example.com:9000/callback" \
     -d "code_verifier=$VERIFIER"
   ```

---

## Common Error Messages and Solutions

| Error | Cause | Solution |
|-------|-------|----------|
| `failed to create OIDC provider` | Can't reach Keycloak | Check network, DNS, firewall |
| `invalid redirect URI` | URI mismatch | Update Keycloak Valid redirect URIs |
| `PKCE verification failed` | PKCE not configured | Enable S256 in Keycloak Advanced settings |
| `user does not have required roles` | Missing role assignment | Assign vpn-user role to user |
| `username claim not found` | Wrong claim path | Check username_claim in config |
| `session not found or expired` | Session timeout | Increase session_timeout in config |
| `token exchange failed` | Various token issues | Check client configuration, scopes |

---

## Getting More Help

1. **Enable debug logging**:
   ```yaml
   # keycloak-sso.yaml
   log:
     level: debug
     format: text  # More readable than JSON for debugging
   ```

2. **Check all logs**:
   - Daemon: `journalctl -u openvpn-keycloak-auth -f`
   - OpenVPN: `journalctl -u openvpn-server@server -f`
   - Keycloak: `journalctl -u keycloak -f` or `podman logs -f keycloak`

3. **Verify configuration**:
   ```bash
   openvpn-keycloak-auth check-config --config /etc/openvpn/keycloak-sso.yaml
   ```

4. **Test OIDC flow manually** (see Debugging Tools above)

5. **Check GitHub issues**: Search for similar problems

6. **Open an issue**: Provide:
   - Daemon logs (with debug enabled)
   - Keycloak version
   - OpenVPN version
   - Configuration (redact secrets!)
   - Exact error message

---

*Last updated: 2026-02-15 for Keycloak 25.0.6*
