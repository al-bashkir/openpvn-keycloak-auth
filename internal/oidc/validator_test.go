package oidc

import (
	"strings"
	"testing"

	"github.com/al-bashkir/openvpn-keycloak/internal/config"
)

func TestValidateToken_UsernameOnly(t *testing.T) {
	oidcCfg := &config.OIDCConfig{}
	authCfg := &config.AuthConfig{
		UsernameClaim: "preferred_username",
	}

	validator := NewValidator(oidcCfg, authCfg)

	tests := []struct {
		name            string
		claims          map[string]interface{}
		expectedUser    string
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "valid username",
			claims: map[string]interface{}{
				"preferred_username": "testuser",
			},
			expectedUser: "testuser",
			wantErr:      false,
		},
		{
			name: "username mismatch",
			claims: map[string]interface{}{
				"preferred_username": "wronguser",
			},
			expectedUser:    "testuser",
			wantErr:         true,
			wantErrContains: "username mismatch",
		},
		{
			name:            "username claim missing",
			claims:          map[string]interface{}{},
			expectedUser:    "testuser",
			wantErr:         true,
			wantErrContains: "not found",
		},
		{
			name: "username claim wrong type",
			claims: map[string]interface{}{
				"preferred_username": 123,
			},
			expectedUser:    "testuser",
			wantErr:         true,
			wantErrContains: "not a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateToken(tt.claims, tt.expectedUser)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error = %v, want error containing %q", err, tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateToken_WithRoles(t *testing.T) {
	oidcCfg := &config.OIDCConfig{
		RequiredRoles: []string{"vpn-user"},
		RoleClaim:     "realm_access.roles",
	}
	authCfg := &config.AuthConfig{
		UsernameClaim: "preferred_username",
	}

	validator := NewValidator(oidcCfg, authCfg)

	tests := []struct {
		name            string
		claims          map[string]interface{}
		expectedUser    string
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "valid user with required role",
			claims: map[string]interface{}{
				"preferred_username": "testuser",
				"realm_access": map[string]interface{}{
					"roles": []interface{}{"vpn-user", "other-role"},
				},
			},
			expectedUser: "testuser",
			wantErr:      false,
		},
		{
			name: "valid user with multiple required roles (has one)",
			claims: map[string]interface{}{
				"preferred_username": "testuser",
				"realm_access": map[string]interface{}{
					"roles": []string{"admin", "vpn-user"},
				},
			},
			expectedUser: "testuser",
			wantErr:      false,
		},
		{
			name: "user missing required role",
			claims: map[string]interface{}{
				"preferred_username": "testuser",
				"realm_access": map[string]interface{}{
					"roles": []string{"other-role"},
				},
			},
			expectedUser:    "testuser",
			wantErr:         true,
			wantErrContains: "does not have required roles",
		},
		{
			name: "realm_access missing",
			claims: map[string]interface{}{
				"preferred_username": "testuser",
			},
			expectedUser:    "testuser",
			wantErr:         true,
			wantErrContains: "not found",
		},
		{
			name: "roles missing in realm_access",
			claims: map[string]interface{}{
				"preferred_username": "testuser",
				"realm_access":       map[string]interface{}{},
			},
			expectedUser:    "testuser",
			wantErr:         true,
			wantErrContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateToken(tt.claims, tt.expectedUser)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error = %v, want error containing %q", err, tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetNestedClaim(t *testing.T) {
	claims := map[string]interface{}{
		"simple": "value",
		"nested": map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": "deep_value",
			},
		},
		"realm_access": map[string]interface{}{
			"roles": []string{"role1", "role2"},
		},
	}

	tests := []struct {
		name      string
		path      string
		wantValue interface{}
		wantErr   bool
	}{
		{
			name:      "simple claim",
			path:      "simple",
			wantValue: "value",
			wantErr:   false,
		},
		{
			name:      "nested claim level 1",
			path:      "realm_access.roles",
			wantValue: []string{"role1", "role2"},
			wantErr:   false,
		},
		{
			name:      "nested claim level 2",
			path:      "nested.level1.level2",
			wantValue: "deep_value",
			wantErr:   false,
		},
		{
			name:    "non-existent claim",
			path:    "nonexistent",
			wantErr: true,
		},
		{
			name:    "invalid nested path",
			path:    "simple.invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := getNestedClaim(claims, tt.path)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// For slice comparison, check length and contents
			if wantSlice, ok := tt.wantValue.([]string); ok {
				gotSlice, ok := value.([]string)
				if !ok {
					t.Errorf("value type = %T, want []string", value)
					return
				}
				if len(gotSlice) != len(wantSlice) {
					t.Errorf("slice length = %d, want %d", len(gotSlice), len(wantSlice))
					return
				}
				for i := range wantSlice {
					if gotSlice[i] != wantSlice[i] {
						t.Errorf("slice[%d] = %v, want %v", i, gotSlice[i], wantSlice[i])
					}
				}
				return
			}

			if value != tt.wantValue {
				t.Errorf("value = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

func TestGetRolesFromClaim(t *testing.T) {
	tests := []struct {
		name      string
		claims    map[string]interface{}
		path      string
		wantRoles []string
		wantErr   bool
	}{
		{
			name: "roles as []string",
			claims: map[string]interface{}{
				"realm_access": map[string]interface{}{
					"roles": []string{"role1", "role2"},
				},
			},
			path:      "realm_access.roles",
			wantRoles: []string{"role1", "role2"},
			wantErr:   false,
		},
		{
			name: "roles as []interface{}",
			claims: map[string]interface{}{
				"realm_access": map[string]interface{}{
					"roles": []interface{}{"role1", "role2"},
				},
			},
			path:      "realm_access.roles",
			wantRoles: []string{"role1", "role2"},
			wantErr:   false,
		},
		{
			name: "roles with mixed types in []interface{}",
			claims: map[string]interface{}{
				"realm_access": map[string]interface{}{
					"roles": []interface{}{"role1", 123, "role2"},
				},
			},
			path:      "realm_access.roles",
			wantRoles: []string{"role1", "role2"},
			wantErr:   false,
		},
		{
			name: "roles as wrong type",
			claims: map[string]interface{}{
				"realm_access": map[string]interface{}{
					"roles": "not-an-array",
				},
			},
			path:    "realm_access.roles",
			wantErr: true,
		},
		{
			name:    "roles missing",
			claims:  map[string]interface{}{},
			path:    "realm_access.roles",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles, err := getRolesFromClaim(tt.claims, tt.path)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(roles) != len(tt.wantRoles) {
				t.Errorf("roles length = %d, want %d", len(roles), len(tt.wantRoles))
				return
			}

			for i, role := range roles {
				if role != tt.wantRoles[i] {
					t.Errorf("roles[%d] = %s, want %s", i, role, tt.wantRoles[i])
				}
			}
		})
	}
}

func TestContainsRole(t *testing.T) {
	roles := []string{"admin", "vpn-user", "developer"}

	tests := []struct {
		name string
		role string
		want bool
	}{
		{"role exists", "vpn-user", true},
		{"role exists at start", "admin", true},
		{"role exists at end", "developer", true},
		{"role does not exist", "nonexistent", false},
		{"empty role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsRole(roles, tt.role); got != tt.want {
				t.Errorf("containsRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateToken_MultipleRequiredRoles(t *testing.T) {
	oidcCfg := &config.OIDCConfig{
		RequiredRoles: []string{"vpn-user", "vpn-admin"},
		RoleClaim:     "realm_access.roles",
	}
	authCfg := &config.AuthConfig{
		UsernameClaim: "preferred_username",
	}

	validator := NewValidator(oidcCfg, authCfg)

	// User has one of the required roles (should pass)
	claims := map[string]interface{}{
		"preferred_username": "testuser",
		"realm_access": map[string]interface{}{
			"roles": []string{"vpn-admin"},
		},
	}

	err := validator.ValidateToken(claims, "testuser")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateToken_NoRequiredRoles(t *testing.T) {
	oidcCfg := &config.OIDCConfig{
		RequiredRoles: []string{}, // No roles required
		RoleClaim:     "realm_access.roles",
	}
	authCfg := &config.AuthConfig{
		UsernameClaim: "preferred_username",
	}

	validator := NewValidator(oidcCfg, authCfg)

	// User doesn't need any roles
	claims := map[string]interface{}{
		"preferred_username": "testuser",
	}

	err := validator.ValidateToken(claims, "testuser")
	if err != nil {
		t.Errorf("expected no error when no roles required, got: %v", err)
	}
}
