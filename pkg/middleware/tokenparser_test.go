package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func newJWKServer(t *testing.T) (*httptest.Server, *rsa.PrivateKey) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	pubKey, err := jwk.FromRaw(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := pubKey.Set(jwk.KeyIDKey, "test-key"); err != nil {
		t.Fatal(err)
	}
	if err := pubKey.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		t.Fatal(err)
	}

	set := jwk.NewSet()
	if err := set.AddKey(pubKey); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(set); err != nil {
			t.Fatal(err)
		}
	}))

	return srv, priv
}

func signToken(t *testing.T, priv *rsa.PrivateKey, issuer string, audience []string, scope string) string {
	token := jwt.New()
	if err := token.Set(jwt.SubjectKey, "123"); err != nil {
		t.Fatal(err)
	}
	if err := token.Set(jwt.IssuerKey, issuer); err != nil {
		t.Fatal(err)
	}
	if err := token.Set(jwt.AudienceKey, audience); err != nil {
		t.Fatal(err)
	}
	if err := token.Set(jwt.ExpirationKey, time.Now().Add(1*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := token.Set("email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := token.Set("scope", scope); err != nil {
		t.Fatal(err)
	}

	jwkPriv, err := jwk.FromRaw(priv)
	if err != nil {
		t.Fatal(err)
	}
	if err := jwkPriv.Set(jwk.KeyIDKey, "test-key"); err != nil {
		t.Fatal(err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, jwkPriv))
	if err != nil {
		t.Fatal(err)
	}
	return string(signed)
}

func TestLoadTokenAuthorizationConfigYAML(t *testing.T) {
	yamlData := []byte(`tokenAuthorization:
  jwk_uri: "https://login.example.com/.well-known/jwks.json"
  validIssuers:
    - "https://login.dev.myorglogin.com/oauth2/realms/root/realms/myorg"
  validAudiences:
    - "appclient"
  expectedScopes:
    appclient:
      - "openid"
      - "profile"
      - "write"
`)

	cfg, err := LoadTokenAuthorizationConfigYAML(yamlData)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.JWKURI != "https://login.example.com/.well-known/jwks.json" {
		t.Fatalf("unexpected JWK URI: %s", cfg.JWKURI)
	}
	if len(cfg.ValidIssuers) != 1 || cfg.ValidIssuers[0] != "https://login.dev.myorglogin.com/oauth2/realms/root/realms/myorg" {
		t.Fatalf("unexpected issuers: %+v", cfg.ValidIssuers)
	}
	if len(cfg.ExpectedScopes["appclient"]) != 3 {
		t.Fatalf("expected scopes not populated")
	}
}

func TestParseBearerTokenValidWithJWK(t *testing.T) {
	srv, priv := newJWKServer(t)
	defer srv.Close()

	token := signToken(t, priv, "https://login.dev.myorglogin.com/oauth2/realms/root/realms/myorg", []string{"appclient"}, "openid profile write")

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	cfg := &TokenAuthorizationConfig{
		JWKURI:         srv.URL,
		ValidIssuers:   []string{"https://login.dev.myorglogin.com/oauth2/realms/root/realms/myorg"},
		ValidAudiences: []string{"appclient"},
		ExpectedScopes: map[string][]string{"appclient": {"openid", "profile", "write"}},
	}

	user, err := ParseBearerToken(req, cfg)
	if err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}
	if user.Subject != "123" || user.Email != "test@example.com" || user.Issuer != cfg.ValidIssuers[0] {
		t.Fatalf("unexpected user info: %+v", user)
	}
}

func TestParseBearerTokenInvalidIssuer(t *testing.T) {
	srv, priv := newJWKServer(t)
	defer srv.Close()

	token := signToken(t, priv, "https://login.other.example.com/oauth2/realms/root/realms/myorg", []string{"appclient"}, "openid profile write")

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	cfg := &TokenAuthorizationConfig{
		JWKURI:         srv.URL,
		ValidIssuers:   []string{"https://login.dev.myorglogin.com/oauth2/realms/root/realms/myorg"},
		ValidAudiences: []string{"appclient"},
	}

	_, err = ParseBearerToken(req, cfg)
	if err == nil || !strings.Contains(err.Error(), "invalid issuer") {
		t.Fatalf("expected invalid issuer error, got %v", err)
	}
}

func TestParseBearerTokenMissingScopes(t *testing.T) {
	srv, priv := newJWKServer(t)
	defer srv.Close()

	token := signToken(t, priv, "https://login.dev.myorglogin.com/oauth2/realms/root/realms/myorg", []string{"appclient"}, "openid profile")

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	cfg := &TokenAuthorizationConfig{
		JWKURI:         srv.URL,
		ValidIssuers:   []string{"https://login.dev.myorglogin.com/oauth2/realms/root/realms/myorg"},
		ValidAudiences: []string{"appclient"},
		ExpectedScopes: map[string][]string{"appclient": {"openid", "profile", "write"}},
	}

	_, err = ParseBearerToken(req, cfg)
	if err == nil || !strings.Contains(err.Error(), "missing required scope") {
		t.Fatalf("expected missing scope error, got %v", err)
	}
}
