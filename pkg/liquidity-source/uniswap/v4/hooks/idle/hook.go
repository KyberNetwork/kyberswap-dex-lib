package idle

import (
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	feePct = uint256.NewInt(3)
)

// Hook of idle takes 3% fee of ETH token
type Hook struct {
	*uniswapv4.BaseHook
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	return &Hook{
		BaseHook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4},
	}
}, HookAddresses...)

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	deltaSpecified := new(uint256.Int)
	if params.ZeroForOne {
		big256.MulDivDown(deltaSpecified, params.AmountSpecified, feePct, big256.U100)
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   deltaSpecified,
		DeltaUnspecified: new(uint256.Int),
	}, nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	hookFeeAmt := new(uint256.Int)
	if !params.ZeroForOne {
		big256.MulDivDown(hookFeeAmt,
			lo.Ternary(params.CalcOut, params.AmountOut, params.AmountIn), feePct, big256.U100)
	}
	return &uniswapv4.AfterSwapResult{
		HookFee: hookFeeAmt,
	}, nil
}
