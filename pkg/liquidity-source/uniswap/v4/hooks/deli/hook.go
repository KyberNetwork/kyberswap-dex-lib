package deli

import (
	"math/big"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

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
		hook.FeeTier = big.NewInt(int64(param.Pool.SwapFee))
		hook.isWBTLToken0 = param.Pool.Tokens[0].Address == wBLT
	}
	return hook
}, HookAddresses...)

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	deltaSpecific := bignumber.ZeroBI
	if params.ZeroForOne == h.isWBTLToken0 {
		deltaSpecific = bignumber.MulDivDown(new(big.Int), params.AmountSpecified, h.FeeTier, FeeDenom)
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   deltaSpecific,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	hookFeeAmt := bignumber.ZeroBI
	if params.ZeroForOne != h.isWBTLToken0 {
		hookFeeAmt = bignumber.MulDivDown(new(big.Int), params.AmountOut, h.FeeTier, FeeDenom)
	}
	return &uniswapv4.AfterSwapResult{
		HookFee: hookFeeAmt,
	}, nil
}
