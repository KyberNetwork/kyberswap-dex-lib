package getcustomroute

import (
	"context"
	"math/big"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IAggregator interface {
	Aggregate(ctx context.Context, params *types.AggregateParams, poolIds []string) (*valueobject.RouteSummary, error)
}

type IPoolManager interface {
	GetPoolByAddress(
		ctx context.Context,
		addresses, dex []string,
		stateRoot common.Hash,
	) (map[string]poolpkg.IPoolSimulator, error)
	GetAEVMClient() aevmclient.Client
}

type IGasRepository interface {
	GetSuggestedGasPrice(ctx context.Context) (*big.Int, error)
}

type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
}

type IPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Price, error)
}
