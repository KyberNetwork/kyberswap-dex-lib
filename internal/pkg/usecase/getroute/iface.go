package getroute

import (
	"context"
	"math/big"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IAggregator interface {
	Aggregate(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummary, error)
	ApplyConfig(config Config)
}

type IBundledAggregator interface {
	Aggregate(ctx context.Context, params *types.AggregateBundledParams) ([]*valueobject.RouteSummary, error)
	ApplyConfig(config Config)
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/getroute/pool_manager.go -package getroute github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute IPoolManager
type IPoolManager interface {
	// GetStateByPoolAddresses return a map of address - pools and a map of dexType- swapLimit for
	GetStateByPoolAddresses(
		ctx context.Context,
		addresses, dex []string,
		stateRoot common.Hash,
		extraData types.PoolManagerExtraData,
	) (*types.FindRouteState, error)
	// GetAEVMClient if using AEVM pools, return the AEVM client, otherwise, return nil. Caller should check for nil.
	GetAEVMClient() aevmclient.Client
}

type IPoolFactory interface {
	NewPoolByAddress(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) map[string]poolpkg.IPoolSimulator
	CloneCurveMetaForBasePools(
		ctx context.Context,
		allPools map[string]poolpkg.IPoolSimulator,
		basePools map[string]poolpkg.IPoolSimulator,
	) []poolpkg.IPoolSimulator
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/getroute/gas_repository.go -package getroute github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute IGasRepository
type IGasRepository interface {
	GetSuggestedGasPrice(ctx context.Context) (*big.Int, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/getroute/pool_rank_repository.go -package getroute github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute IPoolRankRepository
type IPoolRankRepository interface {
	FindBestPoolIDs(
		ctx context.Context,
		tokenIn, tokenOut string,
		amountIn float64,
		opt valueobject.GetBestPoolsOptions,
		index valueobject.IndexType,
	) ([]string, error)
}

type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/getroute/route_cache_repository.go -package getroute github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute IRouteCacheRepository
type IRouteCacheRepository interface {
	Get(ctx context.Context, keys []valueobject.RouteCacheKeyTTL) (map[valueobject.RouteCacheKeyTTL]*valueobject.SimpleRoute, error)
	Set(ctx context.Context, keys []valueobject.RouteCacheKeyTTL, routes []*valueobject.SimpleRoute) error
	Del(ctx context.Context, keys []valueobject.RouteCacheKeyTTL) error
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/getroute/token_repository.go -package getroute github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute ITokenRepository
type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/getroute/onchain_price_repository.go -package getroute github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute IOnchainPriceRepository
type IOnchainPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) (map[string]*routerEntity.OnchainPrice, error)
	RefreshCacheNativePriceInUSD(ctx context.Context)
}

type IPoolsPublisher interface {
	PublishedPoolIDs(storageID string) map[string]struct{}
	PublishedPools(storageID string) map[string]poolpkg.IPoolSimulator
	Publish(ctx context.Context, pools map[string]poolpkg.IPoolSimulator) (string, error)
}
