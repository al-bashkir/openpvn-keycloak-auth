# Security Checklist - OpenVPN Keycloak SSO

This checklist helps ensure your OpenVPN Keycloak SSO deployment is secure. Use this during initial deployment, security audits, and regular reviews.

## Quick Assessment

**Security Score:** ___/100

| Category | Weight | Score | Notes |
|----------|--------|-------|-------|
| Network Security | 20 | __/20 | |
| Authentication | 25 | __/25 | |
| Authorization | 15 | __/15 | |
| System Hardening | 20 | __/20 | |
| Logging & Monitoring | 10 | __/10 | |
| Maintenance | 10 | __/10 | |

**Grade:**
- 90-100: Excellent ✅
- 75-89: Good ✓
- 60-74: Needs Improvement ⚠️
- <60: Urgent Action Required ❌

---

## Network Security (20 points)

### TLS Configuration (8 points)

- [ ] **2 pts** - Keycloak accessible only via HTTPS
- [ ] **2 pts** - TLS 1.2 minimum enforced on Keycloak
- [ ] **2 pts** - Callback endpoint uses HTTPS (reverse proxy)
- [ ] **2 pts** - Valid SSL/TLS certificates (not self-signed in production)

**Verification:**
```bash
# Check Keycloak TLS
curl -I https://keycloak.example.com/realms/myrealm | grep -i "server:\|strict-transport"

# Check callback endpoint
curl -I https://vpn.example.com:9000/health

# Verify certificate validity
openssl s_client -connect keycloak.example.com:443 -showcerts </dev/null 2>/dev/null | grep -A2 "Verify return code"
```

### Firewall Configuration (6 points)

- [ ] **2 pts** - Firewall enabled (firewalld or iptables)
- [ ] **2 pts** - Only necessary ports open (OpenVPN, callback, SSH)
- [ ] **2 pts** - Outbound HTTPS allowed to Keycloak only

**Verification:**
```bash
# Check firewall status
sudo firewall-cmd --state

# List open ports
sudo firewall-cmd --list-all

# Verify only required ports
# Expected: 1194/udp (OpenVPN), 9000/tcp (callback), 22/tcp (SSH)
```

### Rate Limiting (3 points)

- [ ] **1 pt** - Per-IP rate limiting enabled in daemon
- [ ] **1 pt** - Firewall rate limiting configured
- [ ] **1 pt** - Brute force protection enabled in Keycloak

**Verification:**
```bash
# Check daemon rate limiting (in code/config)
grep -A3 "rateLimitMiddleware" internal/httpserver/server.go

# Test rate limiting
for i in {1..60}; do curl http://localhost:9000/health; done
# Should see "Rate limit exceeded" after ~50 requests
```

### Network Segmentation (3 points)

- [ ] **1 pt** - VPN server on dedicated network segment
- [ ] **1 pt** - Management network separate from VPN network
- [ ] **1 pt** - Callback endpoint not exposed to public internet (if possible)

---

## Authentication (25 points)

### OIDC Configuration (10 points)

- [ ] **2 pts** - PKCE enabled and required (S256 method)
- [ ] **2 pts** - Valid redirect URIs configured (exact match)
- [ ] **2 pts** - Client ID matches between Keycloak and daemon
- [ ] **2 pts** - Issuer URL correct and reachable
- [ ] **2 pts** - OIDC discovery working

**Verification:**
```bash
# Check daemon config
sudo grep -A5 "keycloak:" /etc/openvpn/keycloak-sso.yaml

# Test OIDC discovery
curl https://keycloak.example.com/realms/myrealm/.well-known/openid-configuration

# Verify configuration
/usr/local/bin/openvpn-keycloak-auth check-config --config /etc/openvpn/keycloak-sso.yaml
```

### Multi-Factor Authentication (8 points)

- [ ] **4 pts** - MFA enabled for all VPN users in Keycloak
- [ ] **2 pts** - TOTP (OTP) configured
- [ ] **2 pts** - WebAuthn/FIDO2 configured (bonus)

**Verification:**
```bash
# Check in Keycloak Admin Console:
# - Realm → Authentication → Required Actions → "Configure OTP" enabled
# - Realm → Authentication → Flows → Browser → OTP required
```

### Password Policy (4 points)

- [ ] **1 pt** - Minimum password length: 12+ characters
- [ ] **1 pt** - Password complexity required (uppercase, lowercase, numbers, symbols)
- [ ] **1 pt** - Password history enforced (no reuse of last 5 passwords)
- [ ] **1 pt** - Password expiration configured (90 days recommended)

