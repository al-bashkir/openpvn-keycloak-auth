#!/bin/bash
# Generate OpenVPN client profile with embedded CA certificate
# 
# Usage:
#   ./generate-client-profile.sh [ca_cert] [server_hostname] [output_file] [profile_type]
#
# Arguments:
#   ca_cert          Path to CA certificate (default: /etc/openvpn/server/ca.crt)
#   server_hostname  VPN server hostname (default: vpn.example.com)
#   output_file      Output .ovpn file (default: client-generated.ovpn)
#   profile_type     Profile type: universal, cli, tunnelblick, connect (default: universal)
#
# Examples:
#   ./generate-client-profile.sh
#   ./generate-client-profile.sh /etc/openvpn/server/ca.crt vpn.company.com client.ovpn universal
#   ./generate-client-profile.sh /etc/openvpn/server/ca.crt vpn.company.com ios-profile.ovpn connect
#
# Optional: Include client certificate for mutual TLS
#   ./generate-client-profile.sh ca.crt vpn.company.com client.ovpn universal client.crt client.key

set -e

##############################################
# Configuration
##############################################

CA_CERT="${1:-/etc/openvpn/server/ca.crt}"
SERVER_HOSTNAME="${2:-vpn.example.com}"
OUTPUT_FILE="${3:-client-generated.ovpn}"
PROFILE_TYPE="${4:-universal}"
CLIENT_CERT="${5:-}"
CLIENT_KEY="${6:-}"
TA_KEY="${7:-}"

##############################################
# Validation
##############################################

if [ ! -f "$CA_CERT" ]; then
    echo "Error: CA certificate not found: $CA_CERT" >&2
    echo "Usage: $0 [ca_cert] [server_hostname] [output_file] [profile_type]" >&2
    exit 1
fi

if [ -n "$CLIENT_CERT" ] && [ ! -f "$CLIENT_CERT" ]; then
    echo "Error: Client certificate not found: $CLIENT_CERT" >&2
    exit 1
fi

if [ -n "$CLIENT_KEY" ] && [ ! -f "$CLIENT_KEY" ]; then
    echo "Error: Client private key not found: $CLIENT_KEY" >&2
    exit 1
fi

if [ -n "$TA_KEY" ] && [ ! -f "$TA_KEY" ]; then
    echo "Error: TLS auth key not found: $TA_KEY" >&2
    exit 1
fi

# Validate profile type
case "$PROFILE_TYPE" in
    universal|cli|tunnelblick|connect)
        ;;
    *)
        echo "Error: Invalid profile type: $PROFILE_TYPE" >&2
        echo "Valid types: universal, cli, tunnelblick, connect" >&2
        exit 1
        ;;
esac

##############################################
# Generate Profile
##############################################

echo "Generating OpenVPN client profile..."
echo "  Type:   $PROFILE_TYPE"
echo "  Server: $SERVER_HOSTNAME"
echo "  CA:     $CA_CERT"
[ -n "$CLIENT_CERT" ] && echo "  Cert:   $CLIENT_CERT"
[ -n "$CLIENT_KEY" ] && echo "  Key:    $CLIENT_KEY"
[ -n "$TA_KEY" ] && echo "  TLS:    $TA_KEY"
echo "  Output: $OUTPUT_FILE"
echo

##############################################
# Common Header
##############################################

cat > "$OUTPUT_FILE" <<'EOF'
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

##############################################
# Server Configuration
##############################################

cat >> "$OUTPUT_FILE" <<EOF
remote $SERVER_HOSTNAME 1194

EOF

cat >> "$OUTPUT_FILE" <<'EOF'
resolv-retry infinite
nobind
persist-key
persist-tun

EOF

##############################################
# Profile-Specific Settings
##############################################

case "$PROFILE_TYPE" in
    cli)
        cat >> "$OUTPUT_FILE" <<'EOF'
# Optional: Downgrade privileges after initialization
# user nobody
# group nogroup

EOF
        ;;
    tunnelblick)
        cat >> "$OUTPUT_FILE" <<'EOF'
# Tunnelblick-specific optimizations
# Route all traffic through VPN (optional, uncomment to enable)
# redirect-gateway def1

EOF
        ;;
    connect)
        cat >> "$OUTPUT_FILE" <<'EOF'
# OpenVPN Connect optimizations
# Configure split tunneling and other features via app settings

EOF
        ;;
esac

##############################################
# Certificates
##############################################

# CA Certificate (inline)
cat >> "$OUTPUT_FILE" <<'EOF'
<ca>
EOF
cat "$CA_CERT" >> "$OUTPUT_FILE"
cat >> "$OUTPUT_FILE" <<'EOF'
</ca>

EOF

# Client Certificate (inline, if provided)
if [ -n "$CLIENT_CERT" ]; then
    cat >> "$OUTPUT_FILE" <<'EOF'
<cert>
EOF
    cat "$CLIENT_CERT" >> "$OUTPUT_FILE"
    cat >> "$OUTPUT_FILE" <<'EOF'
</cert>

EOF
fi

# Client Private Key (inline, if provided)
if [ -n "$CLIENT_KEY" ]; then
    cat >> "$OUTPUT_FILE" <<'EOF'
<key>
EOF
    cat "$CLIENT_KEY" >> "$OUTPUT_FILE"
    cat >> "$OUTPUT_FILE" <<'EOF'
</key>

EOF
fi

##############################################
# Authentication
##############################################

cat >> "$OUTPUT_FILE" <<'EOF'
auth-user-pass
auth-retry interact

EOF

##############################################
# Security
##############################################

cat >> "$OUTPUT_FILE" <<'EOF'
remote-cert-tls server
data-ciphers AES-256-GCM:AES-128-GCM:AES-256-CBC
tls-version-min 1.2

EOF

# TLS Authentication Key (inline, if provided)
if [ -n "$TA_KEY" ]; then
    cat >> "$OUTPUT_FILE" <<'EOF'
<tls-auth>
EOF
    cat "$TA_KEY" >> "$OUTPUT_FILE"
    cat >> "$OUTPUT_FILE" <<'EOF'
</tls-auth>
key-direction 1

EOF
fi

##############################################
# Performance
##############################################

cat >> "$OUTPUT_FILE" <<'EOF'
sndbuf 393216
rcvbuf 393216

EOF

##############################################
# Logging
##############################################

cat >> "$OUTPUT_FILE" <<'EOF'
verb 3
mute 20

EOF

##############################################
# Profile-Specific Footer
##############################################

case "$PROFILE_TYPE" in
    cli)
        cat >> "$OUTPUT_FILE" <<'EOF'
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
        cat >> "$OUTPUT_FILE" <<'EOF'
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
        cat >> "$OUTPUT_FILE" <<'EOF'
# OpenVPN Connect SSO Instructions:
# 1. Import this profile (File → Import or drag-and-drop)
# 2. Tap/click to connect
# 3. Enter Keycloak username and any password
# 4. Built-in browser opens with Keycloak login
# 5. Log in to Keycloak
# 6. VPN connects automatically
EOF
        ;;
    universal)
        cat >> "$OUTPUT_FILE" <<'EOF'
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

##############################################
# Success Message
##############################################

echo "✓ Client profile generated successfully: $OUTPUT_FILE"
echo
echo "Next steps:"
echo "  1. Import $OUTPUT_FILE into your OpenVPN client"
echo "  2. Connect using your Keycloak username"
echo "  3. Authenticate via browser when prompted"
echo
echo "For detailed setup instructions, see:"
echo "  docs/client-setup.md"
echo

# Set appropriate permissions
chmod 644 "$OUTPUT_FILE"

exit 0
