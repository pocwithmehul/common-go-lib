package launchdarkly

import (
	"errors"
	"testing"

	"github.com/launchdarkly/go-sdk-common/v4/ldcontext"
)

// fakeLDClient implements ldClientIface for testing without a real SDK connection.
type fakeLDClient struct {
	boolVals    map[string]bool
	stringVals  map[string]string
	intVals     map[string]int
	floatVals   map[string]float64
	evalErr     error
	initialized bool
	closed      bool
}

func (f *fakeLDClient) BoolVariation(key string, _ ldcontext.Context, defaultVal bool) (bool, error) {
	if f.evalErr != nil {
		return defaultVal, f.evalErr
	}
	if v, ok := f.boolVals[key]; ok {
		return v, nil
	}
	return defaultVal, nil
}

func (f *fakeLDClient) StringVariation(key string, _ ldcontext.Context, defaultVal string) (string, error) {
	if f.evalErr != nil {
		return defaultVal, f.evalErr
	}
	if v, ok := f.stringVals[key]; ok {
		return v, nil
	}
	return defaultVal, nil
}

func (f *fakeLDClient) IntVariation(key string, _ ldcontext.Context, defaultVal int) (int, error) {
	if f.evalErr != nil {
		return defaultVal, f.evalErr
	}
	if v, ok := f.intVals[key]; ok {
		return v, nil
	}
	return defaultVal, nil
}

func (f *fakeLDClient) Float64Variation(key string, _ ldcontext.Context, defaultVal float64) (float64, error) {
	if f.evalErr != nil {
		return defaultVal, f.evalErr
	}
	if v, ok := f.floatVals[key]; ok {
		return v, nil
	}
	return defaultVal, nil
}

func (f *fakeLDClient) Initialized() bool { return f.initialized }

func (f *fakeLDClient) Close() error {
	f.closed = true
	return nil
}

func TestNewClientValidation(t *testing.T) {
	_, err := NewClient(Config{})
	if err == nil {
		t.Fatal("expected error when SDK key is empty")
	}
}

func TestBoolVariation(t *testing.T) {
	fake := &fakeLDClient{
		boolVals: map[string]bool{"feature-x": true},
	}
	client := NewClientWithLDClient(fake)

	got, err := client.BoolVariation("feature-x", "user-1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected true, got false")
	}
}

func TestBoolVariationDefault(t *testing.T) {
	fake := &fakeLDClient{boolVals: map[string]bool{}}
	client := NewClientWithLDClient(fake)

	got, err := client.BoolVariation("unknown-flag", "user-1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected default value true")
	}
}

func TestStringVariation(t *testing.T) {
	fake := &fakeLDClient{
		stringVals: map[string]string{"theme": "dark"},
	}
	client := NewClientWithLDClient(fake)

	got, err := client.StringVariation("theme", "user-1", "light")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "dark" {
		t.Fatalf("expected %q, got %q", "dark", got)
	}
}

func TestIntVariation(t *testing.T) {
	fake := &fakeLDClient{
		intVals: map[string]int{"max-retries": 5},
	}
	client := NewClientWithLDClient(fake)

	got, err := client.IntVariation("max-retries", "user-1", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 5 {
		t.Fatalf("expected 5, got %d", got)
	}
}

func TestFloat64Variation(t *testing.T) {
	fake := &fakeLDClient{
		floatVals: map[string]float64{"rate-limit": 0.75},
	}
	client := NewClientWithLDClient(fake)

	got, err := client.Float64Variation("rate-limit", "user-1", 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0.75 {
		t.Fatalf("expected 0.75, got %f", got)
	}
}

func TestVariationError(t *testing.T) {
	evalErr := errors.New("flag eval failed")
	fake := &fakeLDClient{evalErr: evalErr}
	client := NewClientWithLDClient(fake)

	if _, err := client.BoolVariation("flag", "user-1", false); err == nil {
		t.Fatal("expected error from BoolVariation")
	}
	if _, err := client.StringVariation("flag", "user-1", ""); err == nil {
		t.Fatal("expected error from StringVariation")
	}
	if _, err := client.IntVariation("flag", "user-1", 0); err == nil {
		t.Fatal("expected error from IntVariation")
	}
	if _, err := client.Float64Variation("flag", "user-1", 0); err == nil {
		t.Fatal("expected error from Float64Variation")
	}
}

func TestIsInitialized(t *testing.T) {
	fake := &fakeLDClient{initialized: true}
	client := NewClientWithLDClient(fake)

	if !client.IsInitialized() {
		t.Fatal("expected client to be initialized")
	}
}

func TestClose(t *testing.T) {
	fake := &fakeLDClient{}
	client := NewClientWithLDClient(fake)

	if err := client.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
	if !fake.closed {
		t.Fatal("expected Close to be delegated to underlying client")
	}
}

func TestNilClient(t *testing.T) {
	client := &Client{}

	if client.IsInitialized() {
		t.Fatal("expected nil client to not be initialized")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("expected nil close to succeed, got %v", err)
	}
	if _, err := client.BoolVariation("flag", "user", false); err == nil {
		t.Fatal("expected error for nil client")
	}
}
