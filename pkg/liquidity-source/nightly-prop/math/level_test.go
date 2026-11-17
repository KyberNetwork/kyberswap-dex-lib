package math

import (
	"math/rand/v2"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func config(lower, upper SignedSpread, bidExtra, askExtra Spread) PriceLevelConfig {
	return PriceLevelConfig{LowerOffset: lower, UpperOffset: upper, BidExtraSpread: bidExtra, AskExtraSpread: askExtra}
}

func mid(t testing.TB, raw uint64, baseDecimals, quoteDecimals uint8) Price {
	t.Helper()
	price, err := PriceFromOracle(raw, baseDecimals, quoteDecimals)
	require.NoError(t, err)
	return price
}

func askRange(t testing.TB, c PriceLevelConfig, m Price, oracleAskSpread Spread) PriceRange {
	t.Helper()
	r, err := AskRange(c, m, oracleAskSpread)
	require.NoError(t, err)
	return r
}

func bidRange(t testing.TB, c PriceLevelConfig, m Price, oracleBidSpread Spread) PriceRange {
	t.Helper()
	r, err := BidRange(c, m, oracleBidSpread)
	require.NoError(t, err)
	return r
}

func dec(t testing.TB, s string) *uint256.Int {
	t.Helper()
	z, err := uint256.FromDecimal(s)
	require.NoError(t, err)
	return z
}

func requireQuote(t *testing.T, want uint64, price Price, baseAmount *uint256.Int) {
	t.Helper()
	quote, err := price.QuoteFor(baseAmount, Ceil)
	require.NoError(t, err)
	require.Equal(t, u(want), quote)
}

func requireRange(t *testing.T, lower, upper, denominator string, r PriceRange) {
	t.Helper()
	require.Equal(t, lower, r.Lower.Numerator.Dec())
	require.Equal(t, upper, r.Upper.Numerator.Dec())
	require.Equal(t, denominator, r.Lower.Denominator.Dec())
	require.Equal(t, denominator, r.Upper.Denominator.Dec())
}

// TestAskBidRange: the vectors of NightlyPoolMath.t.sol. ETH is $2,000, the edges are -0.1% and
// +0.1%, and both total spreads are 0.03%, so 1 ETH costs 1,998.5994 or 2,002.6006 USDC to buy
// and pays 1,997.4006 or 2,001.3994 USDC to sell.
func TestAskBidRange(t *testing.T) {
	c := config(-1_000, 1_000, 200, 100)
	ethUsdc := mid(t, 2_000e8, 18, 6)
	oneEther := scaled(t, 1, 18)
	ask := askRange(t, c, ethUsdc, 200)
	requireQuote(t, 1_998_599_400, ask.Lower, oneEther)
	requireQuote(t, 2_002_600_600, ask.Upper, oneEther)
	bid := bidRange(t, c, ethUsdc, 100)
	requireQuote(t, 1_997_400_600, bid.Lower, oneEther)
	requireQuote(t, 2_001_399_400, bid.Upper, oneEther)
}

// TestAskBidRange_Solidity: ranges that NightlyPoolMath computed in Solidity, for 6/6 decimals
// with offsets, level spreads, and an oracle spread, and for a price above one native unit.
func TestAskBidRange_Solidity(t *testing.T) {
	c := config(-10, 10, 2, 3)
	usdcUsdt := mid(t, 1e8, 6, 6)
	requireRange(t, "999993999960000000000000000000000000", "1000014000040000000000000000000000000",
		"1000000000000000000000000000000000000", askRange(t, c, usdcUsdt, 1))
	requireRange(t, "999987000030000000000000000000000000", "1000006999970000000000000000000000000",
		"1000000000000000000000000000000000000", bidRange(t, c, usdcUsdt, 1))
	requireRange(t, "1998000000000000000000000000000000000000000000000000",
		"2000000000000000000000000000000000000000000000000000", "1000000000000000000000000000000000000",
		bidRange(t, config(-1_000, 0, 0, 0), mid(t, 2_000e8, 6, 18), 0))
}

func TestAskBidRange_Error(t *testing.T) {
	// The lower edge comes first, so it reports its inexact adjustment.
	m := pn(1)
	c := config(1, 2, 0, 0)
	_, err := AskRange(c, m, 0)
	require.ErrorIs(t, err, ErrInexactAdjustment)
	_, err = BidRange(c, m, 0)
	require.ErrorIs(t, err, ErrInexactAdjustment)

	ethUsdc := mid(t, 2_000e8, 18, 6)
	_, err = AskRange(config(-1_000, 1_000, 200, 1), ethUsdc, Spread(^uint32(0)))
	require.ErrorIs(t, err, ErrOverflow)
	// 999,801 + 200 is more than a whole price.
	_, err = BidRange(config(-1_000, 1_000, 200, 100), ethUsdc, 999_801)
	require.ErrorIs(t, err, ErrOverflow)
}

// TestLevelSteps_Solidity: steps that NightlyPoolMath computed in Solidity. The examples are the
// ones in NightlyPoolMath.sol: a level from 2,000 to 2,002 with 10 ETH, and one from 1,998 to
// 2,000 with 19,990 USDC. The others start inside a level, take a price above one native unit,
// add spreads, or use a 0-decimal quote token.
func TestLevelSteps_Solidity(t *testing.T) {
	ethUsdc := mid(t, 2_000e8, 18, 6)
	usdcWeth := mid(t, 2_000e8, 6, 18)
	ask := askRange(t, config(0, 1_000, 0, 0), ethUsdc, 0)
	bid := bidRange(t, config(-1_000, 0, 0, 0), ethUsdc, 0)
	askBig := askRange(t, config(0, 1_000, 0, 0), usdcWeth, 0)
	bidBig := bidRange(t, config(-1_000, 0, 0, 0), usdcWeth, 0)
	ask6 := askRange(t, config(-10, 10, 2, 3), mid(t, 1e8, 6, 6), 1)
	bid6 := bidRange(t, config(-10, 10, 2, 3), mid(t, 1e8, 6, 6), 1)
	zeroDecimals := mid(t, 1e8, 18, 0)
	doubled, err := zeroDecimals.Add(zeroDecimals)
	require.NoError(t, err)
	bid0 := PriceRange{Lower: zeroDecimals, Upper: doubled}

	third := fraction(t, new(uint256.Int).Div(maxRaw, u(3)))
	twoThirds := fraction(t, new(uint256.Int).Mul(third.Raw(), u(2)))
	inside := fraction(t, dec(t, "12345678901234567890123456789"))

	for _, c := range []struct {
		name                         string
		buy                          bool
		depth                        string
		start                        Fraction
		amount                       string
		r                            PriceRange
		wantBase, wantQuote, wantEnd string
	}{
		{"buy example", true, "10000000000000000000", Lower, "10002500000", ask,
			"5000000000000000000", "10002500000", "10141204801825835211973625643007"},
		{"buy drain", true, "10000000000000000000", Lower, "30000000000", ask,
			"10000000000000000000", "20010000000", "20282409603651670423947251286015"},
		{"buy inside", true, "10000000000000000000", third, "1234567891", ask,
			"617065564623708788", "1234567891", "7595174970151000246736301992361"},
		{"sell example", false, "19990000000", Upper, "2500000000000000000", bid,
			"2500000000000000000", "4999374999", "15211807202738752817960438464511"},
		{"sell drain", false, "19990000000", Upper, "20000000000000000000", bid,
			"10000000000000000000", "19990000000", "0"},
		{"sell inside", false, "19990000000", twoThirds, "1234567890123456789", bid,
			"1234567890123456789", "2468211141", "11852550655233336601259115796663"},
		{"sell big", false, "19990000000000000000000", Upper, "2500000", bidBig,
			"2500000", "4999374999999999999999", "15211807202738752817960438464511"},
		{"buy big", true, "10000000", third, "1234567890123456789012", askBig,
			"617065", "1234567890123456789012", "7595174206689044675325951136325"},
		{"usdc buy", true, "2000000000", inside, "1000000000", ask6,
			"1000000990", "1000000000", "10147387674958095247384436119680"},
		{"usdc sell", false, "2000000000", inside, "1000000000", bid6,
			"1000000000", "999987009", "6172919659771676968045456785"},
		{"sell zero decimals", false, "13", Upper, "3333333333333333333", bid0,
			"3333333333333333333", "6", "12481482833016412569363016545380"},
		{"sell zero decimals +1", false, "13", Upper, "3333333333333333334", bid0,
			"3333333333333333334", "6", "12481482833016412567022738514189"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var step Step
			var err error
			if c.buy {
				step, err = BuyWithExactQuote(dec(t, c.depth), c.start, dec(t, c.amount), c.r)
			} else {
				step, err = SellExactBase(dec(t, c.depth), c.start, dec(t, c.amount), c.r)
			}
			require.NoError(t, err)
			require.Equal(t, c.wantBase, step.BaseAmount.Dec())
			require.Equal(t, c.wantQuote, step.QuoteAmount.Dec())
			require.Equal(t, c.wantEnd, step.End.Raw().Dec())
		})
	}
}

