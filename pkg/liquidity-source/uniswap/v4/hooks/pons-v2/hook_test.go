package ponsv2

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// AfterSwap must mirror PonsV2MemeHook._afterSwap: feeAmount + taxAmount = floor(unspecified
// * (hookFeeBps + creatorTaxBps) / BASIS_POINTS), taken from the realized output on exact-in.
func TestAfterSwap_ExactIn_ChargesFeeOnOutput(t *testing.T) {
	h := &Hook{Extra: Extra{Registered: true, FeeBps: 100 + 50}} // hookFeeBps=100, creatorTaxBps=50

	result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountOut:        big.NewInt(1_000_000),
	})
	assert.NoError(t, err)
	// floor(1_000_000 * 150 / 10_000) = 15_000
	assert.Equal(t, big.NewInt(15_000), result.HookFee)
}

// On exact-out the unspecified currency is the input, so the fee is taken from the
// realized input instead -- the swapper pays more than the AMM's own quote, not less.
func TestAfterSwap_ExactOut_ChargesFeeOnInput(t *testing.T) {
	h := &Hook{Extra: Extra{Registered: true, FeeBps: 150}}

	result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: false},
		AmountIn:         big.NewInt(1_000_000),
	})
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(15_000), result.HookFee)
}

// A pool never seen by Track() (RPC failure, or genuinely not yet registered with the
// factory) must price as fee-free rather than error -- unlike b20's LaunchHook, an
// unregistered PonsV2MemeHook pool really does charge nothing (_afterSwap's own
// `if (!info.registered) return 0` early-out), so refusing to quote would be wrong here.
func TestAfterSwap_Unregistered_NoFee(t *testing.T) {
	h := &Hook{}

	result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountOut:        big.NewInt(1_000_000),
	})
	assert.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, result.HookFee)
}

func TestAfterSwap_ZeroFee_NoOp(t *testing.T) {
	h := &Hook{Extra: Extra{Registered: true, FeeBps: 0}}

	result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountOut:        big.NewInt(1_000_000),
	})
	assert.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, result.HookFee)
}
