package printr

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test curve parameters matching a typical PRINTR deployment:
// - maxTokenSupplyE = 9 → maxTokenSupply = 10^9 * 1e18 = 1e27
// - totalCurves = 1
// - virtualReserve = 1e18 (1 ETH)
// - completionThreshold = 5000 * 10^9 = 5e12 tokens (50% of max supply in getCurve format)
// - tradingFee = 100 (1%)

func testMaxTokenSupply() *uint256.Int {
	// 10^9 * 1e18 = 1e27
	return mustFromDecimal("1000000000000000000000000000")
}

func testVirtualReserve() *uint256.Int {
	return mustFromDecimal("1000000000000000000") // 1e18
}

func testCompletionThreshold() *uint256.Int {
	// getCurve returns: completionThreshold * 10^maxTokenSupplyE = 5000 * 10^9 = 5e12
	// In tokens (with 18 decimals): 5e12 * 1e18 = 5e30
	// But completion threshold from getCurve is already in full token units without decimals
	// Actually: getCurve returns completionThreshold * 10^maxTokenSupplyE
	// The raw curve.completionThreshold is in PRECISION-scaled per-chain units
	// For simplicity, let's use completionThreshold = 5e26 (half of 1e27 supply)
	return mustFromDecimal("500000000000000000000000000")
}

func mustFromDecimal(s string) *uint256.Int {
	v, err := uint256.FromDecimal(s)
	if err != nil {
		panic(err)
	}
	return v
}

func TestCalcBuyCost_Basic(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	completionThreshold := testCompletionThreshold()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100) // 1%

	// Buy 1e18 tokens (1 token with 18 decimals)
	tokenAmount := mustFromDecimal("1000000000000000000")

	result := CalcBuyCost(maxTokenSupply, 1, virtualReserve, reserve, completionThreshold, tradingFee, tokenAmount)

	// availableAmount should equal tokenAmount (not capped by threshold)
	assert.True(t, result.AvailableAmount.Eq(tokenAmount), "available should equal requested")

	// cost should be non-zero
	assert.True(t, result.Cost.Sign() > 0, "cost should be positive")

	// fee should be non-zero (1% of curveCost)
	assert.True(t, result.Fee.Sign() > 0, "fee should be positive")

	// cost should be greater than fee
	assert.True(t, result.Cost.Gt(result.Fee), "cost should exceed fee")
}

func TestCalcBuyCost_ZeroAmount(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	completionThreshold := testCompletionThreshold()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100)

	tokenAmount := uint256.NewInt(0)
	result := CalcBuyCost(maxTokenSupply, 1, virtualReserve, reserve, completionThreshold, tradingFee, tokenAmount)

	assert.True(t, result.Cost.IsZero(), "cost should be zero for zero amount")
	assert.True(t, result.Fee.IsZero(), "fee should be zero for zero amount")
}

func TestCalcBuyCost_CappedByCompletionThreshold(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	completionThreshold := testCompletionThreshold()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100)

	// Try to buy more than completionThreshold
	tokenAmount := mustFromDecimal("600000000000000000000000000") // 6e26, exceeds 5e26 threshold
	result := CalcBuyCost(maxTokenSupply, 1, virtualReserve, reserve, completionThreshold, tradingFee, tokenAmount)

	// Should be capped at completionThreshold
	assert.True(t, result.AvailableAmount.Eq(completionThreshold),
		"available should be capped at completion threshold, got %s", result.AvailableAmount.String())
}

func TestCalcBuyCost_ZeroFee(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	completionThreshold := testCompletionThreshold()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(0) // No fee

	tokenAmount := mustFromDecimal("1000000000000000000")
	result := CalcBuyCost(maxTokenSupply, 1, virtualReserve, reserve, completionThreshold, tradingFee, tokenAmount)

	assert.True(t, result.Fee.IsZero(), "fee should be zero when tradingFee is 0")
	assert.True(t, result.Cost.Sign() > 0, "cost should still be positive")
}

