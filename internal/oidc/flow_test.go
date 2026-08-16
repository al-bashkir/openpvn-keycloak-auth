package oidc

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coreosoidc "github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
	"golang.org/x/oauth2"
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

func makeTestProvider(t *testing.T) (*Provider, *rsa.PrivateKey) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	keySet := &coreosoidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&privateKey.PublicKey}}
	verifier := coreosoidc.NewVerifier("https://issuer.example", keySet, &coreosoidc.Config{
		SkipClientIDCheck: true,
	})

	return &Provider{
		accessTokenVerifier: verifier,
		clientID:            "openvpn",
	}, privateKey
}

func makeSignedAccessToken(t *testing.T, privateKey *rsa.PrivateKey, claims map[string]interface{}) string {
	t.Helper()

	if _, ok := claims["iss"]; !ok {
		claims["iss"] = "https://issuer.example"
	}
	if _, ok := claims["sub"]; !ok {
		claims["sub"] = "user123"
	}
	if _, ok := claims["exp"]; !ok {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}
	if _, ok := claims["iat"]; !ok {
		claims["iat"] = time.Now().Add(-time.Minute).Unix()
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("failed to marshal claims: %v", err)
	}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: privateKey}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	serialized, err := signed.CompactSerialize()
	if err != nil {
		t.Fatalf("failed to serialize token: %v", err)
	}

	return serialized
}

// exchangeCodeFixture wires a Provider against an in-memory token endpoint
// that returns id/access tokens signed by privateKey. Tests can mutate the
// returned `id` map between calls to control the issued id_token claims.
type exchangeCodeFixture struct {
	provider *Provider
	id       map[string]interface{}
	omitID   bool
	close    func()
}

func newExchangeCodeFixture(t *testing.T) *exchangeCodeFixture {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	const issuer = "https://issuer.example"
	const audience = "openvpn"

	fx := &exchangeCodeFixture{
		id: map[string]interface{}{
			"iss":   issuer,
			"aud":   audience,
			"sub":   "user-123",
			"exp":   time.Now().Add(time.Hour).Unix(),
			"iat":   time.Now().Add(-time.Minute).Unix(),
			"nonce": "expected-nonce",
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		access := makeSignedAccessToken(t, privateKey, map[string]interface{}{
			"iss": issuer,
			"aud": []string{audience},
			"sub": "user-123",
		})
		body := map[string]interface{}{
			"access_token": access,
			"token_type":   "Bearer",
			"expires_in":   3600,
		}
		if !fx.omitID {
			body["id_token"] = makeSignedAccessToken(t, privateKey, fx.id)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	})
	ts := httptest.NewServer(mux)
	fx.close = ts.Close

	keySet := &coreosoidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&privateKey.PublicKey}}
	fx.provider = &Provider{
		oauth2Config: &oauth2.Config{
			ClientID: audience,
			Endpoint: oauth2.Endpoint{TokenURL: ts.URL + "/token"},
		},
		verifier:            coreosoidc.NewVerifier(issuer, keySet, &coreosoidc.Config{ClientID: audience}),
		accessTokenVerifier: coreosoidc.NewVerifier(issuer, keySet, &coreosoidc.Config{SkipClientIDCheck: true}),
		clientID:            audience,
	}

	t.Cleanup(fx.close)
	return fx
}

func TestExchangeCode_NonceMatch(t *testing.T) {
	fx := newExchangeCodeFixture(t)

	claims, err := fx.provider.ExchangeCode(context.Background(), "code", "verifier", "expected-nonce")
	if err != nil {
		t.Fatalf("ExchangeCode failed: %v", err)
	}
	if claims == nil {
		t.Fatal("expected claims to be returned")
	}
	if claims["sub"] != "user-123" {
		t.Errorf("sub = %v, want user-123", claims["sub"])
	}
}

func TestExchangeCode_NonceMismatch(t *testing.T) {
	fx := newExchangeCodeFixture(t)
	fx.id["nonce"] = "attacker-nonce"

	_, err := fx.provider.ExchangeCode(context.Background(), "code", "verifier", "expected-nonce")
	if err == nil {
		t.Fatal("expected nonce mismatch error")
	}
	if !errors.Is(err, errIDTokenNonceMismatch) {
		t.Fatalf("err = %v, want errIDTokenNonceMismatch", err)
	}
}

