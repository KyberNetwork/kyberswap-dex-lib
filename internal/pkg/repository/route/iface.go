package route

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IFallbackRepository interface {
	Get(ctx context.Context, keys []valueobject.RouteCacheKeyTTL) (map[valueobject.RouteCacheKeyTTL]*valueobject.SimpleRouteWithExtraData, error)
	Set(ctx context.Context, keys []valueobject.RouteCacheKeyTTL, routes []*valueobject.SimpleRouteWithExtraData) ([]*valueobject.SimpleRouteWithExtraData, error)
	Del(ctx context.Context, keys []valueobject.RouteCacheKeyTTL) error
}
