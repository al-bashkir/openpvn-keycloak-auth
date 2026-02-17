#!/bin/bash
# Uninstallation script for OpenVPN Keycloak SSO
#
# This script removes the OpenVPN Keycloak SSO daemon from the system.
# It performs the following tasks:
#   1. Stops and disables the systemd service
#   2. Removes the systemd service file
#   3. Removes the binary
#   4. Removes the auth script (from /etc/openvpn/scripts/)
#   5. Removes OpenVPN service override (drop-in)
#   6. Optionally removes configuration files
#   7. Removes data and runtime directories
#   8. Removes SELinux file contexts
#
# Usage:
#   sudo ./deploy/uninstall.sh
#
# Requirements:
#   - Root privileges

set -e

##############################################
# Configuration
##############################################

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/openvpn"
SCRIPTS_DIR="/etc/openvpn/scripts"
DATA_DIR="/var/lib/openvpn-keycloak-auth"
RUN_DIR="/run/openvpn-keycloak-auth"
BINARY_NAME="openvpn-keycloak-auth"
SERVICE_FILE="openvpn-keycloak-auth.service"
OVPN_OVERRIDE_DIR="/etc/systemd/system/openvpn-server@.service.d"

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

##############################################
# Main Uninstallation Steps
##############################################

main() {
    echo -e "${BOLD}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}║    OpenVPN Keycloak SSO - Uninstallation Script            ║${NC}"
    echo -e "${BOLD}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    # Check root
    check_root
    echo ""

    log_warning "${BOLD}This will remove OpenVPN Keycloak SSO from your system.${NC}"
    echo ""
    read -p "Are you sure you want to continue? (y/N) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Uninstallation cancelled"
        exit 0
    fi
    echo ""

    # Step 1: Stop and disable service
    log_info "Step 1/8: Stopping and disabling service..."
    stop_service
    echo ""

    # Step 2: Remove systemd service file
    log_info "Step 2/8: Removing systemd service file..."
    remove_service
    echo ""

    # Step 3: Remove binary
    log_info "Step 3/8: Removing binary..."
    remove_binary
    echo ""

    # Step 4: Remove auth script
    log_info "Step 4/8: Removing auth script..."
    remove_auth_script
    echo ""

    # Step 5: Remove OpenVPN service override
    log_info "Step 5/8: Removing OpenVPN service override..."
    remove_openvpn_override
    echo ""

    # Step 6: Remove configuration (optional)
    log_info "Step 6/8: Configuration files..."
    remove_config
    echo ""

    # Step 7: Remove data directory (optional)
    log_info "Step 7/8: Data and runtime directories..."
    remove_data
    echo ""

    # Step 8: Remove SELinux contexts
    log_info "Step 8/8: Removing SELinux contexts..."
    remove_selinux
    echo ""

    # Uninstallation complete
    print_success_message
}

##############################################
# Step 1: Stop and Disable Service
##############################################

stop_service() {
    local service_path="/etc/systemd/system/$SERVICE_FILE"

    if [ ! -f "$service_path" ]; then
        log_info "Service file not found (already removed?)"
        return
    fi

    # Check if service is active
    if systemctl is-active --quiet "$SERVICE_FILE" 2>/dev/null; then
        log_info "Stopping service..."
        systemctl stop "$SERVICE_FILE"
        log_success "Service stopped"
    else
        log_info "Service is not running"
    fi

    # Check if service is enabled
    if systemctl is-enabled --quiet "$SERVICE_FILE" 2>/dev/null; then
        log_info "Disabling service..."
        systemctl disable "$SERVICE_FILE" >/dev/null 2>&1
        log_success "Service disabled"
    else
        log_info "Service is not enabled"
    fi
}

##############################################
# Step 2: Remove Service File
##############################################

remove_service() {
    local service_path="/etc/systemd/system/$SERVICE_FILE"

    if [ -f "$service_path" ]; then
        log_info "Removing systemd service file..."
        rm -f "$service_path"
        
        log_info "Reloading systemd..."
        systemctl daemon-reload
        systemctl reset-failed 2>/dev/null || true
        
        log_success "Service file removed"
    else
        log_info "Service file not found (already removed?)"
    fi
}

##############################################
# Step 3: Remove Binary
##############################################

remove_binary() {
    local binary_path="$INSTALL_DIR/$BINARY_NAME"

    if [ -f "$binary_path" ]; then
        log_info "Removing binary: $binary_path"
        rm -f "$binary_path"
        log_success "Binary removed"
    else
        log_info "Binary not found (already removed?)"
    fi
}

##############################################
# Step 4: Remove Auth Script
##############################################

remove_auth_script() {
    local script_path="$SCRIPTS_DIR/auth-keycloak.sh"

    if [ -f "$script_path" ]; then
        log_info "Removing auth script: $script_path"
        rm -f "$script_path"
        log_success "Auth script removed"
    else
        log_info "Auth script not found (already removed?)"
    fi

    # Also remove legacy location if present (pre-v1 installs)
    if [ -f "$CONFIG_DIR/auth-keycloak.sh" ]; then
        log_info "Removing legacy auth script: $CONFIG_DIR/auth-keycloak.sh"
        rm -f "$CONFIG_DIR/auth-keycloak.sh"
    fi

    # Remove scripts directory if empty
    if [ -d "$SCRIPTS_DIR" ] && [ -z "$(ls -A "$SCRIPTS_DIR" 2>/dev/null)" ]; then
        log_info "Removing empty scripts directory: $SCRIPTS_DIR"
        rmdir "$SCRIPTS_DIR" 2>/dev/null || true
    fi
}

