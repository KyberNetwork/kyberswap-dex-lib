package tracing

import (
	"context"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type spanAdapter struct {
	span tracer.Span
}

func (s *spanAdapter) SetTag(key string, value interface{}) {
	s.span.SetTag(key, value)
}

func (s *spanAdapter) Finish() {
	s.span.Finish()
}

type datadogTracer struct {
}

func NewDatadogTracer() ITracer {
	return &datadogTracer{}
}

func (t *datadogTracer) Trace(ctx context.Context, operationName string) (ISpan, context.Context) {
	span, ctx := tracer.StartSpanFromContext(ctx, operationName)

	return &spanAdapter{span}, ctx
}
