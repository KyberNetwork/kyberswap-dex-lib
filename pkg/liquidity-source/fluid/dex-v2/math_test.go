package dexv2

import (
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// packExchangePrices mirrors the Liquidity exchangePricesAndConfig layout used by
// the upstream LiquidityCalcs tests (fluid-contracts liquidityCalcsCalcExchangePrices.t.sol).
// A ratio is stored as mode (bit 0: 1 = invert) | ratio << 1.
func packExchangePrices(rate, fee, utilization, threshold, lastTimestamp, supplyPrice, borrowPrice, supplyRatio, borrowRatio uint64) *big.Int {
	v := new(big.Int)
	put := func(value uint64, offset uint) { v.Or(v, new(big.Int).Lsh(new(big.Int).SetUint64(value), offset)) }
	put(rate, 0)
	put(fee, BITS_EXCHANGE_PRICES_FEE)
	put(utilization, BITS_EXCHANGE_PRICES_UTILIZATION)
	put(threshold, 44)
	put(lastTimestamp, BITS_EXCHANGE_PRICES_LAST_TIMESTAMP)
	put(supplyPrice, BITS_EXCHANGE_PRICES_SUPPLY_EXCHANGE_PRICE)
	put(borrowPrice, BITS_EXCHANGE_PRICES_BORROW_EXCHANGE_PRICE)
	put(supplyRatio, BITS_EXCHANGE_PRICES_SUPPLY_RATIO)
	put(borrowRatio, BITS_EXCHANGE_PRICES_BORROW_RATIO)
	return v
}

const (
	yearSeconds   = 365 * 24 * 60 * 60
	halfYear      = yearSeconds / 2
	precision1e12 = 1_000_000_000_000
)

func TestCalcExchangePricesRevertsOnZeroPrice(t *testing.T) {
	_, _, err := _calcExchangePrice(packExchangePrices(0, 0, 0, 0, 0, 0, precision1e12, 0, 0), 1)
	require.ErrorIs(t, err, ErrFluidLiquidityCalcsError)
	_, _, err = _calcExchangePrice(packExchangePrices(0, 0, 0, 0, 0, precision1e12, 0, 0, 0), 1)
	require.ErrorIs(t, err, ErrFluidLiquidityCalcsError)
}

func TestCalcExchangePricesNoGrowth(t *testing.T) {
	now := uint64(5000 * 86400)
	for name, cfg := range map[string]*big.Int{
		"zero borrow rate": packExchangePrices(0, 5000, 8000, 100, now-halfYear, precision1e12, precision1e12, 0, 0),
		"no time passed":   packExchangePrices(1000, 5000, 8000, 100, now, precision1e12, precision1e12, 0, 0),
	} {
		supply, borrow, err := _calcExchangePrice(cfg, now)
		require.NoError(t, err, name)
		require.Equal(t, int64(precision1e12), supply.Int64(), name)
		require.Equal(t, int64(precision1e12), borrow.Int64(), name)
	}
	_, _, err := _calcExchangePrice(packExchangePrices(1000, 5000, 8000, 100, 10, precision1e12, precision1e12, 0, 0), 9)
	require.ErrorIs(t, err, ErrFluidLiquidityCalcsError, "last update in the future")
}

// Upstream testLiquidityCalcs_calcExchangePrices: 10% borrow rate, 50% fee, supply 10 (+2 interest free),
// borrow 8 (+1 interest free), half a year -> borrow 1.05e12, supply 1.02e12; then another half year ->
// borrow 1.1025e12, supply ~1.041e12 (+-1e7).
func TestCalcExchangePricesUpstreamVector(t *testing.T) {
	now := uint64(2000 * 86400)
	// Amounts are 1e18-scaled, so the fixed-point products need 128 bits.
	div := func(num, den uint64, factor uint64) uint64 {
		r := new(big.Int).Mul(new(big.Int).SetUint64(num), new(big.Int).SetUint64(factor))
		return r.Div(r, new(big.Int).SetUint64(den)).Uint64()
	}
	ratio := func(smaller, bigger uint64) uint64 { return div(smaller, bigger, 10_000) << 1 } // mode 0
	util := func(borrow, supply, borrowFree, supplyFree uint64) uint64 {
		return div(borrow+borrowFree, supply+supplyFree, 10_000)
	}
	ether := uint64(1e18)
	supplyRaw, borrowRaw := 10*ether, 8*ether
	cfg := packExchangePrices(10*100, 5000, util(borrowRaw, supplyRaw, ether, 2*ether), 100, now-halfYear,
		precision1e12, precision1e12, ratio(2*ether, supplyRaw), ratio(ether, borrowRaw))
	supply, borrow, err := _calcExchangePrice(cfg, now)
	require.NoError(t, err)
	require.Equal(t, int64(1_050_000_000_000), borrow.Int64())
	require.Equal(t, int64(1_020_000_000_000), supply.Int64())

	supplyRaw = div(supplyRaw, precision1e12, supply.Uint64())
	borrowRaw = div(borrowRaw, precision1e12, borrow.Uint64())
	cfg = packExchangePrices(10*100, 5000, util(borrowRaw, supplyRaw, ether, 2*ether), 100, now-halfYear,
		supply.Uint64(), borrow.Uint64(), ratio(2*ether, supplyRaw), ratio(ether, borrowRaw))
	supply, borrow, err = _calcExchangePrice(cfg, now)
	require.NoError(t, err)
	require.Equal(t, int64(1_102_500_000_000), borrow.Int64())
	require.InDelta(t, 1_041_000_000_000, supply.Int64(), 1e7)
}

// Supply and borrow ratio modes (bit 0: 0 = interest free / with interest, 1 = inverse) select
// different branches of the supply-yield derivation. The expected values come from an
// independent integer re-implementation of the deployed LiquidityCalcs.calcExchangePrices
// (10% rate, 50% fee, 70% utilization, half a year, prices starting at 1e12).
func TestCalcExchangePricesRatioModes(t *testing.T) {
	now := uint64(2000 * 86400)
	for _, tc := range []struct {
		name                     string
		supplyRatio, borrowRatio uint64
		wantSupply               int64
	}{
		{"supply inverted, borrow inverted", 3000<<1 | 1, 2500<<1 | 1, 1_015_165_000_000},
		{"supply inverted, borrow direct", 3000<<1 | 1, 2500 << 1, 1_060_665_000_000},
		{"supply direct, borrow inverted", 3000 << 1, 2500<<1 | 1, 1_004_550_000_000},
		{"no interest free amounts", 0, 0, 1_017_500_000_000},
	} {
		cfg := packExchangePrices(1000, 5000, 7000, 100, now-halfYear, precision1e12, precision1e12, tc.supplyRatio, tc.borrowRatio)
		supply, borrow, err := _calcExchangePrice(cfg, now)
		require.NoError(t, err, tc.name)
		require.Equal(t, tc.wantSupply, supply.Int64(), tc.name)
		require.Equal(t, int64(1_050_000_000_000), borrow.Int64(), tc.name)
	}
	// A supply ratio of exactly 1 (inverted mode, ratio 0) means no supplier earns yield.
	cfg := packExchangePrices(1000, 5000, 7000, 100, now-halfYear, precision1e12, precision1e12, 1, 0)
	supply, borrow, err := _calcExchangePrice(cfg, now)
	require.NoError(t, err)
	require.Equal(t, int64(precision1e12), supply.Int64())
	require.Equal(t, int64(1_050_000_000_000), borrow.Int64())
}

func TestCalculateVarsDecimals(t *testing.T) {
	pack := func(d0, d1 uint64) *big.Int {
		v := new(big.Int)
		v.Or(v, new(big.Int).Lsh(new(big.Int).SetUint64(d0), BITS_DEX_V2_VARIABLES2_TOKEN_0_DECIMALS))
		return v.Or(v, new(big.Int).Lsh(new(big.Int).SetUint64(d1), BITS_DEX_V2_VARIABLES2_TOKEN_1_DECIMALS))
	}
	cfg := packExchangePrices(0, 0, 0, 0, 0, precision1e12, precision1e12, 0, 0)
	// 6 decimals scale up by 1e3, 15 is the storage code for 18 decimals and scales down by 1e9.
	c, err := _calculateVars(pack(6, 15), cfg, cfg, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1000), c.Token0NumeratorPrecision.Int64())
	require.Equal(t, int64(1), c.Token0DenominatorPrecision.Int64())
	require.Equal(t, int64(1), c.Token1NumeratorPrecision.Int64())
	require.Equal(t, int64(1_000_000_000), c.Token1DenominatorPrecision.Int64())

	zero := packExchangePrices(0, 0, 0, 0, 0, 0, precision1e12, 0, 0)
	_, err = _calculateVars(pack(6, 6), zero, cfg, 1)
	require.ErrorIs(t, err, ErrFluidLiquidityCalcsError)
	_, err = _calculateVars(pack(6, 6), cfg, zero, 1)
	require.ErrorIs(t, err, ErrFluidLiquidityCalcsError)
}

