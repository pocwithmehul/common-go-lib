package locale

import (
	"net/http"
	"testing"
)

func TestGetPreferredLocale(t *testing.T) {
	cases := []struct {
		name     string
		header   string
		expected string
	}{
		{"Standard header", "en-US,en;q=0.9", "en-US"},
		{"Regional locale", "fr-CA,fr;q=0.8", "fr-CA"},
		{"Empty header", "", "en-US"},
		{"Whitespace header", "  ", "en-US"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Accept-Language", tc.header)

			got := GetPreferredLocale(req)
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
