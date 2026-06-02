package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"gopkg.in/yaml.v3"
)

type UserInfo struct {
	Subject  string   `json:"sub,omitempty"`
	Email    string   `json:"email,omitempty"`
	Name     string   `json:"name,omitempty"`
	Issuer   string   `json:"iss,omitempty"`
	Audience []string `json:"aud,omitempty"`
}

type TokenClaims struct {
	Subject  string   `json:"sub,omitempty"`
	Email    string   `json:"email,omitempty"`
	Name     string   `json:"name,omitempty"`
	OID      string   `json:"oid,omitempty"`
	Issuer   string   `json:"iss,omitempty"`
	Audience []string `json:"aud,omitempty"`
	Scope    string   `json:"scope,omitempty"`
	Scp      []string `json:"scp,omitempty"`
}

type TokenAuthorizationConfig struct {
	JWKURI         string              `yaml:"jwk_uri"`
	ValidIssuers   []string            `yaml:"validIssuers"`
	ValidAudiences []string            `yaml:"validAudiences"`
	ExpectedScopes map[string][]string `yaml:"expectedScopes"`
}

type tokenAuthorizationEnvelope struct {
	TokenAuthorization TokenAuthorizationConfig `yaml:"tokenAuthorization"`
}

func LoadTokenAuthorizationConfigYAML(data []byte) (*TokenAuthorizationConfig, error) {
	var envelope tokenAuthorizationEnvelope
	if err := yaml.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	return &envelope.TokenAuthorization, nil
}

func ParseBearerToken(r *http.Request, cfg *TokenAuthorizationConfig) (*UserInfo, error) {
	if cfg == nil {
		return nil, errors.New("token authorization config is required")
	}

	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, errors.New("missing authorization header")
	}

	parts := strings.Fields(auth)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, errors.New("invalid authorization header format")
	}

	claims, err := verifyToken(r.Context(), parts[1], cfg)
	if err != nil {
		return nil, err
	}

	return &UserInfo{
		Subject:  claims.Subject,
		Email:    claims.Email,
		Name:     claims.Name,
		Issuer:   claims.Issuer,
		Audience: claims.Audience,
	}, nil
}

func verifyToken(ctx context.Context, tokenString string, cfg *TokenAuthorizationConfig) (*TokenClaims, error) {
	if cfg.JWKURI == "" {
		return nil, errors.New("missing jwk_uri in token authorization config")
	}

	keySet, err := jwk.Fetch(ctx, cfg.JWKURI)
	if err != nil {
		return nil, fmt.Errorf("failed to load jwk set: %w", err)
	}

	options := []jwt.ParseOption{
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithAcceptableSkew(5 * time.Second),
	}

	token, err := jwt.Parse([]byte(tokenString), options...)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	claims := &TokenClaims{
		Subject:  token.Subject(),
		Issuer:   token.Issuer(),
		Audience: token.Audience(),
	}

	if v, ok := token.Get("email"); ok {
		if email, ok := v.(string); ok {
			claims.Email = email
		}
	}
	if v, ok := token.Get("name"); ok {
		if name, ok := v.(string); ok {
			claims.Name = name
		}
	}
	if v, ok := token.Get("oid"); ok {
		if oid, ok := v.(string); ok {
			claims.OID = oid
		}
	}
	claims.Scope = getStringClaim(token, "scope")
	claims.Scp = getStringSliceClaim(token, "scp")

	if len(cfg.ValidIssuers) > 0 && !isValidIssuer(claims.Issuer, cfg.ValidIssuers) {
		return nil, fmt.Errorf("invalid issuer %s", claims.Issuer)
	}
	if len(cfg.ValidAudiences) > 0 && !isValidAudience(claims.Audience, cfg.ValidAudiences) {
		return nil, fmt.Errorf("invalid audience %v", claims.Audience)
	}
	if err := validateScopes(claims, cfg.ExpectedScopes); err != nil {
		return nil, err
	}

	return claims, nil
}

func getStringClaim(token jwt.Token, key string) string {
	value, ok := token.Get(key)
	if !ok {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func getStringSliceClaim(token jwt.Token, key string) []string {
	value, ok := token.Get(key)
	if !ok {
		return nil
	}

	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}

	return nil
}

func validateScopes(claims *TokenClaims, expected map[string][]string) error {
	if len(expected) == 0 {
		return nil
	}

	actual := parseScopes(claims)
	for _, aud := range claims.Audience {
		expectedScopes, ok := expected[aud]
		if !ok {
			continue
		}

		for _, scope := range expectedScopes {
			if !contains(actual, scope) {
				return fmt.Errorf("missing required scope %q for audience %q", scope, aud)
			}
		}
	}

	return nil
}

func parseScopes(claims *TokenClaims) []string {
	scopes := make([]string, 0, len(claims.Scp))
	scopes = append(scopes, claims.Scp...)
	for _, part := range strings.Fields(claims.Scope) {
		if part != "" {
			scopes = append(scopes, part)
		}
	}
	return unique(scopes)
}

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func isValidIssuer(actual string, allowed []string) bool {
	for _, issuer := range allowed {
		if issuer == actual {
			return true
		}
	}
	return false
}

func isValidAudience(actual []string, allowed []string) bool {
	for _, aud := range actual {
		for _, valid := range allowed {
			if aud == valid {
				return true
			}
		}
	}
	return false
}

func contains(slice []string, target string) bool {
	for _, value := range slice {
		if value == target {
			return true
		}
	}
	return false
}
