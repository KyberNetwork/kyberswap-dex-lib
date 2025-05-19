package usecase

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/aggregator-encoding/pkg/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

//go:generate go run go.uber.org/mock/mockgen -destination ../mocks/usecase/mocks.go -source=iface.go
//go:generate go run go.uber.org/mock/mockgen -destination ../mocks/usecase/pool/pool_simulator.go -package usecase github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool IPoolSimulator

// IPoolRepository receives pool addresses, fetch pool data from datastore, decode them and return []entity.Pool
type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
	FindAllAddresses(ctx context.Context) ([]string, error)
}

// ITokenRepository receives token addresses, fetch token data from datastore, decode them and return []entity.SimplifiedToken
type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.SimplifiedToken, error)
}

type ITokenFullInfoRepository[T entity.Token] interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*T, error)
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
		key string, memberName string, useGlobal bool,
	) error
	RemoveAddressesFromWhitelistIndex(ctx context.Context, key string, pools []string, removeFromGlobal bool) error
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
