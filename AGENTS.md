# OpenVPN Keycloak SSO Authentication - Agent Instructions

## Project Overview

This repository implements SSO authentication for OpenVPN Community Server using Keycloak as the OpenID Connect provider. It uses OpenVPN 2.6 script-based deferred authentication, so deployment is a Go binary plus shell wrapper rather than a C plugin or management-interface integration.

Key facts:

- Target platform: Rocky Linux 9 or compatible Linux
- OpenVPN target: Community Server 2.6.2+; deployment examples target 2.6.19
- Language/toolchain: Go 1.25+
- IdP: Keycloak via OIDC authorization code flow with PKCE
- Binary: `openvpn-keycloak-auth`
- License: MPL-2.0

## Architecture

The binary has four subcommands:

- `serve`: long-running daemon; owns OIDC flow, HTTP callback handling, session tracking, and OpenVPN result-file writes.
- `auth <credentials-file>`: OpenVPN auth-script mode; reads the OpenVPN via-file credentials, sends an IPC request to the daemon, writes an auth-pending response, and returns OpenVPN exit code `2` for deferred auth.
- `check-config`: validates configuration and prints a redacted summary.
- `version`: prints build and Go runtime version information.

Primary components:

- `/etc/openvpn/scripts/auth-keycloak.sh`: installed shell wrapper called by OpenVPN.
- `/usr/local/bin/openvpn-keycloak-auth`: installed Go binary.
- `/etc/openvpn/keycloak-sso.yaml`: daemon configuration.
- `/run/openvpn-keycloak-auth/auth.sock`: Unix socket IPC endpoint.
- `internal/httpserver/templates/*.html`: embedded success/error pages.

Authentication flow:

```text
OpenVPN client -> OpenVPN server -> auth wrapper -> binary auth mode
                                -> Unix socket IPC -> daemon
                                -> Keycloak OIDC browser flow
                                -> callback validates token and roles
                                -> daemon writes OpenVPN auth_control_file
```

## Repository Layout

```text
cmd/openvpn-keycloak-auth/     CLI entry point
internal/auth/                 OpenVPN auth-script mode
internal/config/               YAML/env configuration loading and validation
internal/daemon/               daemon orchestration
internal/httpserver/           callback, redirect, health endpoints, templates
internal/ipc/                  Unix socket protocol
internal/logsanitize/          log-safe string cleanup
internal/oidc/                 OIDC flow, token exchange, claim validation
internal/openvpn/              auth_pending/auth_control/failure file writes
internal/session/              in-memory session tracking and expiry cleanup
config/                        sample daemon and OpenVPN configs
deploy/                        install/uninstall/systemd assets
docs/                          user-facing documentation
scripts/                       OpenVPN wrapper and client profile generator
tasks/                         historical task breakdown
reports/                       historical reports
```

## Configuration Model

The live daemon configuration uses these top-level sections:

- `listen.http`: HTTP callback listen address, for example `:9000`.
- `listen.socket`: absolute Unix socket path.
- `oidc.issuer`: Keycloak realm issuer URL.
- `oidc.client_id`: OIDC client ID.
- `oidc.client_secret`: optional, for confidential clients.
- `oidc.redirect_uri`: callback URI registered in Keycloak.
- `oidc.scopes`, `oidc.required_roles`, `oidc.role_claim`: OIDC claim and authorization settings.
- `auth.session_timeout`, `auth.username_claim`, `auth.allow_username_mismatch`: OpenVPN auth behavior.
- `tls.*`: optional direct TLS listener settings.
- `log.*`: slog level and format.

Do not document or add config options unless they are actually wired in code and tests.

## OpenVPN 2.6 Script Auth Details

OpenVPN calls the wrapper using `auth-user-pass-verify <script> via-file`. In via-file mode OpenVPN provides:

- Argument 1: temporary credentials file with username on line 1 and password on line 2.
- Environment: `auth_control_file`, `auth_pending_file`, `auth_failed_reason_file`, `untrusted_ip`, `untrusted_port`, `common_name`, `IV_SSO`, and related OpenVPN values.

Exit codes:

- `0`: immediate success.
- `1`: immediate failure.
- `2`: deferred authentication pending.

`auth_pending_file` must contain exactly:

```text
<timeout_seconds>
<method>
WEB_AUTH::<url>
```

