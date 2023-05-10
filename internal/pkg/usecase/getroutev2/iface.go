package getroutev2

import (
	"context"
	"math/big"
	"time"

	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IAggregator interface {
	Aggregate(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummary, error)
}

type IPoolManager interface {
	GetPoolByAddress(
		ctx context.Context,
		addresses []string,
		filters ...PoolFilter,
	) (map[string]poolpkg.IPool, error)
}

type IPoolFactory interface {
	NewPoolByAddress(ctx context.Context, pools []*entity.Pool) map[string]poolpkg.IPool
}

type IGasRepository interface {
	GetSuggestedGasPrice(ctx context.Context) (*big.Int, error)
}

type IPoolRankRepository interface {
	FindBestPoolIDs(
		ctx context.Context,
		tokenIn, tokenOut string,
		isTokenInWhitelisted, isTokenOutWhitelisted bool,
		opt types.GetBestPoolsOptions,
	) ([]string, error)
}

type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
}

type IRouteCacheRepository interface {
	Set(ctx context.Context, key *valueobject.RouteCacheKey, route *valueobject.SimpleRoute, ttl time.Duration) error
	Get(ctx context.Context, key *valueobject.RouteCacheKey) (*valueobject.SimpleRoute, error)
}

type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
}

type IPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Price, error)
}
