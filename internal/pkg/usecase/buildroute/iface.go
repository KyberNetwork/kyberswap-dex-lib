package buildroute

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
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

type IEncoder interface {
	Encode(data types.EncodingData) (string, error)
	GetExecutorAddress() string
	GetRouterAddress() string
}

type IGasRepository interface {
	GetSuggestedGasPrice(ctx context.Context) (*big.Int, error)
}

type IGasEstimator interface {
	Execute(ctx context.Context, tx UnsignedTransaction) (uint64, float64, error)
	EstimateGas(ctx context.Context, tx UnsignedTransaction) (uint64, error)
}

//go:generate mockgen -destination ../../mocks/usecase/buildroute/executor_balance_repository.go -package buildroute github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute IExecutorBalanceRepository
type IExecutorBalanceRepository interface {
	HasToken(executorAddress string, queries []string) ([]bool, error)
	HasPoolApproval(executorAddress string, queries []dto.PoolApprovalQuery) ([]bool, error)
}
