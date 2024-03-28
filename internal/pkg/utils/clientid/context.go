package clientid

import "context"

const clientIDContextKey = ctxKey(0)
const KyberSwap = "kyberswap"

type ctxKey int8

func SetClientIDToContext(ctx context.Context, clientID string) context.Context {
	return context.WithValue(ctx, clientIDContextKey, clientID)
}

func GetClientIDFromCtx(ctx context.Context) string {
	v := ctx.Value(clientIDContextKey)
	clientID, _ := v.(string)
	return clientID
}
