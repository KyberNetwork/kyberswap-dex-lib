package getroute

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IAggregator interface {
	Aggregate(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummary, error)
	ApplyConfig(config Config)
}

type IPoolManager interface {
	GetPoolByAddress(
		ctx context.Context,
		addresses, dex []string,
	) (map[string]poolpkg.IPoolSimulator, error)
}

type IPoolFactory interface {
	NewPoolByAddress(ctx context.Context, pools []*entity.Pool) map[string]poolpkg.IPoolSimulator
}

type IGasRepository interface {
	GetSuggestedGasPrice(ctx context.Context) (*big.Int, error)
}

type IPoolRankRepository interface {
	FindBestPoolIDs(
		ctx context.Context,
		tokenIn, tokenOut string,
		opt valueobject.GetBestPoolsOptions,
	) ([]string, error)
}

type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
}

type IRouteCacheRepository interface {
	Set(ctx context.Context, key *valueobject.RouteCacheKey, route *valueobject.SimpleRoute, ttl time.Duration) error
	Get(ctx context.Context, key *valueobject.RouteCacheKey) (*valueobject.SimpleRoute, error)
	Del(ctx context.Context, key *valueobject.RouteCacheKey) error
}

type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
}

type IPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Price, error)
}
