# Quick Start Guide - 5 Minutes to SSO VPN

Get OpenVPN with Keycloak SSO authentication running in 5 minutes.

## What You Need

- **Server:** Rocky Linux 9 with root access
- **Keycloak:** Running instance with admin access
- **Time:** 5 minutes

## Step 1: Install OpenVPN (1 minute)

```bash
sudo dnf install -y epel-release
sudo dnf install -y openvpn

# Verify version (must be 2.6+)
openvpn --version | head -1
# Should show: OpenVPN 2.6.x or later
```

## Step 2: Build and Install SSO Daemon (1 minute)

```bash
# Clone and build
git clone https://github.com/al-bashkir/openvpn-keycloak
cd openvpn-keycloak
make build

# Install
sudo make install

# You'll see:
# ✓ Binary installed
# ✓ Service installed
# ⚠  IMPORTANT: Edit /etc/openvpn/keycloak-sso.yaml
```

## Step 3: Configure Keycloak (2 minutes)

### Create Realm

1. Keycloak Admin → **Create Realm**
2. Name: `vpn`
3. Click **Create**

### Create Client

1. Clients → **Create client**
2. Settings:
   - Client ID: `openvpn`
   - Client type: **Public**
   - Click **Next**
3. Capability config:
   - Standard flow: **ON**
   - Direct access grants: **OFF**
   - Click **Next**
4. Login settings:
   - Valid redirect URIs: `https://vpn.example.com:9000/callback`
   - Web origins: `https://vpn.example.com`
   - Click **Save**
5. Advanced Settings → **Advanced**
   - Proof Key for Code Exchange Code Challenge Method: **S256**
   - Click **Save**

### Create Test User

1. Users → **Create new user**
2. Username: `testuser`
3. Click **Create**
4. Credentials tab → **Set password**
5. Password: Choose a password
6. Temporary: **OFF**
7. Click **Save**

## Step 4: Configure Daemon (30 seconds)

```bash
# Edit configuration
sudo vi /etc/openvpn/keycloak-sso.yaml

# Update these values:
# keycloak:
#   issuer_url: "https://keycloak.example.com/realms/vpn"
#   client_id: "openvpn"
# 
# http:
#   callback_url: "https://vpn.example.com:9000/callback"

# Validate configuration
sudo /usr/local/bin/openvpn-keycloak-sso check-config --config /etc/openvpn/keycloak-sso.yaml

# Start daemon
sudo systemctl enable --now openvpn-keycloak-sso
sudo systemctl status openvpn-keycloak-sso
# Should show: active (running)
```

## Step 5: Configure OpenVPN Server (30 seconds)

```bash
# Copy example config
sudo cp config/openvpn-server.conf.example /etc/openvpn/server/server.conf

# Edit to add SSO directives
sudo vi /etc/openvpn/server/server.conf

# Essential SSO directives (add these):
script-security 3
auth-user-pass-verify /etc/openvpn/auth-keycloak.sh via-file
auth-user-pass-optional
auth-gen-token 0 external-auth
hand-window 120
```

**Or use our complete example:**

```bash
sudo cp config/openvpn-server.conf.example /etc/openvpn/server/server.conf
# Edit VPN network settings as needed

# Generate certificates (if not done)
# See docs/openvpn-server-setup.md for Easy-RSA instructions
```

## Step 6: Start OpenVPN Server (10 seconds)

```bash
sudo systemctl enable --now openvpn-server@server
sudo systemctl status openvpn-server@server
# Should show: active (running)
```

## Step 7: Test Connection (30 seconds)

### Generate Client Profile

```bash
# Generate client config with embedded CA certificate
./scripts/generate-client-profile.sh \
  /etc/openvpn/server/ca.crt \
  vpn.example.com \
  client.ovpn
  
# Copy client.ovpn to your client machine
```

### Connect from Client

**Linux CLI:**
```bash
openvpn --config client.ovpn
# Username: testuser
# Password: sso (any value, ignored)
# Copy the WEB_AUTH:: URL and open in browser
# Log in to Keycloak
# VPN connects!
```

**Windows/macOS (OpenVPN Connect):**
```
1. Import client.ovpn
2. Click Connect
3. Username: testuser, Password: sso
4. Browser opens automatically
5. Log in to Keycloak
6. VPN connects!
```

**macOS (Tunnelblick):**
```
1. Double-click client.ovpn
2. Connect
3. Username: testuser, Password: sso
4. Safari opens automatically
5. Log in to Keycloak
6. VPN connects!
```

## Success! 🎉

You now have OpenVPN with SSO authentication!

**Verify connection:**
```bash
# Check VPN interface
ip addr show tun0

# Ping VPN gateway
ping 10.8.0.1

# Check logs
sudo journalctl -u openvpn-keycloak-sso -f
```

## What's Next?

### Production Deployment

1. **Enable HTTPS** for callback endpoint
   - Use reverse proxy (nginx, Apache)
   - See `docs/deployment.md`

2. **Enable MFA** in Keycloak
   - Realm → Authentication → Required Actions
   - Enable "Configure OTP"

3. **Configure Roles** for access control
   - See `docs/keycloak-setup.md`

4. **Set up monitoring**
   - Check logs regularly
   - Monitor authentication failures

### Troubleshooting

If something doesn't work:

1. **Check daemon logs:**
   ```bash
   sudo journalctl -u openvpn-keycloak-sso -n 50
   ```

2. **Check OpenVPN logs:**
   ```bash
   sudo journalctl -u openvpn-server@server -n 50
   ```

3. **Common issues:**
   - Firewall blocking port 9000: `sudo firewall-cmd --add-port=9000/tcp --permanent && sudo firewall-cmd --reload`
   - Wrong Keycloak URL: Check `issuer_url` in config
   - Certificate issues: Verify `ca.crt` is correct

4. **Get help:**
   - See `docs/troubleshooting.md`
   - Check GitHub issues

## Documentation

- **Full Setup:** `docs/deployment.md`
- **Keycloak Config:** `docs/keycloak-setup.md`
- **OpenVPN Config:** `docs/openvpn-server-setup.md`
- **Client Setup:** `docs/client-setup.md`
- **Security:** `docs/security.md`
- **Testing:** `docs/testing.md`

## Need More Time?

Not quite 5 minutes? That's okay! The detailed guides will walk you through each step:

- **20-minute setup:** `docs/deployment.md` - Step-by-step with explanations
- **Full deployment:** `docs/openvpn-server-setup.md` - Complete PKI setup, certificates, etc.

---

**Happy Secure VPN-ing! 🔒**
