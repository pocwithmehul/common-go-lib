package unleash

import (
	"fmt"
	"net/http"
	"os"
	"time"

	unleash "github.com/Unleash/unleash-client-go/v4"
	"github.com/Unleash/unleash-client-go/v4/api"
	unleashctx "github.com/Unleash/unleash-client-go/v4/context"
)

// Config holds Unleash client configuration.
type Config struct {
	// URL is the Unleash server base URL (e.g. "http://unleash.myapp.com/api").
	// Falls back to the UNLEASH_URL env var.
	URL string
	// AppName identifies the application in the Unleash UI.
	// Falls back to the UNLEASH_APP_NAME env var.
	AppName string
	// APIToken is the server-side API token sent as the Authorization header.
	// Falls back to the UNLEASH_API_TOKEN env var.
	APIToken string
	// InstanceID uniquely identifies this client instance. Optional.
	InstanceID string
	// Environment sets the active Unleash environment. Optional.
	Environment string
	// RefreshInterval controls how often feature toggles are polled. Defaults to 15s.
	RefreshInterval time.Duration
	// MetricsInterval controls how often metrics are flushed. Defaults to 60s.
	MetricsInterval time.Duration
	// DisableMetrics prevents sending usage metrics to the Unleash server.
	DisableMetrics bool
}

// Variant is the result of a variant flag evaluation.
type Variant struct {
	Name    string
	Enabled bool
	Payload VariantPayload
}

// VariantPayload holds the optional payload attached to a variant.
type VariantPayload struct {
	Type  string
	Value string
}

// unleashIface is the subset of unleash.Client used by Client, kept unexported
// so tests can substitute a fake without exposing the interface publicly.
type unleashIface interface {
	IsEnabled(feature string, options ...unleash.FeatureOption) bool
	GetVariant(feature string, options ...unleash.VariantOption) *api.Variant
	Close() error
}

// Client wraps the Unleash SDK client.
type Client struct {
	client unleashIface
}

// NewClient creates and starts an Unleash client. Feature toggles are synced
// in the background; flags return their default values until the first sync
// completes.
func NewClient(cfg Config) (*Client, error) {
	url := cfg.URL
	if url == "" {
		url = os.Getenv("UNLEASH_URL")
	}
	if url == "" {
		return nil, fmt.Errorf("unleash URL cannot be empty")
	}

	appName := cfg.AppName
	if appName == "" {
		appName = os.Getenv("UNLEASH_APP_NAME")
	}
	if appName == "" {
		return nil, fmt.Errorf("unleash app name cannot be empty")
	}

	apiToken := cfg.APIToken
	if apiToken == "" {
		apiToken = os.Getenv("UNLEASH_API_TOKEN")
	}

	opts := []unleash.ConfigOption{
		unleash.WithUrl(url),
		unleash.WithAppName(appName),
		unleash.WithDisableMetrics(cfg.DisableMetrics),
	}

	if apiToken != "" {
		opts = append(opts, unleash.WithCustomHeaders(http.Header{
			"Authorization": []string{apiToken},
		}))
	}
	if cfg.InstanceID != "" {
		opts = append(opts, unleash.WithInstanceId(cfg.InstanceID))
	}
	if cfg.Environment != "" {
		opts = append(opts, unleash.WithEnvironment(cfg.Environment))
	}
	if cfg.RefreshInterval > 0 {
		opts = append(opts, unleash.WithRefreshInterval(cfg.RefreshInterval))
	}
	if cfg.MetricsInterval > 0 {
		opts = append(opts, unleash.WithMetricsInterval(cfg.MetricsInterval))
	}

	u, err := unleash.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create unleash client: %w", err)
	}

	return NewClientWithUnleash(u), nil
}

// NewClientWithUnleash constructs a Client from an existing unleashIface.
// Useful in tests or when sharing an already-created SDK client.
func NewClientWithUnleash(u unleashIface) *Client {
	return &Client{client: u}
}

// Close flushes pending metrics and shuts down the client.
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

// IsEnabled reports whether the named feature toggle is enabled for the given
// user. Returns false when the client is nil or the flag is unknown.
func (c *Client) IsEnabled(flagName, userID string) bool {
	if c.client == nil {
		return false
	}
	return c.client.IsEnabled(flagName, unleash.WithContext(unleashctx.Context{
		UserId: userID,
	}))
}

// GetVariant returns the active variant for the named feature toggle and user.
// If the flag is disabled or unknown, the returned Variant has Enabled=false.
func (c *Client) GetVariant(flagName, userID string) Variant {
	if c.client == nil {
		return Variant{}
	}
	v := c.client.GetVariant(flagName, unleash.WithVariantContext(unleashctx.Context{
		UserId: userID,
	}))
	if v == nil {
		return Variant{}
	}
	return Variant{
		Name:    v.Name,
		Enabled: v.Enabled,
		Payload: VariantPayload{
			Type:  v.Payload.Type,
			Value: v.Payload.Value,
		},
	}
}