func TestExchangeCode_EmptyExpectedNonce(t *testing.T) {
	fx := newExchangeCodeFixture(t)

	_, err := fx.provider.ExchangeCode(context.Background(), "code", "verifier", "")
	if err == nil {
		t.Fatal("expected error for empty expected nonce")
	}
	if !errors.Is(err, errIDTokenNonceMismatch) {
		t.Fatalf("err = %v, want errIDTokenNonceMismatch", err)
	}
}

func TestExchangeCode_MissingIDToken(t *testing.T) {
	fx := newExchangeCodeFixture(t)
	fx.omitID = true

	_, err := fx.provider.ExchangeCode(context.Background(), "code", "verifier", "expected-nonce")
	if err == nil {
		t.Fatal("expected error when id_token missing")
	}
	if !errors.Is(err, errIDTokenMissing) {
		t.Fatalf("err = %v, want errIDTokenMissing", err)
	}
}

func TestGenerateNonce(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		n, err := generateNonce()
		if err != nil {
			t.Fatalf("generateNonce failed: %v", err)
		}
		if len(n) != 32 {
			t.Errorf("nonce length = %d, want 32", len(n))
		}
		if seen[n] {
			t.Errorf("duplicate nonce: %s", n)
		}
		seen[n] = true
	}
}