// market is NightlyPoolMathFuzz._market: one level's ask and bid ranges, a start fraction, and its
// base and quote depths.
type market struct {
	ask, bid              PriceRange
	start                 Fraction
	baseDepth, quoteDepth *uint256.Int
}

// randomMarket follows NightlyPoolMathFuzz._market: any nonzero oracle price, 0 to 18 decimals,
// lower offsets from -99.9999% to +100%, positive widths, extra spreads up to 65,535 millionths,
// and a quote depth of at most 1e24 units, which keeps the square roots within uint256.
func randomMarket(t *testing.T, r *rand.Rand) market {
	t.Helper()
	rawPrice := randomBits(r, 1+r.IntN(64)).Uint64()
	if rawPrice == 0 {
		rawPrice = 1
	}
	m := mid(t, rawPrice, uint8(r.IntN(19)), uint8(r.IntN(19)))
	lower := SignedSpread(r.IntN(2_000_000) - 999_999)
	upper := lower + SignedSpread(r.IntN(1_000_000)+1)
	c := config(lower, upper, Spread(r.IntN(1<<16)), Spread(r.IntN(1<<16)))
	ask := askRange(t, c, m, 0)
	quoteDepth := new(uint256.Int).Mod(randomUint(r), dec(t, "1000000000000000000000000"))
	quoteDepth.AddUint64(quoteDepth, 1)
	baseDepth, err := ask.Upper.BaseFor(quoteDepth, Floor)
	require.NoError(t, err)
	if baseDepth.IsZero() {
		baseDepth = u(1)
	}
	return market{
		ask:        ask,
		bid:        bidRange(t, c, m, 0),
		start:      fraction(t, randomBits(r, 104)),
		baseDepth:  baseDepth,
		quoteDepth: quoteDepth,
	}
}