##############################################
# Step 5: Remove OpenVPN Service Override
##############################################

remove_openvpn_override() {
    local override_file="$OVPN_OVERRIDE_DIR/sso-override.conf"

    if [ -f "$override_file" ]; then
        log_info "Removing OpenVPN service override: $override_file"
        rm -f "$override_file"
        log_success "Override removed"

        # Remove the drop-in directory if empty
        if [ -d "$OVPN_OVERRIDE_DIR" ] && [ -z "$(ls -A "$OVPN_OVERRIDE_DIR" 2>/dev/null)" ]; then
            rmdir "$OVPN_OVERRIDE_DIR" 2>/dev/null || true
        fi

        systemctl daemon-reload 2>/dev/null || true
    else
        log_info "OpenVPN service override not found (already removed?)"
    fi
}

##############################################
# Step 5: Remove Configuration (Optional)
##############################################

remove_config() {
    local config_file="$CONFIG_DIR/keycloak-sso.yaml"
    local backup_file="$CONFIG_DIR/keycloak-sso.yaml.backup"

    if [ ! -f "$config_file" ]; then
        log_info "Configuration file not found (already removed?)"
        return
    fi

    echo ""
    log_warning "Configuration file: $config_file"
    log_warning "This file contains your Keycloak settings and secrets."
    echo ""
    read -p "Remove configuration file? (y/N) " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # Create backup before removing
        log_info "Creating backup: $backup_file"
        cp "$config_file" "$backup_file"
        chmod 600 "$backup_file"
        
        log_info "Removing configuration file..."
        rm -f "$config_file"
        log_success "Configuration removed (backup saved: $backup_file)"
    else
        log_info "Configuration file preserved: $config_file"
    fi
}

##############################################
# Step 7: Remove Data and Runtime Directories (Optional)
##############################################

remove_data() {
    # Runtime directory (socket)
    if [ -d "$RUN_DIR" ]; then
        log_info "Removing runtime directory: $RUN_DIR"
        rm -rf "$RUN_DIR"
        log_success "Runtime directory removed"
    fi

    # Data directory (requires confirmation)
    if [ ! -d "$DATA_DIR" ]; then
        log_info "Data directory not found (already removed?)"
        return
    fi

    echo ""
    log_warning "Data directory: $DATA_DIR"
    log_warning "This directory contains session data and the shared tmp directory."
    echo ""
    read -p "Remove data directory? (y/N) " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "Removing data directory..."
        rm -rf "$DATA_DIR"
        log_success "Data directory removed"
    else
        log_info "Data directory preserved: $DATA_DIR"
    fi
}

##############################################
# Step 8: Remove SELinux Contexts
##############################################

remove_selinux() {
    if ! command -v getenforce &> /dev/null; then
        log_info "SELinux not installed, skipping"
        return
    fi

    local selinux_status
    selinux_status=$(getenforce 2>/dev/null || echo "Disabled")

    if [ "$selinux_status" = "Disabled" ]; then
        log_info "SELinux is disabled, skipping"
        return
    fi

    if command -v semanage &> /dev/null; then
        semanage fcontext -d "$INSTALL_DIR/$BINARY_NAME" &> /dev/null || true
        semanage fcontext -d '/var/run/openvpn-keycloak-auth(/.*)?' &> /dev/null || true
        semanage fcontext -d '/etc/openvpn/scripts(/.*)?' &> /dev/null || true
        log_success "SELinux file contexts removed"
    fi

    # Note: We do NOT revert setsebool (openvpn_run_unconfined,
    # openvpn_can_network_connect) because other OpenVPN plugins
    # or scripts may depend on them.
    log_info "SELinux booleans were NOT reverted (may be used by other components)"
}

##############################################
# Success Message
##############################################

print_success_message() {
    echo ""
    echo -e "${BOLD}${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${GREEN}║        Uninstallation Completed Successfully!               ║${NC}"
    echo -e "${BOLD}${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BOLD}Removed Components:${NC}"
    echo "  ✓ systemd service"
    echo "  ✓ Binary ($INSTALL_DIR/$BINARY_NAME)"
    echo "  ✓ Auth script ($SCRIPTS_DIR/auth-keycloak.sh)"
    echo "  ✓ OpenVPN service override (if installed)"
    echo ""
    
    if [ -f "$CONFIG_DIR/keycloak-sso.yaml" ]; then
        echo -e "${BOLD}Preserved:${NC}"
        echo "  • Configuration: $CONFIG_DIR/keycloak-sso.yaml"
        echo ""
    fi
    
    if [ -d "$DATA_DIR" ]; then
        echo -e "${BOLD}Preserved:${NC}"
        echo "  • Data directory: $DATA_DIR"
        echo ""
    fi

    echo -e "${BOLD}Notes:${NC}"
    echo "  - The 'openvpn' user/group was NOT removed (may be used by OpenVPN)"
    echo "  - Firewall rules were NOT removed"
    echo "  - SELinux contexts were NOT removed"
    echo ""
    echo "If you want to completely clean up:"
    echo "  - Remove firewall rule: firewall-cmd --permanent --remove-port=9000/tcp && firewall-cmd --reload"
    echo "  - Remove data directory: rm -rf $DATA_DIR"
    echo "  - Remove configuration: rm -f $CONFIG_DIR/keycloak-sso.yaml*"
    echo ""
}

##############################################
# Run Main Function
##############################################

main "$@"
