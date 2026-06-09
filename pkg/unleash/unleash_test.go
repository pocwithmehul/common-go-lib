package unleash

import (
	"testing"

	unleash "github.com/Unleash/unleash-client-go/v4"
	"github.com/Unleash/unleash-client-go/v4/api"
	unleashctx "github.com/Unleash/unleash-client-go/v4/context"
)

// fakeUnleashClient implements unleashIface for testing without a real server.
type fakeUnleashClient struct {
	enabledFlags map[string]bool
	variants     map[string]*api.Variant
	closed       bool
}

func (f *fakeUnleashClient) IsEnabled(feature string, options ...unleash.FeatureOption) bool {
	// Apply options to resolve the user context (mirrors real SDK behaviour in tests).
	opts := &struct{ ctx *unleashctx.Context }{}
	_ = opts // options are accepted but not inspected in the fake
	return f.enabledFlags[feature]
}

func (f *fakeUnleashClient) GetVariant(feature string, options ...unleash.VariantOption) *api.Variant {
	if v, ok := f.variants[feature]; ok {
		return v
	}
	return api.GetDefaultVariant()
}

func (f *fakeUnleashClient) Close() error {
	f.closed = true
	return nil
}

func TestNewClientValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "missing URL", cfg: Config{AppName: "svc"}},
		{name: "missing AppName", cfg: Config{URL: "http://localhost:4242/api"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cfg)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if client != nil {
				t.Fatal("expected nil client on validation error")
			}
		})
	}
}

func TestIsEnabled_True(t *testing.T) {
	fake := &fakeUnleashClient{
		enabledFlags: map[string]bool{"dark-mode": true},
	}
	client := NewClientWithUnleash(fake)

	if !client.IsEnabled("dark-mode", "user-1") {
		t.Fatal("expected flag to be enabled")
	}
}

func TestIsEnabled_False(t *testing.T) {
	fake := &fakeUnleashClient{
		enabledFlags: map[string]bool{"dark-mode": false},
	}
	client := NewClientWithUnleash(fake)

	if client.IsEnabled("dark-mode", "user-1") {
		t.Fatal("expected flag to be disabled")
	}
}

func TestIsEnabled_UnknownFlag(t *testing.T) {
	fake := &fakeUnleashClient{enabledFlags: map[string]bool{}}
	client := NewClientWithUnleash(fake)

	if client.IsEnabled("unknown-flag", "user-1") {
		t.Fatal("expected unknown flag to be disabled")
	}
}

func TestGetVariant(t *testing.T) {
	fake := &fakeUnleashClient{
		variants: map[string]*api.Variant{
			"checkout-flow": {
				Name:    "v2",
				Enabled: true,
				Payload: api.Payload{Type: "string", Value: "checkout-v2"},
			},
		},
	}
	client := NewClientWithUnleash(fake)

	v := client.GetVariant("checkout-flow", "user-1")
	if !v.Enabled {
		t.Fatal("expected variant to be enabled")
	}
	if v.Name != "v2" {
		t.Fatalf("expected variant name %q, got %q", "v2", v.Name)
	}
	if v.Payload.Type != "string" || v.Payload.Value != "checkout-v2" {
		t.Fatalf("unexpected variant payload: %+v", v.Payload)
	}
}

func TestGetVariant_DefaultWhenMissing(t *testing.T) {
	fake := &fakeUnleashClient{variants: map[string]*api.Variant{}}
	client := NewClientWithUnleash(fake)

	v := client.GetVariant("unknown-flag", "user-1")
	if v.Enabled {
		t.Fatal("expected default variant to be disabled")
	}
}

func TestClose(t *testing.T) {
	fake := &fakeUnleashClient{}
	client := NewClientWithUnleash(fake)

	if err := client.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
	if !fake.closed {
		t.Fatal("expected Close to be delegated to underlying client")
	}
}

func TestNilClient(t *testing.T) {
	client := &Client{}

	if client.IsEnabled("flag", "user") {
		t.Fatal("expected nil client to return false for IsEnabled")
	}
	v := client.GetVariant("flag", "user")
	if v.Enabled {
		t.Fatal("expected nil client to return disabled variant")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("expected nil client close to succeed, got %v", err)
	}
}

// Ensure fakeUnleashClient satisfies the interface at compile time.
var _ unleashIface = (*fakeUnleashClient)(nil)
