package redis

import (
	"time"

	"context"
)

type DataStoreRepository interface {
	FormatKey(args ...interface{}) string
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, src interface{}) error
}
