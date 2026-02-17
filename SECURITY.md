# Security Policy

## Supported Versions

We release security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

**Note:** This project is currently in active development. We recommend always using the latest version from the `main` branch.

## Security Considerations

The OpenVPN Keycloak SSO authentication daemon handles sensitive authentication data and integrates with critical infrastructure (VPN access, identity provider). We take security very seriously.

### Critical Security Features

This project implements several security measures:

- **PKCE (RFC 7636)** - Proof Key for Code Exchange prevents authorization code interception
- **CSRF Protection** - State parameter validation prevents cross-site request forgery
- **JWT Validation** - Full signature and claims validation
- **Rate Limiting** - Per-IP rate limiting prevents brute force and DoS
- **No Password Transmission** - User passwords never sent to VPN server
- **Secure Defaults** - All security features enabled by default
- **systemd Hardening** - Extensive sandboxing and privilege restrictions

For detailed security documentation, see [docs/security.md](docs/security.md).

## Reporting a Vulnerability

**We take all security reports seriously.**

If you discover a security vulnerability, please follow responsible disclosure:

### DO NOT:

- ❌ Open a public GitHub issue
- ❌ Discuss the vulnerability publicly
- ❌ Exploit the vulnerability

### DO:

1. **Report privately** via one of these methods:

   **Preferred: GitHub Security Advisory**
   - Go to the [Security tab](../../security/advisories)
   - Click "Report a vulnerability"
   - Fill out the form with details

   **Alternative: Email**
   - Send to: [your-security-email@example.com]
   - Subject: "OpenVPN Keycloak SSO Security Vulnerability"
   - Use PGP encryption if possible (key below)

2. **Include these details:**
   - Description of the vulnerability
   - Steps to reproduce
   - Affected versions
   - Potential impact
   - Suggested fix (if you have one)
   - Your contact information (for follow-up)

3. **Allow time for response:**
   - We aim to acknowledge within **48 hours**
   - We aim to provide initial assessment within **7 days**
   - We aim to release a fix within **30 days** for critical issues

### Vulnerability Severity

We classify vulnerabilities using CVSS 3.1:

| Severity | CVSS Score | Examples | Response Time |
|----------|------------|----------|---------------|
| **Critical** | 9.0-10.0 | Remote code execution, authentication bypass | 7 days |
| **High** | 7.0-8.9 | Privilege escalation, token theft | 14 days |
| **Medium** | 4.0-6.9 | Information disclosure, DoS | 30 days |
| **Low** | 0.1-3.9 | Minor issues | 90 days |

## What Happens After You Report

1. **Acknowledgment** - We'll confirm receipt of your report (48 hours)
2. **Investigation** - We'll verify and assess the vulnerability (7 days)
3. **Development** - We'll develop and test a fix
4. **Coordination** - We'll coordinate disclosure timeline with you
5. **Release** - We'll release a security patch
6. **Disclosure** - We'll publish a security advisory
7. **Credit** - We'll credit you in the advisory (if desired)

## Security Advisories

Published security advisories can be found at:
- [GitHub Security Advisories](../../security/advisories)
- [CHANGELOG.md](CHANGELOG.md) - Security fixes section

## Bug Bounty Program

**Status:** We do not currently offer a bug bounty program.

However, we deeply appreciate security researchers who responsibly disclose vulnerabilities. We will:
- Publicly credit you in the security advisory (with your permission)
- Thank you in our CHANGELOG
- Provide a sincere appreciation for your efforts

## Scope

### In Scope

Vulnerabilities in these areas are in scope:

- **Authentication bypass** - Bypassing SSO authentication
- **Authorization bypass** - Accessing VPN without proper authentication
- **Token theft** - Stealing or forging tokens
- **Injection attacks** - SQL, command, LDAP, etc.
- **Information disclosure** - Leaking secrets, tokens, or user data
- **Denial of Service** - Crashing or degrading the service
- **Cryptographic issues** - Weak crypto, improper use of crypto
- **Code execution** - Remote or local code execution
- **Privilege escalation** - Gaining elevated privileges
- **CSRF/SSRF** - Cross-site or server-side request forgery

### Out of Scope

The following are out of scope:

- **Social engineering** - Phishing, etc.
- **Physical attacks** - Physical access to servers
- **DoS requiring massive resources** - DDoS with botnets
- **Issues in dependencies** - Report to upstream projects
- **Keycloak vulnerabilities** - Report to Keycloak project
- **OpenVPN vulnerabilities** - Report to OpenVPN project
- **Theoretical attacks** - Without proof of concept
- **Already disclosed** - CVEs from dependencies we're aware of
- **Self-XSS** - Requires user to paste malicious code
- **Rate limiting bypasses** - Unless allowing critical attacks

**Note:** If you're unsure whether something is in scope, please report it anyway. We'll assess and provide guidance.

## Safe Harbor

We support security researchers who:

- Make a good faith effort to avoid privacy violations, data destruction, and service interruption
- Only interact with accounts you own or with explicit permission
- Don't exploit a vulnerability beyond the minimum necessary to demonstrate it
- Report vulnerabilities promptly
- Keep vulnerability details confidential until we've addressed them

We will not pursue legal action against researchers who follow these guidelines.

## Security Best Practices

For deployers:

1. **Always use HTTPS** for the callback endpoint (TLS termination at reverse proxy)
2. **Enable MFA** in Keycloak for all VPN users
3. **Keep updated** - Apply security patches promptly
4. **Monitor logs** - Watch for unusual authentication patterns
5. **Use strong ciphers** - In both OpenVPN and TLS configurations
6. **Restrict access** - Firewall rules, network segmentation
7. **Regular audits** - Review configurations and logs regularly

See [docs/security.md](docs/security.md) for comprehensive security guidance.

## Contact

- **Security Issues:** [GitHub Security Advisory](../../security/advisories)
- **General Questions:** [GitHub Issues](../../issues)
- **Project Maintainer:** [@al-bashkir](https://github.com/al-bashkir)

## PGP Key

For encrypted communications:

```
-----BEGIN PGP PUBLIC KEY BLOCK-----
[Your PGP public key here]
-----END PGP PUBLIC KEY BLOCK-----
```

**Fingerprint:** `XXXX XXXX XXXX XXXX XXXX  XXXX XXXX XXXX XXXX XXXX`

## Acknowledgments

We thank the following security researchers for responsibly disclosing vulnerabilities:

- *No reported vulnerabilities yet*

---

**Last Updated:** 2026-02-15  
**Version:** 1.0

Thank you for helping keep OpenVPN Keycloak SSO and our users safe!
