package requestid

import "context"

const requestIDContextKey = ctxKey(0)

type ctxKey int8

func SetRequestIDToContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

func RequestIDFromCtx(ctx context.Context) string {
	v := ctx.Value(requestIDContextKey)
	requestID, _ := v.(string)
	return requestID
}