func TestMergeAccessTokenClaims(t *testing.T) {
	t.Run("merges resource_access from access token", func(t *testing.T) {
		provider, privateKey := makeTestProvider(t)
		accessTokenClaims := map[string]interface{}{
			"aud": []string{"openvpn"},
			"resource_access": map[string]interface{}{
				"openvpn": map[string]interface{}{
					"roles": []interface{}{"vpn-user", "vpn-admin"},
				},
			},
			"realm_access": map[string]interface{}{
				"roles": []interface{}{"default-roles"},
			},
		}
		accessToken := makeSignedAccessToken(t, privateKey, accessTokenClaims)

		// ID token claims without resource_access
		dst := map[string]interface{}{
			"sub":                "user123",
			"preferred_username": "testuser",
		}

		if err := provider.mergeAccessTokenClaims(context.Background(), accessToken, "Bearer", dst); err != nil {
			t.Fatalf("mergeAccessTokenClaims failed: %v", err)
		}

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

	t.Run("merges missing nested client role claims", func(t *testing.T) {
		provider, privateKey := makeTestProvider(t)
		accessTokenClaims := map[string]interface{}{
			"aud": []string{"openvpn"},
			"resource_access": map[string]interface{}{
				"openvpn": map[string]interface{}{
					"roles": []interface{}{"vpn-user"},
				},
			},
		}
		accessToken := makeSignedAccessToken(t, privateKey, accessTokenClaims)

		dst := map[string]interface{}{
			"resource_access": map[string]interface{}{
				"account": map[string]interface{}{
					"roles": []interface{}{"manage-account"},
				},
			},
		}

		if err := provider.mergeAccessTokenClaims(context.Background(), accessToken, "Bearer", dst); err != nil {
			t.Fatalf("mergeAccessTokenClaims failed: %v", err)
		}

		resourceAccess := dst["resource_access"].(map[string]interface{})
		if _, ok := resourceAccess["account"]; !ok {
			t.Fatal("existing ID token client roles should be preserved")
		}
		openvpn := resourceAccess["openvpn"].(map[string]interface{})
		roles := openvpn["roles"].([]interface{})
		if len(roles) != 1 || roles[0] != "vpn-user" {
			t.Fatalf("expected openvpn role to be merged, got %v", roles)
		}
	})

	t.Run("does not overwrite existing ID token claims", func(t *testing.T) {
		provider, privateKey := makeTestProvider(t)
		accessTokenClaims := map[string]interface{}{
			"azp": "openvpn",
			"realm_access": map[string]interface{}{
				"roles": []interface{}{"from-access-token"},
			},
		}
		accessToken := makeSignedAccessToken(t, privateKey, accessTokenClaims)

		dst := map[string]interface{}{
			"realm_access": map[string]interface{}{
				"roles": []interface{}{"from-id-token"},
			},
		}

		if err := provider.mergeAccessTokenClaims(context.Background(), accessToken, "Bearer", dst); err != nil {
			t.Fatalf("mergeAccessTokenClaims failed: %v", err)
		}

		// Should keep ID token's realm_access, not overwrite
		ra := dst["realm_access"].(map[string]interface{})
		roles := ra["roles"].([]interface{})
		if roles[0] != "from-id-token" {
			t.Errorf("expected ID token claim to be preserved, got %v", roles[0])
		}
	})

	t.Run("handles empty access token", func(t *testing.T) {
		provider, _ := makeTestProvider(t)
		dst := map[string]interface{}{"sub": "user"}
		if err := provider.mergeAccessTokenClaims(context.Background(), "", "", dst); err != nil {
			t.Fatalf("mergeAccessTokenClaims failed: %v", err)
		}
		// Should not panic or modify dst
		if len(dst) != 1 {
			t.Error("dst should not be modified for empty token")
		}
	})

	t.Run("rejects opaque access token", func(t *testing.T) {
		provider, _ := makeTestProvider(t)
		dst := map[string]interface{}{"sub": "user"}
		if err := provider.mergeAccessTokenClaims(context.Background(), "opaque-token-no-dots", "Bearer", dst); err == nil {
			t.Fatal("expected error for opaque token")
		}
		// Should not panic or modify dst
		if len(dst) != 1 {
			t.Error("dst should not be modified for opaque token")
		}
	})

	t.Run("rejects token with invalid signature", func(t *testing.T) {
		provider, _ := makeTestProvider(t)
		dst := map[string]interface{}{"sub": "user"}
		accessToken := makeTestJWT(t, map[string]interface{}{
			"iss":          "https://issuer.example",
			"aud":          []string{"openvpn"},
			"exp":          time.Now().Add(time.Hour).Unix(),
			"realm_access": map[string]interface{}{"roles": []interface{}{"vpn-user"}},
		})

		err := provider.mergeAccessTokenClaims(context.Background(), accessToken, "Bearer", dst)
		if err == nil {
			t.Fatal("expected error for invalid signature")
		}
		if got := accessTokenClaimsErrorCategory(err); got != "access_token_verify_failed" {
			t.Fatalf("error category = %q, want access_token_verify_failed", got)
		}
		if _, ok := dst["realm_access"]; ok {
			t.Fatal("unverified access token claims must not be merged")
		}
	})

	t.Run("rejects token not issued for client", func(t *testing.T) {
		provider, privateKey := makeTestProvider(t)
		dst := map[string]interface{}{"sub": "user"}
		accessToken := makeSignedAccessToken(t, privateKey, map[string]interface{}{
			"aud":          []string{"other-client"},
			"realm_access": map[string]interface{}{"roles": []interface{}{"vpn-user"}},
		})

		err := provider.mergeAccessTokenClaims(context.Background(), accessToken, "Bearer", dst)
		if err == nil {
			t.Fatal("expected error for token audience mismatch")
		}
		if got := accessTokenClaimsErrorCategory(err); got != "client_binding_failed" {
			t.Fatalf("error category = %q, want client_binding_failed", got)
		}
		if _, ok := dst["realm_access"]; ok {
			t.Fatal("wrong-client access token claims must not be merged")
		}
	})

	t.Run("rejects unsupported token type", func(t *testing.T) {
		provider, privateKey := makeTestProvider(t)
		dst := map[string]interface{}{"sub": "user"}
		accessToken := makeSignedAccessToken(t, privateKey, map[string]interface{}{
			"aud":          []string{"openvpn"},
			"realm_access": map[string]interface{}{"roles": []interface{}{"vpn-user"}},
		})

		err := provider.mergeAccessTokenClaims(context.Background(), accessToken, "DPoP", dst)
		if err == nil {
			t.Fatal("expected error for unsupported token type")
		}
		if got := accessTokenClaimsErrorCategory(err); got != "unsupported_token_type" {
			t.Fatalf("error category = %q, want unsupported_token_type", got)
		}
		if _, ok := dst["realm_access"]; ok {
			t.Fatal("unsupported-token-type access token claims must not be merged")
		}
	})
}
