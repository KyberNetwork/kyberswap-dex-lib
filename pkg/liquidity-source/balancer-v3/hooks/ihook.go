package hooks

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/holiman/uint256"
)

type IHook interface {
	OnBeforeSwap(param shared.PoolSwapParams) (bool, error)
	OnAfterSwap(param shared.AfterSwapParams) (bool, *uint256.Int, error)
	OnComputeDynamicSwapFeePercentage(param shared.PoolSwapParams) (bool, *uint256.Int, error)
}
