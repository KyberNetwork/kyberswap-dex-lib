package flaunch

import (
	"math/big"

	"github.com/daoleno/uniswapv3-sdk/constants"
	"github.com/ethereum/go-ethereum/common"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Define our fee constants
var (
	// 1% fee = 10000 basis points (1% of 1,000,000)
	OnePercentFee = uniswapv4.FeeAmount(10000)
	FeeMax        = big.NewInt(int64(constants.FeeMax))
)

// Define our Uniswap V4 Flaunch hook struct.
// @param hook The hook address
// @param swapFee The swap fee charged
type Hook struct {
	uniswapv4.Hook

	hook    common.Address
	swapFee uniswapv4.FeeAmount
}

// Register a hook factory against each of our Flaunch hooks.
var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		// Define our base hook
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Flaunch},

		// Register the PositionManager hook address
		hook: param.HookAddress,

		// Set our swap fee for the hook
		swapFee: OnePercentFee,
	}

	return hook
}, HookAddresses...)

// The Flaunch protocol does not take fees in a standard way. We don't reward the LP provider directly as the
// liquidity position is held and controlled by Flaunch. Instead we capture the fee in the beforeSwap and afterSwap
// hooks and provide the liquidity directly to the owner of the ERC721 token that represents the memecoin.
//
// There is some internal swap logic in the beforeSwap, but essentially our fees will be captured as a consistent
// percentage of the unspecified token amount.
func (h *Hook) BeforeSwap(swapHookParams *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	// Flaunch does not take fees in the beforeSwap hook.
	return &uniswapv4.BeforeSwapResult{
		SwapFee:         uniswapv4.FeeAmount(0),
		DeltaSpecific:   bignumber.ZeroBI,
		DeltaUnSpecific: bignumber.ZeroBI,
	}, nil
}

// We can calculate the fee based on the output amount and the swap fee. We do not take a protocol fee.
func (h *Hook) AfterSwap(swapHookParams *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	// If the hook's swapFee is zero, we don't take a fee.
	if h.swapFee == 0 {
		return &uniswapv4.AfterSwapResult{
			HookFee: bignumber.ZeroBI,
		}, nil
	}

	// Take the fee based on the output amount and the swap fee.
	hookFeeAmt := new(big.Int)
	hookFeeAmt.Mul(swapHookParams.AmountOut, hookFeeAmt.SetUint64(uint64(h.swapFee))).Div(hookFeeAmt, FeeMax)

	return &uniswapv4.AfterSwapResult{
		HookFee: hookFeeAmt,
	}, nil
}
