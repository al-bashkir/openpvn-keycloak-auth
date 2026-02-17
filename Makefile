# Makefile for OpenVPN Keycloak SSO
#
# Common targets:
#   make build       - Build production binary
#   make test        - Run tests
#   make install     - Install to system (requires root)
#   make uninstall   - Remove from system (requires root)
#   make clean       - Clean build artifacts

##############################################
# Variables
##############################################

# Binary name
BINARY_NAME := openvpn-keycloak-auth

# Version information (from git or default)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Go build flags
LDFLAGS := -ldflags="-s -w \
	-X 'main.version=$(VERSION)' \
	-X 'main.commit=$(COMMIT)' \
	-X 'main.buildDate=$(BUILD_DATE)'"
BUILD_FLAGS := -trimpath

# Directories
SRC_DIR := ./cmd/openvpn-keycloak-auth
BUILD_DIR := ./build
DIST_DIR := ./dist

# Install directories
INSTALL_DIR := /usr/local/bin
CONFIG_DIR := /etc/openvpn
DATA_DIR := /var/lib/openvpn-keycloak-auth

##############################################
# Phony Targets
##############################################

.PHONY: all build build-dev build-static test test-verbose test-coverage \
        lint fmt vet check install uninstall clean dist version help

##############################################
# Default Target
##############################################

all: build

##############################################
# Build Targets
##############################################

# Development build (fast, with debug info)
build-dev:
	@echo "Building $(BINARY_NAME) (development)..."
	go build -o $(BINARY_NAME) $(SRC_DIR)
	@echo "✓ Development build complete: $(BINARY_NAME)"

# Production build (optimized, static)
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@echo "  Commit:     $(COMMIT)"
	@echo "  Build Date: $(BUILD_DATE)"
	CGO_ENABLED=0 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(SRC_DIR)
	@echo "✓ Production build complete: $(BINARY_NAME)"
	@ls -lh $(BINARY_NAME)

# Fully static build (for containers/minimal systems)
build-static:
	@echo "Building static $(BINARY_NAME) $(VERSION)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		$(BUILD_FLAGS) $(LDFLAGS) \
		-tags netgo -a -installsuffix netgo \
		-o $(BINARY_NAME) $(SRC_DIR)
	@echo "✓ Static build complete: $(BINARY_NAME)"
	@file $(BINARY_NAME)

##############################################
# Test Targets
##############################################

# Run tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	go test -v -race -coverprofile=$(BUILD_DIR)/coverage.out ./...
	go tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "✓ Coverage report: $(BUILD_DIR)/coverage.html"

# Run tests verbosely
test-verbose:
	@echo "Running tests (verbose)..."
	go test -v -race -count=1 ./...

# Run specific test
test-one:
	@echo "Running specific test..."
	@if [ -z "$(TEST)" ]; then \
		echo "Error: Specify test with TEST=TestName"; \
		exit 1; \
	fi
	go test -v -race -run $(TEST) ./...

##############################################
# Code Quality Targets
##############################################

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "✓ Vet passed"

# Run all checks
check: fmt vet test
	@echo "✓ All checks passed"

##############################################
# Installation Targets
##############################################

# Install to system (requires root)
install: build
	@echo "Installing $(BINARY_NAME)..."
	@if [ "$$(id -u)" -ne 0 ]; then \
		echo "Error: Installation requires root privileges"; \
		echo "Try: sudo make install"; \
		exit 1; \
	fi
	@chmod +x deploy/install.sh
	@./deploy/install.sh

# Uninstall from system (requires root)
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@if [ "$$(id -u)" -ne 0 ]; then \
		echo "Error: Uninstallation requires root privileges"; \
		echo "Try: sudo make uninstall"; \
		exit 1; \
	fi
	@chmod +x deploy/uninstall.sh
	@./deploy/uninstall.sh

##############################################
# Distribution Targets
##############################################

# Create distribution tarball
dist: build
	@echo "Creating distribution package..."
	@mkdir -p $(DIST_DIR)
	@tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz \
		$(BINARY_NAME) \
		config/ \
		scripts/ \
		deploy/ \
		docs/ \
		README.md \
		LICENSE \
		AGENTS.md
	@echo "✓ Package created: $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz"
	@ls -lh $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz

