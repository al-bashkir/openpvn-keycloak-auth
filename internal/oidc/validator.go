package oidc

import (
	"fmt"
	"strings"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/config"
)

// Validator provides additional token validation beyond what go-oidc does.
// It validates username claims and enforces role/group requirements.
type Validator struct {
	oidcCfg *config.OIDCConfig
	authCfg *config.AuthConfig
}

// NewValidator creates a new token validator.
func NewValidator(oidcCfg *config.OIDCConfig, authCfg *config.AuthConfig) *Validator {
	return &Validator{
		oidcCfg: oidcCfg,
		authCfg: authCfg,
	}
}

// ValidateToken performs additional validation beyond what go-oidc does.
// It validates the username claim and required roles (if configured).
//
// Note: go-oidc already validates:
// - JWT signature via JWKS
// - Standard claims: iss, aud, exp, iat, nbf
//
// This function adds:
// - Username claim extraction and validation
// - Role/group enforcement
func (v *Validator) ValidateToken(claims map[string]interface{}, expectedUsername string) error {
	// 1. Validate username claim
	if err := v.validateUsername(claims, expectedUsername); err != nil {
		return err
	}

	// 2. Validate required roles (if configured)
	if len(v.oidcCfg.RequiredRoles) > 0 {
		if err := v.validateRoles(claims); err != nil {
			return err
		}
	}

	return nil
}

// validateUsername extracts and validates the username claim.
func (v *Validator) validateUsername(claims map[string]interface{}, expectedUsername string) error {
	// Extract username from configured claim
	username, err := getClaimString(claims, v.authCfg.UsernameClaim)
	if err != nil {
		return fmt.Errorf("username claim '%s' not found: %w", v.authCfg.UsernameClaim, err)
	}

	// Check if it matches expected username
	if username != expectedUsername {
		return fmt.Errorf("username mismatch: expected '%s', got '%s'", expectedUsername, username)
	}

	return nil
}

// ValidateRoles validates that the user has at least one of the required roles.
// It is a no-op when no roles are configured.
func (v *Validator) ValidateRoles(claims map[string]interface{}) error {
	if len(v.oidcCfg.RequiredRoles) == 0 {
		return nil
	}
	return v.validateRoles(claims)
}

// validateRoles validates that the user has at least one of the required roles.
func (v *Validator) validateRoles(claims map[string]interface{}) error {
	// Extract roles from configured claim path (e.g., "realm_access.roles")
	roles, err := getRolesFromClaim(claims, v.oidcCfg.RoleClaim)
	if err != nil {
		return fmt.Errorf("failed to extract roles: %w", err)
	}

	// Check if user has at least one of the required roles
	for _, requiredRole := range v.oidcCfg.RequiredRoles {
		if containsRole(roles, requiredRole) {
			return nil // User has required role
		}
	}

	return fmt.Errorf("user does not have required roles: %v (user roles: %v)", v.oidcCfg.RequiredRoles, roles)
}

// getClaimString extracts a string claim, supporting dot notation for nested claims.
// For example: "email", "preferred_username", "realm_access.roles"
func getClaimString(claims map[string]interface{}, path string) (string, error) {
	value, err := getNestedClaim(claims, path)
	if err != nil {
		return "", err
	}

	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("claim '%s' is not a string", path)
	}

	return str, nil
}

// getRolesFromClaim extracts roles as a slice of strings.
// Handles both []string and []interface{} types.
func getRolesFromClaim(claims map[string]interface{}, path string) ([]string, error) {
	value, err := getNestedClaim(claims, path)
	if err != nil {
		return nil, err
	}

	// Handle both []string and []interface{} types
	switch v := value.(type) {
	case []string:
		return v, nil
	case []interface{}:
		roles := make([]string, 0, len(v))
		for _, role := range v {
			if str, ok := role.(string); ok {
				roles = append(roles, str)
			}
		}
		return roles, nil
	default:
		return nil, fmt.Errorf("claim '%s' is not a string array", path)
	}
}

// getNestedClaim retrieves a claim using dot notation.
// For example: "realm_access.roles" navigates through the claims map.
func getNestedClaim(claims map[string]interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")

	var current interface{} = claims
	for i, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("claim path '%s' not found at level %d (%s)", path, i, part)
		}

		current, ok = m[part]
		if !ok {
			return nil, fmt.Errorf("claim '%s' not found in path '%s'", part, path)
		}
	}

	return current, nil
}

// containsRole checks if a role is present in the roles slice.
func containsRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
