# OpenVPN Server Setup for SSO Authentication

This guide explains how to configure OpenVPN Community Server 2.6.19+ with Keycloak SSO authentication on Rocky Linux 9.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Certificate Generation](#certificate-generation)
- [Server Configuration](#server-configuration)
- [Auth Script Setup](#auth-script-setup)
- [Starting the Server](#starting-the-server)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Advanced Configuration](#advanced-configuration)

---

## Prerequisites

### System Requirements

- **Rocky Linux 9** (or RHEL 9 / AlmaLinux 9)
- **OpenVPN 2.6.19 or later** (from EPEL)
- **Root access** or sudo privileges
- **Keycloak** instance configured (see [Keycloak Setup](keycloak-setup.md))
- **SSO daemon** installed and running (see installation guide)

### Required Packages

```bash
# Enable EPEL repository
sudo dnf install -y epel-release

# Install OpenVPN
sudo dnf install -y openvpn

# Verify version (must be 2.6.19+)
openvpn --version | head -1
# Expected: OpenVPN 2.6.19 x86_64-redhat-linux-gnu
```

### Why OpenVPN 2.6+?

OpenVPN 2.6 introduced critical features for SSO:
- **Script-based deferred authentication** (exit code 2)
- **`auth_pending_file` support** for browser opening
- **`auth_failed_reason_file`** for custom error messages
- **`WEB_AUTH::` URL support** in clients

---

## Installation

### Step 1: Install OpenVPN

```bash
# Install from EPEL (Rocky Linux 9)
sudo dnf install -y epel-release
sudo dnf install -y openvpn easy-rsa

# Create OpenVPN directories
sudo mkdir -p /etc/openvpn/server
sudo mkdir -p /etc/openvpn/client
sudo mkdir -p /var/log/openvpn
```

### Step 2: Install Easy-RSA (for certificates)

```bash
# Copy Easy-RSA to OpenVPN directory
sudo cp -r /usr/share/easy-rsa /etc/openvpn/easy-rsa

# Or download latest version
# cd /etc/openvpn
# wget https://github.com/OpenVPN/easy-rsa/releases/download/v3.1.7/EasyRSA-3.1.7.tgz
# tar xzf EasyRSA-3.1.7.tgz
# mv EasyRSA-3.1.7 easy-rsa
```

---

## Certificate Generation

OpenVPN requires a Public Key Infrastructure (PKI) for TLS encryption. We'll use Easy-RSA to generate certificates.

### Initialize PKI

```bash
cd /etc/openvpn/easy-rsa

# Initialize PKI directory
sudo ./easyrsa init-pki
```

### Build Certificate Authority (CA)

```bash
# Build CA (creates ca.crt and ca.key)
sudo ./easyrsa build-ca nopass

# You'll be prompted for:
# - Common Name: Enter something like "OpenVPN CA"

# CA certificate will be at: pki/ca.crt
```

### Generate Server Certificate

```bash
# Build server certificate and key
sudo ./easyrsa build-server-full server nopass

# Server certificate: pki/issued/server.crt
# Server key: pki/private/server.key
```

### Generate Diffie-Hellman Parameters

```bash
# Generate DH params (this takes a while)
sudo ./easyrsa gen-dh

# DH params: pki/dh.pem
```

### Generate TLS Authentication Key (optional but recommended)

```bash
# Generate static key for tls-auth
sudo openvpn --genkey secret pki/ta.key

# TLS auth key: pki/ta.key
```

### Copy Certificates to OpenVPN Directory

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

### Generate Client Certificates (optional)

If you want mutual TLS (client certificates + SSO):

```bash
# Generate client certificate
sudo ./easyrsa build-client-full client1 nopass

# Client certificate: pki/issued/client1.crt
# Client key: pki/private/client1.key
```

---

## Server Configuration

### Step 1: Copy Example Configuration

```bash
# Copy example configuration
sudo cp /path/to/openvpn-keycloak-sso/config/openvpn-server.conf.example \
        /etc/openvpn/server/server.conf
```

### Step 2: Customize Configuration

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

### Step 3: Verify SSO Directives

Ensure these lines are present and uncommented:

```conf
# Enable script-based authentication
script-security 3

# Auth script with via-file mode
auth-user-pass-verify /etc/openvpn/auth-keycloak.sh via-file

# Allow SSO without password in client config
auth-user-pass-optional

# Token for reconnections
auth-gen-token 0 external-auth

# Extended handshake for SSO
hand-window 120
```

**Critical**: These directives are **required** for SSO to work!

---

## Auth Script Setup

### Step 1: Copy Auth Script

```bash
# Copy the shell wrapper
sudo cp /path/to/openvpn-keycloak-sso/scripts/auth-keycloak.sh \
        /etc/openvpn/auth-keycloak.sh

# Make it executable
sudo chmod +x /etc/openvpn/auth-keycloak.sh
```

### Step 2: Verify Binary Location

The script expects the binary at `/usr/local/bin/openvpn-keycloak-sso`.

Edit if your binary is elsewhere:

```bash
sudo vi /etc/openvpn/auth-keycloak.sh
```

Change this line if needed:
```bash
BINARY="/usr/local/bin/openvpn-keycloak-sso"
```

### Step 3: Test Auth Script

```bash
# Verify script is executable
ls -l /etc/openvpn/auth-keycloak.sh
# Should show: -rwxr-xr-x

# Test script execution
/etc/openvpn/auth-keycloak.sh --help 2>&1 || echo "Script can execute"
```

---

## Starting the Server

### Step 1: Enable IP Forwarding

```bash
# Enable IP forwarding (required for VPN)
sudo sysctl -w net.ipv4.ip_forward=1

# Make permanent
echo "net.ipv4.ip_forward = 1" | sudo tee -a /etc/sysctl.conf
```

### Step 2: Configure Firewall

```bash
# Add firewall rules
sudo firewall-cmd --permanent --add-service=openvpn
sudo firewall-cmd --permanent --add-masquerade
sudo firewall-cmd --reload

# Or allow specific port
sudo firewall-cmd --permanent --add-port=1194/udp
sudo firewall-cmd --reload
```

### Step 3: Configure SELinux (if enabled)

```bash
# Check if SELinux is enforcing
getenforce

# If enforcing, allow OpenVPN to execute scripts
sudo setsebool -P openvpn_run_unconfined 1

# Or create custom policy (more secure)
# See: https://fedoraproject.org/wiki/SELinux/openvpn
```

### Step 4: Start SSO Daemon

```bash
# Ensure daemon is running first
sudo systemctl start openvpn-keycloak-sso
sudo systemctl status openvpn-keycloak-sso

# Enable on boot
sudo systemctl enable openvpn-keycloak-sso
```

### Step 5: Start OpenVPN Server

```bash
# Start OpenVPN server
sudo systemctl start openvpn-server@server

# Check status
sudo systemctl status openvpn-server@server

# Enable on boot
sudo systemctl enable openvpn-server@server
```

### Step 6: Verify Server is Running

```bash
# Check if OpenVPN is listening
sudo ss -tuln | grep 1194

# Should show:
# udp   UNCONN 0  0   0.0.0.0:1194   0.0.0.0:*

# Check logs
sudo journalctl -u openvpn-server@server -f
```

---

## Testing

### Test 1: Configuration Syntax

```bash
# Test OpenVPN configuration
sudo openvpn --config /etc/openvpn/server/server.conf --test-crypto

# Should complete without errors
```

### Test 2: Server Startup

```bash
# Check if server started successfully
sudo systemctl status openvpn-server@server

# Should show: "Active: active (running)"
```

### Test 3: Check Listening Port

```bash
# Verify server is listening
sudo ss -tuln | grep :1194

# Or use netstat
sudo netstat -tuln | grep :1194
```

### Test 4: View Server Status

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

### Test 5: Check Logs

```bash
# View OpenVPN logs
sudo journalctl -u openvpn-server@server -n 50

# Look for:
# - "Initialization Sequence Completed"
# - No errors about auth script
```

### Test 6: Daemon Connection

```bash
# Verify daemon is reachable
sudo systemctl status openvpn-keycloak-sso

# Test Unix socket
ls -l /run/openvpn-keycloak-sso/auth.sock
# Should exist with permissions: srw-rw---- openvpn openvpn
```

### Test 7: Manual Auth Script Test

```bash
# Set up test environment
export username="testuser"
export auth_control_file="/tmp/test_acf"
export auth_pending_file="/tmp/test_apf"
export auth_failed_reason_file="/tmp/test_arf"
export untrusted_ip="192.0.2.1"
export untrusted_port="12345"

# Create test credentials file
echo -e "testuser\nsso" > /tmp/test_creds

# Run auth script
sudo -E /etc/openvpn/auth-keycloak.sh /tmp/test_creds

# Check exit code (should be 2 for deferred)
echo "Exit code: $?"

# Check auth_pending_file (should have 3 lines)
cat /tmp/test_apf
# Expected:
# 300
# openurl
# WEB_AUTH::https://keycloak.example.com/...
```

---

## Troubleshooting

### Issue: Server Won't Start

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

### Issue: Auth Script Fails

**Symptom:** Client connection hangs or fails

**Check:**

1. **Script is executable:**
   ```bash
   ls -l /etc/openvpn/auth-keycloak.sh
   # Should be: -rwxr-xr-x
   ```

2. **Binary path is correct:**
   ```bash
   which openvpn-keycloak-sso
   # Should be: /usr/local/bin/openvpn-keycloak-sso
   ```

3. **Daemon is running:**
   ```bash
   systemctl status openvpn-keycloak-sso
   ```

4. **Check daemon logs:**
   ```bash
   sudo journalctl -u openvpn-keycloak-sso -f
   ```

### Issue: Client Can Connect But No Internet

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

### Issue: DNS Not Working in VPN

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

### Issue: SELinux Blocking Script

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

## Advanced Configuration

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

# Disable compression (faster, more secure)
compress lz4-v2
push "compress lz4-v2"
```

### IPv6 Support

```conf
proto udp6
server-ipv6 fd00:1234:5678::/64
push "route-ipv6 2000::/3"
push "dhcp-option DNS6 2001:4860:4860::8888"
```

---

## Security Hardening

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
sudo journalctl -u openvpn-keycloak-sso | grep "auth"
```

---

## Next Steps

After OpenVPN server is configured:

1. **Create client profiles**: See [Client Setup](client-setup.md)
2. **Test connections**: Connect with a client and verify SSO works
3. **Monitor logs**: Watch for successful authentications
4. **Configure backups**: Backup certificates and configuration
5. **Set up monitoring**: Use Prometheus/Grafana for metrics

---

## Reference

### Important File Locations

| File/Directory | Purpose |
|----------------|---------|
| `/etc/openvpn/server/server.conf` | Server configuration |
| `/etc/openvpn/server/*.crt, *.key` | Certificates and keys |
| `/etc/openvpn/auth-keycloak.sh` | Auth script wrapper |
| `/var/log/openvpn/` | Log files |
| `/run/openvpn-keycloak-sso/` | Daemon socket |
| `/etc/openvpn/ccd/` | Per-client configs |

### Useful Commands

```bash
# Start/Stop/Restart
sudo systemctl start openvpn-server@server
sudo systemctl stop openvpn-server@server
sudo systemctl restart openvpn-server@server

# View logs
sudo journalctl -u openvpn-server@server -f
sudo journalctl -u openvpn-keycloak-sso -f

# Check status
sudo systemctl status openvpn-server@server
cat /var/log/openvpn/openvpn-status.log

# Test configuration
sudo openvpn --config /etc/openvpn/server/server.conf --test-crypto

# Generate client config
# See: client-setup.md
```

---

*Last updated: 2026-02-15 for OpenVPN 2.6.19 on Rocky Linux 9*