// below returns a random value from 0 to limit.
func below(r *rand.Rand, limit *uint256.Int) *uint256.Int {
	return new(uint256.Int).Mod(randomUint(r), new(uint256.Int).AddUint64(limit, 1))
}

func (m market) checkBuy(t *testing.T, step Step) {
	t.Helper()
	require.False(t, m.baseDepth.Lt(&step.BaseAmount), "base overdraw")
	require.False(t, step.End.raw.Lt(&m.start.raw), "buy moves down")
	lowerCost, err := m.ask.Lower.QuoteFor(&step.BaseAmount, Ceil)
	require.NoError(t, err)
	require.False(t, step.QuoteAmount.Lt(lowerCost), "buy below lower ask")
}

func (m market) checkSell(t *testing.T, step Step) {
	t.Helper()
	require.False(t, m.quoteDepth.Lt(&step.QuoteAmount), "quote overdraw")
	require.False(t, m.start.raw.Lt(&step.End.raw), "sell moves up")
	upperValue, err := m.bid.Upper.QuoteFor(&step.BaseAmount, Floor)
	require.NoError(t, err)
	require.False(t, upperValue.Lt(&step.QuoteAmount), "sell above upper bid")
}

// TestBuyWithExactQuote_BoundsAndMonotonicity: NightlyPoolMathFuzz's buy properties. The buy spends
// at most the offered quote, and all of it unless it drains the level. One more quote unit buys no
// less base. The cost is at least the lower ask, and the upper ask value drains the level.
func TestBuyWithExactQuote_BoundsAndMonotonicity(t *testing.T) {
	r := rand.New(rand.NewPCG(0x5eed, 7))
	for range 2_000 {
		m := randomMarket(t, r)
		budget, err := m.ask.Upper.QuoteFor(m.baseDepth, Ceil)
		require.NoError(t, err)
		amount := below(r, budget)
		first, err := BuyWithExactQuote(m.baseDepth, m.start, amount, m.ask)
		require.NoError(t, err)
		next, err := BuyWithExactQuote(m.baseDepth, m.start, new(uint256.Int).AddUint64(amount, 1), m.ask)
		require.NoError(t, err)
		require.False(t, amount.Lt(&first.QuoteAmount), "buy exceeds quote budget")
		require.False(t, next.BaseAmount.Lt(&first.BaseAmount), "buy output monotonic")
		require.False(t, next.End.raw.Lt(&first.End.raw), "buy position monotonic")
		m.checkBuy(t, first)
		m.checkBuy(t, next)
		if first.BaseAmount.Lt(m.baseDepth) {
			require.Equal(t, amount, &first.QuoteAmount, "partial buy consumes exact input")
		}
		drain, err := BuyWithExactQuote(m.baseDepth, m.start, budget, m.ask)
		require.NoError(t, err)
		require.True(t, drain.BaseAmount.Eq(m.baseDepth) && drain.End.IsUpper(), "quote buy drain")
	}
}

