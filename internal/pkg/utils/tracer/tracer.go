package tracer

import (
	"context"

	"github.com/KyberNetwork/kyber-trace-go/pkg/tracer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	ddtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type Span struct {
	span   trace.Span
	ddSpan ddtrace.Span
}

func (s Span) SetTag(name string, value string) {
	s.span.SetAttributes(attribute.String(name, value))
	s.ddSpan.SetTag(name, value)
}

func (s Span) End() {
	s.span.End()
	s.ddSpan.Finish()
}

func StartSpanFromContext(ctx context.Context, operationName string) (Span, context.Context) {
	ddSpan, _ := ddtracer.StartSpanFromContext(ctx, operationName)
	ctx, span := tracer.Tracer().Start(ctx, operationName)
	return Span{
		span:   span,
		ddSpan: ddSpan,
	}, ctx
}
