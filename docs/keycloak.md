# Keycloak Configuration for OpenVPN SSO

This guide walks you through configuring Keycloak 25.0.6 as the Identity Provider for OpenVPN SSO authentication.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Step 1: Create Realm](#step-1-create-realm)
- [Step 2: Create OIDC Client](#step-2-create-oidc-client)
- [Step 3: Configure PKCE](#step-3-configure-pkce)
- [Step 4: Configure Client Scopes](#step-4-configure-client-scopes)
- [Step 5: Create Realm Roles](#step-5-create-realm-roles)
- [Step 6: Create and Configure Users](#step-6-create-and-configure-users)
- [Step 7: Verify Configuration](#step-7-verify-configuration)
- [Testing](#testing)
- [Next Steps](#next-steps)
- [Security Recommendations](#security-recommendations)
- Troubleshooting: [Connection](#connection-issues), [Authentication](#authentication-failures),
  [Tokens](#token-issues), [Roles](#rolepermission-issues), [PKCE](#pkce-issues),
  [Redirect URI](#redirect-uri-issues), [Performance](#performance-issues),
  [Debugging Tools](#debugging-tools)

---

## Prerequisites

- **Keycloak 25.0.6** or later installed and running
- Admin access to Keycloak Admin Console
- DNS/hostname for VPN server (e.g., `vpn.example.com`)
- Basic understanding of OIDC/OAuth 2.0 concepts

**Keycloak Installation (Quick Reference):**

```bash
# Using Podman/Docker
podman run -d --name keycloak \
  -p 8080:8080 \
  -e KEYCLOAK_ADMIN=admin \
  -e KEYCLOAK_ADMIN_PASSWORD=admin \
  quay.io/keycloak/keycloak:25.0.6 \
  start-dev

# Production: Use proper database and TLS configuration
# See: https://www.keycloak.org/server/configuration
```

---

## Step 1: Create Realm

A realm in Keycloak is a space where you manage users, credentials, roles, and groups. We'll create a dedicated realm for OpenVPN.

### 1.1 Access Admin Console

1. Open browser and navigate to: `http://your-keycloak-server:8080/admin/`
2. Log in with admin credentials
3. You'll see the **Keycloak Admin Console** home page

### 1.2 Create New Realm

1. **Click the realm dropdown** in the top-left corner (currently shows "Keycloak" or "master")
2. Click **"Create realm"**
3. Fill in the form:
   - **Realm name**: `openvpn` (or your preferred name)
   - **Enabled**: ON (toggle should be blue)
4. Click **"Create"**

**Screenshot Description:** The realm dropdown is in the top-left, next to the Keycloak logo. After clicking "Create realm", you'll see a simple form with a text input for the realm name and an enabled toggle.

### 1.3 Verify Realm Creation

- The realm dropdown now shows `openvpn`
- You're now in the realm's dashboard
- URL should be: `http://your-keycloak-server:8080/admin/master/console/#/openvpn`

---

## Step 2: Create OIDC Client

Now we'll create an OpenID Connect client for the OpenVPN SSO daemon.

### 2.1 Navigate to Clients

1. In the left sidebar, click **"Clients"**
2. Click **"Create client"** button (top-right)

### 2.2 General Settings

You'll see a wizard with multiple steps.

**Step 1 of 3: General settings**

- **Client type**: Select `OpenID Connect` (default)
- **Client ID**: Enter `openvpn`
  - This is the `client_id` you'll use in `keycloak-sso.yaml`
  - Must match exactly (case-sensitive)
- **Name**: `OpenVPN SSO` (optional, user-friendly name)
- **Description**: `OpenVPN SSO Authentication` (optional)

Click **"Next"**

### 2.3 Capability Config

**Step 2 of 3: Capability config**

This is **critical** for security.

- **Client authentication**: `OFF` (toggle should be gray)
  - This makes it a **public client** (no client secret)
  - Required for PKCE-based authentication

- **Authorization**: `OFF` (toggle should be gray)
  - We don't need fine-grained authorization

- **Authentication flow** (check these boxes):
  - ✅ **Standard flow** (Authorization Code Flow)
  - ✅ **Direct access grants** (optional, for testing with curl)
  - ❌ **Implicit flow** (UNCHECK - deprecated and insecure)
  - ❌ **Service accounts roles** (UNCHECK - not needed)
  - ❌ **OAuth 2.0 Device Authorization Grant** (UNCHECK - not needed)
  - ❌ **OIDC CIBA Grant** (UNCHECK - not needed)

Click **"Next"**

### 2.4 Login Settings

**Step 3 of 3: Login settings**

This is where you configure the callback URL.

- **Root URL**: Leave empty
- **Home URL**: Leave empty
- **Valid redirect URIs**:
  - `http://vpn.example.com:9000/callback`
  - `http://localhost:9000/callback` (for local testing)
  - Click **"+"** to add each URL
  - **Important**: Must match EXACTLY what you configure in `keycloak-sso.yaml`

- **Valid post logout redirect URIs**: `*` or leave empty
- **Web origins**: `*` (allow all origins for CORS)
  - In production, specify exact origins: `http://vpn.example.com:9000`

Click **"Save"**

**Screenshot Description:** The form has multiple text inputs. You'll see a "+" button to add multiple redirect URIs. The interface is clean and modern (React-based UI in Keycloak 25.x).

---

## Step 3: Configure PKCE

PKCE (Proof Key for Code Exchange) is **required** for security with public clients. In Keycloak 25.0.6, this is configured in the Advanced settings.

### 3.1 Access Advanced Settings

1. Stay in the `openvpn` client configuration
2. Click the **"Advanced"** tab (top navigation within client)
3. Scroll down to **"Advanced settings"** section

### 3.2 Configure PKCE

Find the following setting:

- **Proof Key for Code Exchange Code Challenge Method**: Select `S256`
  - This matches our implementation which uses SHA256
  - Do **NOT** use `plain` - it's insecure

**Other Important Settings in Advanced Tab:**

- **Access Token Lifespan**: `5 minutes` (default is fine)
- **Client Session Idle**: `5 minutes` (default is fine)
- **Client Session Max**: `10 hours` (default is fine)

Click **"Save"** at the bottom of the page

**Screenshot Description:** The Advanced tab shows many configuration options. The PKCE setting is in a dropdown menu with options: `(not set)`, `plain`, and `S256`. Make sure `S256` is selected.

---

## Step 4: Configure Client Scopes

Client scopes determine what information is included in the ID token and access token.

### 4.1 Check Default Scopes

1. In the `openvpn` client, click the **"Client scopes"** tab
2. You should see **Default scopes assigned**:
   - `acr`
   - `email`
   - `profile`
   - `roles`
   - `web-origins`

These are automatically assigned and sufficient for OpenVPN SSO.

### 4.2 Verify Required Scopes

Ensure these scopes are in the **"Assigned default client scopes"** section:

- **`profile`** - Provides `preferred_username`, `name`, etc.
- **`email`** - Provides `email` claim
- **`roles`** - Provides realm and client roles

**Note:** The `openid` scope is implicit and always included.

### 4.3 Optional: Add Custom Scope

If you want VPN-specific claims, you can create a custom scope:

1. Go to **Client scopes** in the left sidebar (realm-level, not client-level)
2. Click **"Create client scope"**
3. Name: `vpn`
4. Protocol: `OpenID Connect`
5. Add custom mappers if needed

Then assign it to the `openvpn` client.

---

## Step 5: Create Realm Roles

Roles are used to control who can access the VPN. This is optional but recommended for production.

### 5.1 Create VPN User Role

1. In the left sidebar, click **"Realm roles"**
2. Click **"Create role"** button
3. Fill in the form:
   - **Role name**: `vpn-user`
   - **Description**: `Users allowed to connect to VPN`
4. Click **"Save"**

### 5.2 Create Additional Roles (Optional)

You may want different access levels:

- `vpn-admin` - Full VPN access + admin features
- `vpn-dev` - Developer VPN access
- `vpn-contractor` - Limited VPN access

Create each role following the same steps.

### 5.3 Verify Role Creation

- Go back to **"Realm roles"**
- You should see `vpn-user` in the list
- The role is now available for assignment to users

---

## Step 6: Create and Configure Users

Now create users and assign them the VPN role.

### 6.1 Create a Test User

1. In the left sidebar, click **"Users"**
2. Click **"Add user"** button
3. Fill in the form:
   - **Username**: `testuser` (required)
   - **Email**: `testuser@example.com` (optional but recommended)
   - **First name**: `Test` (optional)
   - **Last name**: `User` (optional)
   - **Email verified**: ON (toggle to blue)
   - **Enabled**: ON (toggle should be blue)
4. Click **"Create"**

### 6.2 Set User Password

After creating the user:

1. Click the **"Credentials"** tab (top navigation)
2. Click **"Set password"**
3. Fill in:
   - **Password**: Enter a password
   - **Password confirmation**: Re-enter the same password
   - **Temporary**: OFF (toggle should be gray)
     - If ON, user must change password on first login
4. Click **"Save"**
5. Confirm by clicking **"Save password"** in the modal

### 6.3 Assign Roles to User

1. Still in the user's configuration, click the **"Role mapping"** tab
2. Click **"Assign role"** button
3. In the modal that appears:
   - You'll see a list of available roles
   - **Check the box** next to `vpn-user`
   - You can use the search box to find it quickly
4. Click **"Assign"**

### 6.4 Verify User Configuration

The user's **Role mapping** tab should now show:

- **Assigned roles**: `vpn-user`
- Plus default roles like `default-roles-openvpn`, `offline_access`, `uma_authorization`

---

## Step 7: Verify Configuration

Let's verify everything is configured correctly.

### 7.1 Check Client Configuration

Navigate to **Clients** → `openvpn` and verify:

- ✅ **Client authentication**: OFF (public client)
- ✅ **Valid redirect URIs**: Your VPN server URL with `/callback`
- ✅ **PKCE method**: S256 (in Advanced tab)
- ✅ **Standard flow**: Enabled

### 7.2 Check OIDC Discovery Endpoint

Test the OIDC discovery endpoint (this is what the daemon uses):

```bash
curl -s http://your-keycloak-server:8080/realms/openvpn/.well-known/openid-configuration | jq .
```

Expected output (abbreviated):

```json
{
  "issuer": "http://your-keycloak-server:8080/realms/openvpn",
  "authorization_endpoint": "http://your-keycloak-server:8080/realms/openvpn/protocol/openid-connect/auth",
  "token_endpoint": "http://your-keycloak-server:8080/realms/openvpn/protocol/openid-connect/token",
  "jwks_uri": "http://your-keycloak-server:8080/realms/openvpn/protocol/openid-connect/certs",
  "response_types_supported": ["code", ...],
  "code_challenge_methods_supported": ["plain", "S256"],
  ...
}
```

**Important**: The `issuer` value is what you'll use in `keycloak-sso.yaml`

### 7.3 Configuration Summary

At this point, you should have:

| Component    | Value                                  |
| ------------ | -------------------------------------- |
| Realm        | `openvpn`                              |
| Client ID    | `openvpn`                              |
| Client Type  | Public (no secret)                     |
| PKCE Method  | S256                                   |
| Redirect URI | `http://vpn.example.com:9000/callback` |
| Role         | `vpn-user`                             |
| Test User    | `testuser` with role `vpn-user`        |

---

## Testing

### Test 1: OIDC Discovery

```bash
# Should return JSON configuration
curl -s http://your-keycloak-server:8080/realms/openvpn/.well-known/openid-configuration | jq .issuer
```

Expected: `"http://your-keycloak-server:8080/realms/openvpn"`

### Test 2: Authorization Endpoint

Navigate to this URL in a browser (replace values):

```
http://your-keycloak-server:8080/realms/openvpn/protocol/openid-connect/auth?client_id=openvpn&redirect_uri=http://vpn.example.com:9000/callback&response_type=code&scope=openid
```

You should see:

1. Keycloak login page
2. After login, redirect to your callback URL (may fail if VPN daemon not running, but redirect should happen)

### Test 3: User Login

Try logging in with your test user credentials to verify they work.

---

## Next Steps

After Keycloak is configured:

1. **Configure the daemon**: Edit `/etc/openvpn/keycloak-sso.yaml`:
   ```yaml
   oidc:
     issuer: "http://your-keycloak-server:8080/realms/openvpn"
     client_id: "openvpn"
     redirect_uri: "http://vpn.example.com:9000/callback"
     required_roles:
       - vpn-user
     role_claim: "realm_access.roles"
   ```

2. **Start the daemon**: `sudo systemctl start openvpn-keycloak-auth`

3. **Configure OpenVPN server**: See [deployment.md](deployment.md#openvpn-server-setup)

4. **Test end-to-end**: Connect a VPN client

---

## Security Recommendations

### Production Checklist

- [ ] Use HTTPS for all Keycloak URLs (not HTTP)
- [ ] Configure proper TLS certificates for Keycloak
- [ ] Use a database (PostgreSQL/MySQL) instead of H2
- [ ] Enable HTTPS for redirect URIs
- [ ] Limit **Web origins** to specific domains (not `*`)
- [ ] Configure token lifespans appropriately
- [ ] Enable MFA/2FA in Keycloak
- [ ] Regular security updates for Keycloak
- [ ] Monitor Keycloak logs for suspicious activity
- [ ] Use strong admin passwords
- [ ] Limit admin access to specific IPs

### Role-Based Access Control

Consider creating multiple roles for different access levels:

```yaml
# In keycloak-sso.yaml
oidc:
  required_roles:
    - vpn-user
    - vpn-admin
    - vpn-contractor
```

User needs **at least one** of these roles to access VPN.

---

## Additional Resources

- **Keycloak Documentation**: https://www.keycloak.org/documentation.html
- **OIDC Specification**: https://openid.net/specs/openid-connect-core-1_0.html
- **PKCE RFC 7636**: https://datatracker.ietf.org/doc/html/rfc7636
- **OpenVPN Deferred Auth**: See OpenVPN 2.6 release notes

---

## Support

If you encounter issues not covered here:

1. Check the troubleshooting sections below
2. Review daemon logs: `journalctl -u openvpn-keycloak-auth -f`
3. Review Keycloak logs
4. Open an issue on GitHub with detailed logs

---

_Last updated: 2026-02-15 for Keycloak 25.0.6_

---

# Troubleshooting

Common issues and solutions when using Keycloak with OpenVPN SSO.

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
     issuer: "http://keycloak.example.com:8080/realms/openvpn" # HTTP not HTTPS
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
ERROR token exchange failed error_category=token_exchange_failed
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

Decode the ID token to see what identity claims it contains:

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
ERROR token validation failed error_category=required_role_missing
```

**Diagnosis:**

The user's token doesn't contain the required VPN role.

**Steps to Verify:**

1. **Check user has role assigned**:
   - **Users** → Select user → **Role mapping** tab
   - Should see `vpn-user` in **Assigned roles**

2. **Check role exists**:
   - **Realm roles** → Look for `vpn-user`

3. **Check role is in token claims**:
   - Decode the ID token and, if roles are absent there, decode the access token from the same token response
   - Look for the configured `role_claim` path such as `realm_access.roles`
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
     role_claim: "realm_access.roles" # For realm roles
     # role_claim: "resource_access.openvpn.roles"  # For client roles
   ```

3. **Verify roles scope assigned**:
   - **Clients** → `openvpn` → **Client scopes** tab
   - Ensure `roles` is in **Assigned default client scopes**

### Roles Not Appearing in Token Claims

**Symptom:**
Neither the ID token nor the verified access token contains `realm_access.roles`.

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
       - roles # Add this
   ```

---

## PKCE Issues

### "PKCE verification failed"

**Symptom:**

```
ERROR token exchange failed error_category=token_exchange_failed
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

| Authorization Request                   | Keycloak Config                        | Match?                 |
| --------------------------------------- | -------------------------------------- | ---------------------- |
| `http://vpn.example.com:9000/callback`  | `http://vpn.example.com:9000/callback` | ✅ Yes                 |
| `https://vpn.example.com:9000/callback` | `http://vpn.example.com:9000/callback` | ❌ No (HTTP vs HTTPS)  |
| `http://vpn.example.com:9000/callback`  | `http://vpn.example.com/callback`      | ❌ No (missing port)   |
| `http://10.0.0.1:9000/callback`         | `http://vpn.example.com:9000/callback` | ❌ No (hostname vs IP) |

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

- Monitor Keycloak JWKS endpoint latency
- Monitor network latency to Keycloak

**Solutions:**

1. **Use local Keycloak**:
   - Deploy Keycloak close to VPN server
   - Reduce network latency

2. **Check Keycloak performance**:
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

| Error                               | Cause                   | Solution                                  |
| ----------------------------------- | ----------------------- | ----------------------------------------- |
| `failed to create OIDC provider`    | Can't reach Keycloak    | Check network, DNS, firewall              |
| `invalid redirect URI`              | URI mismatch            | Update Keycloak Valid redirect URIs       |
| `PKCE verification failed`          | PKCE not configured     | Enable S256 in Keycloak Advanced settings |
| `user does not have required roles` | Missing role assignment | Assign vpn-user role to user              |
| `username claim not found`          | Wrong claim path        | Check username_claim in config            |
| `session not found or expired`      | Session timeout         | Increase session_timeout in config        |
| `token exchange failed`             | Various token issues    | Check client configuration, scopes        |

---

## Getting More Help

1. **Enable debug logging**:
   ```yaml
   # keycloak-sso.yaml
   log:
     level: debug
     format: text # More readable than JSON for debugging
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

_Last updated: 2026-02-15 for Keycloak 25.0.6_
