package mooniswap

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestCalcAmountOut_Basic(t *testing.T) {
	srcBalance := uint256.MustFromDecimal("6208659185333448735")
	dstBalance := uint256.MustFromDecimal("12972544827")
	fee := uint256.MustFromDecimal("2650302140801805")
	slippageFee := uint256.MustFromDecimal("835653904615203690")
	amountIn := uint256.MustFromDecimal("1000000000000000000")

	result := calcAmountOut(amountIn, srcBalance, dstBalance, fee, slippageFee)
	require.Equal(t, "1587806769", result.Dec())
}

func TestCalcAmountOut_Reverse(t *testing.T) {
	// 1000 USDT → ETH
	// srcBalance (USDT getBalanceForAddition): 12972544827
	// dstBalance (ETH getBalanceForRemoval): 6208659185333448735
	// same fee/slippageFee (they're pool-level, not direction-dependent)
	// amountIn: 1000000000 (1000 USDT)
	// expected: 416809127717440146 (from on-chain getReturn)

	srcBalance := uint256.MustFromDecimal("12972544827")
	dstBalance := uint256.MustFromDecimal("6208659185333448735")
	fee := uint256.MustFromDecimal("2650302140801805")
	slippageFee := uint256.MustFromDecimal("835653904615203690")
	amountIn := uint256.MustFromDecimal("1000000000")

	result := calcAmountOut(amountIn, srcBalance, dstBalance, fee, slippageFee)
	require.Equal(t, "416809127717440146", result.Dec())
}

func TestCalcAmountOut_ZeroAmount(t *testing.T) {
	srcBalance := uint256.MustFromDecimal("1000000000")
	dstBalance := uint256.MustFromDecimal("1000000000")
	fee := uint256.NewInt(0)
	slippageFee := uint256.MustFromDecimal("1000000000000000000")

	result := calcAmountOut(uint256.NewInt(0), srcBalance, dstBalance, fee, slippageFee)
	require.True(t, result.IsZero())
}

func TestCalcAmountOut_ZeroFee(t *testing.T) {
	// With zero fee and zero slippage fee, should be standard CPMM
	srcBalance := uint256.NewInt(1000000)
	dstBalance := uint256.NewInt(1000000)
	fee := uint256.NewInt(0)
	slippageFee := uint256.NewInt(0)
	amountIn := uint256.NewInt(1000)

	result := calcAmountOut(amountIn, srcBalance, dstBalance, fee, slippageFee)
	// Standard CPMM: 1000 * 1000000 / (1000000 + 1000) = 999
	require.Equal(t, "999", result.Dec())
}

func TestCalcAmountOut_DefaultParams(t *testing.T) {
	// Default Mooniswap V2 params: fee=0, slippageFee=1e18 (100%)
	srcBalance := uint256.NewInt(1000000000)
	dstBalance := uint256.NewInt(1000000000)
	fee := uint256.NewInt(0)
	slippageFee := uint256.MustFromDecimal("1000000000000000000") // 100%
	amountIn := uint256.NewInt(1000000)

	result := calcAmountOut(amountIn, srcBalance, dstBalance, fee, slippageFee)
	// CPMM gives 999000; with 100% slippage fee the result is ~998001 (less due to slippage penalty)
	require.Equal(t, uint64(998001), result.Uint64())
}
