package indexpools

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getpools"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
)

// IPoolRepository receives pool addresses, fetch pool data from datastore, decode them and return []entity.Pool
//
//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/indexpools/pool_repository.go -package indexpools github.com/KyberNetwork/router-service/internal/pkg/usecase/indexpools IPoolRepository
type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
	FindAllAddresses(ctx context.Context) ([]string, error)
	GetPoolsInBlacklist(ctx context.Context) ([]string, error)
	FindAddressesByDex(ctx context.Context, dex string) ([]string, error)
	Count(ctx context.Context) int64
	ScanPools(ctx context.Context, cursor uint64, count int) ([]*entity.Pool, []string, uint64, error)
}

type IBlacklistIndexPoolRepository interface {
	AddToBlacklistIndexPools(ctx context.Context, addresses []string)
	GetBlacklistIndexPools(ctx context.Context) mapset.Set[string]
}

// ITokenRepository receives token addresses, fetch token data from datastore, decode them and return []entity.Token
//
//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/indexpools/token_repository.go -package indexpools github.com/KyberNetwork/router-service/internal/pkg/usecase/indexpools ITokenRepository
type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/indexpools/onchain_price_repository.go -package indexpools github.com/KyberNetwork/router-service/internal/pkg/usecase/indexpools IOnchainPriceRepository
type IOnchainPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) (map[string]*routerEntity.OnchainPrice, error)
	RefreshCacheNativePriceInUSD(ctx context.Context)
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/indexpools/pool_rank_repository.go -package indexpools github.com/KyberNetwork/router-service/internal/pkg/usecase/indexpools IPoolRankRepository
type IPoolRankRepository interface {
	AddToSortedSet(
		ctx context.Context,
		token0, token1 string,
		isToken0Whitelisted, isToken1Whitelisted bool,
		key string, memberName string, score float64,
		useGlobal bool,
	) error
	RemoveFromSortedSet(
		ctx context.Context,
		token0, token1 string,
		isToken0Whitelisted, isToken1Whitelisted bool,
		key string, memberName string, useGlobal bool,
	) error
	RemoveAddressFromIndex(ctx context.Context, key string, pools []string) error
	GetDirectIndexLength(ctx context.Context, key, token0, token1 string) (int64, error)
	AddToWhitelistSortedSet(ctx context.Context, scores []routerEntity.PoolScore, sortBy string, count int64) error
}

type IPoolFactory interface {
	NewPoolByAddress(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) map[string]poolpkg.IPoolSimulator
	NewPools(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) []poolpkg.IPoolSimulator
	NewSwapLimit(limits map[string]map[string]*big.Int) map[string]poolpkg.SwapLimit
}

type IGetPoolsIncludingBasePools interface {
	Handle(ctx context.Context, addresses []string, filter getpools.PoolFilter) ([]*entity.Pool, error)
}
