#!/bin/bash
# Installation script for OpenVPN Keycloak SSO on Rocky Linux 9
#
# This script installs the OpenVPN Keycloak SSO daemon on Rocky Linux 9.
# It performs the following tasks:
#   1. Checks prerequisites (OpenVPN 2.6+, root permissions)
#   2. Creates openvpn user and group
#   3. Installs the binary to /usr/local/bin
#   4. Creates directories (config, data, tmp, runtime, scripts)
#   5. Installs configuration files
#   6. Installs auth script to /etc/openvpn/scripts/
#   7. Installs SSO daemon systemd service
#   8. Installs OpenVPN service override (LimitNPROC, dependency)
#   9. Configures firewall (if firewalld is running)
#  10. Configures SELinux (if enabled)
#
# Usage:
#   sudo ./deploy/install.sh
#
# Requirements:
#   - Rocky Linux 9 (or compatible RHEL 9 derivative)
#   - OpenVPN 2.6+ (will be installed from EPEL if missing)
#   - Built binary: openvpn-keycloak-sso
#   - Root privileges

set -e

##############################################
# Configuration
##############################################

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/openvpn"
SCRIPTS_DIR="/etc/openvpn/scripts"
DATA_DIR="/var/lib/openvpn-keycloak-sso"
TMP_DIR="/var/lib/openvpn-keycloak-sso/tmp"
RUN_DIR="/run/openvpn-keycloak-sso"
BINARY_NAME="openvpn-keycloak-sso"
SERVICE_FILE="openvpn-keycloak-sso.service"
HTTP_PORT="9000"  # Default HTTP callback port

##############################################
# Colors for Output
##############################################

if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    BOLD=''
    NC=''
fi

##############################################
# Helper Functions
##############################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "This script must be run as root"
        log_info "Try: sudo $0"
        exit 1
    fi
}

check_os() {
    if [ ! -f /etc/redhat-release ]; then
        log_warning "This script is designed for Rocky Linux 9"
        log_warning "It may work on other RHEL 9 derivatives, but is not tested"
    else
        local os_version
        os_version=$(cat /etc/redhat-release)
        log_info "Detected OS: $os_version"
    fi
}

##############################################
# Main Installation Steps
##############################################

main() {
    echo -e "${BOLD}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}║     OpenVPN Keycloak SSO - Installation Script             ║${NC}"
    echo -e "${BOLD}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    # Preliminary checks
    log_info "Performing preliminary checks..."
    check_root
    check_os
    echo ""

    # Step 1: Check prerequisites
    log_info "Step 1/10: Checking prerequisites..."
    check_prerequisites
    echo ""

    # Step 2: Create user and group
    log_info "Step 2/10: Creating system user..."
    create_user
    echo ""

    # Step 3: Copy binary
    log_info "Step 3/10: Installing binary..."
    install_binary
    echo ""

    # Step 4: Create directories
    log_info "Step 4/10: Creating directories..."
    create_directories
    echo ""

    # Step 5: Install configuration
    log_info "Step 5/10: Installing configuration files..."
    install_config
    echo ""

    # Step 6: Install auth script
    log_info "Step 6/10: Installing auth script..."
    install_auth_script
    echo ""

    # Step 7: Install systemd services
    log_info "Step 7/10: Installing systemd services..."
    install_service
    echo ""

    # Step 8: Install OpenVPN service override / unit
    log_info "Step 8/10: Installing OpenVPN service unit..."
    install_openvpn_service
    echo ""

    # Step 9: Configure firewall
    log_info "Step 9/10: Configuring firewall..."
    configure_firewall
    echo ""

    # Step 10: Configure SELinux
    log_info "Step 10/10: Configuring SELinux..."
    configure_selinux
    echo ""

    # Installation complete
    print_success_message
}

##############################################
# Step 1: Check Prerequisites
##############################################

check_prerequisites() {
    local all_ok=true

    # Check for binary
    if [ ! -f "$BINARY_NAME" ]; then
        log_error "Binary not found: $BINARY_NAME"
        log_info "Build the binary first: make build"
        all_ok=false
    else
        log_success "Binary found: $BINARY_NAME"
    fi

    # Check OpenVPN
    if ! command -v openvpn &> /dev/null; then
        log_warning "OpenVPN not found. Installing from EPEL..."
        install_openvpn
    else
        check_openvpn_version
    fi

    if [ "$all_ok" = false ]; then
        exit 1
    fi
}