func TestAmountLimits(t *testing.T) {
	max128 := new(uint256.Int).SetOne()
	max128.Lsh(max128, 128).SubUint64(max128, 1)
	require.ErrorIs(t, _verifyAmountLimits(uint256.NewInt(9_999)), ErrAmountOutOfLimits)
	require.NoError(t, _verifyAmountLimits(uint256.NewInt(10_000)))
	require.NoError(t, _verifyAmountLimits(max128))
	require.ErrorIs(t, _verifyAmountLimits(new(uint256.Int).AddUint64(max128, 1)), ErrAmountOutOfLimits)

	max86 := new(uint256.Int).SetOne()
	max86.Lsh(max86, 86).SubUint64(max86, 1)
	require.ErrorIs(t, _verifyAdjustedAmountLimits(uint256.NewInt(9_999)), ErrAdjustedAmountOutOfLimits)
	require.NoError(t, _verifyAdjustedAmountLimits(uint256.NewInt(10_000)))
	require.NoError(t, _verifyAdjustedAmountLimits(max86))
	require.ErrorIs(t, _verifyAdjustedAmountLimits(new(uint256.Int).AddUint64(max86, 1)), ErrAdjustedAmountOutOfLimits)
}

func TestSqrtPriceChangeLimits(t *testing.T) {
	start := uint256.NewInt(1_000_000_000_000)
	at := func(numerator uint64) *uint256.Int { return uint256.NewInt(numerator) }
	// Allowed band is [5e-10, 0.2] of the start price, in either direction.
	require.NoError(t, _verifySqrtPriceX96ChangeLimits(start, at(1_200_000_000_000)))
	require.NoError(t, _verifySqrtPriceX96ChangeLimits(start, at(800_000_000_000)))
	require.ErrorIs(t, _verifySqrtPriceX96ChangeLimits(start, at(1_200_000_000_100)), ErrSqrtPriceChangeOutOfBounds)
	require.ErrorIs(t, _verifySqrtPriceX96ChangeLimits(start, at(799_999_999_900)), ErrSqrtPriceChangeOutOfBounds)
	require.NoError(t, _verifySqrtPriceX96ChangeLimits(start, at(1_000_000_000_500)))
	require.ErrorIs(t, _verifySqrtPriceX96ChangeLimits(start, at(1_000_000_000_499)), ErrSqrtPriceChangeOutOfBounds)
	require.ErrorIs(t, _verifySqrtPriceX96ChangeLimits(start, start), ErrSqrtPriceChangeOutOfBounds)
}

