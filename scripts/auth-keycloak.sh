#!/bin/bash
set -euo pipefail

#
# OpenVPN Keycloak SSO Authentication Script
#
# Called by OpenVPN via --auth-user-pass-verify <script> via-file
# Thin wrapper that execs the Go binary in auth mode.
#
# Exit codes:
#   0 - Auth success (immediate, not used for SSO)
#   1 - Auth failure
#   2 - Auth deferred (SSO flow initiated)
#

BINARY="/usr/local/bin/openvpn-keycloak-sso"
CONFIG="/etc/openvpn/keycloak-sso.yaml"

# --- Fix RLIMIT_NPROC for the Go runtime ---
#
# OpenVPN's systemd unit often sets a very low LimitNPROC (max threads/processes).
# The Go runtime needs at least ~5-10 OS threads for the scheduler, GC, and
# netpoller.  If the limit is too low, Go crashes with "newosproc" / errno=11.
#
# Under SELinux (openvpn_t) setrlimit may be denied; only attempt if the
# current limit is low to avoid noisy AVCs.
_nproc_limit="$(ulimit -u 2>/dev/null || true)"
if [ -n "${_nproc_limit}" ] && [ "${_nproc_limit}" != "unlimited" ] && [ "${_nproc_limit}" -lt 256 ] 2>/dev/null; then
  ulimit -u unlimited 2>/dev/null || ulimit -u 256 2>/dev/null || true
fi

exec "$BINARY" --config "$CONFIG" auth "$1"