install_openvpn() {
    log_info "Installing EPEL repository..."
    dnf install -y epel-release >/dev/null 2>&1

    log_info "Installing OpenVPN..."
    dnf install -y openvpn >/dev/null 2>&1

    check_openvpn_version
}

check_openvpn_version() {
    local version
    version=$(openvpn --version 2>/dev/null | head -n1 | grep -oP '\d+\.\d+\.\d+' || echo "0.0.0")

    log_info "OpenVPN version: $version"

    # Check for 2.6+
    local major minor
    major=$(echo "$version" | cut -d. -f1)
    minor=$(echo "$version" | cut -d. -f2)

    if [ "$major" -lt 2 ] || { [ "$major" -eq 2 ] && [ "$minor" -lt 6 ]; }; then
        log_error "OpenVPN 2.6+ required, found $version"
        log_error "SSO authentication requires OpenVPN 2.6.2 or later"
        exit 1
    fi

    log_success "OpenVPN version OK ($version >= 2.6.0)"
}

##############################################
# Step 2: Create User and Group
##############################################

create_user() {
    if id -u openvpn &> /dev/null; then
        log_info "User 'openvpn' already exists"
    else
        log_info "Creating system user 'openvpn'..."
        useradd --system --shell /sbin/nologin --comment "OpenVPN" openvpn
        log_success "User 'openvpn' created"
    fi

    if getent group openvpn &> /dev/null; then
        log_info "Group 'openvpn' already exists"
    else
        log_info "Creating group 'openvpn'..."
        groupadd --system openvpn
        log_success "Group 'openvpn' created"
    fi
}

##############################################
# Step 3: Install Binary
##############################################

install_binary() {
    log_info "Copying binary to $INSTALL_DIR/$BINARY_NAME..."
    install -m 755 "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    log_success "Binary installed"

    # Verify installation
    if "$INSTALL_DIR/$BINARY_NAME" version &> /dev/null; then
        local version
        version=$("$INSTALL_DIR/$BINARY_NAME" version 2>/dev/null | head -1 || echo "unknown")
        log_success "Binary verified: $version"
    else
        log_error "Binary verification failed"
        exit 1
    fi
}

##############################################
# Step 4: Create Directories
##############################################

create_directories() {
    # Config directory (should already exist from OpenVPN)
    if [ ! -d "$CONFIG_DIR" ]; then
        log_info "Creating $CONFIG_DIR..."
        mkdir -p "$CONFIG_DIR"
        chown openvpn:openvpn "$CONFIG_DIR"
        chmod 750 "$CONFIG_DIR"
    fi

    # Scripts directory
    log_info "Creating $SCRIPTS_DIR..."
    mkdir -p "$SCRIPTS_DIR"
    chown root:openvpn "$SCRIPTS_DIR"
    chmod 750 "$SCRIPTS_DIR"

    # Data directory
    log_info "Creating $DATA_DIR..."
    mkdir -p "$DATA_DIR"
    chown openvpn:openvpn "$DATA_DIR"
    chmod 755 "$DATA_DIR"

    # Shared tmp directory for auth files
    # Both OpenVPN (via tmp-dir) and the SSO daemon write auth control
    # files here.  This avoids PrivateTmp namespace mismatches.
    log_info "Creating $TMP_DIR..."
    mkdir -p "$TMP_DIR"
    chown openvpn:openvpn "$TMP_DIR"
    chmod 750 "$TMP_DIR"

    # Runtime directory for Unix socket
    log_info "Creating $RUN_DIR..."
    mkdir -p "$RUN_DIR"
    chown openvpn:openvpn "$RUN_DIR"
    chmod 770 "$RUN_DIR"

    log_success "Directories created"
}

##############################################
# Step 5: Install Configuration
##############################################

install_config() {
    local config_file="$CONFIG_DIR/keycloak-sso.yaml"

    if [ -f "$config_file" ]; then
        log_warning "Configuration already exists: $config_file"
        log_info "Skipping configuration installation (existing file preserved)"
    else
        if [ ! -f "config/openvpn-keycloak-sso.yaml.example" ]; then
            log_error "Example configuration not found: config/openvpn-keycloak-sso.yaml.example"
            exit 1
        fi

        log_info "Installing example configuration..."
        install -m 640 config/openvpn-keycloak-sso.yaml.example "$config_file"
        chown root:openvpn "$config_file"
        log_success "Configuration installed: $config_file"
        echo ""
        log_warning "${BOLD}IMPORTANT: You MUST edit the configuration file!${NC}"
        log_warning "Edit: $config_file"
        log_warning "Update the following settings:"
        log_warning "  - keycloak.issuer_url (your Keycloak server URL)"
        log_warning "  - keycloak.client_id (your OpenVPN client ID)"
        log_warning "  - http.callback_url (your VPN server's public URL)"
        echo ""
    fi
}

