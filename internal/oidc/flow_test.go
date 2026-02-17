package oidc

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestGenerateCodeVerifier(t *testing.T) {
	// Generate multiple verifiers and ensure they're unique
	seen := make(map[string]bool)

	for i := 0; i < 100; i++ {
		verifier, err := generateCodeVerifier()
		if err != nil {
			t.Fatalf("generateCodeVerifier failed: %v", err)
		}

		// Verify length (RFC 7636: 43-128 characters)
		if len(verifier) < 43 || len(verifier) > 128 {
			t.Errorf("verifier length = %d, want 43-128", len(verifier))
		}

		// Verify it's base64url encoded (no padding)
		if _, err := base64.RawURLEncoding.DecodeString(verifier); err != nil {
			t.Errorf("verifier is not valid base64url: %v", err)
		}

		// Ensure uniqueness
		if seen[verifier] {
			t.Errorf("duplicate verifier generated: %s", verifier)
		}

		seen[verifier] = true
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	tests := []struct {
		name     string
		verifier string
	}{
		{
			name:     "standard verifier",
			verifier: "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
		},
		{
			name:     "another verifier",
			verifier: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge := generateCodeChallenge(tt.verifier)

			// Verify length (SHA256 -> 32 bytes -> 43 chars base64url)
			if len(challenge) != 43 {
				t.Errorf("challenge length = %d, want 43", len(challenge))
			}

			// Verify it's base64url encoded
			decoded, err := base64.RawURLEncoding.DecodeString(challenge)
			if err != nil {
				t.Errorf("challenge is not valid base64url: %v", err)
			}

			// Verify it's a SHA256 hash (32 bytes)
			if len(decoded) != 32 {
				t.Errorf("decoded challenge length = %d, want 32", len(decoded))
			}

			// Manually verify the SHA256
			h := sha256.New()
			h.Write([]byte(tt.verifier))
			expected := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

			if challenge != expected {
				t.Errorf("challenge = %s, want %s", challenge, expected)
			}
		})
	}
}

func TestGenerateState(t *testing.T) {
	// Generate multiple states and ensure they're unique
	seen := make(map[string]bool)

	for i := 0; i < 100; i++ {
		state, err := generateState()
		if err != nil {
			t.Fatalf("generateState failed: %v", err)
		}

		// Verify length (16 bytes -> 32 hex chars)
		if len(state) != 32 {
			t.Errorf("state length = %d, want 32", len(state))
		}

		// Verify it's hex encoded
		for _, c := range state {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				t.Errorf("state contains non-hex character: %c", c)
			}
		}

		// Ensure uniqueness
		if seen[state] {
			t.Errorf("duplicate state generated: %s", state)
		}

		seen[state] = true
	}
}

func TestPKCEFlowConsistency(t *testing.T) {
	// Generate a verifier
	verifier, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier failed: %v", err)
	}

	// Generate challenge from the same verifier twice
	challenge1 := generateCodeChallenge(verifier)
	challenge2 := generateCodeChallenge(verifier)

	// They should be identical (deterministic)
	if challenge1 != challenge2 {
		t.Errorf("challenges differ for same verifier: %s != %s", challenge1, challenge2)
	}

	// Generate a different verifier
	verifier2, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier failed: %v", err)
	}

	// Generate challenge from different verifier
	challenge3 := generateCodeChallenge(verifier2)

	// It should be different
	if challenge1 == challenge3 {
		t.Errorf("challenges should differ for different verifiers")
	}
}

// makeTestJWT builds a fake JWT (header.payload.signature) with the given claims payload.
func makeTestJWT(t *testing.T, claims map[string]interface{}) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("failed to marshal claims: %v", err)
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + encodedPayload + ".fakesignature"
}