func TestCalcBuyCost_MinFeeOneWei(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	completionThreshold := testCompletionThreshold()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(1) // 0.01%

	// Very small buy where fee would round to 0
	tokenAmount := uint256.NewInt(1) // 1 wei of tokens
	result := CalcBuyCost(maxTokenSupply, 1, virtualReserve, reserve, completionThreshold, tradingFee, tokenAmount)

	// With tradingFee > 0, min fee is 1
	if result.Cost.Sign() > 0 {
		assert.True(t, result.Fee.Sign() > 0, "fee should be at least 1 when tradingFee > 0")
	}
}

func TestCalcBuyCost_RoundUpCheck(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	completionThreshold := testCompletionThreshold()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100)

	// Buy a specific amount and verify cost > 0
	tokenAmount := mustFromDecimal("12345678901234567890") // ~12.3 tokens
	result := CalcBuyCost(maxTokenSupply, 1, virtualReserve, reserve, completionThreshold, tradingFee, tokenAmount)

	assert.True(t, result.Cost.Sign() > 0, "cost should be positive")
	assert.True(t, result.AvailableAmount.Eq(tokenAmount), "should get full amount requested")
}

func TestCalcBuyTokenAmount_Basic(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100) // 1%

	// Spend 0.01 ETH
	baseSpend := mustFromDecimal("10000000000000000") // 0.01e18

	tokenAmount := CalcBuyTokenAmount(maxTokenSupply, 1, virtualReserve, reserve, tradingFee, baseSpend)

	assert.True(t, tokenAmount.Sign() > 0, "token amount should be positive")
}

func TestCalcBuyTokenAmount_ZeroSpend(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100)

	baseSpend := uint256.NewInt(0)
	tokenAmount := CalcBuyTokenAmount(maxTokenSupply, 1, virtualReserve, reserve, tradingFee, baseSpend)

	assert.True(t, tokenAmount.IsZero(), "token amount should be zero for zero spend")
}

func TestCalcBuyTokenAmount_RoundTrip(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	completionThreshold := testCompletionThreshold()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100) // 1%

	// Spend 1 ETH
	baseSpend := mustFromDecimal("1000000000000000000") // 1e18

	// Calculate how many tokens for this spend
	tokenAmount := CalcBuyTokenAmount(maxTokenSupply, 1, virtualReserve, reserve, tradingFee, baseSpend)
	require.True(t, tokenAmount.Sign() > 0, "should get some tokens")

	// Now calculate cost for those tokens
	costResult := CalcBuyCost(maxTokenSupply, 1, virtualReserve, reserve, completionThreshold, tradingFee, tokenAmount)

	// The cost should be <= baseSpend (spend quotes conservatively)
	assert.True(t, !costResult.Cost.Gt(baseSpend),
		"cost %s should be <= baseSpend %s", costResult.Cost.String(), baseSpend.String())
}

func TestCalcBuyTokenAmount_ZeroFee(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(0)

	baseSpend := mustFromDecimal("1000000000000000000")
	tokenAmount := CalcBuyTokenAmount(maxTokenSupply, 1, virtualReserve, reserve, tradingFee, baseSpend)

	// With zero fee, should get slightly more tokens
	tokenAmountWithFee := CalcBuyTokenAmount(maxTokenSupply, 1, virtualReserve, reserve, 100, baseSpend)

	assert.True(t, tokenAmount.Gt(tokenAmountWithFee),
		"zero fee should yield more tokens: %s vs %s", tokenAmount.String(), tokenAmountWithFee.String())
}