##############################################
# Step 6: Install Auth Script
##############################################

install_auth_script() {
    local script_file="$SCRIPTS_DIR/auth-keycloak.sh"

    if [ ! -f "scripts/auth-keycloak.sh" ]; then
        log_error "Auth script not found: scripts/auth-keycloak.sh"
        exit 1
    fi

    log_info "Installing auth script..."
    install -m 755 scripts/auth-keycloak.sh "$script_file"
    log_success "Auth script installed: $script_file"
}

##############################################
# Step 7: Install systemd Service
##############################################

install_service() {
    local service_path="/etc/systemd/system/$SERVICE_FILE"

    if [ ! -f "deploy/$SERVICE_FILE" ]; then
        log_error "Service file not found: deploy/$SERVICE_FILE"
        exit 1
    fi

    log_info "Installing SSO daemon systemd service..."
    install -m 644 "deploy/$SERVICE_FILE" "$service_path"

    log_success "Service installed: $SERVICE_FILE"
    log_info "Service is NOT enabled or started yet"
}

##############################################
# Step 8: Install OpenVPN Service Unit
##############################################

install_openvpn_service() {
    # Install a custom OpenVPN systemd unit that depends on the SSO daemon
    # and raises LimitNPROC for the Go auth script.
    #
    # Two options:
    #   A) Full replacement unit (deploy/openvpn-server@.service.example)
    #   B) Drop-in override for the stock unit (deploy/openvpn-sso-override.conf)
    #
    # We install the drop-in by default since it is less invasive.

    local ovpn_override_dir="/etc/systemd/system/openvpn-server@.service.d"

    if [ -f "deploy/openvpn-sso-override.conf" ]; then
        log_info "Installing OpenVPN server systemd override (LimitNPROC=512)..."
        mkdir -p "$ovpn_override_dir"
        install -m 644 "deploy/openvpn-sso-override.conf" \
            "$ovpn_override_dir/sso-override.conf"
        log_success "Override installed: $ovpn_override_dir/sso-override.conf"
    fi

    if [ -f "deploy/openvpn-server@.service.example" ]; then
        log_info "Example OpenVPN service unit available: deploy/openvpn-server@.service.example"
        log_info "To use the full replacement unit instead of the drop-in:"
        log_info "  sudo cp deploy/openvpn-server@.service.example \\"
        log_info "       /etc/systemd/system/openvpn_<instance>.service"
    fi

    log_info "Reloading systemd..."
    systemctl daemon-reload
    log_success "OpenVPN service configuration complete"
}

##############################################
# Step 9: Configure Firewall
##############################################

configure_firewall() {
    if ! command -v firewall-cmd &> /dev/null; then
        log_info "firewalld not installed, skipping firewall configuration"
        return
    fi

    if ! systemctl is-active --quiet firewalld; then
        log_info "firewalld not running, skipping firewall configuration"
        return
    fi

    log_info "Opening port $HTTP_PORT/tcp for HTTP callback..."
    
    if firewall-cmd --permanent --add-port=${HTTP_PORT}/tcp &> /dev/null; then
        firewall-cmd --reload &> /dev/null
        log_success "Firewall configured (port $HTTP_PORT/tcp opened)"
    else
        log_warning "Failed to configure firewall (may require manual configuration)"
    fi
}

##############################################
# Step 10: Configure SELinux
##############################################

