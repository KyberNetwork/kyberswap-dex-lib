package buildroute

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/aggregator-encoding/pkg/encode"
	"github.com/KyberNetwork/aggregator-encoding/pkg/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/buildroute/token_repository.go -package buildroute github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute ITokenRepository
type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
	FindTokenInfoByAddress(ctx context.Context, addresses []string) ([]*routerEntity.TokenInfo, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/encode/clientdata/client_data_encoder.go -package clientdata github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute IClientDataEncoder
type IClientDataEncoder interface {
	Encode(ctx context.Context, data types.ClientData) ([]byte, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/buildroute/gas_repository.go -package buildroute github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute IGasRepository
type IGasRepository interface {
	GetSuggestedGasPrice(ctx context.Context) (*big.Int, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/buildroute/encoder_builder.go -package buildroute github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute IEncodeBuilder
type IEncodeBuilder interface {
	GetEncoder(chainId dexValueObject.ChainID) encode.IEncoder
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/buildroute/gas_estimator.go -package buildroute github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute IGasEstimator
type IGasEstimator interface {
	Execute(ctx context.Context, tx UnsignedTransaction) (uint64, float64, error)
	EstimateGas(ctx context.Context, tx UnsignedTransaction) (uint64, error)
	GetGasTokenPriceUSD(ctx context.Context) (float64, error)
}

type IL1FeeCalculator interface {
	CalculateL1Fee(ctx context.Context, routeSummary valueobject.RouteSummary, encodedSwapData string) (*big.Int, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/buildroute/executor_balance_repository.go -package buildroute github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute IExecutorBalanceRepository
type IExecutorBalanceRepository interface {
	HasToken(ctx context.Context, executorAddress string, queries []string) ([]bool, error)
	HasPoolApproval(ctx context.Context, executorAddress string, queries []dto.PoolApprovalQuery) ([]bool, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/buildroute/pool_repository.go -package buildroute github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute IPoolRepository
type IPoolRepository interface {
	TrackFaultyPools(ctx context.Context, trackers []routerEntity.FaultyPoolTracker) ([]string, error)
	GetFaultyPools(ctx context.Context) ([]string, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/buildroute/onchain_price_repository.go -package buildroute github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute IOnchainPriceRepository
type IOnchainPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) (map[string]*routerEntity.OnchainPrice, error)
}
