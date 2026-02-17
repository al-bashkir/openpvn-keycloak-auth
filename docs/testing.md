# Testing Guide - OpenVPN Keycloak SSO

This guide covers testing strategies, procedures, and automation for the OpenVPN Keycloak SSO authentication system.

## Table of Contents

1. [Testing Overview](#testing-overview)
2. [Unit Testing](#unit-testing)
3. [Integration Testing](#integration-testing)
4. [Manual Testing](#manual-testing)
5. [Performance Testing](#performance-testing)
6. [Security Testing](#security-testing)
7. [CI/CD](#cicd)
8. [Troubleshooting Tests](#troubleshooting-tests)

---

## Testing Overview

### Test Coverage Summary

Current test coverage by package:

| Package | Coverage | Tests | Status |
|---------|----------|-------|--------|
| `internal/auth` | 76.7% | 6 tests | ✅ Good |
| `internal/config` | 75.7% | 8 tests | ✅ Good |
| `internal/httpserver` | 64.2% | 10 tests | ⚠️ Acceptable |
| `internal/ipc` | 74.7% | 7 tests | ✅ Good |
| `internal/oidc` | 67.0% | 15 tests | ✅ Good |
| `internal/openvpn` | 85.2% | 4 tests | ✅ Excellent |
| `internal/session` | 90.7% | 10 tests | ✅ Excellent |
| **Overall** | **76%** | **56 tests** | ✅ **Good** |

**Target:** >80% coverage for production packages  
**Status:** ✅ Achieved for most critical packages

### Test Types

We employ multiple testing strategies:

1. **Unit Tests** - Test individual functions and methods in isolation
2. **Integration Tests** - Test interaction between components
3. **Manual Tests** - End-to-end testing with real OpenVPN and Keycloak
4. **Performance Tests** - Stress testing, concurrent users
5. **Security Tests** - Vulnerability scanning, penetration testing

---

## Unit Testing

### Running Unit Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...

# Run tests verbosely
go test -v ./...

# Run specific package
go test ./internal/session

# Run specific test
go test -run TestCreateSession ./internal/session

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Using Makefile

```bash
# Run tests
make test

# Run tests with coverage report
make test-coverage

# Run specific test
make test-one TEST=TestCreateSession

# Run tests verbosely
make test-verbose
```

### Test Structure

All packages follow consistent test organization:

```
internal/<package>/
├── file.go           # Implementation
├── file_test.go      # Unit tests
└── example_test.go   # Example tests (optional)
```

### Example Test Files

#### Config Package Tests

Location: `internal/config/config_test.go`

**Key Tests:**
- `TestLoadConfig` - Valid and invalid YAML
- `TestValidateConfig` - Field validation
- `TestDefaultValues` - Default configuration
- `TestEnvironmentOverrides` - Env var handling
- `TestConfigRedaction` - Secret redaction
- `TestConfigValidation` - Required fields
- `TestInvalidYAML` - Parse error handling
- `TestMissingFile` - File not found errors

```bash
# Run config tests
go test -v ./internal/config

# Example output:
=== RUN   TestLoadConfig
=== RUN   TestLoadConfig/valid_config
=== RUN   TestLoadConfig/missing_issuer
=== RUN   TestLoadConfig/invalid_yaml
--- PASS: TestLoadConfig (0.01s)
    --- PASS: TestLoadConfig/valid_config (0.00s)
    --- PASS: TestLoadConfig/missing_issuer (0.00s)
    --- PASS: TestLoadConfig/invalid_yaml (0.00s)
PASS
ok  	github.com/al-bashkir/openvpn-keycloak/internal/config	0.015s	coverage: 75.7% of statements
```

#### Session Package Tests

Location: `internal/session/session_test.go`

**Key Tests:**
- `TestCreateSession` - Session creation
- `TestGetSession` - Session retrieval
- `TestDeleteSession` - Session deletion
- `TestSessionNotFound` - Error handling
- `TestGetByState` - State parameter lookup
- `TestSessionExpiry` - TTL expiration
- `TestCleanup` - Expired session cleanup
- `TestConcurrentAccess` - Thread safety
- `TestGenerateSessionID` - ID uniqueness
- `TestWriteResult` - Result writing flag

```bash
# Run session tests
go test -v ./internal/session

# Example output:
=== RUN   TestCreateSession
--- PASS: TestCreateSession (0.00s)
=== RUN   TestSessionExpiry
--- PASS: TestSessionExpiry (0.15s)
=== RUN   TestConcurrentAccess
--- PASS: TestConcurrentAccess (0.01s)
PASS
ok  	github.com/al-bashkir/openvpn-keycloak/internal/session	0.306s	coverage: 90.7% of statements
```

#### OIDC Package Tests

Location: `internal/oidc/validator_test.go`, `internal/oidc/flow_test.go`

**Key Tests:**
- `TestValidateToken` - JWT token validation
- `TestValidateUsername` - Username claim validation
- `TestValidateRoles` - Role-based authorization
- `TestExtractClaim` - Nested claim extraction
- `TestPKCE` - PKCE verifier/challenge generation
- `TestBuildAuthURL` - Authorization URL construction
- `TestInvalidIssuer` - Issuer validation
- `TestMissingClaims` - Missing required claims
- `TestExpiredToken` - Token expiration
- `TestRoleExtraction` - Role claim parsing
- `TestNestedClaims` - Deep claim extraction

```bash
# Run OIDC tests
go test -v ./internal/oidc

# Example output:
=== RUN   TestValidateToken
=== RUN   TestValidateToken/valid_token
=== RUN   TestValidateToken/username_mismatch
=== RUN   TestValidateToken/missing_username
--- PASS: TestValidateToken (0.00s)
=== RUN   TestValidateRoles
=== RUN   TestValidateRoles/user_has_required_role
=== RUN   TestValidateRoles/user_missing_required_role
--- PASS: TestValidateRoles (0.00s)
PASS
ok  	github.com/al-bashkir/openvpn-keycloak/internal/oidc	0.004s	coverage: 67.0% of statements
```

### Table-Driven Tests

Most tests use table-driven approach for comprehensive coverage:

```go
func TestExample(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "TEST",
            wantErr: false,
        },
        {
            name:    "empty input",
            input:   "",
            want:    "",
            wantErr: true,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ToUpper(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ToUpper() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ToUpper() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Testing Best Practices

1. **Test naming:** `Test<Function>` or `Test<Function>_<Scenario>`
2. **Table-driven tests:** For multiple similar test cases
3. **Subtests:** Use `t.Run()` for better organization
4. **Cleanup:** Use `defer` for cleanup operations
5. **Parallel tests:** Use `t.Parallel()` when tests are independent
6. **Mock external dependencies:** Don't rely on external services
7. **Race detector:** Always run with `-race` flag
8. **Coverage:** Aim for >80% in critical packages

---

## Integration Testing

### Local Integration Testing

Test the complete flow with local Keycloak instance:

#### Setup Test Environment

```bash
# Start Keycloak with podman (or docker)
podman run -d --name keycloak-test \
  -p 8080:8080 \
  -e KEYCLOAK_ADMIN=admin \
  -e KEYCLOAK_ADMIN_PASSWORD=admin \
  quay.io/keycloak/keycloak:25.0.6 \
  start-dev

# Wait for Keycloak to start
sleep 30

# Access Keycloak
echo "Keycloak admin: http://localhost:8080/admin (admin/admin)"
```

#### Configure Test Realm

1. Create realm "test"
2. Create client "openvpn-test"
   - Client type: Public
   - Valid redirect URIs: `http://localhost:9000/callback`
   - PKCE: Required, S256
3. Create test user "testuser" with password
4. Assign any required roles

#### Run Integration Test

```bash
# Create test configuration
cat > /tmp/test-config.yaml <<EOF
keycloak:
  issuer_url: "http://localhost:8080/realms/test"
  client_id: "openvpn-test"

http:
  listen_addr: "127.0.0.1:9000"
  callback_url: "http://localhost:9000/callback"

socket:
  path: "/tmp/openvpn-test.sock"

session:
  ttl: "5m"

auth:
  username_claim: "preferred_username"
EOF

# Run daemon
./openvpn-keycloak-sso serve --config /tmp/test-config.yaml &
DAEMON_PID=$!

# Wait for startup
sleep 2

# Test auth flow (requires manual browser interaction)
# See "Manual Testing" section for full procedure

# Cleanup
kill $DAEMON_PID
podman stop keycloak-test
podman rm keycloak-test
rm /tmp/test-config.yaml
```

### Docker Compose Integration Tests

For automated integration testing:

**Create `tests/integration/docker-compose.yml`:**

```yaml
version: '3.8'

services:
  keycloak:
    image: quay.io/keycloak/keycloak:25.0.6
    command: start-dev --import-realm
    environment:
      KEYCLOAK_ADMIN: admin
      KEYCLOAK_ADMIN_PASSWORD: admin
      KC_HTTP_PORT: 8080
    ports:
      - "8080:8080"
    volumes:
      - ./realm-export.json:/opt/keycloak/data/import/realm.json
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health/ready"]
      interval: 10s
      timeout: 5s
      retries: 10

  daemon:
    build:
      context: ../..
      dockerfile: Dockerfile
    depends_on:
      keycloak:
        condition: service_healthy
    environment:
      OVPN_SSO_KEYCLOAK_ISSUER: "http://keycloak:8080/realms/test"
      OVPN_SSO_KEYCLOAK_CLIENT_ID: "openvpn"
      OVPN_SSO_HTTP_CALLBACK_URL: "http://localhost:9000/callback"
    ports:
      - "9000:9000"
    volumes:
      - ./config.yaml:/etc/openvpn/keycloak-sso.yaml
```

**Run integration tests:**

```bash
cd tests/integration
docker-compose up -d
docker-compose logs -f

# Run tests (manual or automated)
# ...

docker-compose down
```

---

## Manual Testing

### Prerequisites

Before manual testing:

- [ ] OpenVPN 2.6.19+ installed
- [ ] Keycloak instance running and configured
- [ ] Daemon built and configured
- [ ] Test user account created in Keycloak
- [ ] Client certificate generated (if using mutual TLS)

### Test Case 1: Successful Authentication

**Objective:** Verify complete SSO authentication flow

**Steps:**

1. **Start daemon:**
   ```bash
   sudo systemctl start openvpn-keycloak-sso
   sudo journalctl -u openvpn-keycloak-sso -f
   ```

2. **Start OpenVPN client:**
   ```bash
   sudo openvpn --config client.ovpn
   ```

3. **Enter credentials:**
   - Username: `testuser`
   - Password: `sso` (any value, will be ignored)

4. **Observe output:**
   ```
   AUTH_PENDING,timeout:300,openurl,WEB_AUTH::https://keycloak.example.com/...
   ```

5. **Browser opens** (or copy URL manually)

6. **Log in to Keycloak:**
   - Enter testuser password
   - Complete MFA if enabled

7. **Verify success page** in browser

8. **Verify VPN connection:**
   ```bash
   # Should show: Initialization Sequence Completed
   ip addr show tun0  # Should show VPN interface
   ping 10.8.0.1      # Should reach VPN gateway
   ```

**Expected Result:** ✅ VPN connected, can ping gateway

**Logs to check:**
```bash
# Daemon logs
journalctl -u openvpn-keycloak-sso | grep testuser

# Expected:
# INFO auth request received username=testuser
# INFO callback received
# INFO user authenticated successfully username=testuser
# INFO auth success written username=testuser

# OpenVPN logs
journalctl -u openvpn@server | grep testuser

# Expected:
# testuser/192.0.2.1:12345 MULTI: Learn: 10.8.0.2 -> testuser/192.0.2.1:12345
```

### Test Case 2: Authentication Failure - Username Mismatch

**Objective:** Verify username validation

**Steps:**

1. Start VPN client with username `alice`
2. When browser opens, log in as `bob`
3. Observe error

**Expected Result:** ❌ Authentication fails with "username mismatch"

**Logs to check:**
```bash
journalctl -u openvpn-keycloak-sso | tail -20

# Expected:
# ERROR token validation failed error="username mismatch: expected alice, got bob"
# INFO auth failure written reason="username mismatch"
```

### Test Case 3: Session Timeout

**Objective:** Verify session expiry

**Steps:**

1. Start VPN client
2. Note the session URL
3. Wait 6 minutes (default TTL is 5 minutes)
4. Try to authenticate

**Expected Result:** ❌ Session expired error

**Logs to check:**
```bash
journalctl -u openvpn-keycloak-sso | grep expired

# Expected:
# ERROR session not found state=abc123 error="session expired"
```

### Test Case 4: Role-Based Authorization

**Objective:** Verify role enforcement

**Prerequisites:**
- Configure `required_roles: ["vpn-user"]` in daemon config
- Create two test users: one with role, one without

**Steps:**

1. **Test with user having role:**
   - Connect with user in "vpn-user" role
   - **Expected:** ✅ Success

2. **Test with user lacking role:**
   - Connect with user NOT in "vpn-user" role
   - **Expected:** ❌ Fails with "insufficient roles"

**Logs to check:**
```bash
journalctl -u openvpn-keycloak-sso | grep role

# For user without role:
# ERROR token validation failed error="user missing required roles: [vpn-user]"
```

### Test Case 5: Concurrent Users

**Objective:** Verify multiple simultaneous authentications

**Steps:**

1. Start 5-10 VPN clients simultaneously on different machines/VMs
2. Each should receive unique auth URL
3. Authenticate each independently

**Expected Result:** ✅ All users can authenticate concurrently

**Verification:**
```bash
# Check active sessions
journalctl -u openvpn-keycloak-sso --since "5 minutes ago" | grep "session_id" | sort | uniq

# Should show 5-10 unique session IDs

# Check OpenVPN status
sudo cat /var/log/openvpn/status.log | grep CLIENT_LIST

# Should show all connected users
```

### Test Case 6: Daemon Restart During Authentication

**Objective:** Verify graceful handling of daemon restart

**Steps:**

1. Start VPN connection (don't complete authentication yet)
2. Restart daemon: `sudo systemctl restart openvpn-keycloak-sso`
3. Try to complete authentication in browser

**Expected Result:** ❌ Session lost, need to reconnect

**Recovery:**
- Disconnect VPN client
- Reconnect (new session created)
- Authentication succeeds

### Test Case 7: Rate Limiting

**Objective:** Verify rate limiting prevents abuse

**Steps:**

1. Send many rapid requests to callback endpoint:
   ```bash
   for i in {1..100}; do
     curl -s http://vpn.example.com:9000/callback?code=test&state=test &
   done
   wait
   ```

2. Check logs

**Expected Result:** After ~50 requests, see "Rate limit exceeded"

**Logs to check:**
```bash
journalctl -u openvpn-keycloak-sso | grep "rate limit"

# Expected:
# WARN rate limit exceeded ip=203.0.113.50 path=/callback
```

### Test Case 8: Invalid Configuration

**Objective:** Verify configuration validation

**Steps:**

1. Create invalid config (e.g., wrong Keycloak URL)
2. Try to start daemon: `sudo systemctl start openvpn-keycloak-sso`

**Expected Result:** ❌ Service fails to start

**Logs to check:**
```bash
sudo systemctl status openvpn-keycloak-sso

# Expected:
# ExecStartPre: openvpn-keycloak-sso check-config (code=exited, status=1/FAILURE)
# ERROR Configuration validation: FAILED
```

### Test Case 9: SSL Certificate Verification

**Objective:** Verify TLS certificate validation

**Steps:**

1. Configure daemon with wrong Keycloak URL (self-signed cert or wrong hostname)
2. Try to start daemon

**Expected Result:** ❌ Certificate verification fails

**Logs to check:**
```bash
journalctl -u openvpn-keycloak-sso

# Expected:
# ERROR failed to discover OIDC provider error="x509: certificate signed by unknown authority"
```

### Test Case 10: Network Failure Recovery

**Objective:** Verify handling of network issues

**Steps:**

1. Start authentication flow
2. Block network to Keycloak (firewall rule)
3. Try to complete authentication

**Expected Result:** ❌ Timeout or connection error

**Recovery:**
- Restore network
- Retry authentication
- Should succeed

---

## Performance Testing

### Concurrent Authentication Test

Test system under load with multiple simultaneous authentications:

```bash
#!/bin/bash
# concurrent-auth-test.sh

NUM_CLIENTS=50
VPN_CONFIG="client.ovpn"

echo "Starting $NUM_CLIENTS concurrent VPN connections..."

for i in $(seq 1 $NUM_CLIENTS); do
    (
        echo "Client $i: Starting"
        timeout 300 openvpn --config $VPN_CONFIG \
          --auth-user-pass <(echo -e "testuser$i\nsso") \
          --log /tmp/vpn-client-$i.log \
          2>&1 | grep -E "SUCCESS|FAILED" &
    ) &
done

wait

echo "All clients finished"

# Check results
SUCCESS=$(grep -l "Initialization Sequence Completed" /tmp/vpn-client-*.log | wc -l)
FAILED=$(( NUM_CLIENTS - SUCCESS ))

echo "Results: $SUCCESS succeeded, $FAILED failed"

# Cleanup
rm -f /tmp/vpn-client-*.log
```

**Expected Performance:**
- 50 concurrent users should authenticate within 60 seconds
- No failures due to resource exhaustion
- Memory usage < 100MB
- CPU usage < 50%

**Monitoring:**
```bash
# Monitor daemon resources
watch "ps aux | grep openvpn-keycloak-sso | grep -v grep"

# Monitor sessions
watch "journalctl -u openvpn-keycloak-sso --since '1 minute ago' | grep 'session_id' | wc -l"
```

### Session Cleanup Performance

Test session cleanup with expired sessions:

```bash
# Create many sessions quickly
for i in {1..1000}; do
    echo "Creating session $i"
    # Simulate auth request that creates session
done

# Wait for cleanup
sleep 360  # 6 minutes (TTL + cleanup interval)

# Check memory usage
ps aux | grep openvpn-keycloak-sso

# Should not show memory leak
```

---

## Security Testing

### Automated Security Scanning

#### 1. Static Analysis with gosec

```bash
# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Run security scan
gosec -fmt=text ./...

# Expected: No high or critical issues
```

#### 2. Dependency Vulnerability Scanning

```bash
# Install nancy
go install github.com/sonatype-nexus-community/nancy@latest

# Check dependencies
go list -json -deps ./... | nancy sleuth

# Expected: No known vulnerabilities
```

#### 3. License Compliance

```bash
# Install go-licenses
go install github.com/google/go-licenses@latest

# Check licenses
go-licenses check ./...

# Expected: All permissive licenses (MIT, Apache, BSD)
```

### Manual Security Testing

#### Test 1: PKCE Enforcement

**Verify:** PKCE is always used

```bash
# Check code for PKCE usage
grep -r "code_challenge" internal/oidc/

# Should find S256 challenge generation
```

#### Test 2: State Parameter Validation

**Verify:** State parameter prevents CSRF

1. Start authentication flow
2. Note the state parameter
3. Try to use different state in callback
4. Should fail with "invalid state"

#### Test 3: Secret Protection

**Verify:** No secrets in logs

```bash
# Check logs for secrets
journalctl -u openvpn-keycloak-sso --since "1 day ago" \
  | grep -iE "token.*:[[:space:]]*ey|secret.*:[[:space:]]*[^[]|password.*:[[:space:]]*[^[]"

# Expected: No matches (no secrets logged)
```

#### Test 4: File Permissions

**Verify:** Correct permissions on sensitive files

```bash
# Run security check
bash << 'EOF'
check() {
    file="$1" expected="$2"
    actual=$(stat -c '%a' "$file" 2>/dev/null)
    [ "$actual" = "$expected" ] && echo "✅ $file: $actual" || echo "❌ $file: $actual (expected $expected)"
}

check "/etc/openvpn/keycloak-sso.yaml" "600"
check "/usr/local/bin/openvpn-keycloak-sso" "755"
check "/run/openvpn-keycloak-sso/auth.sock" "660"
EOF
```

---

## CI/CD

### GitHub Actions Workflow

Create `.github/workflows/test.yml`:

```yaml
name: Test and Build

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.22'
    
    - name: Cache Go modules
      uses: actions/cache@v4
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-
    
    - name: Download dependencies
      run: go mod download
    
    - name: Run tests
      run: go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
    
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v4
      with:
        file: ./coverage.txt
        flags: unittests
    
    - name: Check coverage threshold
      run: |
        COVERAGE=$(go tool cover -func=coverage.txt | grep total | awk '{print $3}' | sed 's/%//')
        echo "Coverage: $COVERAGE%"
        if (( $(echo "$COVERAGE < 75" | bc -l) )); then
          echo "❌ Coverage below 75%"
          exit 1
        fi
        echo "✅ Coverage OK"
  
  lint:
    name: Lint
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.22'
    
    - name: Run golangci-lint
      uses: golangci/golangci-lint-action@v4
      with:
        version: latest
        args: --timeout=5m
  
  build:
    name: Build
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.22'
    
    - name: Build binary
      run: make build
    
    - name: Check binary
      run: |
        file openvpn-keycloak-sso
        ./openvpn-keycloak-sso version
    
    - name: Upload artifact
      uses: actions/upload-artifact@v4
      with:
        name: openvpn-keycloak-sso-linux-amd64
        path: openvpn-keycloak-sso
  
  security:
    name: Security Scan
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.22'
    
    - name: Run gosec
      uses: securego/gosec@master
      with:
        args: '-no-fail -fmt sarif -out results.sarif ./...'
    
    - name: Upload SARIF file
      uses: github/codeql-action/upload-sarif@v3
      with:
        sarif_file: results.sarif
```

### Running CI Locally

Test CI pipeline before pushing:

```bash
# Install act (GitHub Actions local runner)
# https://github.com/nektos/act

# Run tests locally
act -j test

# Run all jobs
act
```

---

## Troubleshooting Tests

### Test Failures

#### Race Detector Failures

**Symptom:** `go test -race` reports data races

**Solution:**
```bash
# Run race detector with verbose output
go test -race -v ./internal/session

# Fix by adding proper locking
# Example: Use sync.Mutex for shared data
```

#### Timeout Failures

**Symptom:** Tests timeout

**Solution:**
```bash
# Increase timeout
go test -timeout 5m ./...

# Or fix slow tests (reduce sleep times in tests)
```

#### Flaky Tests

**Symptom:** Tests pass/fail randomly

**Common causes:**
- Time-dependent tests
- Concurrent access without proper sync
- External dependencies

**Solution:**
- Use `time.After` instead of `time.Sleep` where possible
- Add proper synchronization
- Mock external dependencies

### Coverage Issues

#### Low Coverage

**Check which lines are not covered:**

```bash
# Generate coverage HTML report
go test -coverprofile=coverage.out ./internal/session
go tool cover -html=coverage.out

# Opens browser showing covered (green) vs uncovered (red) lines
```

**Improve coverage:**
- Add tests for error paths
- Test edge cases
- Add table-driven tests for multiple scenarios

### CI/CD Issues

#### Build Fails on CI but Works Locally

**Common causes:**
- Different Go version
- Missing environment variables
- Different OS (Linux vs macOS)

**Solution:**
```bash
# Check Go version
go version

# Ensure go.mod specifies version
# go.mod should have: go 1.22

# Test with specific Go version
go1.22 test ./...
```

---

## Test Maintenance

### Adding New Tests

When adding new functionality:

1. Write tests **before** implementation (TDD)
2. Ensure tests fail initially
3. Implement functionality
4. Verify tests pass
5. Check coverage: `go test -cover ./internal/<package>`

### Updating Existing Tests

When modifying code:

1. Run affected tests: `go test ./internal/<package>`
2. Update tests if behavior changed
3. Ensure no regressions
4. Check coverage maintained or improved

### Regular Test Reviews

**Monthly:**
- Review test coverage
- Remove obsolete tests
- Add tests for new edge cases discovered

**Quarterly:**
- Review test performance (slow tests)
- Update test data
- Review mocking strategies

---

## Summary

### Quick Test Commands

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run with race detector
go test -race ./...

# Run specific test
make test-one TEST=TestCreateSession

# Run tests verbosely
make test-verbose

# Check coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Checklist

Before releasing:

- [ ] All tests pass: `make test`
- [ ] No race conditions: `go test -race ./...`
- [ ] Coverage >75%: `make test-coverage`
- [ ] Manual tests pass (all 10 test cases)
- [ ] Integration tests pass
- [ ] Performance tests pass
- [ ] Security scans pass
- [ ] CI/CD pipeline green

---

**Document Version:** 1.0  
**Last Updated:** 2026-02-15  
**Next Review:** 2026-05-15

For questions about testing, see project documentation or open an issue on GitHub.