// The fixed-width conversions replace amountToAdjusted/adjustedToAmount in the quote path and must
// agree with the big.Int formulas for every precision and price the deployed code can produce.
func TestFixedWidthConversionMatchesBigInt(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for range 20_000 {
		decimals := int64(rng.IntN(19))
		num, den := _calculateNumeratorAndDenominatorPrecisions(decimals)
		supply := new(big.Int).SetUint64(rng.Uint64()>>uint(rng.IntN(24)) | 1_000_000_000_000)
		borrow := new(big.Int).SetUint64(rng.Uint64()>>uint(rng.IntN(24)) | 1_000_000_000_000)
		v, overflow := newTokenVars(num, den, supply, borrow)
		require.False(t, overflow)

		amount := new(big.Int).Rsh(new(big.Int).SetBytes(randomBytes(rng, 16)), uint(rng.IntN(100)))
		var amount256, got uint256.Int
		amount256.SetFromBig(amount)
		v.toAdjusted(&got, &amount256, &v.borrowPrice)
		require.Equal(t, amountToAdjusted(amount, num, den, borrow).String(), got.ToBig().String())

		adjusted := new(big.Int).Rsh(new(big.Int).SetBytes(randomBytes(rng, 11)), uint(rng.IntN(60))+2) // below 2^86
		var adjusted256 uint256.Int
		adjusted256.SetFromBig(adjusted)
		v.toAmount(&got, &adjusted256, &v.supplyPrice)
		require.Equal(t, adjustedToAmount(adjusted, num, den, supply).String(), got.ToBig().String())
	}
}

func randomBytes(rng *rand.Rand, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rng.Uint32())
	}
	return b
}

func TestTokenVarsPriceSelection(t *testing.T) {
	v, overflow := newTokenVars(big.NewInt(1), big.NewInt(1), big.NewInt(7), big.NewInt(9))
	require.False(t, overflow)
	require.Equal(t, uint64(7), v.price(true).Uint64())
	require.Equal(t, uint64(9), v.price(false).Uint64())
	tooBig := new(big.Int).Lsh(big.NewInt(1), 300)
	_, overflow = newTokenVars(tooBig, big.NewInt(1), big.NewInt(1), big.NewInt(1))
	require.True(t, overflow)
}
