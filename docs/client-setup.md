# OpenVPN Client Setup for SSO Authentication

This guide covers installation, configuration, and usage of OpenVPN clients with SSO (Single Sign-On) authentication via Keycloak.

**Important:** SSO authentication requires OpenVPN 2.6+ with `WEB_AUTH::` support. Not all clients support this feature yet.

## Table of Contents

1. [Overview](#overview)
2. [Client Compatibility](#client-compatibility)
3. [Obtaining Your Client Profile](#obtaining-your-client-profile)
4. [Platform-Specific Setup](#platform-specific-setup)
   - [Linux CLI](#linux-cli-openvpn-command-line)
   - [Linux GUI (NetworkManager)](#linux-gui-networkmanager)
   - [macOS (Tunnelblick)](#macos-tunnelblick)
   - [Windows (OpenVPN Connect)](#windows-openvpn-connect)
   - [iOS (OpenVPN Connect)](#ios-openvpn-connect)
   - [Android (OpenVPN Connect)](#android-openvpn-connect)
5. [First Connection](#first-connection)
6. [SSO Authentication Flow](#sso-authentication-flow)
7. [Troubleshooting](#troubleshooting)
8. [Security Best Practices](#security-best-practices)

---

## Overview

This OpenVPN setup uses **SSO authentication** instead of traditional username/password authentication. Here's how it works:

1. **Connect to VPN** - Your client initiates the connection
2. **Enter username** - Provide your Keycloak username (password is ignored)
3. **Browser opens** - Authenticate via your organization's Keycloak login page
4. **Complete MFA** - Complete any multi-factor authentication if required
5. **VPN connects** - Connection completes automatically after successful authentication

**Benefits:**
- **Single Sign-On** - Use your organization's identity provider
- **Stronger security** - Supports MFA, passwordless auth, hardware keys
- **No password sharing** - VPN server never sees your password
- **Centralized management** - IT controls access via Keycloak

---

## Client Compatibility

SSO support varies by client and version:

| Client | Platform | SSO Support | Notes |
|--------|----------|-------------|-------|
| **OpenVPN Connect 3.x** | Windows, macOS, iOS, Android, Linux | ✅ Excellent | Built-in webview, best experience |
| **Tunnelblick 3.8.7+** | macOS | ✅ Excellent | Opens Safari automatically |
| **OpenVPN CLI 2.6+** | Linux, Unix, macOS | ⚠️ Manual | Displays URL to copy/paste |
| **NetworkManager** | Linux (GNOME, KDE) | ⚠️ Limited | May require manual browser opening |
| **OpenVPN GUI 2.6+** | Windows | ⚠️ Manual | Displays URL to copy/paste |
| **Viscosity** | macOS, Windows | ❌ Not tested | May work with manual URL opening |
| **OpenVPN Connect 2.x** | Legacy | ❌ No support | Upgrade to 3.x required |

**Legend:**
- ✅ **Excellent** - Opens browser automatically, seamless experience
- ⚠️ **Manual** - Requires copying URL and opening manually
- ❌ **No support** - Does not work with SSO

**Recommendation:** For the best experience, use **OpenVPN Connect 3.x** on all platforms.

---

## Obtaining Your Client Profile

Your VPN administrator will provide you with a `.ovpn` client profile file. This file contains:

- Server address and port
- CA certificate for server verification
- Security settings (ciphers, TLS version)
- Performance tuning parameters

**Delivery methods:**
- **Email attachment** - Most common, check your inbox
- **Download portal** - Internal website or file share
- **QR code** - For mobile devices (if available)
- **IT help desk** - Request via your organization's support channels

**File naming:**
- `client.ovpn` - Universal profile (works on most clients)
- `client-cli.ovpn` - Linux command-line optimized
- `client-tunnelblick.ovpn` - macOS Tunnelblick optimized
- `client-connect.ovpn` - OpenVPN Connect optimized

**Security note:** Treat your `.ovpn` file carefully:
- Contains your organization's CA certificate
- May contain client certificate/key (if using mutual TLS)
- Don't share with unauthorized users
- Store securely (encrypted disk, password manager)

---

## Platform-Specific Setup

### Linux CLI (OpenVPN Command Line)

**Best for:** Servers, headless systems, power users, automation

**Installation:**

```bash
# Fedora/RHEL/Rocky Linux
sudo dnf install epel-release
sudo dnf install openvpn

# Debian/Ubuntu
sudo apt update
sudo apt install openvpn

# Arch Linux
sudo pacman -S openvpn

# Verify version (need 2.6+)
openvpn --version | head -1
```

**Profile Setup:**

```bash
# Create client config directory
sudo mkdir -p /etc/openvpn/client

# Copy CA certificate (if using external ca= directive)
sudo cp ca.crt /etc/openvpn/client/ca.crt
sudo chmod 644 /etc/openvpn/client/ca.crt

# Copy client profile
sudo cp client-cli.ovpn /etc/openvpn/client/vpn-sso.conf
sudo chmod 644 /etc/openvpn/client/vpn-sso.conf

# Create credentials file (optional, skips username prompt)
cat > ~/.openvpn-creds <<EOF
your-keycloak-username
sso
EOF
chmod 600 ~/.openvpn-creds
```

**Connect (Interactive):**

```bash
# Start OpenVPN (requires root/sudo)
sudo openvpn --config /etc/openvpn/client/vpn-sso.conf

# When prompted:
Enter Auth Username: your-keycloak-username
Enter Auth Password: sso

# Look for this line in the output:
AUTH_PENDING,timeout:300,openurl,WEB_AUTH::https://keycloak.example.com/...

# Copy the URL (everything after WEB_AUTH::)
# Open it in your browser:
firefox "https://keycloak.example.com/realms/myrealm/protocol/openid-connect/auth?..."

# Log in to Keycloak in the browser
# Return to terminal - connection completes automatically
# Look for: "Initialization Sequence Completed"
```

**Connect (with Credentials File):**

```bash
# No username prompt - just open the browser URL
sudo openvpn --config /etc/openvpn/client/vpn-sso.conf \
             --auth-user-pass ~/.openvpn-creds
```

**Run as systemd Service:**

```bash
# Copy config to systemd location
sudo cp /etc/openvpn/client/vpn-sso.conf /etc/openvpn/client/vpn-sso.conf

# Enable and start service
sudo systemctl enable openvpn-client@vpn-sso
sudo systemctl start openvpn-client@vpn-sso

# Check status
sudo systemctl status openvpn-client@vpn-sso

# View logs
sudo journalctl -u openvpn-client@vpn-sso -f

# Note: For SSO, you'll need to open the WEB_AUTH:: URL manually
# Check logs for the URL when connecting
```

**Disconnect:**

```bash
# If running interactively: Ctrl+C

# If running as systemd service:
sudo systemctl stop openvpn-client@vpn-sso
```

---

### Linux GUI (NetworkManager)

**Best for:** Desktop Linux users (GNOME, KDE, Cinnamon)

**Installation:**

```bash
# Fedora/RHEL/Rocky Linux
sudo dnf install NetworkManager-openvpn-gnome

# Debian/Ubuntu
sudo apt install network-manager-openvpn-gnome

# Arch Linux
sudo pacman -S networkmanager-openvpn

# Restart NetworkManager
sudo systemctl restart NetworkManager
```

**Import Profile (Command Line):**

```bash
# Import .ovpn file
sudo nmcli connection import type openvpn file client.ovpn

# The connection will appear in NetworkManager GUI
```

**Import Profile (GUI - GNOME):**

1. Open **Settings** → **Network**
2. Click **VPN** → **+** button
3. Choose **Import from file...**
4. Select your `client.ovpn` file
5. Connection appears in the VPN list

**Configure for SSO:**

```bash
# Edit connection via CLI
nmcli connection edit vpn-sso

# In the nmcli editor:
nmcli> set vpn.data auth-retry=interact
nmcli> set vpn.user-name your-keycloak-username
nmcli> save
nmcli> quit
```

Or via GUI:
1. Click gear icon next to VPN connection
2. **Identity** tab:
   - Username: `your-keycloak-username`
   - Password: (leave empty or type `sso`)
3. Click **Advanced** button
4. **Authentication** tab:
   - Set `auth-retry` to `interact`
5. Click **Apply**

**Connect:**

```bash
# Command line
nmcli connection up vpn-sso

# GUI (GNOME)
Settings → Network → VPN → Toggle switch ON

# GUI (KDE)
System Tray → Networks → VPN → Click connection
```

**SSO Authentication:**

⚠️ **Important:** NetworkManager's SSO support is limited. You may need to:

1. Start the connection
2. Check logs for WEB_AUTH:: URL:
   ```bash
   journalctl -f -u NetworkManager | grep WEB_AUTH
   ```
3. Copy the URL and open manually in your browser
4. Log in to Keycloak
5. Connection completes automatically

**Note:** For better SSO experience on Linux desktop, consider using **OpenVPN Connect** instead of NetworkManager.

---

### macOS (Tunnelblick)

**Best for:** macOS users, excellent SSO support

**Installation:**

1. Download Tunnelblick from https://tunnelblick.net/
2. Open the downloaded `.dmg` file
3. Drag Tunnelblick to Applications folder
4. Launch Tunnelblick
5. Grant permissions when prompted (requires admin password)

**Version Check:**

- **Minimum version:** 3.8.7 (for WEB_AUTH:: support)
- **Recommended:** Latest stable version
- Check version: Tunnelblick icon → About Tunnelblick

**Import Profile:**

**Method 1: Double-click**
1. Locate your `.ovpn` file in Finder
2. Double-click the file
3. Tunnelblick opens and prompts: "Install configuration"
4. Choose **Only Me** (recommended) or **All Users**
5. Enter your macOS password if prompted
6. Configuration installed successfully

**Method 2: Drag-and-drop**
1. Launch Tunnelblick
2. Drag `.ovpn` file onto Tunnelblick icon in menu bar
3. Follow prompts to install

**Method 3: Tunnelblick Menu**
1. Click Tunnelblick icon in menu bar
2. **VPN Details...** → **Configurations** tab
3. Click **+** (bottom left) → **Add a VPN Configuration**
4. Select your `.ovpn` file
5. Choose installation type (Only Me / All Users)

**Connect:**

1. Click **Tunnelblick icon** in menu bar
2. Hover over your VPN configuration name
3. Click **Connect**
4. Credential prompt appears:
   - **Username:** `your-keycloak-username`
   - **Password:** `sso` (or anything - will be ignored)
   - **Save in Keychain:** ☑️ (optional, saves username only)
5. Click **OK**

**SSO Authentication (Automatic):**

1. **Safari opens automatically** with Keycloak login page
2. Log in with your Keycloak credentials
3. Complete any MFA challenges (TOTP, WebAuthn, etc.)
4. Success page appears in Safari
5. **Close Safari tab** - you can close the browser window
6. Return to Tunnelblick - **connection completes automatically**
7. Menu bar icon turns **solid** when connected

**Connection Status:**

- **Solid icon** - Connected
- **Hollow icon** - Disconnected
- **Animated icon** - Connecting

**Disconnect:**

- Click Tunnelblick icon → **Disconnect**
- Or: Click icon → Hover over VPN → **Disconnect**

**View Logs:**

- Tunnelblick icon → **VPN Details...**
- Select your configuration
- Click **Log** tab
- Useful for troubleshooting

**Advanced Settings:**

Tunnelblick icon → VPN Details → Select configuration → Settings:

- **Connect automatically:** When computer starts
- **Set DNS/WINS:** Set nameserver (prevents DNS leaks)
- **Monitor network settings:** Reconnect on network changes
- **Route all traffic:** Force all traffic through VPN

---

### Windows (OpenVPN Connect)

**Best for:** Windows users, modern SSO support

**Installation:**

1. Download from https://openvpn.net/client/
2. Run the installer (`OpenVPN-Connect-Setup.msi`)
3. Follow installation wizard
4. Launch **OpenVPN Connect** from Start menu

**System Requirements:**
- Windows 10/11 (64-bit)
- Administrator privileges for first connection
- .NET Framework 4.8+

**Import Profile:**

**Method 1: File → Import**
1. Launch OpenVPN Connect
2. Click **File** → **Import Profile** → **From Local File**
3. Browse and select your `.ovpn` file
4. Profile appears in the main window

**Method 2: Drag-and-drop**
1. Open File Explorer, locate `.ovpn` file
2. Drag file onto OpenVPN Connect window
3. Profile imports automatically

**Connect:**

1. In OpenVPN Connect main window, find your profile
2. Click the **toggle switch** to connect
3. Or click profile name → **Connect** button
4. Credential prompt appears:
   - **Username:** `your-keycloak-username`
   - **Password:** `sso` (or anything - will be ignored)
   - **Save:** ☑️ (optional, saves username only)
5. Click **OK**

**SSO Authentication:**

1. **Built-in browser opens** with Keycloak login page
2. Log in with your Keycloak credentials
3. Complete MFA if required
4. Success page displays
5. Browser closes automatically
6. VPN connection completes
7. Status changes to **Connected**

**Connection Status:**

- Green checkmark - Connected
- Gray - Disconnected
- Spinning - Connecting
- System tray icon also shows status

**Disconnect:**

- Click toggle switch to disconnect
- Or: Right-click system tray icon → Disconnect

**View Logs:**

- Click profile → **Settings** (gear icon) → **Log**
- Useful for troubleshooting

**Advanced Features:**

Settings (gear icon):
- **Launch on startup:** Auto-start OpenVPN Connect
- **Connect on launch:** Auto-connect to VPN
- **Reconnect on network change:** Yes (recommended)

---

### iOS (OpenVPN Connect)

**Best for:** iPhone and iPad users

**Installation:**

1. Open **App Store**
2. Search for **"OpenVPN Connect"**
3. Download and install (it's free)
4. Open the app

**Import Profile:**

**Method 1: Email**
1. Email the `.ovpn` file to yourself
2. Open email on your iPhone/iPad
3. Tap the `.ovpn` attachment
4. Tap **Share** → **Copy to OpenVPN**
5. OpenVPN Connect opens with import screen
6. Tap **Add**

**Method 2: Cloud Storage**
1. Upload `.ovpn` to iCloud Drive, Dropbox, or Google Drive
2. Open file in Files app or cloud storage app
3. Tap file → **Share** → **OpenVPN**
4. Tap **Add** in OpenVPN Connect

**Method 3: AirDrop**
1. On Mac: Right-click `.ovpn` → Share → AirDrop
2. Select your iPhone/iPad
3. File transfers to device
4. Tap notification → **Open with OpenVPN**

**Connect:**

1. Open **OpenVPN Connect** app
2. Tap your profile
3. Tap **Connect** button
4. Enter credentials:
   - **Username:** `your-keycloak-username`
   - **Password:** `sso`
5. Tap **OK**
6. iOS VPN permission prompt appears
7. Tap **Allow** (first time only)

**SSO Authentication:**

1. **Built-in browser opens** within the app
2. Keycloak login page loads
3. Log in with your credentials
4. Complete Touch ID / Face ID if enabled
5. Complete MFA if required
6. Success page appears
7. Browser **closes automatically**
8. VPN status changes to **Connected**
9. **VPN icon** appears in status bar

**Connection Status:**

- **Connected:** Green checkmark, VPN icon in status bar
- **Connecting:** Spinner animation
- **Disconnected:** Gray

**Disconnect:**

- Open OpenVPN Connect → Tap toggle to disconnect
- Or: Settings → VPN → Toggle off

**Always-On VPN:**

Settings → VPN → OpenVPN → **Connect On Demand**

Configure rules:
- Connect when: Wi-Fi, Cellular, or Both
- Disconnect when: Never (stay connected)

**View Logs:**

- Tap profile → **Log** tab
- Shows connection events and errors

**Troubleshooting Tips:**

- If browser doesn't open, update OpenVPN Connect to latest version
- For Touch ID/Face ID, enable in iOS Settings → OpenVPN
- Check network connectivity (Wi-Fi or Cellular)

---

### Android (OpenVPN Connect)

**Best for:** Android phone and tablet users

**Installation:**

1. Open **Google Play Store**
2. Search for **"OpenVPN Connect"**
3. Install (by OpenVPN Inc.)
4. Open the app
5. Grant permissions when prompted

**Import Profile:**

**Method 1: Email**
1. Email `.ovpn` file to yourself
2. Open email on Android device
3. Tap the attachment
4. Choose **OpenVPN** from app list
5. Tap **Import** → **Add**

**Method 2: File Manager**
1. Download or copy `.ovpn` to device
2. Open Files app / File Manager
3. Navigate to `.ovpn` file
4. Tap file → **Open with** → **OpenVPN**
5. Tap **Import** → **Add**

**Method 3: Cloud Storage**
1. Upload to Google Drive, Dropbox, etc.
2. Open file in cloud app
3. Tap **Open in** → **OpenVPN**
4. Tap **Import** → **Add**

**Connect:**

1. Open **OpenVPN Connect**
2. Tap your profile
3. Tap **Connect** button
4. Enter credentials:
   - **Username:** `your-keycloak-username`
   - **Password:** `sso`
5. Tap **OK**
6. Android VPN permission prompt (first time)
7. Tap **OK** to allow VPN connection

**SSO Authentication:**

1. **WebView opens** within the app
2. Keycloak login page loads
3. Enter your Keycloak credentials
4. Complete MFA if required
5. Success page displays
6. WebView **closes automatically**
7. VPN connects
8. **Key icon** appears in notification bar

**Connection Status:**

- **Connected:** Key icon in notification bar, timer showing duration
- **Connecting:** Spinner
- **Disconnected:** No key icon

**Disconnect:**

- Swipe down notification bar → Tap VPN notification → Disconnect
- Or: Open OpenVPN Connect → Tap profile → Disconnect

**Always-On VPN:**

Settings → Network & Internet → VPN → ⚙️ (gear icon) → OpenVPN:
- **Always-on VPN:** Toggle ON
- **Block connections without VPN:** Toggle ON (optional, more secure)

**Battery Optimization:**

For reliable VPN:
1. Settings → Apps → OpenVPN Connect
2. Battery → **Unrestricted**
3. Prevents Android from killing VPN in background

**View Logs:**

- Tap profile → **Log** tab
- Shows connection events and errors

---

## First Connection

### What to Expect

**First time connecting:**

1. **Client prompts for credentials**
   - Username: Your Keycloak username (e.g., `john.doe`)
   - Password: Anything (e.g., `sso`) - will be ignored by SSO

2. **Browser opens** (automatic on modern clients)
   - OpenVPN Connect, Tunnelblick: Opens automatically
   - CLI clients: Displays URL to copy/paste manually

3. **Keycloak login page appears**
   - Your organization's login page
   - May have custom branding/logo
   - URL should match your organization's Keycloak server

4. **Authenticate**
   - Enter your Keycloak password
   - Complete MFA if enabled (TOTP, SMS, WebAuthn, etc.)
   - May require accepting terms of service

5. **Success page**
   - "Authentication successful" message
   - Safe to close browser tab
   - Return to VPN client

6. **VPN connects**
   - Connection completes automatically
   - Status changes to "Connected"
   - You're now on the corporate network

### Typical Timeline

- **Connection initiation:** Immediate
- **Browser opening:** 1-5 seconds
- **Keycloak login:** User-dependent (30-60 seconds typical)
- **Token validation:** 1-3 seconds
- **VPN connection:** 2-5 seconds
- **Total:** 30-90 seconds (mostly depends on how fast you log in)

### Subsequent Connections

**Saved credentials:**
- Most clients can save your **username** (not password)
- Next time: just click connect, browser opens automatically
- No need to re-enter username

**Token lifetime:**
- Keycloak issues tokens with expiration (typically 5-60 minutes)
- You'll need to re-authenticate when token expires
- Some clients handle this automatically with refresh tokens

---

## SSO Authentication Flow

### How SSO Works Behind the Scenes

```
┌──────────┐          ┌──────────┐          ┌──────────┐          ┌──────────┐
│  Client  │          │   VPN    │          │  Daemon  │          │ Keycloak │
│          │          │  Server  │          │          │          │          │
└─────┬────┘          └─────┬────┘          └─────┬────┘          └─────┬────┘
      │                     │                     │                     │
      │  1. Connect         │                     │                     │
      ├────────────────────>│                     │                     │
      │                     │                     │                     │
      │  2. Auth Deferred   │                     │                     │
      │  WEB_AUTH:: URL     │                     │                     │
      │<────────────────────┤                     │                     │
      │                     │                     │                     │
      │  3. Open Browser    │                     │                     │
      │────────────────────────────────────────────────────────────────>│
      │                     │                     │                     │
      │  4. User Login      │                     │                     │
      │<───────────────────────────────────────────────────────────────>│
      │                     │                     │                     │
      │  5. Callback        │                     │                     │
      │────────────────────────────────────────>  │                     │
      │                     │                     │                     │
      │                     │                     │  6. Validate Token  │
      │                     │                     ├────────────────────>│
      │                     │                     │                     │
      │                     │                     │  7. Token Valid     │
      │                     │                     │<────────────────────┤
      │                     │                     │                     │
      │                     │  8. Write Auth OK   │                     │
      │                     │<────────────────────┤                     │
      │                     │                     │                     │
      │  9. Connected       │                     │                     │
      │<────────────────────┤                     │                     │
      │                     │                     │                     │
```

### Step-by-Step Technical Flow

1. **Client initiates connection**
   - Sends username (Keycloak username) and password (ignored)
   - OpenVPN server calls auth script

2. **Auth script defers authentication**
   - Generates session ID and PKCE verifier/challenge
   - Sends auth request to daemon via Unix socket
   - Returns exit code 2 (deferred)

3. **Daemon generates authorization URL**
   - Creates OIDC authorization URL with PKCE challenge
   - Includes state parameter (session ID) for CSRF protection
   - Writes URL to `auth_pending_file`

4. **Server sends AUTH_PENDING to client**
   - Client receives WEB_AUTH:: URL
   - Modern clients open browser automatically
   - CLI clients display URL for manual opening

5. **User authenticates in browser**
   - Browser loads Keycloak login page
   - User enters Keycloak password
   - Completes MFA if required
   - Keycloak validates credentials

6. **Keycloak redirects to callback**
   - Browser redirects to daemon's HTTP callback endpoint
   - URL includes authorization code and state

7. **Daemon exchanges code for token**
   - Validates state matches session ID (CSRF protection)
   - Exchanges authorization code + PKCE verifier for token
   - Validates token signature via JWKS
   - Extracts username from token claims

8. **Daemon validates token claims**
   - Checks issuer (iss) matches Keycloak URL
   - Checks audience (aud) matches client ID
   - Checks expiration (exp) is in the future
   - Checks issued at (iat) and not before (nbf)
   - Optionally checks group/role membership

9. **Daemon writes auth result**
   - If valid: writes "1" to `auth_control_file`
   - If invalid: writes reason to `auth_failed_reason_file`, then "0" to `auth_control_file`

10. **Server completes connection**
    - OpenVPN server reads auth_control_file
    - If "1": connection succeeds, assigns IP, routes, etc.
    - If "0": connection fails, displays error message

### Security Features

**PKCE (Proof Key for Code Exchange):**
- Prevents authorization code interception attacks
- Client generates random verifier and challenge
- Challenge sent in authorization request
- Verifier sent when exchanging code for token
- Server validates verifier matches challenge

**State Parameter:**
- CSRF (Cross-Site Request Forgery) protection
- Random state value tied to session ID
- Validated on callback to ensure request originated from this client

**Token Validation:**
- JWT signature verified using Keycloak's public keys (JWKS)
- All claims validated (issuer, audience, expiration, etc.)
- Username extracted from preferred_username claim
- Optional: group/role membership enforced

**No Password Transmission:**
- VPN server never sees user's Keycloak password
- Only receives cryptographically signed token
- Reduces attack surface significantly

---

## Troubleshooting

### Browser Doesn't Open

**Symptoms:**
- Click connect, but no browser window appears
- Connection hangs or times out

**Solutions:**

1. **Check client version:**
   - OpenVPN Connect: Update to 3.x or later
   - Tunnelblick: Update to 3.8.7 or later
   - CLI clients: Browser never opens automatically (expected)

2. **Manual browser opening:**
   - Check client logs for `WEB_AUTH::` URL
   - Copy the URL and paste into browser manually
   - Example URL:
     ```
     WEB_AUTH::https://keycloak.example.com/realms/myrealm/protocol/openid-connect/auth?client_id=openvpn&...
     ```

3. **Check default browser:**
   - macOS: Set Safari as default (Tunnelblick uses Safari)
   - Windows/Linux: Ensure you have a default browser set

4. **Firewall/proxy:**
   - Check firewall isn't blocking browser requests
   - If behind proxy, ensure browser can access Keycloak

### Authentication Hangs After Browser Login

**Symptoms:**
- Successfully log in to Keycloak
- Success page appears
- But VPN connection doesn't complete
- Eventually times out

**Solutions:**

1. **Check server version:**
   - OpenVPN server must be 2.6.2 or later
   - Earlier versions have bugs with AUTH_PENDING
   - Verify: `ssh server "openvpn --version"`

2. **Verify daemon is running:**
   ```bash
   # On VPN server
   sudo systemctl status openvpn-keycloak-sso
   ```
   - Should show "active (running)"
   - If not running: `sudo systemctl start openvpn-keycloak-sso`

3. **Check daemon logs:**
   ```bash
   # On VPN server
   sudo journalctl -u openvpn-keycloak-sso -f
   ```
   - Look for callback errors
   - Check for token validation failures

4. **Verify auth-retry setting:**
   - Must be set to "interact" in client config
   - Check your `.ovpn` file for: `auth-retry interact`

5. **Network connectivity:**
   - Ensure daemon can reach Keycloak (outbound HTTPS)
   - Check firewall rules on server

### "Authentication Failed" Error

**Symptoms:**
- Browser login succeeds
- But client shows "Authentication failed"
- Connection rejected

**Solutions:**

1. **Username mismatch:**
   - VPN client username must match Keycloak username exactly
   - Check capitalization (usually case-sensitive)
   - Check for typos or extra spaces

2. **Missing roles/groups:**
   - Check Keycloak configuration requires specific roles
   - Verify you're a member of required groups
   - Contact IT to grant necessary permissions

3. **Token validation failure:**
   - Check server logs: `sudo journalctl -u openvpn-keycloak-sso -n 50`
   - Look for "token validation failed" messages
   - Common causes:
     - Clock skew between server and Keycloak
     - Wrong client ID in daemon config
     - Wrong issuer URL in daemon config

4. **Expired token:**
   - Token lifetime too short
   - Check Keycloak session settings
   - Increase Access Token Lifespan in Keycloak client settings

### Connection Timeout

**Symptoms:**
- Browser doesn't open within timeout period
- Or user doesn't complete login in time
- Connection times out after 5 minutes

**Solutions:**

1. **Complete authentication faster:**
   - Default timeout is 5 minutes
   - Ensure you complete login before timeout

2. **Check timeout configuration:**
   - Server side: Check `hand-window` directive in `openvpn-server.conf`
   - Should be at least 120 seconds
   - Client side: Some clients have separate timeout settings

3. **Network issues:**
   - Check internet connectivity
   - Verify Keycloak server is accessible
   - Try accessing Keycloak URL directly in browser

### Certificate Errors

**Symptoms:**
- "TLS Error: TLS handshake failed"
- "Certificate verify failed"
- "SSL: error"

**Solutions:**

1. **Verify CA certificate:**
   - Ensure CA certificate in `.ovpn` matches server's CA
   - Check certificate hasn't expired
   - Verify certificate is properly formatted (PEM format)

2. **Check server hostname:**
   - `remote` directive in config must match server's certificate CN/SAN
   - If using IP address, ensure certificate includes IP in SAN
   - Or use `remote-cert-tls server` to skip CN check (already in configs)

3. **TLS version mismatch:**
   - Server and client TLS version settings must overlap
   - Check `tls-version-min` in both configs
   - Try removing client-side TLS restrictions temporarily

4. **System time:**
   - Certificate validation requires accurate time
   - Check system clock is correct
   - Sync with NTP: `sudo ntpdate pool.ntp.org`

### Can't Import Profile

**Symptoms:**
- Client rejects `.ovpn` file
- "Invalid configuration" error
- Import fails silently

**Solutions:**

1. **File format:**
   - Ensure file extension is `.ovpn` (not `.conf` or `.txt`)
   - Verify file encoding is UTF-8
   - Check for Windows line endings (CRLF vs LF)

2. **Syntax errors:**
   - Open file in text editor
   - Look for malformed directives
   - Check certificate markers are intact (BEGIN/END)

3. **Platform-specific:**
   - Tunnelblick: Requires `.ovpn` extension
   - NetworkManager: Can import `.ovpn` or `.conf`
   - OpenVPN Connect: Requires `.ovpn`
   - Try renaming file if needed

4. **Inline certificates:**
   - Some clients require certificates inline (`<ca>...</ca>`)
   - Others prefer external files (`ca /path/to/ca.crt`)
   - Check client documentation for preference

### Frequent Disconnections

**Symptoms:**
- VPN connects successfully
- But disconnects frequently (every few minutes)
- Reconnects automatically or requires manual reconnection

**Solutions:**

1. **Network stability:**
   - Check Wi-Fi signal strength
   - Try wired connection
   - Switch from UDP to TCP (more reliable, slightly slower)
     - Change `proto udp` to `proto tcp-client` in `.ovpn`

2. **NAT timeout:**
   - Many routers drop NAT mappings after inactivity
   - Add keepalive directive to client config:
     ```
     keepalive 10 60
     ```
   - Or use `ping` directive (already in server config)

3. **Mobile-specific:**
   - **iOS:** Enable "Connect On Demand" in VPN settings
   - **Android:** Disable battery optimization for OpenVPN Connect
   - **Android:** Enable "Always-on VPN" in system settings

4. **Token expiration:**
   - If disconnecting every N minutes consistently
   - May be token expiration issue
   - Check Keycloak token lifetimes
   - Server should handle token refresh automatically

5. **Firewall/IDS:**
   - Corporate firewall may be dropping long-lived connections
   - Try connecting from different network to isolate issue
   - Contact IT if corporate firewall is the issue

### Logs and Debugging

**Enable verbose logging:**

Edit your `.ovpn` file, change:
```
verb 3
```
To:
```
verb 4
```

Reimport profile and reconnect. Check logs for detailed output.

**View client logs:**

- **Tunnelblick:** Icon → VPN Details → Log tab
- **OpenVPN Connect:** Profile → Settings → Log
- **Linux CLI:** Output in terminal or `/var/log/openvpn/`
- **NetworkManager:** `journalctl -u NetworkManager -f`

**View server logs:**

```bash
# SSH to VPN server
ssh admin@vpn.example.com

# OpenVPN server logs
sudo journalctl -u openvpn@server -f

# SSO daemon logs
sudo journalctl -u openvpn-keycloak-sso -f

# Both together
sudo journalctl -f -u openvpn@server -u openvpn-keycloak-sso
```

**Test OpenVPN connectivity (without SSO):**

```bash
# Test UDP connectivity to server
nc -u -v vpn.example.com 1194

# Test TCP connectivity (if server supports it)
nc -v vpn.example.com 1194

# If these fail, firewall is blocking OpenVPN
```

---

## Security Best Practices

### Password Security

1. **Use a strong Keycloak password**
   - Minimum 12 characters
   - Mix of uppercase, lowercase, numbers, symbols
   - Don't reuse passwords from other services
   - Consider using a password manager

2. **Enable MFA (Multi-Factor Authentication)**
   - TOTP (Time-based One-Time Password) - Google Authenticator, Authy
   - WebAuthn - YubiKey, TouchID, Windows Hello
   - SMS (less secure, but better than nothing)
   - Contact IT to enable MFA for your account

### Profile Security

1. **Protect your `.ovpn` file**
   - Contains your organization's CA certificate
   - May contain client certificate/key (if mutual TLS)
   - Don't share with unauthorized users
   - Store on encrypted disk
   - Don't email unencrypted

2. **Delete old profiles**
   - If you receive an updated profile, delete the old one
   - Old CA certificates may be revoked or expired
   - Prevents confusion about which profile to use

### Client Security

1. **Keep software updated**
   - Update OpenVPN client regularly (security patches)
   - Update OS (Windows, macOS, iOS, Android)
   - Enable automatic updates if possible

2. **Use Always-On VPN**
   - Prevents accidental unprotected connections
   - Especially important on mobile devices
   - iOS: Settings → VPN → Connect On Demand
   - Android: Settings → Network → VPN → Always-on VPN

3. **DNS leak prevention**
   - Ensure VPN routes DNS queries
   - Test at https://dnsleaktest.com/
   - Tunnelblick: Enable "Set nameserver"
   - OpenVPN Connect: Enabled by default

4. **Public Wi-Fi protection**
   - ALWAYS connect to VPN before accessing corporate resources
   - Public Wi-Fi is untrusted - assume it's monitored
   - VPN encrypts all traffic, protecting from eavesdropping

### Keycloak Session Security

1. **Log out when done**
   - Disconnect VPN when not needed
   - Reduce attack surface

2. **Monitor active sessions**
   - Check Keycloak account settings for active sessions
   - Revoke suspicious sessions
   - Contact IT if you see unauthorized access

3. **Report suspicious activity**
   - Unexpected login notifications
   - Sessions from unknown locations/devices
   - Contact IT/security team immediately

### Device Security

1. **Use device encryption**
   - macOS: FileVault
   - Windows: BitLocker
   - Linux: LUKS
   - iOS/Android: Enabled by default on modern devices

2. **Lock screen when away**
   - Use password/PIN/biometric lock
   - Set auto-lock timeout (5 minutes or less)
   - Never leave unlocked device unattended

3. **Lost/stolen device**
   - Report to IT immediately
   - IT can revoke Keycloak sessions remotely
   - Use remote wipe if available (Find My iPhone, etc.)

---

## Getting Help

### Self-Service Resources

1. **This documentation:**
   - `docs/client-setup.md` (this file)
   - `docs/troubleshooting.md` (general troubleshooting)
   - `README.md` (project overview)

2. **Check logs:**
   - Client logs (see "Logs and Debugging" section above)
   - Server logs (if you have access)

3. **Test connectivity:**
   - Can you access Keycloak directly in browser?
   - Can you ping/telnet to VPN server?
   - Does the problem occur on different networks?

### Contact IT Support

**When contacting IT, provide:**

1. **Your information:**
   - Full name and username
   - Department/team

2. **Problem description:**
   - What were you trying to do?
   - What happened instead?
   - Error messages (exact text or screenshot)

3. **Environment:**
   - Operating system and version
   - OpenVPN client type and version
   - Network location (office, home, public Wi-Fi, etc.)

4. **Logs (if possible):**
   - Copy relevant log entries
   - Or take screenshot of error

**Support channels:**
- Email: it-support@example.com
- Help desk: https://helpdesk.example.com
- Phone: +1-555-1234 (8am-5pm EST)
- Slack: #it-support

---

## Appendix: Command Reference

### Linux CLI Quick Reference

```bash
# Install OpenVPN
sudo dnf install openvpn  # RHEL/Fedora/Rocky
sudo apt install openvpn  # Debian/Ubuntu

# Connect (interactive)
sudo openvpn --config client.ovpn

# Connect (with credentials file)
sudo openvpn --config client.ovpn --auth-user-pass ~/.openvpn-creds

# Connect as systemd service
sudo systemctl start openvpn-client@vpn-sso

# Stop VPN
sudo systemctl stop openvpn-client@vpn-sso

# View logs
sudo journalctl -u openvpn-client@vpn-sso -f
```

### NetworkManager Quick Reference

```bash
# Install NetworkManager OpenVPN plugin
sudo dnf install NetworkManager-openvpn-gnome

# Import profile
sudo nmcli connection import type openvpn file client.ovpn

# Connect
nmcli connection up vpn-sso

# Disconnect
nmcli connection down vpn-sso

# View status
nmcli connection show vpn-sso

# Edit connection
nmcli connection edit vpn-sso

# View logs
journalctl -u NetworkManager -f
```

### macOS Quick Reference

```bash
# Install Tunnelblick
# Download from https://tunnelblick.net/ and install manually

# Check Tunnelblick version
# Tunnelblick menu → About Tunnelblick

# View logs
# Tunnelblick menu → VPN Details → Log tab
```

### Windows Quick Reference

```powershell
# Install OpenVPN Connect
# Download from https://openvpn.net/client/ and install

# View logs
# OpenVPN Connect → Profile → Settings → Log
```

---

**Document Version:** 1.0  
**Last Updated:** 2026-02-15  
**Maintained By:** OpenVPN SSO Project Team

For questions or feedback about this documentation, please contact the project maintainers or open an issue on the project repository.
