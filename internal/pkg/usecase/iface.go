package usecase

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/aggregator-encoding/pkg/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
)

//go:generate mockgen -destination ../mocks/usecase/pool_repository.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IPoolRepository
//go:generate mockgen -destination ../mocks/usecase/token_repository.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase ITokenRepository
//go:generate mockgen -destination ../mocks/usecase/price_repository.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IPriceRepository
//go:generate mockgen -destination ../mocks/usecase/config_fetcher_repository.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IConfigFetcherRepository
//go:generate mockgen -destination ../mocks/usecase/client_data_encoder.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IClientDataEncoder
//go:generate mockgen -destination ../mocks/usecase/encoder.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IEncoder
//go:generate mockgen -destination ../mocks/usecase/pool_rank_repository.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IPoolRankRepository

// IPoolRepository receives pool addresses, fetch pool data from datastore, decode them and return []entity.Pool
type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
	FindAllAddresses(ctx context.Context) ([]string, error)
}

// ITokenRepository receives token addresses, fetch token data from datastore, decode them and return []entity.Token
type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
}

// IPriceRepository receives token addresses, fetch price data from datastore, decode them and return []entity.Price
type IPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Price, error)
}

type IOnchainPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) (map[string]*routerEntity.OnchainPrice, error)
}

type IConfigFetcherRepository interface {
	GetConfigs(ctx context.Context, serviceCode string, currentHash string) (valueobject.RemoteConfig, error)
}

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
		key string, memberName string, score float64,
		useGlobal bool,
	) error
	RemoveAddressFromIndex(ctx context.Context, key string, pools []string) error
	GetDirectIndexLength(ctx context.Context, key, token0, token1 string) (int64, error)
}

type IGasRepository interface {
	UpdateSuggestedGasPrice(ctx context.Context) (*big.Int, error)
	GetSuggestedGasPrice(ctx context.Context) (*big.Int, error)
}

type IClientDataEncoder interface {
	Encode(ctx context.Context, data types.ClientData) ([]byte, error)
}

// IEncoder encodes swap data
type IEncoder interface {
	Encode(data types.EncodingData) (string, error)
	GetExecutorAddress(clientID string) string
	GetRouterAddress() string
}

type IPoolFactory interface {
	NewPoolByAddress(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) map[string]poolpkg.IPoolSimulator
	NewPools(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) []poolpkg.IPoolSimulator
}