# Create distribution with checksums
dist-checksums: dist
	@echo "Generating checksums..."
	@cd $(DIST_DIR) && sha256sum $(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz > $(BINARY_NAME)-$(VERSION)-checksums.txt
	@echo "✓ Checksums: $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-checksums.txt"
	@cat $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-checksums.txt

##############################################
# Utility Targets
##############################################

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@go clean
	@echo "✓ Clean complete"

# Show version information
version:
	@echo "Version Information:"
	@echo "  Version:    $(VERSION)"
	@echo "  Commit:     $(COMMIT)"
	@echo "  Build Date: $(BUILD_DATE)"

# Show help
help:
	@echo "OpenVPN Keycloak Auth - Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build Targets:"
	@echo "  build          Build production binary (default)"
	@echo "  build-dev      Build development binary (faster, with debug info)"
	@echo "  build-static   Build fully static binary"
	@echo ""
	@echo "Test Targets:"
	@echo "  test           Run tests with race detector"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  test-verbose   Run tests verbosely"
	@echo "  test-one       Run specific test (TEST=TestName)"
	@echo ""
	@echo "Code Quality:"
	@echo "  lint           Run golangci-lint"
	@echo "  fmt            Format code with go fmt"
	@echo "  vet            Run go vet"
	@echo "  check          Run fmt, vet, and tests"
	@echo ""
	@echo "Installation:"
	@echo "  install        Install to system (requires root)"
	@echo "  uninstall      Remove from system (requires root)"
	@echo ""
	@echo "Distribution:"
	@echo "  dist           Create distribution tarball"
	@echo "  dist-checksums Create tarball with checksums"
	@echo ""
	@echo "Utilities:"
	@echo "  clean          Clean build artifacts"
	@echo "  version        Show version information"
	@echo "  help           Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build              # Build production binary"
	@echo "  make test               # Run tests"
	@echo "  sudo make install       # Install to system"
	@echo "  make dist               # Create distribution package"
	@echo "  make test-one TEST=TestValidateToken  # Run specific test"
	@echo ""

##############################################
# Dependencies
##############################################

# Ensure go.mod and go.sum are up to date
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	@echo "✓ Dependencies downloaded"

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	go mod tidy
	@echo "✓ Dependencies tidied"

# Update dependencies
.PHONY: update
update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
	@echo "✓ Dependencies updated"

##############################################
# Development Targets
##############################################

# Run the daemon locally (for development)
.PHONY: run
run: build-dev
	@echo "Running $(BINARY_NAME) serve..."
	@if [ ! -f config/openvpn-keycloak-auth.yaml ]; then \
		echo "Error: config/openvpn-keycloak-auth.yaml not found"; \
		echo "Copy from config/openvpn-keycloak-auth.yaml.example and edit"; \
		exit 1; \
	fi
	./$(BINARY_NAME) serve --config config/openvpn-keycloak-auth.yaml

# Check configuration
.PHONY: check-config
check-config: build-dev
	@echo "Checking configuration..."
	@if [ ! -f config/openvpn-keycloak-auth.yaml ]; then \
		echo "Error: config/openvpn-keycloak-auth.yaml not found"; \
		exit 1; \
	fi
	./$(BINARY_NAME) check-config --config config/openvpn-keycloak-auth.yaml

# Watch and rebuild on changes (requires entr)
.PHONY: watch
watch:
	@if ! command -v entr >/dev/null 2>&1; then \
		echo "Error: 'entr' not installed"; \
		echo "Install: dnf install entr (Rocky/RHEL) or apt install entr (Debian/Ubuntu)"; \
		exit 1; \
	fi
	@echo "Watching for changes..."
	find . -name '*.go' | entr -r make build-dev

##############################################
# Documentation
##############################################

# Generate documentation
.PHONY: docs
docs:
	@echo "Documentation files:"
	@echo "  README.md                      - Project overview"
	@echo "  AGENTS.md                      - Agent instructions"
	@echo "  docs/keycloak-setup.md         - Keycloak configuration"
	@echo "  docs/openvpn-server-setup.md   - OpenVPN server setup"
	@echo "  docs/client-setup.md           - Client setup guide"
	@echo ""
	@echo "Configuration examples:"
	@echo "  config/openvpn-keycloak-auth.yaml.example  - Daemon config"
	@echo "  config/openvpn-server.conf.example        - OpenVPN server"
	@echo "  config/client.ovpn.example                - Universal client"
	@echo ""
