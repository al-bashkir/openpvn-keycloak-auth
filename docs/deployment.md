# Deployment Guide - OpenVPN Keycloak SSO

This guide covers deployment of the OpenVPN Keycloak SSO authentication daemon on Rocky Linux 9.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Building the Binary](#building-the-binary)
3. [Installation](#installation)
4. [Configuration](#configuration)
5. [Service Management](#service-management)
6. [Verification](#verification)
7. [Troubleshooting](#troubleshooting)
8. [Uninstallation](#uninstallation)
9. [Manual Installation](#manual-installation)
10. [Security Hardening](#security-hardening)

---

## Prerequisites

### System Requirements

- **Operating System:** Rocky Linux 9 (or RHEL 9 derivative)
- **Architecture:** x86_64 (amd64)
- **OpenVPN:** Version 2.6.2 or later
- **Go:** Version 1.22+ (for building from source)
- **RAM:** Minimum 512MB (1GB recommended)
- **Disk Space:** 50MB for binary and dependencies

### Required Software

```bash
# Install OpenVPN 2.6+ from EPEL
sudo dnf install epel-release
sudo dnf install openvpn

# Verify version
openvpn --version | head -1
# Should show: OpenVPN 2.6.x or later
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

1. **Checks prerequisites** - Verifies OpenVPN 2.6+ is installed
2. **Creates user/group** - Creates `openvpn` system user and group
3. **Installs binary** - Copies to `/usr/local/bin/openvpn-keycloak-auth`
4. **Creates directories:**
   - `/etc/openvpn` - Configuration files
   - `/var/lib/openvpn-keycloak-auth` - Data directory
5. **Installs files:**
   - `/etc/openvpn/keycloak-sso.yaml` - Configuration (if not exists)
   - `/etc/openvpn/auth-keycloak.sh` - Auth script
   - `/etc/systemd/system/openvpn-keycloak-auth.service` - systemd unit
6. **Configures firewall** - Opens port 9000/tcp (if firewalld is running)
7. **Configures SELinux** - Sets file contexts (if SELinux is enabled)

### Installation Output

```
╔══════════════════════════════════════════════════════════════╗
║     OpenVPN Keycloak SSO - Installation Script             ║
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
keycloak:
  issuer_url: "https://keycloak.example.com/realms/myrealm"
  client_id: "openvpn"
  # ... other settings

http:
  listen_addr: "0.0.0.0:9000"
  callback_url: "https://vpn.example.com:9000/callback"
  # ... other settings
```

See [`config/openvpn-keycloak-auth.yaml.example`](../config/openvpn-keycloak-auth.yaml.example) for all available options.

### Validate Configuration

```bash
# Check configuration syntax and connectivity
sudo /usr/local/bin/openvpn-keycloak-auth check-config \
  --config /etc/openvpn/keycloak-sso.yaml
```

Expected output:

```
Configuration validation: PASSED
✓ YAML syntax valid
✓ All required fields present
✓ Keycloak issuer URL reachable
✓ OIDC discovery successful
✓ HTTP server configuration valid
✓ Socket path valid
```

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
{"status":"ok","version":"895062d"}
```

### Step 4: Test OIDC Discovery

```bash
# Check logs for OIDC discovery
sudo journalctl -u openvpn-keycloak-auth | grep "OIDC provider discovered"

# Should show:
INFO OIDC provider discovered issuer=https://keycloak.example.com/realms/myrealm
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

# Create credentials file
echo -e "testuser\nsso" > /tmp/test_creds
chmod 600 /tmp/test_creds

# Run auth script
/etc/openvpn/auth-keycloak.sh /tmp/test_creds
echo "Exit code: $?"

# Should show: Exit code: 2 (deferred)

# Check auth_pending_file
cat /tmp/test_apf

# Should show 3 lines:
# 300
# openurl
# WEB_AUTH::https://keycloak.example.com/realms/...

# Cleanup
rm -f /tmp/test_*
```

### Step 6: Full Integration Test

See [docs/openvpn-server-setup.md](./openvpn-server-setup.md) section "Testing the SSO Flow" for complete end-to-end testing.

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
   **Solution:** Check directory permissions, ensure RuntimeDirectory in service file

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
# Sessions should expire after session_ttl (default 5 minutes)
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
4. **Removes auth script** - Deletes `/etc/openvpn/auth-keycloak.sh`
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
sudo rm -f /etc/openvpn/auth-keycloak.sh

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
sudo mkdir -p /etc/openvpn
sudo mkdir -p /var/lib/openvpn-keycloak-auth
sudo chown openvpn:openvpn /var/lib/openvpn-keycloak-auth
sudo chmod 755 /var/lib/openvpn-keycloak-auth
```

### 4. Install Configuration

```bash
sudo install -m 600 config/openvpn-keycloak-auth.yaml.example \
  /etc/openvpn/keycloak-sso.yaml
sudo chown root:openvpn /etc/openvpn/keycloak-sso.yaml

# Edit configuration
sudo vim /etc/openvpn/keycloak-sso.yaml
```

### 5. Install Auth Script

```bash
sudo install -m 755 scripts/auth-keycloak.sh /etc/openvpn/auth-keycloak.sh
```

### 6. Install systemd Service

```bash
sudo install -m 644 deploy/openvpn-keycloak-auth.service \
  /etc/systemd/system/openvpn-keycloak-auth.service
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

The systemd service file includes extensive security hardening. Review these settings:

### Filesystem Protection

```ini
ProtectSystem=strict         # Read-only /usr, /boot, /efi
ProtectHome=true            # Inaccessible /home
ReadWritePaths=/var/lib/openvpn-keycloak-auth
PrivateTmp=true             # Private /tmp
```

### Kernel Protection

```ini
ProtectKernelTunables=true  # Read-only /proc/sys, /sys
ProtectKernelModules=true   # No kernel module loading
ProtectKernelLogs=true      # No access to kernel logs
ProtectControlGroups=true   # Read-only /sys/fs/cgroup
```

### Privilege Restrictions

```ini
NoNewPrivileges=true        # Can't gain new privileges
RestrictSUIDSGID=true       # SUID/SGID bits have no effect
LockPersonality=true        # No personality changes
PrivateUsers=true           # Private user namespace
```

### System Call Filtering

```ini
SystemCallFilter=@system-service
SystemCallFilter=~@privileged @resources @obsolete @debug @mount
```

### Capabilities

```ini
# If using port >= 1024 (default 9000), no capabilities needed
CapabilityBoundingSet=
AmbientCapabilities=

# If using port < 1024, uncomment:
# CapabilityBoundingSet=CAP_NET_BIND_SERVICE
# AmbientCapabilities=CAP_NET_BIND_SERVICE
```

### Resource Limits

```ini
LimitNOFILE=65536           # Max open files
LimitNPROC=512              # Max processes
TasksMax=512                # Max threads
```

### Additional Hardening

To further harden the system:

1. **Run on non-standard port** (reduces automated attacks)
   ```yaml
   http:
     listen_addr: "0.0.0.0:9443"  # Instead of 9000
   ```

2. **Use TLS for HTTP server** (requires certificates)
   ```yaml
   http:
     tls:
       cert_file: "/etc/openvpn/certs/server.crt"
       key_file: "/etc/openvpn/certs/server.key"
   ```

3. **Restrict callback URL to VPN network only**
   ```yaml
   http:
     listen_addr: "10.8.0.1:9000"  # VPN interface only
   ```

4. **Enable rate limiting** (already enabled in config)
   ```yaml
   http:
     rate_limit:
       requests_per_minute: 60
       burst: 10
   ```

5. **Regularly update** the binary and dependencies
   ```bash
   cd /path/to/source
   git pull
   make build
   sudo make install
sudo systemctl restart openvpn-keycloak-auth
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

1. **Configure Keycloak** - See [docs/keycloak-setup.md](./keycloak-setup.md)
2. **Configure OpenVPN Server** - See [docs/openvpn-server-setup.md](./openvpn-server-setup.md)
3. **Configure Clients** - See [docs/client-setup.md](./client-setup.md)
4. **Test SSO Flow** - Follow testing procedures in server setup guide

---

**Document Version:** 1.0  
**Last Updated:** 2026-02-15  
**Platform:** Rocky Linux 9

For questions or issues, consult the troubleshooting section or check project documentation.