**Verification:**
```bash
# Check in Keycloak Admin Console:
# - Realm → Authentication → Password Policy
```

### Session Management (3 points)

- [ ] **1 pt** - Session timeout ≤ 5 minutes (daemon)
- [ ] **1 pt** - Token lifetime ≤ 5 minutes (Keycloak)
- [ ] **1 pt** - Idle timeout configured (Keycloak)

**Verification:**
```bash
# Check daemon config
grep -i "session_ttl" /etc/openvpn/keycloak-sso.yaml

# Check Keycloak Admin Console:
# - Client → Settings → Access Token Lifespan
```

---

## Authorization (15 points)

### Role-Based Access Control (8 points)

- [ ] **2 pts** - Required roles configured in daemon
- [ ] **2 pts** - Roles assigned to users in Keycloak
- [ ] **2 pts** - Role membership verified in token validation
- [ ] **2 pts** - Principle of least privilege applied

**Verification:**
```bash
# Check daemon config
grep -A3 "required_roles" /etc/openvpn/keycloak-sso.yaml

# Test with user lacking required role (should fail)
```

### Group Management (4 points)

- [ ] **2 pts** - VPN users organized into groups
- [ ] **2 pts** - Group-based access control implemented

### Token Validation (3 points)

- [ ] **1 pt** - JWT signature validation enabled
- [ ] **1 pt** - All claims validated (iss, aud, exp, iat, nbf)
- [ ] **1 pt** - Username matching enforced (unless explicitly disabled)

**Verification:**
```bash
# Check code in internal/oidc/validator.go
# Verify ValidateToken function checks all claims
```

---

## System Hardening (20 points)

### File Permissions (8 points)

- [ ] **2 pts** - Config file: 0600, root:openvpn
- [ ] **2 pts** - Binary: 0755, root:root
- [ ] **2 pts** - Socket directory: 0770, openvpn:openvpn
- [ ] **2 pts** - Data directory: 0755, openvpn:openvpn

**Verification:**
```bash
# Run verification script
bash << 'EOF'
check() {
    local file="$1" expected_perms="$2" expected_owner="$3"
    [ ! -e "$file" ] && echo "❌ $file missing" && return 1
    local perms=$(stat -c '%a' "$file")
    local owner=$(stat -c '%U:%G' "$file")
    [ "$perms" != "$expected_perms" ] && echo "❌ $file: $perms (expected $expected_perms)" && return 1
    [ "$owner" != "$expected_owner" ] && echo "⚠️  $file: $owner (expected $expected_owner)"
    echo "✅ $file: $perms $owner"
}

check "/etc/openvpn/keycloak-sso.yaml" "600" "root:openvpn"
check "/usr/local/bin/openvpn-keycloak-auth" "755" "root:root"
check "/var/lib/openvpn-keycloak-auth" "755" "openvpn:openvpn"
EOF
```

### systemd Hardening (6 points)

- [ ] **1 pt** - NoNewPrivileges=true
- [ ] **1 pt** - ProtectSystem=strict
- [ ] **1 pt** - ProtectHome=true
- [ ] **1 pt** - SystemCallFilter enabled
- [ ] **1 pt** - PrivateTmp=true
- [ ] **1 pt** - CapabilityBoundingSet minimal or empty

**Verification:**
```bash
# Check service file
grep -E "NoNewPrivileges|ProtectSystem|PrivateTmp" /etc/systemd/system/openvpn-keycloak-auth.service

# Verify security score
systemd-analyze security openvpn-keycloak-auth.service | head -1
# Should show score < 3.0 (lower is better)
```

### SELinux (3 points)

- [ ] **1 pt** - SELinux enabled (Enforcing mode)
- [ ] **1 pt** - File contexts correct
- [ ] **1 pt** - No denials in audit log

**Verification:**
```bash
# Check SELinux status
getenforce  # Should show "Enforcing"

# Check file contexts
ls -Z /usr/local/bin/openvpn-keycloak-auth
# Should show: system_u:object_r:bin_t:s0

# Check for denials
ausearch -m avc -ts recent | grep openvpn-keycloak-auth
# Should show no results
```

### User Privileges (3 points)

- [ ] **1 pt** - Daemon runs as non-root user (openvpn)
- [ ] **1 pt** - Daemon runs with minimal group membership
- [ ] **1 pt** - No unnecessary capabilities granted