func TestCalcSellRefund_Basic(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	tradingFee := uint16(100) // 1%

	// First buy some tokens (simulate reserve build-up)
	// reserve = 0.5 ETH
	reserve := mustFromDecimal("500000000000000000") // 0.5e18

	// Sell 1e18 tokens (1 token)
	tokenAmount := mustFromDecimal("1000000000000000000")

	result := CalcSellRefund(maxTokenSupply, 1, virtualReserve, reserve, tradingFee, tokenAmount)

	assert.True(t, result.Refund.Sign() > 0, "refund should be positive")
	assert.True(t, result.Fee.Sign() > 0, "fee should be positive")
	assert.True(t, result.TokenAmountIn.Eq(tokenAmount), "should sell full amount")
}

func TestCalcSellRefund_CappedByIssuedSupply(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	reserve := mustFromDecimal("500000000000000000") // 0.5e18
	tradingFee := uint16(100)

	// Try to sell way more than issued supply
	tokenAmount := mustFromDecimal("999000000000000000000000000") // ~1e27
	result := CalcSellRefund(maxTokenSupply, 1, virtualReserve, reserve, tradingFee, tokenAmount)

	// tokenAmountIn should be capped at currentIssuedSupply
	assert.True(t, !result.TokenAmountIn.Gt(tokenAmount), "should be capped")
	assert.True(t, result.Refund.Sign() > 0, "refund should be positive when there's issued supply")
}

func TestCalcSellRefund_ZeroReserve(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100)

	// With zero reserve, there are no issued tokens to sell
	tokenAmount := mustFromDecimal("1000000000000000000")
	result := CalcSellRefund(maxTokenSupply, 1, virtualReserve, reserve, tradingFee, tokenAmount)

	// issuedSupply = 0, so tokenAmountIn should be 0
	assert.True(t, result.TokenAmountIn.IsZero(), "no tokens to sell when reserve is 0")
	assert.True(t, result.Refund.IsZero(), "refund should be 0")
}

func TestCalcSellRefund_ZeroFee(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	reserve := mustFromDecimal("500000000000000000")
	tradingFee := uint16(0)

	tokenAmount := mustFromDecimal("1000000000000000000")

	result := CalcSellRefund(maxTokenSupply, 1, virtualReserve, reserve, tradingFee, tokenAmount)

	assert.True(t, result.Fee.IsZero(), "fee should be zero when tradingFee is 0")
	assert.True(t, result.Refund.Sign() > 0, "refund should still be positive")
}

func TestBuySellRoundTrip(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	completionThreshold := testCompletionThreshold()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100) // 1%

	// Buy tokens with 1 ETH
	baseSpend := mustFromDecimal("1000000000000000000")
	tokenAmount := CalcBuyTokenAmount(maxTokenSupply, 1, virtualReserve, reserve, tradingFee, baseSpend)
	require.True(t, tokenAmount.Sign() > 0)

	buyResult := CalcBuyCost(maxTokenSupply, 1, virtualReserve, reserve, completionThreshold, tradingFee, tokenAmount)
	require.True(t, buyResult.Cost.Sign() > 0)

	// Update reserve after buy
	newReserve := new(uint256.Int).Sub(buyResult.Cost, buyResult.Fee)

	// Now sell all tokens back
	sellResult := CalcSellRefund(maxTokenSupply, 1, virtualReserve, newReserve, tradingFee, tokenAmount)

	// Due to trading fees (both buy and sell), refund should be less than original spend
	assert.True(t, sellResult.Refund.Lt(baseSpend),
		"refund %s should be less than original spend %s due to fees",
		sellResult.Refund.String(), baseSpend.String())

	// But refund should be positive
	assert.True(t, sellResult.Refund.Sign() > 0, "refund should be positive")
}

func TestMultipleCurves(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	completionThreshold := testCompletionThreshold()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100)

	// With 3 curves, each curve has 1/3 of the supply
	tokenAmount := mustFromDecimal("1000000000000000000")

	result1 := CalcBuyCost(maxTokenSupply, 1, virtualReserve, reserve, completionThreshold, tradingFee, tokenAmount)
	result3 := CalcBuyCost(maxTokenSupply, 3, virtualReserve, reserve, completionThreshold, tradingFee, tokenAmount)

	// With 3 curves, same token amount should cost MORE per token (smaller pool)
	assert.True(t, result3.Cost.Gt(result1.Cost),
		"cost with 3 curves %s should be > cost with 1 curve %s",
		result3.Cost.String(), result1.Cost.String())
}