The pending method is selected from OpenVPN's `IV_SSO`; current code prefers `webauth` and falls back to `openurl`.

`auth_control_file` must contain `1` for success or `0` for failure. Write failure reasons before writing `0` to `auth_control_file`.

## Development Commands

Use the Makefile when possible:

```bash
make build
make test
make lint
make check
```

Equivalent direct commands:

```bash
go build -o openvpn-keycloak-auth ./cmd/openvpn-keycloak-auth
go test ./...
go test -race ./...
go vet ./...
golangci-lint run
shellcheck scripts/*.sh deploy/*.sh
```

If `golangci-lint` was built with an older Go version than `go.mod` targets, record that as a blocked validation instead of changing global tooling without user approval.

## Coding Rules

- Preserve intended OpenVPN behavior; do not change exit codes, file formats, or callback flow casually.
- Keep changes small and reviewable.
- Do not remove code or assets until you verify they are not referenced by embeds, tests, scripts, docs, build tags, registration, reflection, or runtime paths.
- Use `context.Context` for network and shutdown paths.
- Use `crypto/rand` for session IDs, states, and PKCE values; never use `math/rand` for security values.
- Use `log/slog` for structured logs.
- Never log passwords, tokens, authorization codes, PKCE verifiers, full auth URLs, or client secrets.
- Sanitize externally controlled values with `internal/logsanitize` before logging.
- Keep OpenVPN control-file writes simple and reliable with restrictive permissions; OpenVPN provides the target paths.
- Preserve IPC hardening: peer credentials, bounded/deadline reads, and absolute canonical OpenVPN result paths are security-sensitive.
- Preserve callback result-write claiming; do not reintroduce separate check/write/mark sequences that can race duplicate callbacks.
- Prefer standard library functionality unless a dependency already exists for the task.

## Shell Script Rules

Shell scripts in `scripts/` and `deploy/` should follow these conventions when edited:

- Start with `#!/usr/bin/env bash` and `set -euo pipefail`.
- Put executable logic in `main()` and call `main "$@"` at the end.
- Use `local` or `local -r` for function variables.
- Send diagnostics to stderr with `printf`, not `echo`.
- Validate external commands before use when practical.
- Avoid pipelines through `head`, `tail`, `less`, or `more`; use command-specific flags or Bash parsing.
- Run `shellcheck scripts/*.sh deploy/*.sh` after edits.

## Documentation Rules

- Keep README, QUICKSTART, `config/*.example`, and `docs/` aligned with live code and installer behavior.
- Current installed wrapper path is `/etc/openvpn/scripts/auth-keycloak.sh`.
- Current config path is `/etc/openvpn/keycloak-sso.yaml`.
- Current config permissions are `0640 root:openvpn` so the daemon user can read the file.
- `check-config` validates local configuration only; it does not perform Keycloak connectivity or OIDC discovery.
- Do not claim exact coverage, audit, performance, or end-to-end validation results unless the corresponding command or environment was actually run in the current work.
- Full browser SSO must be validated with a real OpenVPN server and Keycloak realm; local unit tests cannot prove that deployment end-to-end.

## Security Checklist For Changes

- No tokens, passwords, codes, PKCE verifiers, or client secrets in logs.
- Unknown YAML keys should remain rejected so config typos do not silently disable security controls.
- OIDC issuer and redirect URLs are parsed and validated, not only prefix-checked.
- Unix socket paths are absolute and protected by directory/socket permissions.
- Session IDs and OIDC state values are cryptographically random.
- `auth_control_file` gets a final result on all callback success/failure paths where OpenVPN provided a session.
- Failure reason is written before `auth_control_file=0`.
- Generated client profiles containing private keys should not be world-readable.
- Update docs when behavior, paths, config keys, or security posture changes.

## Validation Expectations

Before reporting completion after code changes, run and report:

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `shellcheck scripts/*.sh deploy/*.sh` when shell files changed
- `golangci-lint run`, unless blocked by local toolchain mismatch

For build/config changes, also run:

- `go build -o openvpn-keycloak-auth ./cmd/openvpn-keycloak-auth`
- `./openvpn-keycloak-auth check-config --config config/openvpn-keycloak-auth.yaml.example`

If a validation command is not run or is blocked, state that explicitly in the final report.
