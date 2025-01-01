package hooks

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/holiman/uint256"
)

type StableSurgeHook struct {
	BaseHook
}

func NewStableSurgeHook() *StableSurgeHook {
	return &StableSurgeHook{}
}

func (h *StableSurgeHook) OnComputeDynamicSwapFeePercentage(
	staticSwapFeePercentage,
	amountGivenScaled18,
	balanceIn,
	balanceOut *uint256.Int,
) (bool, *uint256.Int, error) {
	return false, staticSwapFeePercentage, nil
}

func (h *StableSurgeHook) getSurgeFeePercentage(params shared.VaultSwapParams) (*uint256.Int, error) {
	return nil, nil
}
