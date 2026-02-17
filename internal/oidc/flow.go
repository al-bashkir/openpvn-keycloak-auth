package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// AuthFlowData contains the data needed to initiate an OIDC authorization flow.
type AuthFlowData struct {
	// State is the OIDC state parameter for CSRF protection
	State string

	// CodeVerifier is the PKCE code verifier (must be stored for token exchange)
	CodeVerifier string

	// AuthURL is the complete authorization URL to redirect the user to
	AuthURL string
}

// TokenData contains the tokens and claims returned from the OIDC provider.
type TokenData struct {
	// AccessToken is the OAuth2 access token
	AccessToken string `json:"-"`

	// RefreshToken is the OAuth2 refresh token (if available)
	RefreshToken string `json:"-"`

	// IDToken is the raw OIDC ID token (JWT); tagged json:"-" because it
	// encodes identity claims in a base64-decodable payload.
	IDToken string `json:"-"`

	// Claims are the parsed claims from the ID token
	Claims map[string]interface{}

	// Expiry is when the access token expires
	Expiry time.Time
}

// StartAuthFlow initiates an OIDC authorization flow with PKCE.
// It generates the PKCE verifier/challenge and state parameter,
// constructs the authorization URL, and returns the flow data.
func (p *Provider) StartAuthFlow(ctx context.Context) (*AuthFlowData, error) {
	// Generate PKCE verifier and challenge
	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}

	challenge := generateCodeChallenge(verifier)

	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Construct authorization URL with PKCE parameters
	authURL := p.oauth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	return &AuthFlowData{
		State:        state,
		CodeVerifier: verifier,
		AuthURL:      authURL,
	}, nil
}

// ExchangeCode exchanges an authorization code for tokens.
// It uses the PKCE code verifier to complete the flow.
// The ID token is verified (signature, issuer, audience, expiry) before returning.
func (p *Provider) ExchangeCode(ctx context.Context, code, codeVerifier string) (*TokenData, error) {
	// Exchange authorization code for tokens
	token, err := p.oauth2Config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Extract ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}

	// Verify ID token (signature, issuer, audience, expiry)
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Parse claims from ID token
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Merge access token claims into the claims map.
	// Keycloak puts resource_access (client-specific roles) and realm_access
	// in the access token, not the ID token. We decode the access token JWT
	// payload and merge selected claims so the validator can find them.
	mergeAccessTokenClaims(token.AccessToken, claims)

	return &TokenData{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      rawIDToken,
		Claims:       claims,
		Expiry:       token.Expiry,
	}, nil
}

// mergeAccessTokenClaims decodes a JWT access token's payload and merges
// role-related claims into the destination claims map.
// Only claims not already present in dst are merged (ID token takes precedence).
// This is best-effort: errors are logged but do not fail the auth flow,
// since not all access tokens are JWTs (e.g., opaque tokens).
func mergeAccessTokenClaims(accessToken string, dst map[string]interface{}) {
	if accessToken == "" {
		return
	}

	atClaims, err := decodeJWTPayload(accessToken)
	if err != nil {
		slog.Debug("could not decode access token as JWT (may be opaque)", "error", err)
		return
	}

	// Claims to merge from access token if not present in ID token
	mergeKeys := []string{"resource_access", "realm_access", "groups"}

	for _, key := range mergeKeys {
		if _, exists := dst[key]; !exists {
			if val, ok := atClaims[key]; ok {
				dst[key] = val
				slog.Debug("merged claim from access token", "claim", key)
			}
		}
	}
}

// decodeJWTPayload extracts and decodes the payload (second segment) of a JWT.
// It does NOT verify the signature — that's already handled by the OIDC provider
// during the token exchange. This is only used to extract claims from the
// Keycloak access token which is a JWT.
func decodeJWTPayload(token string) (map[string]interface{}, error) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("not a valid JWT: expected 3 parts, got %d", len(parts))
	}

	// Decode base64url payload (second segment)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT payload: %w", err)
	}

	return claims, nil
}

// generateCodeVerifier creates a cryptographically random PKCE code verifier.
// The verifier is 32 random bytes encoded as base64url (43 characters).
// Per RFC 7636, the verifier must be 43-128 characters.
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCodeChallenge creates a PKCE code challenge from the verifier.
// It uses the S256 method: BASE64URL(SHA256(ASCII(verifier)))
func generateCodeChallenge(verifier string) string {
	h := sha256.New()
	h.Write([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// generateState creates a random state parameter for CSRF protection.
// The state is 16 random bytes encoded as hex (32 characters).
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
