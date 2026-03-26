package canonic

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestCountDigits(t *testing.T) {
	require.Equal(t, uint32(1), countDigits(uint256.NewInt(1)))
	require.Equal(t, uint32(1), countDigits(uint256.NewInt(9)))
	require.Equal(t, uint32(2), countDigits(uint256.NewInt(10)))
	require.Equal(t, uint32(2), countDigits(uint256.NewInt(99)))
	require.Equal(t, uint32(3), countDigits(uint256.NewInt(100)))
	require.Equal(t, uint32(7), countDigits(uint256.NewInt(1234567)))
	require.Equal(t, uint32(19), countDigits(uint256.MustFromDecimal("1000000000000000000")))
}

func TestRoundPriceQ(t *testing.T) {
	require.Equal(t, uint256.NewInt(123457000), roundPriceQ(uint256.NewInt(123456789), 6))
	require.Equal(t, uint256.NewInt(123456000), roundPriceQ(uint256.NewInt(123456499), 6))
	require.Equal(t, uint256.NewInt(123457000), roundPriceQ(uint256.NewInt(123456500), 6))
	require.Equal(t, uint256.NewInt(100000), roundPriceQ(uint256.NewInt(100000), 6))
	require.Equal(t, uint256.NewInt(12345), roundPriceQ(uint256.NewInt(12345), 6))
	require.Equal(t, uint256.NewInt(0), roundPriceQ(uint256.NewInt(0), 6))
}

func TestCalcAskRungPrice(t *testing.T) {
	midPrice := uint256.MustFromDecimal("3000000000")
	midPrec := uint256.MustFromDecimal("1000000000")
	quoteScale := uint256.MustFromDecimal("1000000")

	priceQ := calcAskRungPrice(midPrice, midPrec, 10, quoteScale)
	require.True(t, priceQ.Gt(uint256.NewInt(0)))

	expectedRaw := uint256.NewInt(3000300)
	require.True(t, priceQ.Cmp(expectedRaw) >= 0, "ask price should be >= mid * (100000+10)/100000 * quoteScale/midPrec")
}

func TestCalcBidRungPrice(t *testing.T) {
	midPrice := uint256.MustFromDecimal("3000000000")
	midPrec := uint256.MustFromDecimal("1000000000")
	quoteScale := uint256.MustFromDecimal("1000000")

	priceQ := calcBidRungPrice(midPrice, midPrec, 10, quoteScale)
	require.True(t, priceQ.Gt(uint256.NewInt(0)))

	midQ := uint256.NewInt(3000000)
	require.True(t, priceQ.Lt(midQ), "bid price should be < mid price")
}

func TestCalcSellBaseTargetIn(t *testing.T) {
	midPrice := uint256.MustFromDecimal("2000000000000000000")
	midPrec := uint256.MustFromDecimal("1000000000000000000")
	takerFee := uint256.NewInt(300)
	baseScale := uint256.MustFromDecimal("1000000000000000000")
	quoteScale := uint256.MustFromDecimal("1000000")

	bidBps := []uint16{10, 20, 30}
	bidVols := []*uint256.Int{
		uint256.MustFromDecimal("4000000"),
		uint256.MustFromDecimal("4000000"),
		uint256.MustFromDecimal("4000000"),
	}

	baseIn := uint256.MustFromDecimal("1000000000000000000")

	quoteOut, fee, used := calcSellBaseTargetIn(
		baseIn, midPrice, midPrec, takerFee, baseScale, quoteScale,
		bidBps, bidVols,
	)

	require.True(t, quoteOut.Gt(uint256.NewInt(0)), "should get some quote out")
	require.True(t, fee.Gt(uint256.NewInt(0)), "should have fee")
	require.True(t, used.Gt(uint256.NewInt(0)), "should use some base")

	t.Logf("sell 1 ETH: quoteOut=%s fee=%s baseUsed=%s", quoteOut, fee, used)
}

