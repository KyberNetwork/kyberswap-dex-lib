package buildroute

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/aggregator-encoding/pkg/encode"
	"github.com/KyberNetwork/aggregator-encoding/pkg/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	routerEntities "github.com/KyberNetwork/router-service/internal/pkg/entity"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
}

type IPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Price, error)
}

type IClientDataEncoder interface {
	Encode(ctx context.Context, data types.ClientData) ([]byte, error)
}

type IGasRepository interface {
	GetSuggestedGasPrice(ctx context.Context) (*big.Int, error)
}

type IEncodeBuilder interface {
	GetEncoder(chainId dexValueObject.ChainID) encode.IEncoder
}

//go:generate mockgen -destination ../../mocks/usecase/buildroute/gas_estimator.go -package buildroute github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute IGasEstimator
type IGasEstimator interface {
	Execute(ctx context.Context, tx UnsignedTransaction) (uint64, float64, error)
	EstimateGas(ctx context.Context, tx UnsignedTransaction) (uint64, error)
	GetGasTokenPriceUSD(ctx context.Context) (float64, error)
}

type IL1FeeCalculator interface {
	CalculateL1Fee(ctx context.Context, chainId valueobject.ChainID, encodedSwapData string) (*big.Int, error)
}

//go:generate mockgen -destination ../../mocks/usecase/buildroute/executor_balance_repository.go -package buildroute github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute IExecutorBalanceRepository
type IExecutorBalanceRepository interface {
	HasToken(executorAddress string, queries []string) ([]bool, error)
	HasPoolApproval(executorAddress string, queries []dto.PoolApprovalQuery) ([]bool, error)
}

//go:generate mockgen -destination ../../mocks/usecase/buildroute/pool_repository.go -package buildroute github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute IPoolRepository
type IPoolRepository interface {
	TrackFaultyPools(ctx context.Context, trackers []routerEntities.FaultyPoolTracker) ([]string, error)
	GetFaultyPools(ctx context.Context, offset, count int64) ([]string, error)
}

type IOnchainPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) (map[string]*routerEntity.OnchainPrice, error)
}