func TestCalcBuyCost_ZeroCurves(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	completionThreshold := testCompletionThreshold()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100)
	tokenAmount := mustFromDecimal("1000000000000000000")

	result := CalcBuyCost(maxTokenSupply, 0, virtualReserve, reserve, completionThreshold, tradingFee, tokenAmount)

	assert.True(t, result.AvailableAmount.IsZero(), "available should be zero with 0 curves")
	assert.True(t, result.Cost.IsZero(), "cost should be zero with 0 curves")
	assert.True(t, result.Fee.IsZero(), "fee should be zero with 0 curves")
}

func TestCalcBuyTokenAmount_ZeroCurves(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100)
	baseSpend := mustFromDecimal("1000000000000000000")

	tokenAmount := CalcBuyTokenAmount(maxTokenSupply, 0, virtualReserve, reserve, tradingFee, baseSpend)

	assert.True(t, tokenAmount.IsZero(), "token amount should be zero with 0 curves")
}

func TestCalcSellRefund_ZeroCurves(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	reserve := mustFromDecimal("500000000000000000")
	tradingFee := uint16(100)
	tokenAmount := mustFromDecimal("1000000000000000000")

	result := CalcSellRefund(maxTokenSupply, 0, virtualReserve, reserve, tradingFee, tokenAmount)

	assert.True(t, result.TokenAmountIn.IsZero(), "tokenAmountIn should be zero with 0 curves")
	assert.True(t, result.Refund.IsZero(), "refund should be zero with 0 curves")
	assert.True(t, result.Fee.IsZero(), "fee should be zero with 0 curves")
}

func TestCalcBuyTokenAmount_LargeSpend(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	reserve := uint256.NewInt(0)
	tradingFee := uint16(100)

	// Spend a large amount (100 ETH)
	baseSpend := mustFromDecimal("100000000000000000000") // 100e18

	tokenAmount := CalcBuyTokenAmount(maxTokenSupply, 1, virtualReserve, reserve, tradingFee, baseSpend)

	assert.True(t, tokenAmount.Sign() > 0, "should get tokens for large spend")
	// Should not exceed maxTokenSupply / totalCurves
	initialTokenReserve := new(uint256.Int).Div(maxTokenSupply, uint256.NewInt(1))
	assert.True(t, !tokenAmount.Gt(initialTokenReserve),
		"token amount %s should not exceed initial token reserve %s",
		tokenAmount.String(), initialTokenReserve.String())
}

func TestCalcSellRefund_MultipleCurves(t *testing.T) {
	maxTokenSupply := testMaxTokenSupply()
	virtualReserve := testVirtualReserve()
	reserve := mustFromDecimal("500000000000000000") // 0.5 ETH
	tradingFee := uint16(100)
	tokenAmount := mustFromDecimal("1000000000000000000") // 1e18

	result1 := CalcSellRefund(maxTokenSupply, 1, virtualReserve, reserve, tradingFee, tokenAmount)
	result3 := CalcSellRefund(maxTokenSupply, 3, virtualReserve, reserve, tradingFee, tokenAmount)

	// With 3 curves (smaller pool), selling should yield a different refund
	assert.True(t, result1.Refund.Sign() > 0, "refund should be positive for 1 curve")
	assert.True(t, result3.Refund.Sign() > 0, "refund should be positive for 3 curves")
	// Refund from smaller pool should differ from larger pool
	assert.False(t, result1.Refund.Eq(result3.Refund),
		"refund with 1 curve should differ from 3 curves")
}
