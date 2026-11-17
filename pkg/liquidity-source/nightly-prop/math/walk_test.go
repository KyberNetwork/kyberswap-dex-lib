package math

// The exact-in vectors of evm_contracts/test/NightlyPoolQuotes.t.sol, as the Rust port of the
// Nightly trading engine (crates/nightly-pool/src/pool.rs) copies them, on the same fixture:
// 20,000 USDC in level -1 (bid 1,998 to 2,000), 10 ETH in level 0 (ask 2,000 to 2,002), and 20 ETH
// in level 1, with zero extra spreads, the position at the lower edge of level 0, and ETH at $2,000
// with zero oracle spreads.

import (
	"math"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

const (
	timeMs           = 1_000_000_000
	fixtureBlockTime = timeMs / 1_000
	maxPriceAgeMs    = 2_500
	fixtureMid       = 200_000_000_000
)

type fixture struct {
	pool           Pool
	oracle         OraclePrice
	blockTimestamp uint64
}

// levelConfigs is _levels(): IDs -1, 0, and 1 without extra spreads.
func levelConfigs() []PriceLevelConfig {
	return []PriceLevelConfig{config(-1_000, 0, 0, 0), config(0, 1_000, 0, 0), config(1_000, 2_000, 0, 0)}
}

// fourLevels is four adjoining levels from -2 to 1.
func fourLevels() []PriceLevelConfig {
	return append([]PriceLevelConfig{config(-2_000, -1_000, 0, 0)}, levelConfigs()...)
}

// deploy is _deploy(lowest, initial, configs): no liquidity, and the position at the lower edge of
// initial.
func deploy(baseDecimals, quoteDecimals uint8, lowest, initial int16, configs []PriceLevelConfig) *fixture {
	levels := make([]Level, len(configs))
	for i, c := range configs {
		levels[i].Config = c
	}
	return &fixture{
		pool: Pool{
			BaseDecimals:  baseDecimals,
			QuoteDecimals: quoteDecimals,
			MaxPriceAgeMs: maxPriceAgeMs,
			LowestLevelID: lowest,
			Levels:        levels,
			Position:      Position{LevelID: initial, Fraction: Lower},
		},
		oracle:         OraclePrice{MidPrice: fixtureMid, TimestampMs: timeMs},
		blockTimestamp: fixtureBlockTime,
	}
}

// funded is _deploy(-1, 0, configs) and _fund() with ETH (18) and USDC (6).
func funded(t testing.TB, configs []PriceLevelConfig) *fixture {
	return fundedWithDecimals(t, 18, 6, configs)
}

// fundedWithDecimals is _deploy(-1, 0, configs) and _fund(): 20,000 quote tokens in level -1, 10
// base tokens in level 0, and 20 base tokens in level 1.
func fundedWithDecimals(t testing.TB, baseDecimals, quoteDecimals uint8, configs []PriceLevelConfig) *fixture {
	f := deploy(baseDecimals, quoteDecimals, -1, 0, configs)
	f.seedLevel(-1, u(0), scaled(t, 20_000, quoteDecimals))
	f.seedLevel(0, scaled(t, 10, baseDecimals), u(0))
	f.seedLevel(1, scaled(t, 20, baseDecimals), u(0))
	return f
}

func (f *fixture) seedLevel(levelID int16, baseAmount, quoteAmount *uint256.Int) {
	f.pool.Levels[int(levelID)-int(f.pool.LowestLevelID)].Liquidity = Liquidity{
		BaseAmount:  *baseAmount,
		QuoteAmount: *quoteAmount,
	}
}

func (f *fixture) seedPosition(levelID int16, position Fraction) {
	f.pool.Position = Position{LevelID: levelID, Fraction: position}
}

func (f *fixture) buy(quoteIn *uint256.Int) (Quote, error) {
	swap, err := f.pool.BuyWithExactQuote(quoteIn, f.oracle, f.blockTimestamp)
	return swap.Quote, err
}

func (f *fixture) sell(baseIn *uint256.Int) (Quote, error) {
	swap, err := f.pool.SellExactBase(baseIn, f.oracle, f.blockTimestamp)
	return swap.Quote, err
}

func ether(t testing.TB, amount uint64) *uint256.Int {
	return scaled(t, amount, 18)
}

func usdc(t testing.TB, amount uint64) *uint256.Int {
	return scaled(t, amount, 6)
}

func maxPositionOver(t *testing.T, divisor uint64) Fraction {
	return fraction(t, new(uint256.Int).Div(maxRaw, u(divisor)))
}

func requireWalk(t *testing.T, q Quote, err error, amountIn, amountOut *uint256.Int, levelID int16, end Fraction) {
	t.Helper()
	require.NoError(t, err)
	require.Equal(t, amountIn.Dec(), q.AmountIn.Dec(), "amount in")
	require.Equal(t, amountOut.Dec(), q.AmountOut.Dec(), "amount out")
	require.Equal(t, levelID, q.EndPosition.LevelID, "end level")
	require.Equal(t, end.Raw().Dec(), q.EndPosition.Fraction.Raw().Dec(), "end fraction")
}

func TestPool_BuyWithinLevel(t *testing.T) {
	// 7,000 USDC from the lower edge of level 0, whose ask range is 2,000 to 2,002.
	q, err := funded(t, levelConfigs()).buy(usdc(t, 7_000))
	requireWalk(t, q, err, usdc(t, 7_000), dec(t, "3499387714231273582"), 0,
		fraction(t, dec(t, "7097601498202505053776590463358")))
}

func TestPool_BuyMatchesExactBaseOut(t *testing.T) {
	// 10,002.5 USDC is the exact-base-out cost of 5 ETH.
	q, err := funded(t, levelConfigs()).buy(u(10_002_500_000))
	requireWalk(t, q, err, u(10_002_500_000), ether(t, 5), 0, maxPositionOver(t, 2))
}

func TestPool_BuyFromMixedBalances(t *testing.T) {
	f := funded(t, levelConfigs())
	f.seedLevel(0, ether(t, 5), u(10_002_500_000))
	f.seedPosition(0, maxPositionOver(t, 2))
	// The exact-base-out cost of 2.5 ETH. The start price and values round up, so the output is
	// slightly less than 2.5 ETH.
	q, err := f.buy(u(5_003_125_000))
	requireWalk(t, q, err, u(5_003_125_000), dec(t, "2499999999875078076"), 0,
		fraction(t, dec(t, "15211807202485381054856014054993")))
}

func TestPool_BuyExactDrainAndFullDepth(t *testing.T) {
	f := funded(t, levelConfigs())
	q, err := f.buy(usdc(t, 20_010))
	requireWalk(t, q, err, usdc(t, 20_010), ether(t, 10), 0, Upper)
	q, err = f.buy(usdc(t, 60_070))
	requireWalk(t, q, err, usdc(t, 60_070), ether(t, 30), 1, Upper)
}

func TestPool_BuyDrainRoundsOnce(t *testing.T) {
	// 0.0001 ETH from 2,000 to 2,002 costs 0.2001 whole quote tokens. Rounding the start and extra
	// amounts separately would charge 2.
	f := deploy(18, 0, -1, 0, levelConfigs())
	base := u(100_000_000_000_000)
	f.seedLevel(0, base, u(0))
	q, err := f.buy(u(1))
	requireWalk(t, q, err, u(1), base, 0, Upper)
}

func TestPool_BuyCrossesIntoSecondLevel(t *testing.T) {
	// 20,010 USDC buys all 10 ETH of level 0; the other 4,990 USDC buys about 2.49 ETH of level 1.
	q, err := funded(t, levelConfigs()).buy(usdc(t, 25_000))
	requireWalk(t, q, err, usdc(t, 25_000), dec(t, "12492352352132474569"), 1,
		fraction(t, dec(t, "2527545564128776602018949236608")))
}

func TestPool_BuyAppliesEachLevelsSpread(t *testing.T) {
	configs := levelConfigs()
	configs[2].AskExtraSpread = 1_000
	q, err := funded(t, configs).buy(u(30_031_261_250))
	requireWalk(t, q, err, u(30_031_261_250), ether(t, 15), 1, maxPositionOver(t, 4))
}

func TestPool_BuySkipsEmptyLevel(t *testing.T) {
	f := funded(t, levelConfigs())
	f.seedLevel(0, u(0), u(0))
	q, err := f.buy(u(10_011_250_000))
	requireWalk(t, q, err, u(10_011_250_000), ether(t, 5), 1, maxPositionOver(t, 4))
}

func TestPool_BuyNativeRounding(t *testing.T) {
	// One native USDC unit is worth 500,000,000 wei at 2,000; the base output rounds down.
	q, err := funded(t, levelConfigs()).buy(u(1))
	require.NoError(t, err)
	require.Equal(t, u(499_999_999), &q.AmountOut)
}

func TestPool_BuyTokenDecimalCombinations(t *testing.T) {
	for baseDecimals := uint8(0); baseDecimals <= 18; baseDecimals++ {
		for quoteDecimals := uint8(0); quoteDecimals <= 18; quoteDecimals++ {
			f := fundedWithDecimals(t, baseDecimals, quoteDecimals, levelConfigs())
			// At least the exact-base-out cost of 5 base tokens.
			quoteIn, err := CeilDiv(scaled(t, 20_005, quoteDecimals), u(2))
			require.NoError(t, err)
			q, err := f.buy(&quoteIn)
			require.NoError(t, err, "decimals %d/%d", baseDecimals, quoteDecimals)
			require.False(t, q.AmountOut.Lt(scaled(t, 5, baseDecimals)), "decimals %d/%d", baseDecimals, quoteDecimals)
			require.Equal(t, int16(0), q.EndPosition.LevelID, "decimals %d/%d", baseDecimals, quoteDecimals)
		}
	}
}

func TestPool_BuyError(t *testing.T) {
	_, err := funded(t, levelConfigs()).buy(u(0))
	require.ErrorIs(t, err, ErrZeroSwapAmount)

	// One whole base token costs about 2,000 USDC, so one native USDC unit buys less than one base
	// unit.
	_, err = fundedWithDecimals(t, 0, 6, levelConfigs()).buy(u(1))
	require.ErrorIs(t, err, ErrAmountTooSmall)

	// All 30 ETH cost 60,070 USDC; one more native unit has no base to buy.
	_, err = funded(t, levelConfigs()).buy(new(uint256.Int).AddUint64(usdc(t, 60_070), 1))
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
	require.ErrorContains(t, err, ": 1 left")

	f := funded(t, levelConfigs())
	f.seedLevel(0, ether(t, 10), maxUint256)
	_, err = f.buy(usdc(t, 1_000))
	require.ErrorIs(t, err, ErrLiquidityOverflow)
	require.ErrorContains(t, err, "level 0")

	// The level's base is worth more than 2^128 native USDC units, so squaring quoteAtStartPrice
	// overflows.
	f = funded(t, levelConfigs())
	f.seedLevel(0, maxUint256, u(0))
	_, err = f.buy(usdc(t, 1_000))
	require.ErrorIs(t, err, ErrOverflow)

	// The highest level ID is the largest int16, and the walk stops there.
	_, err = deploy(18, 6, math.MaxInt16, math.MaxInt16, []PriceLevelConfig{config(0, 1_000, 0, 0)}).buy(usdc(t, 1))
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// TestPool_PriceAge: a price is fresh up to its time plus the maximum age, and the zero-amount check
// comes before the age check.
func TestPool_PriceAge(t *testing.T) {
	f := funded(t, levelConfigs())
	f.blockTimestamp = fixtureBlockTime + 2
	_, err := f.buy(usdc(t, 1_000))
	require.NoError(t, err)
	_, err = f.sell(ether(t, 1))
	require.NoError(t, err)

	f.blockTimestamp = fixtureBlockTime + 3
	_, err = f.buy(usdc(t, 1_000))
	require.ErrorIs(t, err, ErrStaleOraclePrice)
	_, err = f.sell(ether(t, 1))
	require.ErrorIs(t, err, ErrStaleOraclePrice)
	_, err = f.buy(u(0))
	require.ErrorIs(t, err, ErrZeroSwapAmount)
}

func TestPool_SellWithinLowerLevel(t *testing.T) {
	// 5 ETH from the top of level -1, whose bid range is 1,998 to 2,000.
	q, err := funded(t, levelConfigs()).sell(ether(t, 5))
	requireWalk(t, q, err, ether(t, 5), u(9_997_501_249), -1,
		fraction(t, dec(t, "10146275404226748129579612455829")))
}

func TestPool_SellExactRefill(t *testing.T) {
	// 20,000 USDC / 1,999, the mean of 2,000 and 1,998, rounded up in wei.
	baseIn := dec(t, "10005002501250625313")
	q, err := funded(t, levelConfigs()).sell(baseIn)
	requireWalk(t, q, err, baseIn, usdc(t, 20_000), -1, Lower)
}

func TestPool_SellCrossesIntoLowerLevel(t *testing.T) {
	f := funded(t, levelConfigs())
	f.seedLevel(0, ether(t, 5), u(10_002_500_000))
	f.seedPosition(0, maxPositionOver(t, 2))
	// 5 ETH buy back level 0's 10,002.5 USDC; the last 1 ETH sells into level -1.
	q, err := f.sell(ether(t, 6))
	requireWalk(t, q, err, ether(t, 6), u(12_002_400_049), -1,
		fraction(t, dec(t, "18255182763766685967100950359862")))
}

func TestPool_SellAppliesEachLevelsSpread(t *testing.T) {
	configs := levelConfigs()
	configs[0].BidExtraSpread = 1_000
	f := funded(t, configs)
	// Oracle spread code 80 is 203 millionths.
	f.oracle = OraclePrice{MidPrice: fixtureMid, BidSpread: 203, TimestampMs: timeMs + 1}
	f.seedLevel(0, ether(t, 5), u(10_002_500_000))
	f.seedPosition(0, maxPositionOver(t, 2))
	q, err := f.sell(ether(t, 6))
	requireWalk(t, q, err, ether(t, 6), u(11_997_966_523), -1,
		fraction(t, dec(t, "18259677094844523613557270880618")))
}

func TestPool_SellMaximumSpreads(t *testing.T) {
	configs := levelConfigs()
	configs[0].BidExtraSpread = math.MaxUint16
	f := funded(t, configs)
	// Oracle spread code 255 is 10,000 millionths.
	f.oracle = OraclePrice{MidPrice: fixtureMid, BidSpread: 10_000, TimestampMs: timeMs + 1}
	q, err := f.sell(ether(t, 1))
	require.NoError(t, err)
	require.Equal(t, u(1_848_844_579), &q.AmountOut)
}

func TestPool_SellSkipsEmptyLevels(t *testing.T) {
	f := deploy(18, 6, -2, 0, fourLevels())
	f.seedLevel(-2, u(0), usdc(t, 20_000))
	q, err := f.sell(ether(t, 1))
	requireWalk(t, q, err, ether(t, 1), u(1_997_900_149), -2,
		fraction(t, dec(t, "18257211004727051132116118245106")))
}

func TestPool_SellWideQuoteBalance(t *testing.T) {
	// The fraction moves down by at least the exact distance, so the price is just below 2,000.
	f := funded(t, levelConfigs())
	f.seedLevel(-1, u(0), new(uint256.Int).SubUint64(two128, 1))
	q, err := f.sell(ether(t, 1))
	requireWalk(t, q, err, ether(t, 1), u(1_999_999_999), -1,
		fraction(t, dec(t, "20282409603651670423947251285895")))
}

func TestPool_SellTokenDecimalCombinations(t *testing.T) {
	for baseDecimals := uint8(0); baseDecimals <= 18; baseDecimals++ {
		for quoteDecimals := uint8(0); quoteDecimals <= 18; quoteDecimals++ {
			q, err := fundedWithDecimals(t, baseDecimals, quoteDecimals, levelConfigs()).sell(scaled(t, 1, baseDecimals))
			pair := []any{"decimals %d/%d", baseDecimals, quoteDecimals}
			require.NoError(t, err, pair...)
			require.False(t, q.AmountOut.Lt(scaled(t, 1_998, quoteDecimals)), pair...)
			require.False(t, scaled(t, 2_000, quoteDecimals).Lt(&q.AmountOut), pair...)
			require.Equal(t, int16(-1), q.EndPosition.LevelID, pair...)
		}
	}
}

func TestPool_SellError(t *testing.T) {
	_, err := funded(t, levelConfigs()).sell(u(0))
	require.ErrorIs(t, err, ErrZeroSwapAmount)

	// One wei is worth 0.000000002 native USDC units.
	_, err = funded(t, levelConfigs()).sell(u(1))
	require.ErrorIs(t, err, ErrAmountTooSmall)

	// Level -1 buys back 10.005... ETH; no level is below it.
	_, err = funded(t, levelConfigs()).sell(ether(t, 11))
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
	left := new(uint256.Int).Sub(ether(t, 11), dec(t, "10005002501250625313"))
	require.ErrorContains(t, err, ": "+left.Dec()+" left")

	// The lowest level ID is the smallest int16, and the walk stops there.
	_, err = deploy(18, 6, math.MinInt16, math.MinInt16, []PriceLevelConfig{config(0, 1_000, 0, 0)}).sell(ether(t, 1))
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
	require.ErrorContains(t, err, ": "+ether(t, 1).Dec()+" left")

	f := funded(t, levelConfigs())
	f.seedLevel(0, maxUint256, usdc(t, 20_000))
	_, err = f.sell(ether(t, 1))
	require.ErrorIs(t, err, ErrLiquidityOverflow)
	require.ErrorContains(t, err, "level 0")

	// A maximum quote balance buys back more base than uint256 can hold.
	f = funded(t, levelConfigs())
	f.seedLevel(-1, u(0), maxUint256)
	_, err = f.sell(ether(t, 1))
	require.ErrorIs(t, err, ErrOverflow)
}

func TestPool_InvalidPosition(t *testing.T) {
	f := funded(t, levelConfigs())
	f.seedPosition(5, Lower)
	_, err := f.buy(usdc(t, 1))
	require.ErrorIs(t, err, ErrInvalidPriceLevel)
}

// TestPool_ApplyBuy: an applied buy writes what swapExactIn writes: level 0 sells all of its base
// and keeps the 20,010 USDC paid for it, level 1 keeps the rest of the quote, and the position
// moves to the end of the walk.
func TestPool_ApplyBuy(t *testing.T) {
	f := funded(t, levelConfigs())
	swap, err := f.pool.BuyWithExactQuote(usdc(t, 25_000), f.oracle, f.blockTimestamp)
	require.NoError(t, err)
	require.NoError(t, f.pool.Apply(swap))
	require.Equal(t, swap.Quote.EndPosition, f.pool.Position)
	require.Equal(t, Liquidity{QuoteAmount: *usdc(t, 20_000)}, f.pool.Levels[0].Liquidity)
	require.Equal(t, Liquidity{QuoteAmount: *usdc(t, 20_010)}, f.pool.Levels[1].Liquidity)
	// Level 1 sold 12.49... - 10 ETH for the other 4,990 USDC.
	levelOneBase := new(uint256.Int).Sub(ether(t, 30), &swap.Quote.AmountOut)
	require.Equal(t, Liquidity{BaseAmount: *levelOneBase, QuoteAmount: *usdc(t, 4_990)}, f.pool.Levels[2].Liquidity)
}

// TestPool_ApplySell: an applied sell refills level -1 with the base and moves the position.
func TestPool_ApplySell(t *testing.T) {
	f := funded(t, levelConfigs())
	swap, err := f.pool.SellExactBase(ether(t, 5), f.oracle, f.blockTimestamp)
	require.NoError(t, err)
	require.NoError(t, f.pool.Apply(swap))
	require.Equal(t, swap.Quote.EndPosition, f.pool.Position)
	levelMinusOneQuote := new(uint256.Int).Sub(usdc(t, 20_000), &swap.Quote.AmountOut)
	require.Equal(t, Liquidity{BaseAmount: *ether(t, 5), QuoteAmount: *levelMinusOneQuote}, f.pool.Levels[0].Liquidity)
	require.Equal(t, Liquidity{BaseAmount: *ether(t, 10)}, f.pool.Levels[1].Liquidity)
}

// TestPool_ApplyRoundTrip: a buy applied, then a sell of the bought base applied, quotes again
// from the moved state, as two swaps on chain.
func TestPool_ApplyRoundTrip(t *testing.T) {
	f := funded(t, levelConfigs())
	buy, err := f.pool.BuyWithExactQuote(usdc(t, 7_000), f.oracle, f.blockTimestamp)
	require.NoError(t, err)
	require.NoError(t, f.pool.Apply(buy))
	sell, err := f.pool.SellExactBase(&buy.Quote.AmountOut, f.oracle, f.blockTimestamp)
	require.NoError(t, err)
	// The sell refills level 0 at bid prices below the ask, so it pays less than the buy cost.
	require.True(t, sell.Quote.AmountOut.Lt(usdc(t, 7_000)))
	require.Equal(t, int16(0), sell.Quote.EndPosition.LevelID)
	require.NoError(t, f.pool.Apply(sell))
	require.Equal(t, sell.Quote.EndPosition, f.pool.Position)
}

// TestPool_WalkWritesNothing: a walk does not change the pool, and Apply with a level outside the
// pool writes nothing.
func TestPool_WalkWritesNothing(t *testing.T) {
	f := funded(t, levelConfigs())
	before := f.pool
	before.Levels = append([]Level(nil), f.pool.Levels...)

	swap, err := f.pool.BuyWithExactQuote(usdc(t, 25_000), f.oracle, f.blockTimestamp)
	require.NoError(t, err)
	require.Equal(t, before, f.pool)

	swap.Writes = append(swap.Writes, LevelWrite{LevelID: 5})
	require.ErrorIs(t, f.pool.Apply(swap), ErrInvalidPriceLevel)
	require.Equal(t, before, f.pool)
}
