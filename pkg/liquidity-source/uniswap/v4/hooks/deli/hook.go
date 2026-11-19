package deli

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// slot0LpFee is the LP fee in the pool's slot0, which DeliHook reads as its own fee. Pool.SwapFee also
// holds the zeroForOne protocol fee, so with a protocol fee the LP fee is the high half of Extra pF.
func slot0LpFee(pool *entity.Pool) uint64 {
	var extra struct {
		Fees uint64 `json:"pF"`
	}
	if json.Unmarshal([]byte(pool.Extra), &extra) == nil && extra.Fees != 0 {
		return extra.Fees >> 24
	}
	return uint64(pool.SwapFee)
}

// Hook of idle takes 3% fee of ETH token
type Hook struct {
	*uniswapv4.BaseHook
	FeeTier      *big.Int
	isWBTLToken0 bool
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		BaseHook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Deli},
	}
	if pool := param.Pool; pool != nil {
		hook.FeeTier = big.NewInt(int64(slot0LpFee(param.Pool)))
		hook.isWBTLToken0 = param.Pool.Tokens[0].Address == wBLT
	}
	return hook
}, HookAddresses...)

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	deltaSpecified := bignumber.ZeroBI
	if params.ZeroForOne == h.isWBTLToken0 {
		deltaSpecified = bignumber.MulDivUp(new(big.Int), params.AmountSpecified, h.FeeTier, FeeDenom)
	}
	// The hook overrides the pool's LP fee to 0 and charges its own, so the pool adds only the protocol fee.
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   deltaSpecified,
		DeltaUnspecified: bignumber.ZeroBI,
		SwapFee:          uniswapv4.OverrideFeeFlag,
	}, nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	hookFeeAmt := bignumber.ZeroBI
	if params.ZeroForOne != h.isWBTLToken0 {
		if params.CalcOut {
			hookFeeAmt = h.wBLTFeeOnInput(params)
		} else {
			hookFeeAmt = bignumber.MulDivDown(new(big.Int), params.AmountIn, h.FeeTier, FeeDenom)
		}
	}
	return &uniswapv4.AfterSwapResult{
		HookFee: hookFeeAmt,
	}, nil
}

// wBLTFeeOnInput is the wBLT fee DeliHook.beforeSwap computes for an exact-input swap that buys wBLT: the
// input amount's fee, rounded up, converted to wBLT at the pre-swap spot price (rounded up twice).
func (h *Hook) wBLTFeeOnInput(params *uniswapv4.AfterSwapParams) *big.Int {
	var fee, inter uint256.Int
	feeTier := uint256.NewInt(h.FeeTier.Uint64())
	big256.MulDivUp(&fee, big256.FromBig(params.AmountSpecified), feeTier, UFeeDenom)
	mul, div := params.SqrtPriceX96, Q96 // token0 in, so wBLT is token1: fee * price
	if !params.ZeroForOne {
		mul, div = Q96, params.SqrtPriceX96
	}
	big256.MulDivUp(&inter, &fee, mul, div)
	return big256.MulDivUp(&fee, &inter, mul, div).ToBig()
}
