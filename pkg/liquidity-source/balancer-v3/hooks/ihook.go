package hooks

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/holiman/uint256"
)

type IHook interface {
	OnBeforeSwap()
	OnAfterSwap()
	OnComputeDynamicSwapFeePercentage(param shared.PoolSwapParams) (bool, *uint256.Int, error)
}
