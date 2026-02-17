package oidc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/config"
)

func newTestIssuer(t *testing.T) string {
	t.Helper()

	var baseURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issuer := baseURL + "/realms/test"

		if r.URL.Path != "/realms/test/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/auth",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/keys",
		})
	}))
	baseURL = ts.URL
	t.Cleanup(ts.Close)

	return baseURL + "/realms/test"
}

func TestNewProviderAndStartAuthFlow(t *testing.T) {
	issuer := newTestIssuer(t)

	p, err := NewProvider(context.Background(), &config.OIDCConfig{
		Issuer:      issuer,
		ClientID:    "test-client",
		RedirectURI: "http://localhost/callback",
		Scopes:      []string{"openid"},
	})
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	flow, err := p.StartAuthFlow(context.Background())
	if err != nil {
		t.Fatalf("StartAuthFlow failed: %v", err)
	}
	if flow.State == "" {
		t.Fatal("expected state to be set")
	}
	if flow.CodeVerifier == "" {
		t.Fatal("expected code verifier to be set")
	}
	if flow.AuthURL == "" {
		t.Fatal("expected auth URL to be set")
	}
	if !strings.HasPrefix(flow.AuthURL, issuer+"/auth") {
		t.Fatalf("expected auth URL to start with %q, got %q", issuer+"/auth", flow.AuthURL)
	}

	u, err := url.Parse(flow.AuthURL)
	if err != nil {
		t.Fatalf("failed to parse auth URL: %v", err)
	}

	q := u.Query()
	if q.Get("client_id") != "test-client" {
		t.Fatalf("client_id = %q, want %q", q.Get("client_id"), "test-client")
	}
	if q.Get("redirect_uri") != "http://localhost/callback" {
		t.Fatalf("redirect_uri = %q, want %q", q.Get("redirect_uri"), "http://localhost/callback")
	}
	if q.Get("response_type") != "code" {
		t.Fatalf("response_type = %q, want %q", q.Get("response_type"), "code")
	}
	if q.Get("scope") != "openid" {
		t.Fatalf("scope = %q, want %q", q.Get("scope"), "openid")
	}
	if q.Get("state") != flow.State {
		t.Fatalf("state = %q, want %q", q.Get("state"), flow.State)
	}
	if q.Get("code_challenge") == "" {
		t.Fatal("expected code_challenge to be set")
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Fatalf("code_challenge_method = %q, want %q", q.Get("code_challenge_method"), "S256")
	}
}

func TestNewProvider_DiscoveryFailure(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(ts.Close)

	issuer := ts.URL + "/realms/test"
	_, err := NewProvider(context.Background(), &config.OIDCConfig{
		Issuer:      issuer,
		ClientID:    "test-client",
		RedirectURI: "http://localhost/callback",
		Scopes:      []string{"openid"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
