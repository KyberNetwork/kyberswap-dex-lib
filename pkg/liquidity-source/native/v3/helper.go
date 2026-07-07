package v3

import "context"

type contextKey string

const (
	underlyingScannedKey contextKey = "scanned"
)

func NewContextWithUnderlyingScanned(ctx context.Context, scanned bool) context.Context {
	return context.WithValue(ctx, underlyingScannedKey, scanned)
}

func IsUnderlyingScanned(ctx context.Context) bool {
	if v := ctx.Value(underlyingScannedKey); v != nil {
		if scanned, ok := v.(bool); ok {
			return scanned
		}
	}
	return false
}