configure_selinux() {
    if ! command -v getenforce &> /dev/null; then
        log_info "SELinux not installed, skipping SELinux configuration"
        return
    fi

    local selinux_status
    selinux_status=$(getenforce 2>/dev/null || echo "Disabled")

    if [ "$selinux_status" = "Disabled" ]; then
        log_info "SELinux is disabled, skipping SELinux configuration"
        return
    fi

    log_info "SELinux is $selinux_status, configuring contexts..."

    # File contexts (matches production Puppet fcontext resources)
    if command -v semanage &> /dev/null; then
        # Binary: bin_t so it can be executed from openvpn_t domain
        if semanage fcontext -a -t bin_t "$INSTALL_DIR/$BINARY_NAME" &> /dev/null; then
            log_success "SELinux file context added for binary (bin_t)"
        else
            log_warning "SELinux binary context may already exist"
        fi

        # Runtime / socket directory: openvpn_var_run_t so the OpenVPN
        # auth helper (openvpn_t) can connect to the daemon's Unix socket.
        if semanage fcontext -a -t openvpn_var_run_t '/var/run/openvpn-keycloak-sso(/.*)?' &> /dev/null; then
            log_success "SELinux file context added for socket directory (openvpn_var_run_t)"
        else
            log_warning "SELinux socket directory context may already exist"
        fi

        # Scripts directory: openvpn_unconfined_script_exec_t so OpenVPN
        # can execute the auth script in an unconfined helper domain.
        if semanage fcontext -a -t openvpn_unconfined_script_exec_t '/etc/openvpn/scripts(/.*)?' &> /dev/null; then
            log_success "SELinux file context added for scripts directory (openvpn_unconfined_script_exec_t)"
        else
            log_warning "SELinux scripts context may already exist"
        fi
    fi

    # Restore contexts on all managed paths
    if command -v restorecon &> /dev/null; then
        restorecon -v "$INSTALL_DIR/$BINARY_NAME" &> /dev/null || true
        restorecon -Rv /run/openvpn-keycloak-sso &> /dev/null || true
        restorecon -Rv /etc/openvpn/scripts &> /dev/null || true
        log_success "SELinux contexts restored"
    fi

    # SELinux booleans required for OpenVPN + SSO auth scripts
    if command -v setsebool &> /dev/null; then
        # Allow OpenVPN to run unconfined scripts (needed for Go auth binary)
        if setsebool -P openvpn_run_unconfined 1 &> /dev/null; then
            log_success "SELinux boolean set: openvpn_run_unconfined=on"
        else
            log_warning "Failed to set openvpn_run_unconfined (may not exist in this policy)"
        fi

        # Allow OpenVPN to make outbound network connections (OIDC callbacks)
        if setsebool -P openvpn_can_network_connect 1 &> /dev/null; then
            log_success "SELinux boolean set: openvpn_can_network_connect=on"
        else
            log_warning "Failed to set openvpn_can_network_connect (may not exist in this policy)"
        fi
    fi

    log_success "SELinux configuration complete"
    log_info "Note: If you encounter SELinux denials, check: ausearch -m avc -ts recent"
    log_info "Quick fix for testing: sudo setenforce 0  (sets SELinux to permissive)"
}

##############################################
# Success Message
##############################################

print_success_message() {
    echo ""
    echo -e "${BOLD}${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${GREEN}║          Installation Completed Successfully!               ║${NC}"
    echo -e "${BOLD}${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BOLD}Next Steps:${NC}"
    echo ""
    echo "1. ${BOLD}Edit Configuration:${NC}"
    echo "   vim $CONFIG_DIR/keycloak-sso.yaml"
    echo ""
    echo "2. ${BOLD}Validate Configuration:${NC}"
    echo "   $INSTALL_DIR/$BINARY_NAME check-config --config $CONFIG_DIR/keycloak-sso.yaml"
    echo ""
    echo "3. ${BOLD}Enable Service:${NC}"
    echo "   systemctl enable $SERVICE_FILE"
    echo ""
    echo "4. ${BOLD}Start Service:${NC}"
    echo "   systemctl start $SERVICE_FILE"
    echo ""
    echo "5. ${BOLD}Check Status:${NC}"
    echo "   systemctl status $SERVICE_FILE"
    echo ""
    echo "6. ${BOLD}View Logs:${NC}"
    echo "   journalctl -u $SERVICE_FILE -f"
    echo ""
    echo "7. ${BOLD}Configure OpenVPN Server:${NC}"
    echo "   See: config/openvpn-server.conf.example"
    echo "   See: docs/openvpn-server-setup.md"
    echo ""
    echo -e "${BOLD}Documentation:${NC}"
    echo "  - Keycloak setup: docs/keycloak-setup.md"
    echo "  - Server setup:   docs/openvpn-server-setup.md"
    echo "  - Client setup:   docs/client-setup.md"
    echo ""
}

##############################################
# Run Main Function
##############################################

main "$@"
