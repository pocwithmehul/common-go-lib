package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pocwithmehul/common-go-lib/pkg/httputils"
)

type DatadogConfig struct {
	Site   string
	APIKey string
	Env    string
}

type Logger struct {
	Service string
	env     string
	site    string
	apiKey  string
	client  httputils.HTTPClient
}

func NewLogger(service string, cfg DatadogConfig) *Logger {
	return NewLoggerWithClient(service, cfg, http.DefaultClient)
}

func NewLoggerWithClient(service string, cfg DatadogConfig, client httputils.HTTPClient) *Logger {
	site := cfg.Site
	if site == "" {
		site = "datadoghq.com"
	}

	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("DATADOG_API_KEY")
	}

	env := cfg.Env
	if env == "" {
		env = os.Getenv("DATADOG_ENV")
	}

	if client == nil {
		client = http.DefaultClient
	}

	return &Logger{Service: service, env: env, site: site, apiKey: apiKey, client: client}
}

func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.emit(msg, "info", fields)
}

func (l *Logger) Error(msg string, fields map[string]interface{}) {
	l.emit(msg, "error", fields)
}

func (l *Logger) emit(msg, level string, fields map[string]interface{}) {
	if l.apiKey == "" {
		fmt.Printf("%s [%s] %s %v\n", time.Now().UTC().Format(time.RFC3339), level, msg, fields)
		return
	}

	payload := map[string]interface{}{
		"message":   msg,
		"ddsource":  "go-service",
		"service":   l.Service,
		"hostname":  os.Getenv("HOSTNAME"),
		"ddtags":    fmt.Sprintf("env:%s", l.env),
		"level":     level,
		"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
	}

	if fields != nil {
		payload["attributes"] = fields
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://http-intake.logs.%s/v1/input", l.site)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("datadog request error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", l.apiKey)

	resp, err := l.client.Do(req)
	if err != nil {
		fmt.Printf("datadog send error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		fmt.Printf("datadog response status: %s\n", resp.Status)
	}
}
