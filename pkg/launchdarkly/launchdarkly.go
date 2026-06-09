package launchdarkly

import (
	"fmt"
	"os"
	"time"

	ld "github.com/launchdarkly/go-server-sdk/v7"
	"github.com/launchdarkly/go-sdk-common/v4/ldcontext"
)

// Config holds LaunchDarkly client configuration.
type Config struct {
	// SDKKey is the server-side SDK key. Falls back to the LAUNCHDARKLY_SDK_KEY env var.
	SDKKey string
	// InitWait is how long to block waiting for the client to connect. Defaults to 5s.
	InitWait time.Duration
	// Offline runs the client in offline mode; all flags return their default values.
	Offline bool
}

// ldClientIface is the subset of ld.LDClient used by Client, kept unexported so
// tests can substitute a fake without exposing the interface publicly.
type ldClientIface interface {
	BoolVariation(key string, context ldcontext.Context, defaultVal bool) (bool, error)
	StringVariation(key string, context ldcontext.Context, defaultVal string) (string, error)
	IntVariation(key string, context ldcontext.Context, defaultVal int) (int, error)
	Float64Variation(key string, context ldcontext.Context, defaultVal float64) (float64, error)
	Initialized() bool
	Close() error
}

// Client wraps the LaunchDarkly SDK client.
type Client struct {
	client ldClientIface
}

// NewClient creates and initialises a LaunchDarkly client.
func NewClient(cfg Config) (*Client, error) {
	sdkKey := cfg.SDKKey
	if sdkKey == "" {
		sdkKey = os.Getenv("LAUNCHDARKLY_SDK_KEY")
	}
	if sdkKey == "" {
		return nil, fmt.Errorf("LaunchDarkly SDK key cannot be empty")
	}

	initWait := cfg.InitWait
	if initWait == 0 {
		initWait = 5 * time.Second
	}

	ldConfig := ld.Config{
		Offline: cfg.Offline,
	}

	ldClient, err := ld.MakeCustomClient(sdkKey, ldConfig, initWait)
	if err != nil && !cfg.Offline {
		return nil, fmt.Errorf("failed to initialise LaunchDarkly client: %w", err)
	}

	return NewClientWithLDClient(ldClient), nil
}

// NewClientWithLDClient constructs a Client from an existing ldClientIface. Useful
// in tests or when sharing an already-initialised SDK client.
func NewClientWithLDClient(ldClient ldClientIface) *Client {
	return &Client{client: ldClient}
}

// Close flushes pending events and shuts down the client.
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

// IsInitialized reports whether the client has successfully connected to
// LaunchDarkly and received its initial flag data.
func (c *Client) IsInitialized() bool {
	if c.client == nil {
		return false
	}
	return c.client.Initialized()
}

// BoolVariation returns the boolean value of a feature flag for the given user key.
func (c *Client) BoolVariation(flagKey, userKey string, defaultVal bool) (bool, error) {
	if c.client == nil {
		return defaultVal, fmt.Errorf("LaunchDarkly client is not initialised")
	}
	return c.client.BoolVariation(flagKey, ldcontext.New(userKey), defaultVal)
}

// StringVariation returns the string value of a feature flag for the given user key.
func (c *Client) StringVariation(flagKey, userKey string, defaultVal string) (string, error) {
	if c.client == nil {
		return defaultVal, fmt.Errorf("LaunchDarkly client is not initialised")
	}
	return c.client.StringVariation(flagKey, ldcontext.New(userKey), defaultVal)
}

// IntVariation returns the integer value of a feature flag for the given user key.
func (c *Client) IntVariation(flagKey, userKey string, defaultVal int) (int, error) {
	if c.client == nil {
		return defaultVal, fmt.Errorf("LaunchDarkly client is not initialised")
	}
	return c.client.IntVariation(flagKey, ldcontext.New(userKey), defaultVal)
}

// Float64Variation returns the float64 value of a feature flag for the given user key.
func (c *Client) Float64Variation(flagKey, userKey string, defaultVal float64) (float64, error) {
	if c.client == nil {
		return defaultVal, fmt.Errorf("LaunchDarkly client is not initialised")
	}
	return c.client.Float64Variation(flagKey, ldcontext.New(userKey), defaultVal)
}
