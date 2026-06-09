package tracer

import (
	"context"
	"os"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// Config holds Datadog APM tracer configuration.
type Config struct {
	// Service is the name of the service being traced.
	Service string
	// Env is the deployment environment (e.g. "production", "staging").
	Env string
	// Version is the service version.
	Version string
	// AgentAddr is the Datadog agent address (host:port). Defaults to "localhost:8126".
	AgentAddr string
}

// Start initialises the global Datadog tracer. Call Stop() (typically via defer)
// when the application shuts down.
func Start(cfg Config) {
	agentAddr := cfg.AgentAddr
	if agentAddr == "" {
		agentAddr = os.Getenv("DD_AGENT_HOST")
		if agentAddr == "" {
			agentAddr = "localhost:8126"
		}
	}

	env := cfg.Env
	if env == "" {
		env = os.Getenv("DD_ENV")
	}

	version := cfg.Version
	if version == "" {
		version = os.Getenv("DD_VERSION")
	}

	opts := []tracer.StartOption{
		tracer.WithAgentAddr(agentAddr),
	}
	if cfg.Service != "" {
		opts = append(opts, tracer.WithServiceName(cfg.Service))
	}
	if env != "" {
		opts = append(opts, tracer.WithEnv(env))
	}
	if version != "" {
		opts = append(opts, tracer.WithServiceVersion(version))
	}

	tracer.Start(opts...)
}

// Stop flushes buffered traces and shuts down the global tracer.
func Stop() {
	tracer.Stop()
}

// StartSpan creates and returns a new span for the given operation name.
func StartSpan(operationName string, opts ...ddtrace.StartSpanOption) ddtrace.Span {
	return tracer.StartSpan(operationName, opts...)
}

// StartSpanFromContext creates a child span from the given context and returns
// both the span and the updated context containing that span.
func StartSpanFromContext(ctx context.Context, operationName string, opts ...ddtrace.StartSpanOption) (ddtrace.Span, context.Context) {
	return tracer.StartSpanFromContext(ctx, operationName, opts...)
}

// SpanFromContext retrieves the active span from the context, if any.
func SpanFromContext(ctx context.Context) (ddtrace.Span, bool) {
	return tracer.SpanFromContext(ctx)
}
