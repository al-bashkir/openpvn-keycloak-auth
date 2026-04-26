#!/usr/bin/env bash
set -euo pipefail

# OpenVPN Keycloak SSO Authentication Script
#
# Called by OpenVPN via --auth-user-pass-verify <script> via-file.
# Thin wrapper that execs the Go binary in auth mode.

adjust_nproc_limit() {
  local nproc_limit

  nproc_limit="$(ulimit -u 2>/dev/null || true)"
  if [[ -n "$nproc_limit" && "$nproc_limit" != "unlimited" && "$nproc_limit" -lt 256 ]] 2>/dev/null; then
    ulimit -u unlimited 2>/dev/null || ulimit -u 256 2>/dev/null || true
  fi
}

main() {
  local -r binary="/usr/local/bin/openvpn-keycloak-auth"
  local -r config="/etc/openvpn/keycloak-sso.yaml"

  if [[ $# -ne 1 ]]; then
    printf 'Error: expected OpenVPN credentials file argument\n' >&2
    return 1
  fi

  if [[ ! -x "$binary" ]]; then
    printf 'Error: executable not found: %s\n' "$binary" >&2
    return 1
  fi

  # OpenVPN's systemd unit may set a low LimitNPROC. The Go runtime needs
  # enough OS threads for the scheduler, GC, and netpoller.
  adjust_nproc_limit

  exec "$binary" --config "$config" auth "$1"
}

main "$@"
