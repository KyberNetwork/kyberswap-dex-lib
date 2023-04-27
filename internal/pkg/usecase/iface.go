package usecase

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

//go:generate mockgen -destination ../mocks/usecase/pool_repository.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IPoolRepository
//go:generate mockgen -destination ../mocks/usecase/token_repository.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase ITokenRepository
//go:generate mockgen -destination ../mocks/usecase/price_repository.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IPriceRepository
//go:generate mockgen -destination ../mocks/usecase/config_fetcher_repository.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IConfigFetcherRepository
//go:generate mockgen -destination ../mocks/usecase/route_cache_repository.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IRouteCacheRepository
//go:generate mockgen -destination ../mocks/usecase/scanner_state_repository.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IScannerStateRepository
//go:generate mockgen -destination ../mocks/usecase/client_data_encoder.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IClientDataEncoder
//go:generate mockgen -destination ../mocks/usecase/encoder.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IEncoder
//go:generate mockgen -destination ../mocks/usecase/l2fee_calculator.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IL2FeeCalculator
//go:generate mockgen -destination ../mocks/usecase/index_pools_route_repository.go -package usecase github.com/KyberNetwork/router-service/internal/pkg/usecase IIndexPoolsRouteRepository

// IPoolRepository receives pool addresses, fetch pool data from datastore, decode them and return []entity.Pool
type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]entity.Pool, error)
	FindAllAddresses(ctx context.Context) ([]string, error)
}

// ITokenRepository receives token addresses, fetch token data from datastore, decode them and return []entity.Token
type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]entity.Token, error)
}

// IPriceRepository receives token addresses, fetch price data from datastore, decode them and return []entity.Price
type IPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]entity.Price, error)
}

type IConfigFetcherRepository interface {
	GetConfigs(ctx context.Context, serviceCode string, currentHash string) (valueobject.RemoteConfig, error)
}

type IRouteRepository interface {
	GetBestPools(ctx context.Context, directPairKey, tokenIn, tokenOut string, options GetBestPoolsOptions, whitelistI, whitelistJ bool) (*types.BestPools, error)
}

// IIndexPoolsRouteRepository is used in IndexPoolsUseCase
// Can not put AddToSortedSetScoreByReserveUsd and AddToSortedSetScoreByAmplifiedTvl into IRouteRepository because of  cyclic dependency when generating mock test
type IIndexPoolsRouteRepository interface {
	AddToSortedSetScoreByReserveUsd(ctx context.Context, pool entity.Pool, key string, tokenIAddress, tokenJAddress string, whiteListI, whiteListJ bool) error
	AddToSortedSetScoreByAmplifiedTvl(ctx context.Context, pool entity.Pool, key string, tokenIAddress, tokenJAddress string, whiteListI, whiteListJ bool) error
}

// IRouteCacheRepository collects, manage and store route cache
type IRouteCacheRepository interface {
	// Set stores route cache
	Set(ctx context.Context, key string, data string, ttl time.Duration) error
	// Get receives key and return data, ttl and error if exists
	Get(ctx context.Context, key string) ([]byte, time.Duration, error)
}

type IScannerStateRepository interface {
	GetGasPrice(ctx context.Context) (*big.Float, error)
	GetL2Fee(ctx context.Context) (*entity.L2Fee, error)
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
	GetExecutorAddress() string
	GetRouterAddress() string
	GetKyberLOAddress() string
}

type IL2FeeCalculator interface {
	SetParams(l2Fee *entity.L2Fee)
	CreateRawTxFromInputData(encodedSwapData string) ([]byte, error)
	GetL1Fee(_data []byte) *big.Int
}

type IL2FeeCalculatorUseCase interface {
	GetL1Fee(ctx context.Context, encodedSwapData string) (*big.Int, error)
}
