# Contributing to OpenVPN Keycloak SSO

Thank you for your interest in contributing to OpenVPN Keycloak SSO! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Coding Standards](#coding-standards)
- [Testing Requirements](#testing-requirements)
- [Documentation Guidelines](#documentation-guidelines)
- [Pull Request Process](#pull-request-process)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Enhancements](#suggesting-enhancements)
- [Code Review Process](#code-review-process)

## Code of Conduct

### Our Pledge

We are committed to providing a welcoming and inclusive environment for all contributors, regardless of background or experience level.

### Our Standards

**Positive behavior includes:**
- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

**Unacceptable behavior includes:**
- Harassment, trolling, or derogatory comments
- Publishing others' private information without permission
- Other conduct which could reasonably be considered inappropriate

### Enforcement

Instances of abusive, harassing, or otherwise unacceptable behavior may be reported by opening an issue or contacting the project maintainers. All complaints will be reviewed and investigated promptly and fairly.

## How Can I Contribute?

### Reporting Bugs

**Before Submitting a Bug Report:**
1. Check the [documentation](docs/) for common issues
2. Search [existing issues](../../issues) to avoid duplicates
3. Verify the bug exists in the latest version

**How to Submit a Bug Report:**

Open an issue with:
- **Clear title** - Descriptive summary of the issue
- **Environment** - OS, OpenVPN version, Keycloak version, Go version
- **Steps to reproduce** - Exact steps that trigger the bug
- **Expected behavior** - What you expected to happen
- **Actual behavior** - What actually happened
- **Logs** - Relevant log output (redact any secrets!)
- **Configuration** - Sanitized config file (remove secrets)

**Example:**

```markdown
**Environment:**
- OS: Rocky Linux 9.3
- OpenVPN: 2.6.19
- Keycloak: 25.0.6
- openvpn-keycloak-auth: v1.0.0

**Steps to Reproduce:**
1. Start daemon with config X
2. Connect client with profile Y
3. Observe error Z

**Expected:** Client should authenticate successfully
**Actual:** Authentication fails with "invalid token"

**Logs:**
```
[error logs here]
```

**Config:**
```yaml
[sanitized config]
```
```

### Suggesting Enhancements

**Before Submitting an Enhancement:**
1. Check the [roadmap](README.md#roadmap) for planned features
2. Search [existing issues](../../issues) for similar suggestions
3. Consider whether it fits the project's scope

**How to Submit an Enhancement:**

Open an issue with:
- **Clear title** - Summary of the enhancement
- **Motivation** - Why this would be useful
- **Use case** - Concrete example of how it would be used
- **Proposed solution** - Your idea for implementation (optional)
- **Alternatives** - Other approaches you've considered

### Contributing Code

We welcome pull requests for:
- **Bug fixes** - Fixes for reported issues
- **Features** - Implementations of accepted feature requests
- **Documentation** - Improvements to docs, examples, guides
- **Tests** - Additional test coverage
- **Performance** - Optimizations with benchmarks

## Development Setup

### Prerequisites

```bash
# Go 1.22 or later
go version

# OpenVPN 2.6.19 (for testing)
openvpn --version

# Git
git --version
```

### Clone and Build

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR-USERNAME/openvpn-keycloak.git
cd openvpn-keycloak

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run linter
make lint
```

### Development Workflow

1. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/my-feature
   # or
   git checkout -b fix/issue-123
   ```

2. **Make your changes** following the coding standards

3. **Write tests** for your changes

4. **Run tests and linter**:
   ```bash
   make test
   make lint
   make test-coverage
   ```

5. **Commit your changes** with clear commit messages:
   ```bash
   git add .
   git commit -m "Add feature X to improve Y"
   ```

6. **Push to your fork**:
   ```bash
   git push origin feature/my-feature
   ```

7. **Open a Pull Request** on GitHub

## Coding Standards

### Go Style Guide

Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) and [Effective Go](https://golang.org/doc/effective_go).

**Key principles:**
- **gofmt** - All code must be formatted with `gofmt` (run via `make fmt`)
- **golangci-lint** - Must pass all linter checks (run via `make lint`)
- **Simplicity** - Prefer simple, readable code over clever solutions
- **Error handling** - Always handle errors explicitly, never ignore them
- **Context** - Use `context.Context` for cancellation and timeouts

### Code Organization

```go
// Package comment - Every package should have a doc comment
package mypackage

import (
    // Standard library imports first
    "context"
    "fmt"

    // Third-party imports second
    "github.com/coreos/go-oidc/v3/oidc"

    // Local imports last
    "github.com/yourusername/openvpn-keycloak/internal/config"
)

// Public functions should have doc comments
// that start with the function name
func DoSomething(ctx context.Context, input string) (string, error) {
    // Implementation
}

// privateFunction doesn't require a doc comment unless complex
func privateFunction() {
    // Implementation
}
```

### Naming Conventions

- **Packages** - Short, lowercase, single word (no underscores): `session`, `oidc`, `httpserver`
- **Variables** - camelCase for local, PascalCase for exported: `sessionID`, `MaxTimeout`
- **Constants** - PascalCase or ALL_CAPS depending on scope: `DefaultTimeout`, `MAX_RETRIES`
- **Interfaces** - "-er" suffix when possible: `Handler`, `Validator`, `Provider`

### Error Handling

**Always handle errors explicitly:**

```go
// Good
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Bad - never ignore errors
result, _ := doSomething()
```

**Use error wrapping:**

```go
// Wrap errors with context
if err := validate(input); err != nil {
    return fmt.Errorf("validation failed for %s: %w", input, err)
}
```

**Create custom error types when needed:**

```go
type ValidationError struct {
    Field string
    Value string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("invalid %s: %s", e.Field, e.Value)
}
```

### Logging

Use structured logging with `log/slog`:

```go
import "log/slog"

// Good - structured logging
slog.Info("session created",
    "session_id", sessionID,
    "username", username,
    "ip", clientIP,
)

// Bad - unstructured logging
log.Printf("Session %s created for user %s from %s", sessionID, username, clientIP)

// NEVER log secrets
slog.Debug("token received") // ✓ Good
slog.Debug("token", "value", token) // ✗ NEVER DO THIS
```

**Log levels:**
- **Debug** - Detailed information for debugging (disabled in production)
- **Info** - General informational messages
- **Warn** - Warning messages for recoverable issues
- **Error** - Error messages for failures

### Security Best Practices

**Critical rules:**

1. **Never log secrets** - Tokens, passwords, keys must never appear in logs
2. **Use crypto/rand** - Never use `math/rand` for security-sensitive values
3. **Validate all inputs** - Especially data from external sources
4. **Use contexts** - Always pass `context.Context` for timeouts/cancellation
5. **Atomic file writes** - Use `os.WriteFile` for critical file operations
6. **Error in all paths** - Especially for `auth_control_file` writes

**Example - Session ID generation:**

```go
import "crypto/rand"

// Good - cryptographically secure
func generateSessionID() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", fmt.Errorf("failed to generate session ID: %w", err)
    }
    return hex.EncodeToString(b), nil
}

// Bad - predictable, insecure
func generateSessionID() string {
    return fmt.Sprintf("%d", time.Now().UnixNano())
}
```

### Comments

**Package comments:**

```go
// Package session implements session management for OpenVPN authentication.
// It provides session storage, cleanup, and retrieval functionality.
package session
```

**Function comments:**

```go
// ValidateToken verifies the JWT signature and validates all claims.
// It checks the issuer, audience, expiration, and optional role requirements.
// Returns an error if any validation fails.
func ValidateToken(ctx context.Context, token string, cfg *config.Config) error {
    // Implementation
}
```

**Inline comments:**

Use sparingly, only for complex logic:

```go
// Calculate PKCE challenge: base64url(sha256(verifier))
h := sha256.Sum256([]byte(verifier))
challenge := base64.RawURLEncoding.EncodeToString(h[:])
```

## Testing Requirements

### Unit Tests

**All new code must include unit tests:**

- **Coverage target:** 70%+ for new code
- **Table-driven tests** for multiple scenarios
- **Mock external dependencies** (HTTP, filesystem, etc.)
- **Test error paths** as well as success paths

**Example - Table-driven test:**

```go
func TestValidateToken(t *testing.T) {
    tests := []struct {
        name    string
        token   string
        wantErr bool
    }{
        {
            name:    "valid token",
            token:   "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
            wantErr: false,
        },
        {
            name:    "expired token",
            token:   "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
            wantErr: true,
        },
        {
            name:    "invalid signature",
            token:   "invalid.token.here",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateToken(context.Background(), tt.token, testConfig)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detector
go test -race ./...

# Run specific test
go test -v -run TestValidateToken ./internal/oidc

# Run benchmarks
go test -bench=. ./...
```

### Integration Tests

For changes affecting the full authentication flow:

1. Set up test environment (see [docs/testing.md](docs/testing.md))
2. Run integration tests: `make test-integration`
3. Test with real OpenVPN and Keycloak instances

## Documentation Guidelines

### What to Document

**Code documentation:**
- Public functions, types, constants
- Complex algorithms or logic
- Security-sensitive code
- Non-obvious behavior

**User documentation:**
- New features or options
- Configuration changes
- Breaking changes
- Migration guides

### Documentation Structure

**For new features, update:**
1. Code comments (inline documentation)
2. `README.md` (if it's a major feature)
3. Relevant guide in `docs/` directory
4. Configuration examples in `config/`
5. `CHANGELOG.md` with the change

### Writing Style

- **Be clear and concise** - Avoid jargon unless necessary
- **Use examples** - Code examples are worth a thousand words
- **Be specific** - "The token expires after 5 minutes" vs. "The token expires quickly"
- **Use active voice** - "The daemon validates the token" vs. "The token is validated"
- **Format code** - Use triple backticks with language hints

## Pull Request Process

### Before Submitting

**Checklist:**

- [ ] Code follows the style guide
- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] New code has unit tests (70%+ coverage)
- [ ] Documentation is updated
- [ ] Commit messages are clear and descriptive
- [ ] No secrets or sensitive data in code/logs
- [ ] Changes are focused (one feature/fix per PR)

### PR Template

When opening a PR, include:

```markdown
## Description
Brief description of what this PR does.

## Motivation
Why is this change needed? What problem does it solve?

## Changes
- List of specific changes made
- Another change
- Yet another change

## Testing
How was this tested?
- [ ] Unit tests added/updated
- [ ] Integration tests run
- [ ] Manual testing performed

## Related Issues
Fixes #123
Relates to #456

## Screenshots (if applicable)
[Add screenshots for UI changes]

## Checklist
- [ ] Code follows style guide
- [ ] Tests pass
- [ ] Linter passes
- [ ] Documentation updated
- [ ] CHANGELOG.md updated (if user-facing change)
```

### PR Review Process

1. **Automated checks** - CI/CD pipeline must pass (tests, lint, build)
2. **Code review** - At least one maintainer will review your code
3. **Feedback** - Address review comments and push updates
4. **Approval** - Once approved, a maintainer will merge

### After Merge

- Your PR will be included in the next release
- You'll be added to the contributors list
- Thank you for contributing! 🎉

## Commit Message Guidelines

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat** - New feature
- **fix** - Bug fix
- **docs** - Documentation changes
- **test** - Adding or updating tests
- **refactor** - Code refactoring (no functional changes)
- **perf** - Performance improvements
- **style** - Code style changes (formatting, etc.)
- **build** - Build system or dependency changes
- **ci** - CI/CD pipeline changes

### Examples

```
feat(oidc): add support for custom claims validation

Implement custom claims validation to support role-based access control.
Users can now specify required claims in the configuration.

Closes #123
```

```
fix(auth): handle timeout in auth script gracefully

Previously, if the daemon didn't respond within 5 seconds, the auth
script would hang indefinitely. Now it returns exit code 1 after timeout.

Fixes #456
```

```
docs(deployment): add troubleshooting section for systemd

Added common systemd issues and solutions to the deployment guide.
```

## Release Process

**For Maintainers:**

1. Update version in `go.mod` and relevant files
2. Update `CHANGELOG.md` with release notes
3. Create git tag: `git tag -a v1.x.x -m "Release v1.x.x"`
4. Push tag: `git push origin v1.x.x`
5. GitHub Actions will build and create release automatically
6. Update release notes on GitHub

## Getting Help

**Need help contributing?**

- **Documentation** - Check [docs/](docs/) directory
- **Issues** - Search [existing issues](../../issues)
- **Discussions** - Use GitHub Discussions for questions
- **Email** - Contact maintainers (see README.md)

## Recognition

All contributors will be recognized:
- Added to `README.md` contributors section
- Mentioned in release notes
- Listed in GitHub contributors

Thank you for making OpenVPN Keycloak SSO better!

---

**Last Updated:** 2026-02-15
**Version:** 1.0.0
