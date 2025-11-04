package flaunch

import (
	"math/big"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	FeeDivBy = big.NewInt(100) // 1%
)

type Hook struct {
	uniswapv4.Hook
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	return &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Flaunch},
	}
}, HookAddresses...)

// BeforeSwap of Flaunch protocol does not take fees in a standard way. The LP providers don't get rewarded directly as
// the liquidity position is held and controlled by Flaunch. Instead, the fee is captured in the beforeSwap and
// afterSwap hooks and provide the liquidity directly to the owner of the ERC721 token that represents the memecoin.
//
// There is some internal swap logic in the beforeSwap, but essentially the fees will be captured as a consistent
// percentage of the unspecified token amount.
func (h *Hook) BeforeSwap(_ *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	// Flaunch does not take fees in the beforeSwap hook.
	return &uniswapv4.BeforeSwapResult{
		SwapFee:          uniswapv4.FeeAmount(0),
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}

// AfterSwap calculates the fee based on the output amount and the swap fee. There's no protocol fee.
func (h *Hook) AfterSwap(swapHookParams *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	var hookFeeAmt big.Int
	return &uniswapv4.AfterSwapResult{
		HookFee: hookFeeAmt.Div(swapHookParams.AmountOut, FeeDivBy),
	}, nil
}