**Verification:**
```bash
# Check running process
ps aux | grep openvpn-keycloak-auth | grep -v grep
# Should show user: openvpn (not root)

# Check capabilities
sudo cat /proc/$(pgrep openvpn-keycloak-auth)/status | grep Cap
# All cap values should be 0000000000000000 (no capabilities)
```

---

## Logging & Monitoring (10 points)

### Log Security (4 points)

- [ ] **2 pts** - No secrets logged (tokens, passwords, client secrets)
- [ ] **2 pts** - Audit trail for all authentication attempts

**Verification:**
```bash
# Check for secrets in logs
journalctl -u openvpn-keycloak-auth --since "24 hours ago" \
  | grep -iE "token.*:[[:space:]]*ey|secret.*:[[:space:]]*[^[]"
# Should show no results

# Verify audit trail
journalctl -u openvpn-keycloak-auth --since "1 hour ago" \
  | grep -E "authenticated successfully|auth failure"
```

### Log Retention (2 points)

- [ ] **1 pt** - Log rotation configured
- [ ] **1 pt** - Logs retained for compliance period (30+ days)

**Verification:**
```bash
# Check journal configuration
cat /etc/systemd/journald.conf | grep -E "SystemMaxUse|MaxRetentionSec"

# Check current log size
journalctl --disk-usage
```

### Monitoring (4 points)

- [ ] **1 pt** - Failed authentication monitoring
- [ ] **1 pt** - Rate limit alerts configured
- [ ] **1 pt** - Service health checks
- [ ] **1 pt** - Anomaly detection (unusual login times/locations)

**Verification:**
```bash
# Check health endpoint
curl http://localhost:9000/health

# Monitor failed authentications
journalctl -u openvpn-keycloak-auth --since "1 hour ago" \
  | grep "auth failure" | wc -l
```

---

## Maintenance (10 points)

### Updates (4 points)

- [ ] **1 pt** - System packages up to date
- [ ] **1 pt** - OpenVPN up to date (2.6.2+)
- [ ] **1 pt** - Keycloak up to date (latest security patches)
- [ ] **1 pt** - Daemon up to date (latest release)

**Verification:**
```bash
# Check system updates
sudo dnf check-update

# Check OpenVPN version
openvpn --version | head -1

# Check daemon version
/usr/local/bin/openvpn-keycloak-auth version
```

### Backup & Recovery (3 points)

- [ ] **1 pt** - Configuration backed up
- [ ] **1 pt** - Certificates backed up (encrypted)
- [ ] **1 pt** - Recovery procedure tested

**Verification:**
```bash
# Check backup script exists
ls -l /usr/local/bin/backup-vpn-config.sh

# Test recovery (in non-prod environment)
```

### Documentation (3 points)

- [ ] **1 pt** - Network diagram current
- [ ] **1 pt** - Runbooks for common tasks
- [ ] **1 pt** - Incident response plan documented

---

## Automated Security Checks

Save this script as `/usr/local/bin/security-check.sh`:

