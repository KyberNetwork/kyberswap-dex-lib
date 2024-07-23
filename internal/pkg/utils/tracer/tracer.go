package tracer

import (
	"context"

	"github.com/KyberNetwork/kyber-trace-go/pkg/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Span struct {
	span trace.Span
}

func (s Span) SetTag(name string, value string) {
	s.span.SetAttributes(attribute.String(name, value))
}

func (s Span) End() {
	s.span.End()
}

func StartSpanFromGinContext(ginCtx *gin.Context, operationName string) (Span, context.Context) {
	span, ctx := StartSpanFromContext(ginCtx.Request.Context(), operationName)
	if reqLogger, ok := ginCtx.Get(string(constant.CtxLoggerKey)); ok {
		return span, context.WithValue(ctx, constant.CtxLoggerKey, reqLogger)
	}

	return span, ctx
}

func StartSpanFromContext(ctx context.Context, operationName string) (Span, context.Context) {
	ctx, span := tracer.Tracer().Start(ctx, operationName)
	return Span{
		span: span,
	}, ctx
}
