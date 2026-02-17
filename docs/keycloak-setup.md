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
- [Troubleshooting](#troubleshooting)

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

| Component | Value |
|-----------|-------|
| Realm | `openvpn` |
| Client ID | `openvpn` |
| Client Type | Public (no secret) |
| PKCE Method | S256 |
| Redirect URI | `http://vpn.example.com:9000/callback` |
| Role | `vpn-user` |
| Test User | `testuser` with role `vpn-user` |

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

## Troubleshooting

### Issue: "Invalid redirect URI"

**Symptom**: Error message after OIDC login

**Cause**: Mismatch between:
- Redirect URI in Keycloak client configuration
- Redirect URI in `keycloak-sso.yaml`
- Redirect URI in authorization request

**Solution**:
1. Go to **Clients** → `openvpn` → **Settings**
2. Check **Valid redirect URIs**
3. Ensure exact match: `http://vpn.example.com:9000/callback`
4. Click **"Save"**

### Issue: "Client not found"

**Symptom**: Error when starting daemon or during auth

**Cause**: Client ID mismatch

**Solution**:
- Verify client ID in Keycloak is exactly `openvpn`
- Verify `client_id: openvpn` in `keycloak-sso.yaml`
- Client IDs are case-sensitive

### Issue: "PKCE verification failed"

**Symptom**: Token exchange fails

**Cause**: PKCE not configured or wrong method

**Solution**:
1. Go to **Clients** → `openvpn` → **Advanced** tab
2. Set **Proof Key for Code Exchange Code Challenge Method**: `S256`
3. Click **"Save"**
4. Restart the daemon

### Issue: User can't access VPN despite logging in

**Symptom**: Authentication succeeds but token validation fails

**Cause**: User doesn't have required role

**Solution**:
1. Go to **Users** → Select user → **Role mapping** tab
2. Click **"Assign role"**
3. Assign `vpn-user` role
4. User should retry authentication

### Issue: Token doesn't contain roles

**Symptom**: "User does not have required roles" error

**Cause**: Roles scope not assigned or roles not in token

**Solution**:
1. Go to **Clients** → `openvpn` → **Client scopes** tab
2. Ensure `roles` is in **Assigned default client scopes**
3. Verify role claim path in `keycloak-sso.yaml`: `role_claim: "realm_access.roles"`

### Issue: Keycloak unreachable from VPN server

**Symptom**: Daemon fails to start with "failed to create OIDC provider"

**Cause**: Network connectivity issues

**Solution**:
1. Test from VPN server:
   ```bash
   curl -v http://your-keycloak-server:8080/realms/openvpn/.well-known/openid-configuration
   ```
2. Check firewall rules
3. Verify DNS resolution
4. Check Keycloak service status

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

2. **Start the daemon**: `sudo systemctl start openvpn-keycloak-sso`

3. **Configure OpenVPN server**: See [OpenVPN Server Setup](openvpn-server-setup.md)

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

1. Check the [Troubleshooting Guide](troubleshooting.md)
2. Review daemon logs: `journalctl -u openvpn-keycloak-sso -f`
3. Review Keycloak logs
4. Open an issue on GitHub with detailed logs

---

*Last updated: 2026-02-15 for Keycloak 25.0.6*
