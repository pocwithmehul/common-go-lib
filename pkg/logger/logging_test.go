package logger

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pocwithmehul/common-go-lib/mocks"
)

func TestLoggerEmitSendsDatadogRequest(t *testing.T) {
	var capturedReq *http.Request
	mockClient := &mocks.MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedReq = req
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
		},
	}

	logger := NewLoggerWithClient("test-service", DatadogConfig{APIKey: "fake-key", Env: "prod", Site: "datadoghq.com"}, mockClient)
	logger.Info("hello world", map[string]interface{}{"foo": "bar"})

	if capturedReq == nil {
		t.Fatal("expected request to be sent")
	}
	if got := capturedReq.Header.Get("DD-API-KEY"); got != "fake-key" {
		t.Fatalf("expected DD-API-KEY header to be fake-key, got %q", got)
	}
	if !strings.Contains(capturedReq.URL.String(), "http-intake.logs.datadoghq.com/v1/input") {
		t.Fatalf("unexpected request URL: %s", capturedReq.URL.String())
	}
}

func TestNewLoggerWithClientDefaultsToHTTPClient(t *testing.T) {
	logger := NewLoggerWithClient("svc", DatadogConfig{APIKey: "x", Env: "prod"}, nil)
	if logger.client == nil {
		t.Fatal("expected logger client not to be nil")
	}
}
