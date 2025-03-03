package getcustomroute

import (
	"context"
	"math/big"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IAggregator interface {
	Aggregate(ctx context.Context, params *types.AggregateParams, poolIds []string) (*valueobject.RouteSummary, error)
}

type IPoolFactory interface {
	NewPools(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) []poolpkg.IPoolSimulator
	NewSwapLimit(limits map[string]map[string]*big.Int, poolManagerExtraData types.PoolManagerExtraData) map[string]poolpkg.SwapLimit
}
type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
}

type IPoolManager interface {
	// GetAEVMClient if using AEVM pools, return the AEVM client, otherwise, return nil. Caller should check for nil.
	GetAEVMClient() aevmclient.Client
}

type IGasRepository interface {
	GetSuggestedGasPrice(ctx context.Context) (*big.Int, error)
}

type IL1FeeEstimator interface {
	EstimateL1Fees(ctx context.Context) (*big.Int, *big.Int, error)
}

type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
}

type IOnchainPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) (map[string]*routerEntity.OnchainPrice, error)
}
