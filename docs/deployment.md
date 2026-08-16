# Deployment Guide - OpenVPN Keycloak SSO

This guide covers deployment of the OpenVPN Keycloak SSO authentication daemon **and** the OpenVPN server it authenticates for, on Rocky Linux 9.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Building the Binary](#building-the-binary)
3. [Installation](#installation)
4. [Configuration](#configuration)
5. [Service Management](#service-management)
6. [Verification](#verification)
7. [Troubleshooting](#troubleshooting)
8. [OpenVPN Server Setup](#openvpn-server-setup)
9. [Uninstallation](#uninstallation)
10. [Manual Installation](#manual-installation)
11. [Security Hardening](#security-hardening)
12. [Advanced OpenVPN Configuration](#advanced-openvpn-configuration)
13. [Distribution Package](#distribution-package)

---

## Prerequisites

### System Requirements

- **Operating System:** Rocky Linux 9 (or RHEL 9 derivative)
- **Architecture:** x86_64 (amd64)
- **OpenVPN:** Version 2.6.2 or later
- **Go:** Version 1.26+ (for building from source)
- **RAM:** Minimum 512MB (1GB recommended)
- **Disk Space:** 50MB for binary and dependencies

### Required Software

```bash
# Install OpenVPN 2.6.2+ from EPEL (easy-rsa is needed for the PKI later)
sudo dnf install epel-release
sudo dnf install openvpn easy-rsa

# Verify version; the first line should show OpenVPN 2.6.x or later
openvpn --version
```

### Network Requirements

- **Outbound HTTPS (443):** To Keycloak server
- **Inbound HTTP (9000):** For OIDC callback (configurable)
- **Unix Socket:** `/run/openvpn-keycloak-auth/auth.sock` (local only)

### Firewall Configuration

```bash
# Open callback port (if firewalld is running)
sudo firewall-cmd --permanent --add-port=9000/tcp
sudo firewall-cmd --reload

# Verify
sudo firewall-cmd --list-ports
```

---

## Building the Binary

### Method 1: Using Makefile (Recommended)

```bash
# Clone repository
git clone https://github.com/al-bashkir/openvpn-keycloak-auth
cd openvpn-keycloak-auth

# Build production binary
make build

# Verify build
./openvpn-keycloak-auth version
```

### Method 2: Manual Build

```bash
# Production build (static, optimized)
CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w" \
  -o openvpn-keycloak-auth \
  ./cmd/openvpn-keycloak-auth

# Verify
./openvpn-keycloak-auth version
```

### Method 3: Development Build

```bash
# Fast build with debug info
make build-dev

# Or manually
go build -o openvpn-keycloak-auth ./cmd/openvpn-keycloak-auth
```

### Build Output

The build produces a single static binary:

```bash
$ ls -lh openvpn-keycloak-auth
-rwxr-xr-x. 1 user user 9.8M Feb 15 12:00 openvpn-keycloak-auth

$ file openvpn-keycloak-auth
openvpn-keycloak-auth: ELF 64-bit LSB executable, x86-64, statically linked, stripped
```

---

## Installation

### Automated Installation (Recommended)

The installation script handles all setup automatically:

```bash
# Build first
make build

# Install (requires root)
sudo make install

# Or run script directly
sudo ./deploy/install.sh
```

### What the Installer Does

1. **Checks prerequisites** - Verifies OpenVPN 2.6.2+ is installed
2. **Creates user/group** - Creates `openvpn` system user and group
3. **Installs binary** - Copies to `/usr/local/bin/openvpn-keycloak-auth`
4. **Creates directories:**
   - `/etc/openvpn` - Configuration files
   - `/etc/openvpn/scripts` - OpenVPN auth wrapper scripts
   - `/var/lib/openvpn-keycloak-auth` - Data directory
   - `/var/lib/openvpn-keycloak-auth/tmp` - Shared OpenVPN temp directory
   - `/run/openvpn-keycloak-auth` - Runtime socket directory
5. **Installs files:**
   - `/etc/openvpn/keycloak-sso.yaml` - Configuration (if not exists)
   - `/etc/openvpn/scripts/auth-keycloak.sh` - Auth script
   - `/etc/systemd/system/openvpn-keycloak-auth.service` - systemd unit
   - `/etc/systemd/system/openvpn-server@.service.d/sso-override.conf` - OpenVPN service override
6. **Configures firewall** - Opens port 9000/tcp (if firewalld is running)
7. **Configures SELinux** - Sets file contexts (if SELinux is enabled)

### Installation Output

```
╔══════════════════════════════════════════════════════════════╗
║     OpenVPN Keycloak SSO - Installation Script               ║
╚══════════════════════════════════════════════════════════════╝

[INFO] Performing preliminary checks...
[INFO] Detected OS: Rocky Linux release 9.3 (Blue Onyx)
...
[SUCCESS] Installation Completed Successfully!
```

---

## Configuration

### Initial Configuration

After installation, you **must** edit the configuration file:

```bash
# Edit configuration
sudo vim /etc/openvpn/keycloak-sso.yaml
```

### Required Settings

Update these settings with your actual values:

```yaml
listen:
  http: ":9000"

oidc:
  issuer: "https://keycloak.example.com/realms/myrealm"
  client_id: "openvpn"
  redirect_uri: "https://vpn.example.com:9000/callback"
  # ... other settings
```

See [`config/openvpn-keycloak-auth.yaml.example`](../config/openvpn-keycloak-auth.yaml.example) for all available options.

### Validate Configuration

```bash
# Check local configuration syntax and values
sudo /usr/local/bin/openvpn-keycloak-auth check-config \
  --config /etc/openvpn/keycloak-sso.yaml
```

Expected output:

```
Configuration is valid

Configuration Summary:
  OIDC Issuer: https://keycloak.example.com/realms/myrealm
  Client ID: openvpn
  Redirect URI: https://vpn.example.com:9000/callback
  ...
```

`check-config` validates YAML, known keys, required fields, URL syntax, socket path, log settings, and TLS certificate/key file existence when direct TLS is enabled. It does not contact Keycloak or perform OIDC discovery; discovery is performed when the daemon starts.

---

## Service Management

### Enable and Start Service

```bash
# Enable service (start on boot)
sudo systemctl enable openvpn-keycloak-auth

# Start service
sudo systemctl start openvpn-keycloak-auth

# Check status
sudo systemctl status openvpn-keycloak-auth
```

### Service Status Output

```
● openvpn-keycloak-auth.service - OpenVPN Keycloak Authentication Daemon
     Loaded: loaded (/etc/systemd/system/openvpn-keycloak-auth.service; enabled; preset: disabled)
     Active: active (running) since Sat 2026-02-15 12:00:00 UTC; 5min ago
       Docs: https://github.com/al-bashkir/openvpn-keycloak
   Main PID: 12345 (openvpn-keycloa)
      Tasks: 8 (limit: 512)
     Memory: 12.5M
     CGroup: /system.slice/openvpn-keycloak-auth.service
             └─12345 /usr/local/bin/openvpn-keycloak-auth serve --config /etc/openvpn/keycloak-sso.yaml

Feb 15 12:00:00 vpn systemd[1]: Starting OpenVPN Keycloak SSO Authentication Daemon...
Feb 15 12:00:00 vpn openvpn-keycloak-auth[12345]: INFO starting daemon version=895062d
Feb 15 12:00:00 vpn openvpn-keycloak-auth[12345]: INFO OIDC provider discovered issuer=https://keycloak.example.com/realms/myrealm
Feb 15 12:00:00 vpn openvpn-keycloak-auth[12345]: INFO HTTP server listening addr=0.0.0.0:9000
Feb 15 12:00:00 vpn openvpn-keycloak-auth[12345]: INFO IPC server listening socket=/run/openvpn-keycloak-auth/auth.sock
Feb 15 12:00:00 vpn systemd[1]: Started OpenVPN Keycloak SSO Authentication Daemon.
```

### View Logs

```bash
# Follow logs in real-time
sudo journalctl -u openvpn-keycloak-auth -f

# View recent logs
sudo journalctl -u openvpn-keycloak-auth -n 100

# View logs since specific time
sudo journalctl -u openvpn-keycloak-auth --since "10 minutes ago"

# View logs for specific date
sudo journalctl -u openvpn-keycloak-auth --since "2026-02-15" --until "2026-02-16"
```

### Service Control Commands

```bash
# Start service
sudo systemctl start openvpn-keycloak-auth

# Stop service
sudo systemctl stop openvpn-keycloak-auth

# Restart service
sudo systemctl restart openvpn-keycloak-auth

# Reload configuration (not supported, requires restart)
sudo systemctl restart openvpn-keycloak-auth

# Check status
sudo systemctl status openvpn-keycloak-auth

# Enable autostart
sudo systemctl enable openvpn-keycloak-auth

# Disable autostart
sudo systemctl disable openvpn-keycloak-auth

# Check if enabled
sudo systemctl is-enabled openvpn-keycloak-auth

# Check if running
sudo systemctl is-active openvpn-keycloak-auth
```

---

## Verification

### Step 1: Check Service Status

```bash
sudo systemctl status openvpn-keycloak-auth
```

Should show: `Active: active (running)`

### Step 2: Verify Unix Socket

```bash
# Check socket exists
ls -l /run/openvpn-keycloak-auth/auth.sock

# Should show:
srwxrwx---. 1 openvpn openvpn 0 Feb 15 12:00 /run/openvpn-keycloak-auth/auth.sock
```

### Step 3: Verify HTTP Server

```bash
# Test HTTP server is listening
curl -v http://localhost:9000/health

# Should return:
{"status":"ok","version":"dev"}
```

### Step 4: Test OIDC Discovery

```bash
# Check daemon logs for successful OIDC provider initialization
sudo journalctl -u openvpn-keycloak-auth

# Should include a successful OIDC provider initialization message.
```

### Step 5: Test Auth Script

```bash
# Create test environment
export username="testuser"
export auth_control_file="/tmp/test_acf"
export auth_pending_file="/tmp/test_apf"
export auth_failed_reason_file="/tmp/test_arf"
export untrusted_ip="192.0.2.1"
export untrusted_port="12345"
export IV_SSO="webauth,openurl"

# Create credentials file
echo -e "testuser\nsso" > /tmp/test_creds
chmod 600 /tmp/test_creds

# Run auth script
/etc/openvpn/scripts/auth-keycloak.sh /tmp/test_creds
echo "Exit code: $?"

# Should show: Exit code: 2 (deferred)

# Check auth_pending_file
cat /tmp/test_apf

# Should show 3 lines:
# 300
# webauth
# WEB_AUTH::https://keycloak.example.com/realms/...

# Cleanup
rm -f /tmp/test_*
```

### Step 6: Full Integration Test

See [OpenVPN Server Setup](#openvpn-server-setup) below, section "Testing", for complete end-to-end testing.

---

## Troubleshooting

### Service Won't Start

**Check logs:**

```bash
sudo journalctl -u openvpn-keycloak-auth -n 50
```

**Common issues:**

1. **Configuration error**
   ```
   ERROR failed to load config: yaml: unmarshal errors
   ```
   **Solution:** Check YAML syntax in `/etc/openvpn/keycloak-sso.yaml`

2. **Port already in use**
   ```
   ERROR failed to start HTTP server: listen tcp :9000: bind: address already in use
   ```
   **Solution:** Change port in config or stop conflicting service
   ```bash
   sudo lsof -i :9000  # Find what's using the port
   ```

3. **Can't reach Keycloak**
   ```
   ERROR failed to discover OIDC provider: Get "https://keycloak.example.com/...": dial tcp: lookup keycloak.example.com: no such host
   ```
   **Solution:** Check DNS, firewall, and Keycloak URL

4. **Permission denied**
   ```
   ERROR failed to create socket: listen unix /run/openvpn-keycloak-auth/auth.sock: bind: permission denied
   ```
   **Solution:** Check directory permissions, ensure `RuntimeDirectory` is set in the service file

### Socket Not Created

```bash
# Check if directory exists
ls -ld /run/openvpn-keycloak-auth/

# Should show:
drwxrwx---. 2 openvpn openvpn 60 Feb 15 12:00 /run/openvpn-keycloak-auth/

# If directory missing, restart service
sudo systemctl restart openvpn-keycloak-auth
```

### HTTP Server Not Responding

```bash
# Check if listening
sudo ss -tlnp | grep 9000

# Should show:
LISTEN 0  4096  0.0.0.0:9000  0.0.0.0:*  users:(("openvpn-keycloak",pid=12345,fd=8))

# Test locally
curl http://localhost:9000/health

# Test from outside (if firewall allows)
curl http://vpn.example.com:9000/health
```

### SELinux Denials

```bash
# Check for SELinux denials
sudo ausearch -m avc -ts recent

# If denials found, check context
ls -Z /usr/local/bin/openvpn-keycloak-auth

# Should show:
-rwxr-xr-x. root root system_u:object_r:bin_t:s0 /usr/local/bin/openvpn-keycloak-auth

# If wrong context, restore
sudo restorecon -v /usr/local/bin/openvpn-keycloak-auth

# If issues persist, create custom policy or set to permissive
sudo semanage permissive -a openvpn_keycloak_sso_t
```

### High Memory Usage

```bash
# Check memory usage
sudo systemctl status openvpn-keycloak-auth | grep Memory

# If excessive, check for session leaks
sudo journalctl -u openvpn-keycloak-auth | grep "cleaned up"

# Should periodically show:
INFO cleaned up expired sessions count=X

# Check session cleanup is working
# Sessions should expire after auth.session_timeout (default 5 minutes)
```

---

### OpenVPN Server Troubleshooting

#### Issue: Server Won't Start

**Check logs:**

```bash
sudo journalctl -u openvpn-server@server -xe
```

**Common causes:**

1. **Port already in use:**
   ```bash
   sudo ss -tuln | grep :1194
   # Change port in server.conf if needed
   ```

2. **Certificate files not found:**
   ```bash
   ls -l /etc/openvpn/server/
   # Verify ca.crt, server.crt, server.key, dh.pem exist
   ```

3. **Permission errors:**
   ```bash
   # Check OpenVPN can read certificate files
   sudo -u openvpn cat /etc/openvpn/server/server.key
   ```

#### Issue: Auth Script Fails

**Symptom:** Client connection hangs or fails

**Check:**

1. **Script is executable:**
   ```bash
   ls -l /etc/openvpn/scripts/auth-keycloak.sh
   # Should be: -rwxr-xr-x
   ```

2. **Binary path is correct:**
   ```bash
   which openvpn-keycloak-auth
   # Should be: /usr/local/bin/openvpn-keycloak-auth
   ```

3. **Daemon is running:**
   ```bash
   systemctl status openvpn-keycloak-auth
   ```

4. **Check daemon logs:**
   ```bash
   sudo journalctl -u openvpn-keycloak-auth -f
   ```

#### Issue: Client Can Connect But No Internet

**Causes:**

1. **IP forwarding not enabled:**
   ```bash
   sudo sysctl net.ipv4.ip_forward
   # Should output: net.ipv4.ip_forward = 1
   ```

2. **NAT/Masquerade not configured:**
   ```bash
   sudo firewall-cmd --query-masquerade
   # Should output: yes
   ```

3. **Routes not pushed:**
   ```bash
   # Check server.conf has:
   # push "redirect-gateway def1 bypass-dhcp"
   ```

#### Issue: DNS Not Working in VPN

**Solutions:**

1. **Push DNS servers:**
   ```conf
   push "dhcp-option DNS 8.8.8.8"
   push "dhcp-option DNS 8.8.4.4"
   ```

2. **Check client received DNS:**
   ```bash
   # On client
   cat /etc/resolv.conf
   # Or: resolvectl status (systemd-resolved)
   ```

#### Issue: SELinux Blocking Script

**Check SELinux denials:**

```bash
sudo ausearch -m avc -ts recent | grep openvpn
```

**Solutions:**

```bash
# Quick fix (permissive for testing)
sudo setenforce 0

# Permanent fix
sudo setsebool -P openvpn_run_unconfined 1

# Or create custom policy (recommended)
```

---

## OpenVPN Server Setup

The daemon is only half the deployment: OpenVPN itself needs a PKI, a server
config that defers authentication to the auth script, and the SSO override.
Everything below assumes the daemon from the previous sections is installed
and running.

### Install Easy-RSA

```bash
# Copy Easy-RSA to OpenVPN directory
sudo cp -r /usr/share/easy-rsa /etc/openvpn/easy-rsa

# Or download latest version
# cd /etc/openvpn
# wget https://github.com/OpenVPN/easy-rsa/releases/download/v3.1.7/EasyRSA-3.1.7.tgz
# tar xzf EasyRSA-3.1.7.tgz
# mv EasyRSA-3.1.7 easy-rsa
```

### Certificate Generation

OpenVPN requires a Public Key Infrastructure (PKI) for TLS encryption. We'll use Easy-RSA to generate certificates.

#### Initialize PKI

```bash
cd /etc/openvpn/easy-rsa

# Initialize PKI directory
sudo ./easyrsa init-pki
```

#### Build Certificate Authority (CA)

```bash
# Build CA (creates ca.crt and ca.key)
sudo ./easyrsa build-ca nopass

# You'll be prompted for:
# - Common Name: Enter something like "OpenVPN CA"

# CA certificate will be at: pki/ca.crt
```

#### Generate Server Certificate

```bash
# Build server certificate and key
sudo ./easyrsa build-server-full server nopass

# Server certificate: pki/issued/server.crt
# Server key: pki/private/server.key
```

#### Generate Diffie-Hellman Parameters

```bash
# Generate DH params (this takes a while)
sudo ./easyrsa gen-dh

# DH params: pki/dh.pem
```

#### Generate TLS Authentication Key (optional but recommended)

```bash
# Generate static key for tls-auth
sudo openvpn --genkey secret pki/ta.key

# TLS auth key: pki/ta.key
```

#### Copy Certificates to OpenVPN Directory

```bash
# Copy to server directory
sudo cp pki/ca.crt /etc/openvpn/server/
sudo cp pki/issued/server.crt /etc/openvpn/server/
sudo cp pki/private/server.key /etc/openvpn/server/
sudo cp pki/dh.pem /etc/openvpn/server/
sudo cp pki/ta.key /etc/openvpn/server/

# Set permissions
sudo chmod 600 /etc/openvpn/server/server.key
sudo chmod 600 /etc/openvpn/server/ta.key
```

#### Generate Client Certificates (optional)

If you want mutual TLS (client certificates + SSO):

```bash
# Generate client certificate
sudo ./easyrsa build-client-full client1 nopass

# Client certificate: pki/issued/client1.crt
# Client key: pki/private/client1.key
```

---

### Server Configuration

#### Step 1: Copy Example Configuration

```bash
# Copy example configuration
sudo cp /path/to/openvpn-keycloak-auth/config/openvpn-server.conf.example \
        /etc/openvpn/server/server.conf
```

#### Step 2: Customize Configuration

Edit `/etc/openvpn/server/server.conf`:

```bash
sudo vi /etc/openvpn/server/server.conf
```

**Required Changes:**

1. **Port and Protocol** (if needed):
   ```conf
   port 1194          # Change if port 1194 is already used
   proto udp          # Or 'tcp' if UDP is blocked
   ```

2. **VPN Subnet** (if conflicts with your network):
   ```conf
   server 10.8.0.0 255.255.255.0  # Change to avoid conflicts
   ```

3. **DNS Servers**:
   ```conf
   # For full tunnel (all traffic through VPN)
   push "dhcp-option DNS 8.8.8.8"
   push "dhcp-option DNS 8.8.4.4"

   # Or use your internal DNS for split tunnel
   push "dhcp-option DNS 192.168.1.1"
   ```

4. **Routes** (for split tunnel):
   ```conf
   # Comment out this line for split tunnel:
   # push "redirect-gateway def1 bypass-dhcp"

   # Add specific routes instead:
   push "route 10.0.0.0 255.0.0.0"
   push "route 192.168.1.0 255.255.255.0"
   ```

#### Step 3: Verify SSO Directives

Ensure these lines are present and uncommented:

```conf
# Enable script-based authentication
script-security 3

# Auth script with via-file mode
auth-user-pass-verify /etc/openvpn/scripts/auth-keycloak.sh via-file

# Allow SSO without password in client config
auth-user-pass-optional

# Token for reconnections
auth-gen-token 0 external-auth

# Extended handshake for SSO
hand-window 120

# Shared temp directory visible to both OpenVPN and the auth daemon
tmp-dir /var/lib/openvpn-keycloak-auth/tmp
```

**Critical**: These directives are **required** for SSO to work!

---

### Auth Script Setup

#### Step 1: Copy Auth Script

```bash
# Copy the shell wrapper
sudo mkdir -p /etc/openvpn/scripts
sudo cp /path/to/openvpn-keycloak-auth/scripts/auth-keycloak.sh \
        /etc/openvpn/scripts/auth-keycloak.sh

# Make it executable
sudo chmod +x /etc/openvpn/scripts/auth-keycloak.sh
```

#### Step 2: Verify Binary Location

The script expects the binary at `/usr/local/bin/openvpn-keycloak-auth`.

Edit if your binary is elsewhere:

```bash
sudo vi /etc/openvpn/scripts/auth-keycloak.sh
```

Change this line if needed:

```bash
local -r binary="/usr/local/bin/openvpn-keycloak-auth"
```

#### Step 3: Test Auth Script

```bash
# Verify script is executable
ls -l /etc/openvpn/scripts/auth-keycloak.sh
# Should show: -rwxr-xr-x

# Test script execution
/etc/openvpn/scripts/auth-keycloak.sh --help 2>&1 || printf 'Script can execute\n'
```

---

### Starting the Server

#### Step 1: Enable IP Forwarding

```bash
# Enable IP forwarding (required for VPN)
sudo sysctl -w net.ipv4.ip_forward=1

# Make permanent
echo "net.ipv4.ip_forward = 1" | sudo tee -a /etc/sysctl.conf
```

#### Step 2: Configure Firewall

```bash
# Add firewall rules
sudo firewall-cmd --permanent --add-service=openvpn
sudo firewall-cmd --permanent --add-masquerade
sudo firewall-cmd --reload

# Or allow specific port
sudo firewall-cmd --permanent --add-port=1194/udp
sudo firewall-cmd --reload
```

#### Step 3: Configure SELinux (if enabled)

```bash
# Check if SELinux is enforcing
getenforce

# If enforcing, allow OpenVPN to execute scripts
sudo setsebool -P openvpn_run_unconfined 1

# Or create custom policy (more secure)
# See: https://fedoraproject.org/wiki/SELinux/openvpn
```

#### Step 4: Start SSO Daemon

```bash
# Ensure daemon is running first
sudo systemctl start openvpn-keycloak-auth
sudo systemctl status openvpn-keycloak-auth

# Enable on boot
sudo systemctl enable openvpn-keycloak-auth
```

#### Step 5: Start OpenVPN Server

```bash
# Start OpenVPN server
sudo systemctl start openvpn-server@server

# Check status
sudo systemctl status openvpn-server@server

# Enable on boot
sudo systemctl enable openvpn-server@server
```

#### Step 6: Verify Server is Running

```bash
# Check if OpenVPN is listening
sudo ss -tuln | grep 1194

# Should show:
# udp   UNCONN 0  0   0.0.0.0:1194   0.0.0.0:*

# Check logs
sudo journalctl -u openvpn-server@server -f
```

---

### Testing

#### Test 1: Configuration Syntax

```bash
# Test OpenVPN configuration
sudo openvpn --config /etc/openvpn/server/server.conf --test-crypto

# Should complete without errors
```

#### Test 2: Server Startup

```bash
# Check if server started successfully
sudo systemctl status openvpn-server@server

# Should show: "Active: active (running)"
```

#### Test 3: Check Listening Port

```bash
# Verify server is listening
sudo ss -tuln | grep :1194

# Or use netstat
sudo netstat -tuln | grep :1194
```

#### Test 4: View Server Status

```bash
# Check OpenVPN status file
cat /var/log/openvpn/openvpn-status.log
```

Should show:

```
OpenVPN CLIENT LIST
Updated,Fri Feb 15 20:00:00 2026
Common Name,Real Address,Bytes Received,Bytes Sent,Connected Since
ROUTING TABLE
Virtual Address,Common Name,Real Address,Last Ref
GLOBAL STATS
Max bcast/mcast queue length,0
END
```

#### Test 5: Check Logs

```bash
# View OpenVPN logs
sudo journalctl -u openvpn-server@server -n 50

# Look for:
# - "Initialization Sequence Completed"
# - No errors about auth script
```

#### Test 6: Daemon Connection

(The manual auth-script test lives in [Verification](#step-5-test-auth-script) above.)

```bash
# Verify daemon is reachable
sudo systemctl status openvpn-keycloak-auth

# Test Unix socket
ls -l /run/openvpn-keycloak-auth/auth.sock
# Should exist with permissions: srw-rw---- openvpn openvpn
```

---

## Uninstallation

### Automated Uninstallation (Recommended)

```bash
# Uninstall (requires root)
sudo make uninstall

# Or run script directly
sudo ./deploy/uninstall.sh
```

### What the Uninstaller Does

1. **Stops service** - Stops and disables systemd service
2. **Removes service file** - Deletes systemd unit file
3. **Removes binary** - Deletes `/usr/local/bin/openvpn-keycloak-auth`
4. **Removes auth script** - Deletes `/etc/openvpn/scripts/auth-keycloak.sh`
5. **Prompts for config removal** - Optionally removes `/etc/openvpn/keycloak-sso.yaml`
6. **Prompts for data removal** - Optionally removes `/var/lib/openvpn-keycloak-auth`

### Manual Cleanup

If automated uninstall fails or you need manual cleanup:

```bash
# Stop and disable service
sudo systemctl stop openvpn-keycloak-auth
sudo systemctl disable openvpn-keycloak-auth

# Remove service file
sudo rm -f /etc/systemd/system/openvpn-keycloak-auth.service
sudo systemctl daemon-reload

# Remove binary
sudo rm -f /usr/local/bin/openvpn-keycloak-auth

# Remove auth script
sudo rm -f /etc/openvpn/scripts/auth-keycloak.sh

# Remove configuration (optional)
sudo rm -f /etc/openvpn/keycloak-sso.yaml

# Remove data directory (optional)
sudo rm -rf /var/lib/openvpn-keycloak-auth

# Remove firewall rule (optional)
sudo firewall-cmd --permanent --remove-port=9000/tcp
sudo firewall-cmd --reload
```

---

## Manual Installation

If you can't use the automated installer, follow these steps:

### 1. Create User and Group

```bash
sudo useradd --system --shell /sbin/nologin openvpn
```

### 2. Install Binary

```bash
sudo install -m 755 openvpn-keycloak-auth /usr/local/bin/openvpn-keycloak-auth
```

### 3. Create Directories

```bash
sudo mkdir -p /etc/openvpn/scripts
sudo mkdir -p /var/lib/openvpn-keycloak-auth
sudo mkdir -p /var/lib/openvpn-keycloak-auth/tmp
sudo mkdir -p /run/openvpn-keycloak-auth
sudo chown root:openvpn /etc/openvpn/scripts
sudo chmod 750 /etc/openvpn/scripts
sudo chown openvpn:openvpn /var/lib/openvpn-keycloak-auth
sudo chown openvpn:openvpn /var/lib/openvpn-keycloak-auth/tmp
sudo chown openvpn:openvpn /run/openvpn-keycloak-auth
sudo chmod 755 /var/lib/openvpn-keycloak-auth
sudo chmod 750 /var/lib/openvpn-keycloak-auth/tmp
sudo chmod 770 /run/openvpn-keycloak-auth
```

### 4. Install Configuration

```bash
sudo install -m 640 config/openvpn-keycloak-auth.yaml.example \
  /etc/openvpn/keycloak-sso.yaml
sudo chown root:openvpn /etc/openvpn/keycloak-sso.yaml

# Edit configuration
sudo vim /etc/openvpn/keycloak-sso.yaml
```

### 5. Install Auth Script

```bash
sudo mkdir -p /etc/openvpn/scripts
sudo install -m 755 scripts/auth-keycloak.sh /etc/openvpn/scripts/auth-keycloak.sh
```

### 6. Install systemd Service

```bash
sudo install -m 644 deploy/openvpn-keycloak-auth.service \
  /etc/systemd/system/openvpn-keycloak-auth.service
sudo mkdir -p /etc/systemd/system/openvpn-server@.service.d
sudo install -m 644 deploy/openvpn-sso-override.conf \
  /etc/systemd/system/openvpn-server@.service.d/sso-override.conf
sudo systemctl daemon-reload
```

### 7. Configure Firewall

```bash
sudo firewall-cmd --permanent --add-port=9000/tcp
sudo firewall-cmd --reload
```

### 8. Enable and Start Service

```bash
sudo systemctl enable openvpn-keycloak-auth
sudo systemctl start openvpn-keycloak-auth
sudo systemctl status openvpn-keycloak-auth
```

---

## Security Hardening

Daemon hardening first, then the OpenVPN server. The reasoning behind these
settings is in [security.md](security.md).

### Daemon Hardening

The systemd service file includes extensive security hardening. Review these settings:

#### Filesystem Protection

```ini
ProtectSystem=strict         # Read-only /usr, /boot, /efi
ProtectHome=true            # Inaccessible /home
ReadWritePaths=/var/lib/openvpn-keycloak-auth
PrivateTmp=true             # Private /tmp
```

#### Kernel Protection

```ini
ProtectKernelTunables=true  # Read-only /proc/sys, /sys
ProtectKernelModules=true   # No kernel module loading
ProtectKernelLogs=true      # No access to kernel logs
ProtectControlGroups=true   # Read-only /sys/fs/cgroup
```

#### Privilege Restrictions

```ini
NoNewPrivileges=true        # Can't gain new privileges
RestrictSUIDSGID=true       # SUID/SGID bits have no effect
LockPersonality=true        # No personality changes
```

`PrivateUsers=true` is intentionally not enabled in the shipped daemon unit because it can break Unix socket ownership and peer-credential behavior between OpenVPN and the auth daemon.

#### System Call Filtering

```ini
SystemCallFilter=@system-service
SystemCallFilter=~@privileged @resources @obsolete @debug @mount
```

#### Capabilities

```ini
# If using port >= 1024 (default 9000), no capabilities needed
CapabilityBoundingSet=
AmbientCapabilities=

# If using port < 1024, uncomment:
# CapabilityBoundingSet=CAP_NET_BIND_SERVICE
# AmbientCapabilities=CAP_NET_BIND_SERVICE
```

#### Resource Limits

```ini
LimitNOFILE=65536           # Max open files
LimitNPROC=512              # Max processes
TasksMax=512                # Max threads
```

#### Additional Hardening

To further harden the system:

1. **Run on non-standard port** (reduces automated attacks)
   ```yaml
   listen:
     http: "0.0.0.0:9443" # Instead of :9000
   ```

2. **Use TLS for HTTP server** (requires certificates)
   ```yaml
   tls:
     enabled: true
     cert_file: "/etc/openvpn/certs/server.crt"
     key_file: "/etc/openvpn/certs/server.key"
   ```

3. **Restrict callback URL to VPN network only**
   ```yaml
   listen:
     http: "10.8.0.1:9000" # VPN interface only
   ```

4. **Keep the callback endpoint behind firewall or reverse-proxy controls**
   - The daemon includes a built-in global request limiter, but it is not currently configurable.
   - Use firewalld, nginx, Apache, or load-balancer controls for deployment-specific limits.

5. **Regularly update** the binary and dependencies
   ```bash
   cd /path/to/source
   git pull
   make build
   sudo make install
   sudo systemctl restart openvpn-keycloak-auth
   ```

### OpenVPN Hardening

### Recommended Settings

```conf
# Use strong ciphers only
data-ciphers AES-256-GCM:AES-128-GCM
tls-cipher TLS-ECDHE-RSA-WITH-AES-256-GCM-SHA384

# Minimum TLS 1.2
tls-version-min 1.2

# TLS authentication
tls-auth /etc/openvpn/server/ta.key 0

# Drop privileges
user openvpn
group openvpn

# Limit clients
max-clients 100

# Limit script execution time
script-timeout 60
```

### Monitoring

```bash
# Watch connections in real-time
watch -n 1 'sudo cat /var/log/openvpn/openvpn-status.log'

# Monitor logs
sudo journalctl -u openvpn-server@server -f

# Check authentication events
sudo journalctl -u openvpn-keycloak-auth | grep "auth"
```

---

## Advanced OpenVPN Configuration

### Per-Client Configuration

Create `/etc/openvpn/ccd/` directory:

```bash
sudo mkdir -p /etc/openvpn/ccd
```

Enable in `server.conf`:

```conf
client-config-dir /etc/openvpn/ccd
```

Create per-client config (filename = common name or username):

```bash
# /etc/openvpn/ccd/testuser
ifconfig-push 10.8.0.10 255.255.255.0
push "route 192.168.100.0 255.255.255.0"
```

### Multiple Server Instances

```bash
# Create additional configs
sudo cp /etc/openvpn/server/server.conf /etc/openvpn/server/server2.conf

# Edit server2.conf:
# - Change port (e.g., 1195)
# - Change management port if using
# - Change log files

# Start second instance
sudo systemctl start openvpn-server@server2
```

### Logging to File

Add to `server.conf`:

```conf
log /var/log/openvpn/openvpn.log
status /var/log/openvpn/status.log 60
```

### Performance Tuning

For high-throughput scenarios:

```conf
# Enable fast I/O
fast-io

# Increase buffer sizes
sndbuf 393216
rcvbuf 393216
push "sndbuf 393216"
push "rcvbuf 393216"

# Leave compression disabled unless you fully understand the security tradeoffs.
# Do not add compress/push "compress" directives for most deployments.
```

### IPv6 Support

```conf
proto udp6
server-ipv6 fd00:1234:5678::/64
push "route-ipv6 2000::/3"
push "dhcp-option DNS6 2001:4860:4860::8888"
```

---

## Distribution Package

To create a distribution tarball for deployment on multiple servers:

```bash
# Create tarball
make dist

# Creates: dist/openvpn-keycloak-auth-<version>-linux-amd64.tar.gz
```

### Deploy from Tarball

```bash
# On target server
tar -xzf openvpn-keycloak-auth-<version>-linux-amd64.tar.gz
cd openvpn-keycloak-auth-<version>-linux-amd64
sudo ./deploy/install.sh
```

---

## Next Steps

After successful installation:

1. **Configure Keycloak** - See [keycloak.md](./keycloak.md)
2. **Configure OpenVPN Server** - See [OpenVPN Server Setup](#openvpn-server-setup) above
3. **Configure Clients** - See [client-setup.md](./client-setup.md)
4. **Test SSO Flow** - Follow the testing procedures in that section

## Reference

### Important File Locations

| File/Directory                          | Purpose               |
| --------------------------------------- | --------------------- |
| `/etc/openvpn/server/server.conf`       | Server configuration  |
| `/etc/openvpn/server/*.crt, *.key`      | Certificates and keys |
| `/etc/openvpn/scripts/auth-keycloak.sh` | Auth script wrapper   |
| `/var/log/openvpn/`                     | Log files             |
| `/run/openvpn-keycloak-auth/`           | Daemon socket         |
| `/etc/openvpn/ccd/`                     | Per-client configs    |

### Useful Commands

```bash
# Start/Stop/Restart
sudo systemctl start openvpn-server@server
sudo systemctl stop openvpn-server@server
sudo systemctl restart openvpn-server@server

# View logs
sudo journalctl -u openvpn-server@server -f
sudo journalctl -u openvpn-keycloak-auth -f

# Check status
sudo systemctl status openvpn-server@server
cat /var/log/openvpn/openvpn-status.log

# Test configuration
sudo openvpn --config /etc/openvpn/server/server.conf --test-crypto

# Generate client config
# See: client-setup.md
```

---

## _Last updated: 2026-02-15 for OpenVPN 2.6.2+; examples target OpenVPN 2.6.19 on Rocky Linux 9_

**Document Version:** 1.0\
**Last Updated:** 2026-02-15\
**Platform:** Rocky Linux 9

For questions or issues, consult the troubleshooting section or check project documentation.
