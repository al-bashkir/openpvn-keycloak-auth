#!/usr/bin/env bash
set -euo pipefail

# Generate an OpenVPN client profile with embedded certificates for SSO auth.

usage() {
  printf 'Usage: %s [ca_cert] [server_hostname] [output_file] [profile_type] [client_cert] [client_key] [tls_auth_key]\n' "$0" >&2
  printf 'Profile types: universal, cli, tunnelblick, connect\n' >&2
}

require_command() {
  local -r command_name="$1"

  command -v "$command_name" >/dev/null 2>&1 || {
    printf 'Error: %s not found in PATH\n' "$command_name" >&2
    return 1
  }
}

require_file() {
  local -r label="$1"
  local -r file_path="$2"

  if [[ ! -f "$file_path" ]]; then
    printf 'Error: %s not found: %s\n' "$label" "$file_path" >&2
    usage
    return 1
  fi
}

validate_profile_type() {
  local -r profile_type="$1"

  case "$profile_type" in
    universal | cli | tunnelblick | connect)
      return 0
      ;;
    *)
      printf 'Error: invalid profile type: %s\n' "$profile_type" >&2
      usage
      return 1
      ;;
  esac
}

validate_server_hostname() {
  local -r server_hostname="$1"

  if [[ -z "$server_hostname" ]]; then
    printf 'Error: server hostname must not be empty\n' >&2
    return 1
  fi
  if [[ "$server_hostname" =~ [[:space:][:cntrl:]] ]]; then
    printf 'Error: server hostname must not contain whitespace or control characters\n' >&2
    return 1
  fi
}

output_directory() {
  local -r output_file="$1"

  if [[ "$output_file" == */* ]]; then
    printf '%s\n' "${output_file%/*}"
  else
    printf '.\n'
  fi
}

write_profile_safely() {
  local -r ca_cert="$1"
  local -r server_hostname="$2"
  local -r output_file="$3"
  local -r profile_type="$4"
  local -r client_cert="$5"
  local -r client_key="$6"
  local -r tls_auth_key="$7"
  local -r output_name="${output_file##*/}"
  local output_dir
  local temp_file

  output_dir="$(output_directory "$output_file")"

  if [[ -d "$output_file" ]]; then
    printf 'Error: output path is a directory: %s\n' "$output_file" >&2
    return 1
  fi

  if [[ ! -d "$output_dir" || ! -w "$output_dir" ]]; then
    printf 'Error: output directory must exist and be writable: %s\n' "$output_dir" >&2
    return 1
  fi

  umask 077
  temp_file="$(mktemp "${output_dir}/.${output_name}.tmp.XXXXXX")"
  if write_profile "$ca_cert" "$server_hostname" "$temp_file" "$profile_type" "$client_cert" "$client_key" "$tls_auth_key" \
    && chmod 600 "$temp_file" \
    && mv -f "$temp_file" "$output_file"; then
    return 0
  fi

  rm -f "$temp_file"
  return 1
}

print_summary() {
  local -r ca_cert="$1"
  local -r server_hostname="$2"
  local -r output_file="$3"
  local -r profile_type="$4"
  local -r client_cert="$5"
  local -r client_key="$6"
  local -r tls_auth_key="$7"

  printf 'Generating OpenVPN client profile...\n'
  printf '  Type:   %s\n' "$profile_type"
  printf '  Server: %s\n' "$server_hostname"
  printf '  CA:     %s\n' "$ca_cert"
  [[ -n "$client_cert" ]] && printf '  Cert:   %s\n' "$client_cert"
  [[ -n "$client_key" ]] && printf '  Key:    %s\n' "$client_key"
  [[ -n "$tls_auth_key" ]] && printf '  TLS:    %s\n' "$tls_auth_key"
  printf '  Output: %s\n\n' "$output_file"
}

write_profile() {
  local -r ca_cert="$1"
  local -r server_hostname="$2"
  local -r output_file="$3"
  local -r profile_type="$4"
  local -r client_cert="$5"
  local -r client_key="$6"
  local -r tls_auth_key="$7"

  {
    cat <<'EOF'
# OpenVPN Client Configuration for SSO Authentication
# Auto-generated profile - DO NOT EDIT MANUALLY
#
# To use this profile:
#   1. Import into your OpenVPN client
#   2. Connect with your Keycloak username
#   3. Authenticate via browser when prompted
#
# For detailed instructions, see docs/client-setup.md

client
dev tun
proto udp

EOF

    printf 'remote %s 1194\n\n' "$server_hostname"

    cat <<'EOF'
resolv-retry infinite
nobind
persist-key
persist-tun

EOF

    case "$profile_type" in
      cli)
        cat <<'EOF'
# Optional: Downgrade privileges after initialization
# user nobody
# group nogroup

EOF
        ;;
      tunnelblick)
        cat <<'EOF'
# Tunnelblick-specific optimizations
# Route all traffic through VPN (optional, uncomment to enable)
# redirect-gateway def1

EOF
        ;;
      connect)
        cat <<'EOF'
