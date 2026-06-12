package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var (
	errUnsupportedAccessTokenType = errors.New("unsupported access token type")
	errAccessTokenVerifyFailed    = errors.New("access token verification failed")
	errAccessTokenParseFailed     = errors.New("access token claim parsing failed")
	errAccessTokenClientMismatch  = errors.New("access token client binding failed")
	errIDTokenMissing             = errors.New("no id_token in token response")
	errIDTokenNonceMismatch       = errors.New("id_token nonce mismatch")
)

// AuthFlowData contains the data needed to initiate an OIDC authorization flow.
type AuthFlowData struct {
	// State is the OIDC state parameter for CSRF protection
	State string

	// CodeVerifier is the PKCE code verifier (must be stored for token exchange)
	CodeVerifier string

	// Nonce is the OIDC nonce; the issued ID token must echo it back.
	Nonce string

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

	// Generate nonce; the ID token must echo it back to bind the response to this flow.
	nonce, err := generateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Construct authorization URL with PKCE + nonce parameters
	authURL := p.oauth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oidc.Nonce(nonce),
	)

	return &AuthFlowData{
		State:        state,
		CodeVerifier: verifier,
		Nonce:        nonce,
		AuthURL:      authURL,
	}, nil
}

// ExchangeCode exchanges an authorization code for tokens.
// It uses the PKCE code verifier to complete the flow.
// The ID token is verified (signature, issuer, audience, expiry) and its nonce
// is checked against expectedNonce before returning.
func (p *Provider) ExchangeCode(ctx context.Context, code, codeVerifier, expectedNonce string) (*TokenData, error) {
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
		return nil, errIDTokenMissing
	}

	// Verify ID token (signature, issuer, audience, expiry)
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Bind the response to this flow: the id_token nonce must match the value
	// we sent on the authorization request.
	if expectedNonce == "" || idToken.Nonce != expectedNonce {
		return nil, errIDTokenNonceMismatch
	}

	// Parse claims from ID token
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Merge verified access-token role claims into the claims map. Keycloak often
	// puts resource_access and realm_access in the access token, not the ID token.
	if err := p.mergeAccessTokenClaims(ctx, token.AccessToken, token.TokenType, claims); err != nil {
		slog.Debug("could not merge access token claims", "error_category", accessTokenClaimsErrorCategory(err))
		return nil, fmt.Errorf("failed to verify access token claims: %w", err)
	}

	return &TokenData{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      rawIDToken,
		Claims:       claims,
		Expiry:       token.Expiry,
	}, nil
}

// mergeAccessTokenClaims verifies a JWT access token and merges role-related
// claims into the destination claims map.
// Only missing claims are merged (ID token takes precedence).
// A non-empty access token must be a verified JWT issued for this client.
func (p *Provider) mergeAccessTokenClaims(ctx context.Context, accessToken, tokenType string, dst map[string]interface{}) error {
	if accessToken == "" {
		return nil
	}

	atClaims, err := p.verifyAccessTokenClaims(ctx, accessToken, tokenType)
	if err != nil {
		return err
	}

	// Claims to merge from access token if not present in ID token
	mergeKeys := []string{"resource_access", "realm_access", "groups"}

	for _, key := range mergeKeys {
		val, ok := atClaims[key]
		if !ok {
			continue
		}
		if mergeMissingClaim(dst, key, val) {
			slog.Debug("merged claim from access token", "claim", key)
		}
	}

	return nil
}

func mergeMissingClaim(dst map[string]interface{}, key string, src interface{}) bool {
	dstVal, exists := dst[key]
	if !exists {
		dst[key] = src
		return true
	}

	dstMap, dstOK := dstVal.(map[string]interface{})
	srcMap, srcOK := src.(map[string]interface{})
	if !dstOK || !srcOK {
		return false
	}

	return mergeMissingMapValues(dstMap, srcMap)
}

func mergeMissingMapValues(dst, src map[string]interface{}) bool {
	merged := false
	for key, srcVal := range src {
		dstVal, exists := dst[key]
		if !exists {
			dst[key] = srcVal
			merged = true
			continue
		}

		dstMap, dstOK := dstVal.(map[string]interface{})
		srcMap, srcOK := srcVal.(map[string]interface{})
		if dstOK && srcOK && mergeMissingMapValues(dstMap, srcMap) {
			merged = true
		}
	}
	return merged
}

func (p *Provider) verifyAccessTokenClaims(ctx context.Context, accessToken, tokenType string) (map[string]interface{}, error) {
	if tokenType != "" && !strings.EqualFold(tokenType, "Bearer") {
		return nil, fmt.Errorf("%w: %q", errUnsupportedAccessTokenType, tokenType)
	}

	verifiedToken, err := p.accessTokenVerifier.Verify(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errAccessTokenVerifyFailed, err)
	}

	var claims map[string]interface{}
	if err := verifiedToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("%w: %w", errAccessTokenParseFailed, err)
	}

	if !accessTokenIssuedForClient(verifiedToken.Audience, claims, p.clientID) {
		return nil, errAccessTokenClientMismatch
	}

	return claims, nil
}

func accessTokenClaimsErrorCategory(err error) string {
	switch {
	case errors.Is(err, errUnsupportedAccessTokenType):
		return "unsupported_token_type"
	case errors.Is(err, errAccessTokenVerifyFailed):
		return "access_token_verify_failed"
	case errors.Is(err, errAccessTokenParseFailed):
		return "access_token_parse_failed"
	case errors.Is(err, errAccessTokenClientMismatch):
		return "client_binding_failed"
	default:
		return "access_token_claims_failed"
	}
}

func accessTokenIssuedForClient(audience []string, claims map[string]interface{}, clientID string) bool {
	if slices.Contains(audience, clientID) {
		return true
	}

	for _, claim := range []string{"azp", "client_id"} {
		value, ok := claims[claim].(string)
		if ok && value == clientID {
			return true
		}
	}

	return false
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

// generateNonce creates a random OIDC nonce. The id_token returned by the
// provider must echo this value, binding the token to this auth request.
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
