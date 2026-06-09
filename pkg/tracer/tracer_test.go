package tracer_test

import (
	"context"
	"testing"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/mocktracer"

	"github.com/pocwithmehul/common-go-lib/pkg/tracer"
)

func TestStartSpan(t *testing.T) {
	mt := mocktracer.Start()
	defer mt.Stop()

	span := tracer.StartSpan("test.operation")
	if span == nil {
		t.Fatal("expected a non-nil span")
	}
	span.Finish()

	finished := mt.FinishedSpans()
	if len(finished) != 1 {
		t.Fatalf("expected 1 finished span, got %d", len(finished))
	}
	if finished[0].OperationName() != "test.operation" {
		t.Errorf("expected operation name %q, got %q", "test.operation", finished[0].OperationName())
	}
}

func TestStartSpanFromContext(t *testing.T) {
	mt := mocktracer.Start()
	defer mt.Stop()

	parent := tracer.StartSpan("parent.operation")
	ctx := context.Background()

	child, childCtx := tracer.StartSpanFromContext(ctx, "child.operation")
	if child == nil {
		t.Fatal("expected a non-nil child span")
	}
	if childCtx == nil {
		t.Fatal("expected a non-nil child context")
	}

	child.Finish()
	parent.Finish()
}

func TestSpanFromContext(t *testing.T) {
	mt := mocktracer.Start()
	defer mt.Stop()

	span, ctx := tracer.StartSpanFromContext(context.Background(), "test.operation")
	defer span.Finish()

	retrieved, ok := tracer.SpanFromContext(ctx)
	if !ok {
		t.Fatal("expected span to be found in context")
	}
	if retrieved == nil {
		t.Fatal("expected a non-nil span from context")
	}
}

func TestSpanFromContext_Empty(t *testing.T) {
	_, ok := tracer.SpanFromContext(context.Background())
	if ok {
		t.Fatal("expected no span in empty context")
	}
}

func TestStart_Defaults(t *testing.T) {
	// Verify Start does not panic when all config fields are empty.
	tracer.Start(tracer.Config{})
	tracer.Stop()
}
