package hooklet

import (
	"encoding/json"
	"math/big"

	"github.com/holiman/uint256"
)

type IHooklet interface {
	BeforeSwap(*SwapParams) (feeOverriden bool, fee *uint256.Int, priceOverridden bool, sqrtPriceX96 *uint256.Int)
	AfterSwap(*SwapParams)
}

type SwapParams struct {
	ZeroForOne bool
}

type noopHooklet struct{}

func NewNoopHooklet(_ string) *noopHooklet {
	return &noopHooklet{}
}

func (h *noopHooklet) BeforeSwap(params *SwapParams) (bool, *uint256.Int, bool, *uint256.Int) {
	return false, new(uint256.Int), false, new(uint256.Int)
}

func (h *noopHooklet) AfterSwap(_ *SwapParams) {}

type feeOverrideHooklet struct {
	overrideZeroToOne, overrideOneToZero bool
	feeZeroToOne, feeOneToZero           *uint256.Int
}

type FeeOverride struct {
	OverrideZeroToOne bool
	FeeZeroToOne      *big.Int
	OverrideOneToZero bool
	FeeOneToZero      *big.Int
}

func NewFeeOverrideHooklet(hookletExtra string) IHooklet {
	var feeOverride FeeOverride
	if err := json.Unmarshal([]byte(hookletExtra), &feeOverride); err != nil {
		return nil
	}

	return &feeOverrideHooklet{
		overrideZeroToOne: feeOverride.OverrideZeroToOne,
		feeZeroToOne:      uint256.MustFromBig(feeOverride.FeeZeroToOne),
		overrideOneToZero: feeOverride.OverrideOneToZero,
		feeOneToZero:      uint256.MustFromBig(feeOverride.FeeOneToZero),
	}
}

func (h *feeOverrideHooklet) BeforeSwap(params *SwapParams) (bool, *uint256.Int, bool, *uint256.Int) {
	if params.ZeroForOne {
		return h.overrideZeroToOne, h.feeZeroToOne, false, new(uint256.Int)
	}

	return h.overrideOneToZero, h.feeOneToZero, false, new(uint256.Int)
}

func (h *feeOverrideHooklet) AfterSwap(_ *SwapParams) {}
