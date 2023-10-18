package buildroute

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
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
	EstimateGas(sender string, recipient string, data string,
		value *big.Int, gasPrice *big.Int, ctx context.Context) (uint64, error)
}

type IGasEstimator interface {
	Execute(ctx context.Context, tx UnsignedTransaction) (uint64, error)
}
