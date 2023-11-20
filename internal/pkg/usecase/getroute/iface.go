package getroute

import (
	"context"
	"math/big"
	"time"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IAggregator interface {
	Aggregate(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummary, error)
	ApplyConfig(config Config)
}

//go:generate mockgen -destination ../../mocks/usecase/getroute/pool_manager.go -package getroute github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute IPoolManager
type IPoolManager interface {
	// GetStateByPoolAddresses return a map of address - pools and a map of dexType- swapLimit for
	GetStateByPoolAddresses(
		ctx context.Context,
		addresses, dex []string,
		stateRoot common.Hash,
	) (map[string]poolpkg.IPoolSimulator, map[string]poolpkg.SwapLimit, error)
	GetAEVMClient() aevmclient.Client
}

type IPoolFactory interface {
	NewPoolByAddress(ctx context.Context, pools []*entity.Pool) map[string]poolpkg.IPoolSimulator
}

//go:generate mockgen -destination ../../mocks/usecase/getroute/gas_repository.go -package getroute github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute IGasRepository
type IGasRepository interface {
	GetSuggestedGasPrice(ctx context.Context) (*big.Int, error)
}

//go:generate mockgen -destination ../../mocks/usecase/getroute/pool_rank_repository.go -package getroute github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute IPoolRankRepository
type IPoolRankRepository interface {
	FindBestPoolIDs(
		ctx context.Context,
		tokenIn, tokenOut string,
		opt valueobject.GetBestPoolsOptions,
	) ([]string, error)
}

//go:generate mockgen -destination ../../mocks/usecase/getroute/best_path_repository.go -package getroute github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute IBestPathRepository
type IBestPathRepository interface {
	GetBestPaths(sourceHash uint64, tokenIn, tokenOut string) []*entity.MinimalPath
}

type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
}

//go:generate mockgen -destination ../../mocks/usecase/getroute/route_cache_repository.go -package getroute github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute IRouteCacheRepository
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
