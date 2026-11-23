package deli

import (
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Hook of idle takes 3% fee of ETH token
type Hook struct {
	*uniswapv4.BaseHook
	FeeTier      *uint256.Int
	isWBTLToken0 bool
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		BaseHook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Deli},
	}
	if pool := param.Pool; pool != nil {
		hook.FeeTier = uint256.NewInt(uint64(param.Pool.SwapFee))
		hook.isWBTLToken0 = param.Pool.Tokens[0].Address == wBLT
	}
	return hook
}, HookAddresses...)

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	deltaSpecified := new(uint256.Int)
	if params.ZeroForOne == h.isWBTLToken0 {
		big256.MulDivDown(deltaSpecified, params.AmountSpecified, h.FeeTier, UFeeDenom)
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   deltaSpecified,
		DeltaUnspecified: new(uint256.Int),
	}, nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	hookFeeAmt := new(uint256.Int)
	if params.ZeroForOne != h.isWBTLToken0 {
		big256.MulDivDown(hookFeeAmt, lo.Ternary(params.CalcOut, params.AmountOut, params.AmountIn),
			h.FeeTier, UFeeDenom)
	}
	return &uniswapv4.AfterSwapResult{
		HookFee: hookFeeAmt,
	}, nil
}