# OpenVPN Connect optimizations
# Configure split tunneling and other features via app settings

EOF
        ;;
    esac

    cat <<'EOF'
<ca>
EOF
    cat "$ca_cert"
    cat <<'EOF'
</ca>

EOF

    if [[ -n "$client_cert" ]]; then
      cat <<'EOF'
<cert>
EOF
      cat "$client_cert"
      cat <<'EOF'
</cert>

EOF
    fi

    if [[ -n "$client_key" ]]; then
      cat <<'EOF'
<key>
EOF
      cat "$client_key"
      cat <<'EOF'
</key>

EOF
    fi

    cat <<'EOF'
auth-user-pass
auth-retry interact

remote-cert-tls server
data-ciphers AES-256-GCM:AES-128-GCM:AES-256-CBC
tls-version-min 1.2

EOF

    if [[ -n "$tls_auth_key" ]]; then
      cat <<'EOF'
<tls-auth>
EOF
      cat "$tls_auth_key"
      cat <<'EOF'
</tls-auth>
key-direction 1

EOF
    fi

    cat <<'EOF'
sndbuf 393216
rcvbuf 393216

verb 3
mute 20

EOF

    case "$profile_type" in
      cli)
        cat <<'EOF'
# CLI SSO Instructions:
# 1. Start OpenVPN: openvpn --config client.ovpn
# 2. Enter username (Keycloak username) and password (anything, e.g., "sso")
# 3. Copy the WEB_AUTH:: URL from the output
# 4. Open the URL in your browser
# 5. Log in to Keycloak
# 6. Return to terminal - connection completes automatically
EOF
        ;;
      tunnelblick)
        cat <<'EOF'
# Tunnelblick SSO Instructions:
# 1. Double-click this file to import into Tunnelblick
# 2. Click "Connect" in Tunnelblick menu
# 3. Enter Keycloak username and any password
# 4. Safari opens automatically with Keycloak login
# 5. Log in to Keycloak
# 6. VPN connects automatically
EOF
        ;;
      connect)
        cat <<'EOF'
# OpenVPN Connect SSO Instructions:
# 1. Import this profile (File > Import or drag-and-drop)
# 2. Tap/click to connect
# 3. Enter Keycloak username and any password
# 4. Built-in browser opens with Keycloak login
# 5. Log in to Keycloak
# 6. VPN connects automatically
EOF
        ;;
      universal)
        cat <<'EOF'
# SSO Authentication Instructions:
# 1. Import this profile into your OpenVPN client
# 2. Connect and enter your Keycloak username
# 3. Password can be anything (e.g., "sso") - it will be ignored
# 4. Authenticate via browser when prompted
# 5. VPN connection completes after successful login
#
# Modern clients (OpenVPN Connect, Tunnelblick) open browser automatically
# CLI clients display a URL that you must open manually
EOF
        ;;
    esac
  } >"$output_file"
}

main() {
  require_command cat
  require_command chmod
  require_command mktemp
  require_command mv
  require_command rm

  local -r ca_cert="${1:-/etc/openvpn/server/ca.crt}"
  local -r server_hostname="${2:-vpn.example.com}"
  local -r output_file="${3:-client-generated.ovpn}"
  local -r profile_type="${4:-universal}"
  local -r client_cert="${5:-}"
  local -r client_key="${6:-}"
  local -r tls_auth_key="${7:-}"

  require_file "CA certificate" "$ca_cert"
  [[ -n "$client_cert" ]] && require_file "Client certificate" "$client_cert"
  [[ -n "$client_key" ]] && require_file "Client private key" "$client_key"
  [[ -n "$tls_auth_key" ]] && require_file "TLS auth key" "$tls_auth_key"
  validate_profile_type "$profile_type"
  validate_server_hostname "$server_hostname"

  print_summary "$ca_cert" "$server_hostname" "$output_file" "$profile_type" "$client_cert" "$client_key" "$tls_auth_key"
  write_profile_safely "$ca_cert" "$server_hostname" "$output_file" "$profile_type" "$client_cert" "$client_key" "$tls_auth_key"

  printf 'Client profile generated successfully: %s\n\n' "$output_file"
  printf 'Next steps:\n'
  printf '  1. Import %s into your OpenVPN client\n' "$output_file"
  printf '  2. Connect using your Keycloak username\n'
  printf '  3. Authenticate via browser when prompted\n\n'
  printf 'For detailed setup instructions, see:\n'
  printf '  docs/client-setup.md\n'
}

main "$@"