// TestSellExactBase_BoundsAndMonotonicity: NightlyPoolMathFuzz's sell properties, mirrored.
func TestSellExactBase_BoundsAndMonotonicity(t *testing.T) {
	r := rand.New(rand.NewPCG(0x5eed, 8))
	for range 2_000 {
		m := randomMarket(t, r)
		maxBase, err := m.bid.Lower.BaseFor(m.quoteDepth, Ceil)
		require.NoError(t, err)
		amount := below(r, maxBase)
		first, err := SellExactBase(m.quoteDepth, m.start, amount, m.bid)
		require.NoError(t, err)
		next, err := SellExactBase(m.quoteDepth, m.start, new(uint256.Int).AddUint64(amount, 1), m.bid)
		require.NoError(t, err)
		require.False(t, amount.Lt(&first.BaseAmount), "sell exceeds base budget")
		require.False(t, next.QuoteAmount.Lt(&first.QuoteAmount), "sell output monotonic")
		require.False(t, first.End.raw.Lt(&next.End.raw), "sell position monotonic")
		m.checkSell(t, first)
		m.checkSell(t, next)
		if first.QuoteAmount.Lt(m.quoteDepth) {
			require.Equal(t, amount, &first.BaseAmount, "partial sell consumes exact input")
		}
		drain, err := SellExactBase(m.quoteDepth, m.start, maxBase, m.bid)
		require.NoError(t, err)
		require.True(t, drain.QuoteAmount.Eq(m.quoteDepth) && drain.End.IsLower(), "sell drain")
	}
}

// TestSellExactBase_MonotonicAtQuoteRoundingBoundary: one more base unit at a quote rounding
// boundary does not reduce the quote output.
func TestSellExactBase_MonotonicAtQuoteRoundingBoundary(t *testing.T) {
	r := rand.New(rand.NewPCG(0x5eed, 9))
	for range 2_000 {
		m := randomMarket(t, r)
		startPrice, err := m.bid.PriceAt(m.start, Floor)
		require.NoError(t, err)
		refillPrice, err := startPrice.Mean(m.bid.Lower, Floor)
		require.NoError(t, err)
		quoteBoundary := new(uint256.Int).Mod(randomUint(r), m.quoteDepth)
		baseAtBoundary, err := refillPrice.BaseFor(quoteBoundary, Floor)
		require.NoError(t, err)
		before, err := SellExactBase(m.quoteDepth, m.start, baseAtBoundary, m.bid)
		require.NoError(t, err)
		after, err := SellExactBase(m.quoteDepth, m.start, new(uint256.Int).AddUint64(baseAtBoundary, 1), m.bid)
		require.NoError(t, err)
		require.False(t, after.QuoteAmount.Lt(&before.QuoteAmount), "one more base unit reduces quote output")
	}
}
