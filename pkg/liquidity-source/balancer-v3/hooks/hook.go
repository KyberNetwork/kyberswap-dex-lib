package hooks

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
)

type IHook interface {
	OnBeforeSwap(param shared.PoolSwapParams) (bool, error)
	OnAfterSwap(param shared.AfterSwapParams) (bool, *uint256.Int, error)
	OnComputeDynamicSwapFeePercentage(param shared.PoolSwapParams) (bool, *uint256.Int, error)
}

type NoOpHook struct{}

var _ IHook = (*NoOpHook)(nil)

func NewNoOpHook() *NoOpHook {
	return &NoOpHook{}
}

func (h *NoOpHook) OnBeforeSwap(shared.PoolSwapParams) (bool, error) {
	return false, nil
}

func (h *NoOpHook) OnAfterSwap(shared.AfterSwapParams) (bool, *uint256.Int, error) {
	return false, math.ZERO, nil
}

func (h *NoOpHook) OnComputeDynamicSwapFeePercentage(shared.PoolSwapParams) (bool, *uint256.Int, error) {
	return false, math.ZERO, nil
}
