package hooks

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/holiman/uint256"
)

type baseHook struct{}

func NewBaseHook() *baseHook {
	return &baseHook{}
}

func (h *baseHook) OnBeforeSwap(shared.PoolSwapParams) (bool, error) {
	return false, nil
}

func (h *baseHook) OnAfterSwap(shared.AfterSwapParams) (bool, *uint256.Int, error) {
	return false, math.ZERO, nil

}

func (h *baseHook) OnComputeDynamicSwapFeePercentage(shared.PoolSwapParams) (bool, *uint256.Int, error) {
	return false, math.ZERO, nil
}