```bash
#!/bin/bash
# Automated security checks for OpenVPN Keycloak SSO

SCORE=0
MAX_SCORE=100

echo "╔════════════════════════════════════════════════════════════╗"
echo "║        OpenVPN Keycloak SSO - Security Check              ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Network Security (20 points)
echo "=== Network Security ==="

# TLS on Keycloak (2 pts)
if curl -s -I https://keycloak.example.com/realms/myrealm 2>/dev/null | grep -q "HTTP/2 200"; then
    echo "✅ Keycloak HTTPS: OK (+2)"
    ((SCORE+=2))
else
    echo "❌ Keycloak HTTPS: FAIL"
fi

# Firewall enabled (2 pts)
if systemctl is-active --quiet firewalld || systemctl is-active --quiet iptables; then
    echo "✅ Firewall enabled: OK (+2)"
    ((SCORE+=2))
else
    echo "❌ Firewall enabled: FAIL"
fi

# Rate limiting (3 pts)
if grep -q "rateLimitMiddleware" /home/*/Projects/WORK/openvpn-sso-plugin-v3/internal/httpserver/server.go 2>/dev/null; then
    echo "✅ Rate limiting: OK (+3)"
    ((SCORE+=3))
else
    echo "⚠️  Rate limiting: Not verified"
fi

echo ""

# Authentication (25 points)
echo "=== Authentication ==="

# PKCE in config (2 pts)
if grep -q "S256" /home/*/Projects/WORK/openvpn-sso-plugin-v3/internal/oidc/flow.go 2>/dev/null; then
    echo "✅ PKCE enabled: OK (+2)"
    ((SCORE+=2))
else
    echo "❌ PKCE enabled: FAIL"
fi

# Config valid (2 pts)
if /usr/local/bin/openvpn-keycloak-auth check-config --config /etc/openvpn/keycloak-sso.yaml 2>/dev/null | grep -q "PASSED"; then
    echo "✅ Configuration valid: OK (+2)"
    ((SCORE+=2))
else
    echo "❌ Configuration valid: FAIL"
fi

echo ""

# System Hardening (20 points)
echo "=== System Hardening ==="

# Config permissions (2 pts)
if [ "$(stat -c '%a' /etc/openvpn/keycloak-sso.yaml 2>/dev/null)" = "600" ]; then
    echo "✅ Config permissions: OK (+2)"
    ((SCORE+=2))
else
    echo "❌ Config permissions: FAIL"
fi

# SELinux (3 pts)
if command -v getenforce >/dev/null && [ "$(getenforce)" = "Enforcing" ]; then
    echo "✅ SELinux enforcing: OK (+3)"
    ((SCORE+=3))
else
    echo "⚠️  SELinux enforcing: Not enforcing"
fi

# Service runs as non-root (1 pt)
if ps aux | grep -v grep | grep openvpn-keycloak-auth | grep -q "^openvpn"; then
    echo "✅ Non-root service: OK (+1)"
    ((SCORE+=1))
else
    echo "❌ Non-root service: FAIL"
fi

echo ""

# Logging (10 points)
echo "=== Logging & Monitoring ==="

# No secrets in logs (2 pts)
if ! journalctl -u openvpn-keycloak-auth --since "24 hours ago" 2>/dev/null \
  | grep -iE "token.*:[[:space:]]*ey|secret.*:[[:space:]]*[^[]" >/dev/null; then
    echo "✅ No secrets in logs: OK (+2)"
    ((SCORE+=2))
else
    echo "❌ No secrets in logs: FAIL"
fi

# Service health (1 pt)
if curl -s http://localhost:9000/health 2>/dev/null | grep -q "ok"; then
    echo "✅ Service health: OK (+1)"
    ((SCORE+=1))
else
    echo "❌ Service health: FAIL"
fi

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "Final Score: $SCORE / $MAX_SCORE"
echo ""

if [ $SCORE -ge 90 ]; then
    echo "Grade: ✅ EXCELLENT (90-100)"
elif [ $SCORE -ge 75 ]; then
    echo "Grade: ✓ GOOD (75-89)"
elif [ $SCORE -ge 60 ]; then
    echo "Grade: ⚠️  NEEDS IMPROVEMENT (60-74)"
else
    echo "Grade: ❌ URGENT ACTION REQUIRED (<60)"
fi

echo ""
exit 0
```

Make executable:
```bash
sudo chmod +x /usr/local/bin/security-check.sh
```

Run regularly:
```bash
sudo /usr/local/bin/security-check.sh
```

---

## Compliance Frameworks

### PCI DSS

If processing payment card data over VPN:

- [ ] Strong authentication (MFA) ✅ Supported
- [ ] Encryption in transit (TLS) ✅ Supported
- [ ] Audit logging ✅ Supported
- [ ] Access control (role-based) ✅ Supported
- [ ] Regular security testing ⚠️ Manual

### HIPAA

If transmitting PHI over VPN:

- [ ] Access control (unique user IDs) ✅ Supported
- [ ] Audit logging ✅ Supported
- [ ] Encryption ✅ Supported
- [ ] Session timeout ✅ Supported
- [ ] Emergency access procedure ⚠️ Manual

### GDPR

For EU user data:

- [ ] Data minimization (only log necessary data) ✅ Implemented
- [ ] Right to erasure (user deletion) ⚠️ Manual in Keycloak
- [ ] Data breach notification ⚠️ Manual process
- [ ] Privacy by design ✅ Implemented

---

## Next Steps

Based on your security assessment:

**Score 90-100 (Excellent):**
- Maintain current posture
- Regular quarterly reviews
- Stay updated on security advisories

**Score 75-89 (Good):**
- Address any failed checks
- Implement missing MFA if not enabled
- Review and update documentation

**Score 60-74 (Needs Improvement):**
- Prioritize failed checks in Network and Authentication sections
- Schedule security audit
- Implement missing controls within 30 days

**Score <60 (Urgent):**
- Stop deployment to production
- Fix critical issues immediately
- Engage security team for review
- Retest before deployment

---

**Document Version:** 1.0  
**Last Updated:** 2026-02-15  
**Review Frequency:** Quarterly

For questions about this checklist, see [docs/security.md](security.md) or consult your security team.
