package consumer

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

// IgnoreNilRedis ignores redis.Nil in XReadGroup to avoid gtrs error.
type IgnoreNilRedis struct {
	redis.Cmdable
}

// XReadGroup ignores redis.Nil in XReadGroup to avoid gtrs error.
func (r *IgnoreNilRedis) XReadGroup(ctx context.Context, args *redis.XReadGroupArgs) *redis.XStreamSliceCmd {
	ret := r.Cmdable.XReadGroup(ctx, args)
	if errors.Is(ret.Err(), redis.Nil) {
		return &redis.XStreamSliceCmd{}
	}
	return ret
}
