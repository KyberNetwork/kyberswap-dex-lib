package hooks

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/holiman/uint256"
)

type BaseHook struct{}

var _ IHook = (*BaseHook)(nil)

func NewBaseHook() *BaseHook {
	return &BaseHook{}
}

func (h *BaseHook) OnBeforeSwap(shared.PoolSwapParams) (bool, error) {
	return false, nil
}

func (h *BaseHook) OnAfterSwap(shared.AfterSwapParams) (bool, *uint256.Int, error) {
	return false, math.ZERO, nil

}

func (h *BaseHook) OnComputeDynamicSwapFeePercentage(shared.PoolSwapParams) (bool, *uint256.Int, error) {
	return false, math.ZERO, nil
}