func TestCalcBuyBaseTargetIn(t *testing.T) {
	midPrice := uint256.MustFromDecimal("2000000000000000000")
	midPrec := uint256.MustFromDecimal("1000000000000000000")
	takerFee := uint256.NewInt(300)
	baseScale := uint256.MustFromDecimal("1000000000000000000")
	quoteScale := uint256.MustFromDecimal("1000000")

	askBps := []uint16{10, 20, 30}
	askVols := []*uint256.Int{
		uint256.MustFromDecimal("500000000000000000"),
		uint256.MustFromDecimal("500000000000000000"),
		uint256.MustFromDecimal("500000000000000000"),
	}

	quoteIn := uint256.MustFromDecimal("2000000")

	baseOut, fee, used := calcBuyBaseTargetIn(
		quoteIn, midPrice, midPrec, takerFee, baseScale, quoteScale,
		askBps, askVols,
	)

	require.True(t, baseOut.Gt(uint256.NewInt(0)), "should get some base out")
	require.True(t, fee.Gt(uint256.NewInt(0)), "should have fee")
	require.True(t, used.Gt(uint256.NewInt(0)), "should use some quote")

	t.Logf("buy with 2 USDC: baseOut=%s fee=%s quoteUsed=%s", baseOut, fee, used)
}

func TestCalcSellBaseTargetIn_ZeroLiquidity(t *testing.T) {
	midPrice := uint256.MustFromDecimal("2000000000000000000")
	midPrec := uint256.MustFromDecimal("1000000000000000000")
	takerFee := uint256.NewInt(300)
	baseScale := uint256.MustFromDecimal("1000000000000000000")
	quoteScale := uint256.MustFromDecimal("1000000")

	bidBps := []uint16{10}
	bidVols := []*uint256.Int{uint256.NewInt(0)}

	quoteOut, _, _ := calcSellBaseTargetIn(
		uint256.MustFromDecimal("1000000000000000000"),
		midPrice, midPrec, takerFee, baseScale, quoteScale,
		bidBps, bidVols,
	)
	require.True(t, quoteOut.IsZero(), "should get 0 with no liquidity")
}

func TestCalcSellBaseAmountIn(t *testing.T) {
	midPrice := uint256.MustFromDecimal("2000000000000000000")
	midPrec := uint256.MustFromDecimal("1000000000000000000")
	takerFee := uint256.NewInt(300)
	baseScale := uint256.MustFromDecimal("1000000000000000000")
	quoteScale := uint256.MustFromDecimal("1000000")

	bidBps := []uint16{10, 20, 30}
	bidVols := []*uint256.Int{
		uint256.MustFromDecimal("4000000"),
		uint256.MustFromDecimal("4000000"),
		uint256.MustFromDecimal("4000000"),
	}

	quoteDesired := uint256.MustFromDecimal("1000000")

	baseNeeded, fee := calcSellBaseAmountIn(
		quoteDesired, midPrice, midPrec, takerFee, baseScale, quoteScale,
		bidBps, bidVols,
	)

	require.NotNil(t, baseNeeded)
	require.True(t, baseNeeded.Gt(uint256.NewInt(0)))
	require.True(t, fee.Gt(uint256.NewInt(0)))

	t.Logf("get 1 USDC out: baseNeeded=%s fee=%s", baseNeeded, fee)
}

func TestCalcBuyBaseAmountIn(t *testing.T) {
	midPrice := uint256.MustFromDecimal("2000000000000000000")
	midPrec := uint256.MustFromDecimal("1000000000000000000")
	takerFee := uint256.NewInt(300)
	baseScale := uint256.MustFromDecimal("1000000000000000000")
	quoteScale := uint256.MustFromDecimal("1000000")

	askBps := []uint16{10, 20, 30}
	askVols := []*uint256.Int{
		uint256.MustFromDecimal("500000000000000000"),
		uint256.MustFromDecimal("500000000000000000"),
		uint256.MustFromDecimal("500000000000000000"),
	}

	baseDesired := uint256.MustFromDecimal("100000000000000000")

	quoteNeeded, fee := calcBuyBaseAmountIn(
		baseDesired, midPrice, midPrec, takerFee, baseScale, quoteScale,
		askBps, askVols,
	)

	require.NotNil(t, quoteNeeded)
	require.True(t, quoteNeeded.Gt(uint256.NewInt(0)))
	require.True(t, fee.Gt(uint256.NewInt(0)))

	t.Logf("get 0.1 ETH: quoteNeeded=%s fee=%s", quoteNeeded, fee)
}
