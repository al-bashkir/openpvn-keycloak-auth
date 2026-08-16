package oidc

import (
	"fmt"
	"slices"
	"strings"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/config"
)

// Validate performs the policy checks that go-oidc does not: the username
// claim must match the VPN username (unless auth.allow_username_mismatch is
// set), and the user must hold one of oidc.required_roles.
//
// go-oidc already validates the JWT signature via JWKS and the standard
// iss/aud/exp/iat/nbf claims.
func Validate(cfg *config.Config, claims map[string]interface{}, expectedUsername string) error {
	if !cfg.Auth.AllowUsernameMismatch {
		value, err := getNestedClaim(claims, cfg.Auth.UsernameClaim)
		if err != nil {
			return fmt.Errorf("username claim '%s' not found: %w", cfg.Auth.UsernameClaim, err)
		}
		username, ok := value.(string)
		if !ok {
			return fmt.Errorf("username claim '%s' is not a string", cfg.Auth.UsernameClaim)
		}
		if username != expectedUsername {
			return fmt.Errorf("username mismatch: expected '%s', got '%s'", expectedUsername, username)
		}
	}

	// Roles are enforced even when the VPN username is allowed to differ.
	if len(cfg.OIDC.RequiredRoles) == 0 {
		return nil
	}

	roles, err := getRolesFromClaim(claims, cfg.OIDC.RoleClaim)
	if err != nil {
		return fmt.Errorf("failed to extract roles: %w", err)
	}
	for _, requiredRole := range cfg.OIDC.RequiredRoles {
		if slices.Contains(roles, requiredRole) {
			return nil
		}
	}

	return fmt.Errorf("user does not have required roles: %v (user roles: %v)", cfg.OIDC.RequiredRoles, roles)
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