func TestDecodeJWTPayload(t *testing.T) {
	t.Run("valid JWT", func(t *testing.T) {
		original := map[string]interface{}{
			"sub":  "user123",
			"name": "Test User",
		}
		token := makeTestJWT(t, original)

		claims, err := decodeJWTPayload(token)
		if err != nil {
			t.Fatalf("decodeJWTPayload failed: %v", err)
		}

		if claims["sub"] != "user123" {
			t.Errorf("expected sub=user123, got %v", claims["sub"])
		}
		if claims["name"] != "Test User" {
			t.Errorf("expected name=Test User, got %v", claims["name"])
		}
	})

	t.Run("not a JWT", func(t *testing.T) {
		_, err := decodeJWTPayload("not-a-jwt")
		if err == nil {
			t.Error("expected error for non-JWT token")
		}
	})

	t.Run("invalid base64 payload", func(t *testing.T) {
		_, err := decodeJWTPayload("header.!!!invalid!!!.signature")
		if err == nil {
			t.Error("expected error for invalid base64 payload")
		}
	})

	t.Run("invalid JSON payload", func(t *testing.T) {
		badPayload := base64.RawURLEncoding.EncodeToString([]byte("not json"))
		_, err := decodeJWTPayload("header." + badPayload + ".signature")
		if err == nil {
			t.Error("expected error for invalid JSON payload")
		}
	})
}

func TestMergeAccessTokenClaims(t *testing.T) {
	t.Run("merges resource_access from access token", func(t *testing.T) {
		accessTokenClaims := map[string]interface{}{
			"sub": "user123",
			"resource_access": map[string]interface{}{
				"openvpn": map[string]interface{}{
					"roles": []interface{}{"vpn-user", "vpn-admin"},
				},
			},
			"realm_access": map[string]interface{}{
				"roles": []interface{}{"default-roles"},
			},
		}
		accessToken := makeTestJWT(t, accessTokenClaims)

		// ID token claims without resource_access
		dst := map[string]interface{}{
			"sub":                "user123",
			"preferred_username": "testuser",
		}

		mergeAccessTokenClaims(accessToken, dst)

		// resource_access should be merged
		ra, ok := dst["resource_access"]
		if !ok {
			t.Fatal("expected resource_access to be merged")
		}
		raMap, ok := ra.(map[string]interface{})
		if !ok {
			t.Fatal("expected resource_access to be a map")
		}
		openvpn, ok := raMap["openvpn"].(map[string]interface{})
		if !ok {
			t.Fatal("expected resource_access.openvpn to be a map")
		}
		roles, ok := openvpn["roles"].([]interface{})
		if !ok {
			t.Fatal("expected roles to be an array")
		}
		if len(roles) != 2 {
			t.Errorf("expected 2 roles, got %d", len(roles))
		}

		// realm_access should also be merged
		if _, ok := dst["realm_access"]; !ok {
			t.Error("expected realm_access to be merged")
		}

		// Original ID token claims should be preserved
		if dst["preferred_username"] != "testuser" {
			t.Error("ID token claims should be preserved")
		}
	})

	t.Run("does not overwrite existing ID token claims", func(t *testing.T) {
		accessTokenClaims := map[string]interface{}{
			"realm_access": map[string]interface{}{
				"roles": []interface{}{"from-access-token"},
			},
		}
		accessToken := makeTestJWT(t, accessTokenClaims)

		dst := map[string]interface{}{
			"realm_access": map[string]interface{}{
				"roles": []interface{}{"from-id-token"},
			},
		}

		mergeAccessTokenClaims(accessToken, dst)

		// Should keep ID token's realm_access, not overwrite
		ra := dst["realm_access"].(map[string]interface{})
		roles := ra["roles"].([]interface{})
		if roles[0] != "from-id-token" {
			t.Errorf("expected ID token claim to be preserved, got %v", roles[0])
		}
	})

	t.Run("handles empty access token", func(t *testing.T) {
		dst := map[string]interface{}{"sub": "user"}
		mergeAccessTokenClaims("", dst)
		// Should not panic or modify dst
		if len(dst) != 1 {
			t.Error("dst should not be modified for empty token")
		}
	})

	t.Run("handles opaque access token gracefully", func(t *testing.T) {
		dst := map[string]interface{}{"sub": "user"}
		mergeAccessTokenClaims("opaque-token-no-dots", dst)
		// Should not panic or modify dst
		if len(dst) != 1 {
			t.Error("dst should not be modified for opaque token")
		}
	})
}
