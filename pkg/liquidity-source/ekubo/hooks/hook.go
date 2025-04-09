package ekubo

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type HooksConfig struct {
	ShouldCallBeforeSwap bool `json:"befSwap,omitempty"`
	ShouldCallAfterSwap  bool `json:"aftSwap,omitempty"`
}

type PoolSwapParams struct {
	PoolKey        quoting.PoolKey
	Amount         *big.Int
	IsToken1       bool
	SqrtRatioLimit *big.Int
}

type SwapResult struct {
	ConsumedAmount   *big.Int
	CalculatedAmount *big.Int
	FeesPaid         *big.Int
	SkipAhead        uint32

	SqrtRatio       *big.Int
	Liquidity       *big.Int
	ActiveTickIndex int

	InitializedTicksCrossed uint32
	TickSpacingsCrossed     uint32
}

type IHook interface {
	OnBeforeSwap(param *PoolSwapParams) (uint64, error)
	OnAfterSwap(param *SwapResult) (uint64, error)
}

type NoOpHook struct{}

var _ IHook = (*NoOpHook)(nil)

func NewNoOpHook() *NoOpHook {
	return &NoOpHook{}
}

func (h *NoOpHook) OnBeforeSwap(*PoolSwapParams) (gas uint64, err error) {
	return 0, nil
}

func (h *NoOpHook) OnAfterSwap(*SwapResult) (gas uint64, err error) {
	return 0, nil
}
