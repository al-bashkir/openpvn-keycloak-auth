// Package oidc implements OpenID Connect (OIDC) authentication with PKCE.
package oidc

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/al-bashkir/openvpn-keycloak/internal/config"
)

// Provider wraps the OIDC provider and OAuth2 configuration.
// It handles provider discovery, token exchange, and ID token verification.
type Provider struct {
	oidcProvider *oidc.Provider
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

// NewProvider creates a new OIDC provider using the specified configuration.
// It performs OIDC discovery via /.well-known/openid-configuration
// and sets up the OAuth2 configuration and ID token verifier.
func NewProvider(ctx context.Context, cfg *config.OIDCConfig) (*Provider, error) {
	// Discover OIDC configuration from issuer
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	// Create OAuth2 config
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes,
	}

	// Create ID token verifier
	// This will verify the token signature, issuer, audience, and expiry
	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	return &Provider{
		oidcProvider: provider,
		oauth2Config: oauth2Config,
		verifier:     verifier,
	}, nil
}
