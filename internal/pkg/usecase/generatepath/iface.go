package generatepath

import (
	"context"
	"math/big"
	"time"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
)

type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
}

type IPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Price, error)
}

type IGasRepository interface {
	GetSuggestedGasPrice(ctx context.Context) (*big.Int, error)
}

type IBestPathRepository interface {
	SetBestPaths(sourcesHash uint64, tokenIn, tokenOut string, data []*entity.MinimalPath, ttl time.Duration) error
	GetPregenTokenAmounts(ctx context.Context) ([]TokenAndAmounts, int64, error)
}

type IPoolManager interface {
	GetStateByPoolAddresses(
		ctx context.Context,
		addresses, dex []string,
		stateRoot common.Hash,
	) (map[string]poolpkg.IPoolSimulator, map[string]poolpkg.SwapLimit, error)
	Reload() error
	GetAEVMClient() aevmclient.Client
}

type ITokenAmountsRepository interface {
	GetTokenAmounts(limit int) error
}
