package rateob

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func u(s string) uint256.Int {
	var z uint256.Int
	if err := z.SetFromDecimal(s); err != nil {
		panic(err)
	}
	return z
}

// withPool adds a band pool anchored at lmp (2000): a 1% pair limit both ways, a 0.1% engine fee,
// a 1% swap rail, and three bands at 20%, 60% and 100% of the limit (0.2%, 0.6%, 1%) charging
// 1x, 2x and 3x the fee. Each band holds 1 base and 3000 USDC unless reserves are given.
func withPool(reserves ...[2]string) func(*Extra) {
	return func(e *Extra) {
		bp := &BandPool{LimitBuy: 1_000_000, LimitSell: 1_000_000, EngineFee: 100_000, RailUp: 1_000_000,
			RailDown: 1_000_000}
		bp.Anchor.SetUint64(2000e8)
		for i, frac := range []uint32{20_000_000, 60_000_000, 100_000_000} {
			b := Band{SpreadFrac: frac, FeeMultiplier: uint32(i+1) * 100_000_000, HasShares: true,
				BaseReserve: u("1000000000000000000"), QuoteReserve: u("3000000000")}
			if i < len(reserves) {
				b.BaseReserve, b.QuoteReserve = u(reserves[i][0]), u(reserves[i][1])
			}
			bp.Bands = append(bp.Bands, b)
		}
		e.BandPool = bp
	}
}

func TestPool_BuyRemainderWalksBandsTightestFirst(t *testing.T) {
	// One ask: 1 base at 2000. Buying with 5000 USDC clears it (2000 USDC -> 0.999 base after the
	// 0.1% taker fee) and leaves 3000 USDC for the pool.
	sim := newSim(t, nil, []Level{level(2000e8, order("1000000000000000000"))}, withPool())
	res, err := quote(t, sim, tQuote, "5000000000")
	require.NoError(t, err)

	// Band 0 (0.2%): bound ceil(2000*1.002) = 2004. It sells all its base for 2004 USDC:
	// out 1e18, fee 1e18*0.1% = 1e15.
	// Band 1 (0.6%): bound 2012, takes the other 996 USDC: out floor(996e6*1e8/2012e8)*1e12 =
	// 495029e12, fee 495029e12*0.2% = 990058e9.
	// minOut at the make price min(price, ceil(2000*1.03)) = 2060: floor(3000e6*1e8/2060e8)*1e12 =
	// 1456310e12, less band 0's 0.1% -- 1454853690e9, which 1493038942e9 clears.
	require.Equal(t, "2492038942000000000", res.TokenAmountOut.Amount.String())
	require.Equal(t, "2990058000000000", res.Fee.Amount.String())
	require.Equal(t, "0", res.RemainingTokenAmountIn.Amount.String())
	require.Equal(t, int64(gasBase+gasMatch+gasPoolLeg+2*gasPoolBand), res.Gas)

	si := res.SwapInfo.(SwapInfo)
	require.Equal(t, "206000000000", si.LimitPrice)
	require.Equal(t, "3000000000", si.PoolAmountIn)
	require.Equal(t, 1, si.Matches)
	require.Len(t, si.Fills, 2)
	// Band 1's fill is reportable (out >= reserve/1e6) and inside the 1% rail: lmp = 2012.
	require.Equal(t, "201200000000", si.Lmp.Dec())
}

func TestPool_SellRemainderBoundsRoundDown(t *testing.T) {
	// One bid: 2000 USDC at 2000 (1 base). Selling 3 base clears it (1998 USDC) and leaves 2.
	sim := newSim(t, []Level{level(2000e8, order("2000000000"))}, nil, withPool())
	res, err := quote(t, sim, tBase, "3000000000000000000")
	require.NoError(t, err)

	// Band 0: floor(2000*0.998) = 1996. available floor(3000e6*1e8/1996e8)*1e12 = 1503006e12,
	// out 1503006*1996 = 2999999976 (all but 24 of its 3000 USDC), fee 2999999.
	// Band 1: 1988, takes the other 496994e12: out 988024072, fee 0.2% = 1976048.
	// minOut at max(price, floor(2000*0.97)) = 1940: 3880 USDC less 0.1% = 3876.12 USDC.
	require.Equal(t, "5981048001", res.TokenAmountOut.Amount.String())
	require.Equal(t, "0", res.RemainingTokenAmountIn.Amount.String())
	si := res.SwapInfo.(SwapInfo)
	require.Equal(t, "194000000000", si.LimitPrice)
	require.Equal(t, "2000000000000000000", si.PoolAmountIn)
	require.Equal(t, "198800000000", si.Lmp.Dec())
}

func TestPool_ExhaustedBandsReturnTheRest(t *testing.T) {
	// Pool-only (no asks): 10,000 USDC against three bands of 1 base each. Every band empties and
	// the rest is returned rather than sent, since the engine would refund it.
	sim := newSim(t, nil, nil, withPool())
	res, err := quote(t, sim, tQuote, "10000000000")
	require.NoError(t, err)
	si := res.SwapInfo.(SwapInfo)
	// 2004 + 2012 + 2020 USDC buy the three bases.
	require.Equal(t, "6036000000", si.PoolAmountIn)
	require.Equal(t, "3964000000", res.RemainingTokenAmountIn.Amount.String())
	require.Equal(t, 0, si.Matches)
	require.Len(t, si.Fills, 3)
	// Fees 0.1%, 0.2%, 0.3% of 1 base each.
	require.Equal(t, "2994000000000000000", res.TokenAmountOut.Amount.String())
	// With no match the engine leaves lmp alone and starts the limit at the snapshot's reach.
	require.Equal(t, "206000000000", si.LimitPrice)
	require.Equal(t, int64(gasBase+gasPoolLeg+3*gasPoolBand), res.Gas)

	// A band with no shares, no payable reserve, or a tolerance that rounds to zero is skipped.
	sim = newSim(t, nil, nil, withPool([2]string{"0", "0"}), func(e *Extra) {
		e.BandPool.Bands[1].HasShares = false
		e.BandPool.Bands[2].SpreadFrac = 1 // 1 * 1e6 / 1e8 rounds to 0
	})
	_, err = quote(t, sim, tQuote, "1000000000")
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func TestPool_FeeIsCappedAtMaxFeeRate(t *testing.T) {
	// 1% engine fee x5 would be 5%: PoolBands caps every band at 3%.
	sim := newSim(t, nil, nil, withPool(), func(e *Extra) {
		e.BandPool.EngineFee = 1_000_000
		e.BandPool.Bands[0].FeeMultiplier = 500_000_000
	})
	res, err := quote(t, sim, tQuote, "1002000000") // half of band 0 at 2004
	require.NoError(t, err)
	// out floor(1002e6*1e8/2004e8)*1e12 = 0.5e18, fee 3% of it.
	require.Equal(t, "15000000000000000", res.Fee.Amount.String())
	require.Equal(t, "485000000000000000", res.TokenAmountOut.Amount.String())
}

func TestPool_RefusedByTheOrdersOwnPrice(t *testing.T) {
	// MaxMatches 1 stops the book after one of two asks at 2000, so the ask head is still 2000 and
	// the make price is min(..., askHead) = 2000. The pool sells at 2004+ less its fee: below the
	// bound at the whole remainder and at band 0's boundary alike, so the quote is book-only.
	sim := newSim(t, nil, []Level{level(2000e8, order("1000000000000000000"), order("1000000000000000000"))},
		withPool(), func(e *Extra) { e.MaxMatches = 1 })
	res, err := quote(t, sim, tQuote, "5000000000")
	require.NoError(t, err)
	require.Equal(t, "999000000000000000", res.TokenAmountOut.Amount.String())
	require.Equal(t, "3000000000", res.RemainingTokenAmountIn.Amount.String())
	require.Empty(t, res.SwapInfo.(SwapInfo).Fills)

	// At 2006 the make price clears band 0 (2004) but not band 1 (2012 at 2x the fee): the whole
	// 2994 USDC remainder fails, band 0's boundary (2004 USDC) passes and the rest is returned.
	sim = newSim(t, nil, []Level{level(2006e8, order("1000000000000000000"), order("1000000000000000000"))},
		withPool(), func(e *Extra) { e.MaxMatches = 1 })
	res, err = quote(t, sim, tQuote, "5000000000")
	require.NoError(t, err)
	si := res.SwapInfo.(SwapInfo)
	require.Equal(t, "2004000000", si.PoolAmountIn)
	require.Equal(t, "990000000", res.RemainingTokenAmountIn.Amount.String())
	require.Equal(t, "1998000000000000000", res.TokenAmountOut.Amount.String()) // 0.999 + 0.999
}

func TestPool_NotQuotedWhereTheWalkCannotPredictTheEngine(t *testing.T) {
	asks := []Level{level(2000e8, order("1000000000000000000"))}
	cases := map[string]func(*Extra){
		"no pool":       func(e *Extra) { e.BandPool = nil },
		"limit zero":    func(e *Extra) { e.BandPool.LimitBuy = 0 },
		"side not read": func(e *Extra) { e.AsksFull = true },
		"level not read": func(e *Extra) {
			e.Asks[0].More = true
		},
		"order may expire": func(e *Extra) {
			e.Asks = append(e.Asks, level(2001e8, order("1000000000000000000", now+60)))
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			sim := newSim(t, nil, slicesCloneLevels(asks), withPool(), mutate)
			res, err := quote(t, sim, tQuote, "5000000000")
			require.NoError(t, err)
			require.Equal(t, "999000000000000000", res.TokenAmountOut.Amount.String())
			require.Equal(t, "3000000000", res.RemainingTokenAmountIn.Amount.String())
			require.Equal(t, "200000000000", res.SwapInfo.(SwapInfo).LimitPrice)
		})
	}
	// With no book fill and no pool leg there is nothing to quote.
	sim := newSim(t, nil, nil, withPool(), func(e *Extra) { e.BandPool.LimitBuy = 0 })
	_, err := quote(t, sim, tQuote, "5000000000")
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func slicesCloneLevels(levels []Level) []Level {
	out := make([]Level, len(levels))
	for i, l := range levels {
		out[i] = Level{Price: l.Price, Orders: append([]Order(nil), l.Orders...), More: l.More}
	}
	return out
}

func TestPool_DustLevelEndsTheWalkAndSetsLmp(t *testing.T) {
	// A level holding only dust is entered, evicted and ends the sweep with no match -- and
	// MatchingLib still sets lmp to it, so the make price is measured from 2001.
	sim := newSim(t, nil, []Level{level(2000e8, order("1000000000000000000")), level(2001e8, order("1"))},
		withPool())
	res, err := quote(t, sim, tQuote, "3000000000")
	require.NoError(t, err)
	si := res.SwapInfo.(SwapInfo)
	require.Equal(t, "206103000000", si.LimitPrice) // ceil(2001 * 1.03)
	require.Equal(t, "1000000000", si.PoolAmountIn)
	require.Equal(t, int64(gasBase+gasMatch+gasEviction+gasPoolLeg+gasPoolBand), res.Gas)

	// The dust order went with the swap, and with it the level.
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: si})
	require.Empty(t, sim.Asks)
}

func TestPool_ReportIsClampedAndNeedsAReportableFill(t *testing.T) {
	// A 0.1% rail caps band 0's 2004 at floor(2000*1.001) = 2002.
	sim := newSim(t, nil, nil, withPool(), func(e *Extra) { e.BandPool.RailUp = 100_000 })
	res, err := quote(t, sim, tQuote, "1000000000")
	require.NoError(t, err)
	si := res.SwapInfo.(SwapInfo)
	require.Equal(t, "200200000000", si.Lmp.Dec())

	// A fill under 1/1e6 of the band's payable reserve executes but does not print.
	sim = newSim(t, nil, nil, withPool([2]string{"1000000000000000000000000", "0"}))
	res, err = quote(t, sim, tQuote, "1000000000")
	require.NoError(t, err)
	si = res.SwapInfo.(SwapInfo)
	require.Equal(t, "200000000000", si.Lmp.Dec())
}

func TestPool_UpdateBalanceMovesReservesAndLmp(t *testing.T) {
	sim := newSim(t, nil, []Level{level(2000e8, order("1000000000000000000"))}, withPool())
	before := sim.BandPool
	res, err := quote(t, sim, tQuote, "5000000000")
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})

	require.Empty(t, sim.Asks)
	require.Equal(t, "201200000000", sim.Lmp.Dec())
	// Band 0 sold all its base for 2004 USDC, band 1 sold 495029e12 for 996 USDC.
	require.Equal(t, "0", sim.BandPool.Bands[0].BaseReserve.Dec())
	require.Equal(t, "5004000000", sim.BandPool.Bands[0].QuoteReserve.Dec())
	require.Equal(t, "504971000000000000", sim.BandPool.Bands[1].BaseReserve.Dec())
	require.Equal(t, "3996000000", sim.BandPool.Bands[1].QuoteReserve.Dec())
	// Copy-on-write: the pre-swap state is untouched, and the anchor does not move in-transaction.
	require.Equal(t, "1000000000000000000", before.Bands[0].BaseReserve.Dec())
	require.Equal(t, "200000000000", sim.BandPool.Anchor.Dec())

	// The next quote starts where this one left the pool: band 1's rest, then band 2.
	res, err = quote(t, sim, tQuote, "2000000000")
	require.NoError(t, err)
	require.Equal(t, 0, res.SwapInfo.(SwapInfo).Matches)
	require.Len(t, res.SwapInfo.(SwapInfo).Fills, 2)
}

func TestPool_CloneStateAndDeterminism(t *testing.T) {
	sim := newSim(t, []Level{level(2000e8, order("1000000000"))}, []Level{level(2001e8, order("500000000000000000"))},
		withPool())
	for _, in := range []struct{ token, amount string }{{tBase, "2000000000000000000"}, {tQuote, "4000000000"}} {
		amount, _ := new(big.Int).SetString(in.amount, 10)
		out := tQuote
		if in.token == tQuote {
			out = tBase
		}
		testutil.TestCloneState(t, sim, pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in.token, Amount: amount}, TokenOut: out}, nil)
	}
	// Selling 1 base three times:
	//  1. the bid's 0.5 base (999 USDC after fee), then 0.5 base into band 0 at 1996 (998 USDC,
	//     997.002 after its 0.1%);
	//  2. no bids left: 1 base into band 0 at 1996 again (the anchor holds in-transaction),
	//     1994.004 USDC;
	//  3. band 0 has 6 USDC left: 3006e12 base for 5999976 (5993977 net), then band 1 at 1988
	//     at 0.2%: 996994e12 base for 1982024072 (1978060024 net).
	testutil.TestCalcAmountOutWithUpdateBalance(t, sim, map[int]map[int][][][2]string{
		0: {1: {{{"1000000000000000000", "1996002000"}, {"1000000000000000000", "1994004000"},
			{"1000000000000000000", "1984054001"}}}},
	})
}

func TestMakePriceMirrorsMarketMakePriceLib(t *testing.T) {
	n := func(v uint64) *uint256.Int { return uint256.NewInt(v) }
	var z uint256.Int
	const s = 3_000_000
	// limitBuy: min(lp, ceil(lmp*1.03)), capped at a nonzero ask head.
	require.True(t, mmLimitBuy(&z, n(2000), n(9999), n(0), n(0), s))
	require.Equal(t, uint64(2060), z.Uint64())
	require.True(t, mmLimitBuy(&z, n(2000), n(9999), n(1990), n(2050), s))
	require.Equal(t, uint64(2050), z.Uint64())
	require.True(t, mmLimitBuy(&z, n(0), n(2100), n(1990), n(2050), s))
	require.Equal(t, uint64(2050), z.Uint64())
	require.True(t, mmLimitBuy(&z, n(0), n(9999), n(2000), n(0), s)) // no lmp, bid head only
	require.Equal(t, uint64(2060), z.Uint64())
	// limitSell: max(lp, floor(lmp*0.97)), floored at the bid head only when there is no ask.
	require.True(t, mmLimitSell(&z, n(2000), n(1), n(1950), n(0), s))
	require.Equal(t, uint64(1950), z.Uint64())
	require.True(t, mmLimitSell(&z, n(2000), n(1), n(1950), n(2010), s))
	require.Equal(t, uint64(1940), z.Uint64())
	require.True(t, mmLimitSell(&z, n(2000), n(1), n(0), n(0), s))
	require.Equal(t, uint64(1940), z.Uint64())
	require.True(t, mmLimitSell(&z, n(0), n(1), n(1950), n(2010), s)) // no lmp, both heads
	require.Equal(t, uint64(1950), z.Uint64())
}

func TestTracker_AnchorMustBeFrozen(t *testing.T) {
	lmp := uint256.NewInt(2000e8)
	reads := func(mutate func(*poolReads)) *poolReads {
		pr := &poolReads{ok: true, anchor: big.NewInt(2000e8), limitBuy: 1e5, limitSell: 1e5, engineFee: 1e5,
			bandCount: big.NewInt(1)}
		pr.newest.Price, pr.newest.ActualWindow = big.NewInt(2000e8), anchorWindow
		pr.bands[0].SpreadFrac, pr.bands[0].Shares = 20_000_000, big.NewInt(1)
		pr.reserves[0].BaseReserve, pr.reserves[0].QuoteReserve = big.NewInt(5), big.NewInt(7)
		pr.multipliers[0] = 100_000_000
		if mutate != nil {
			mutate(pr)
		}
		return pr
	}
	var base, quote uint256.Int
	bp := reads(nil).bandPool(lmp, &base, &quote)
	require.NotNil(t, bp)
	require.Equal(t, "200000000000", bp.Anchor.Dec())
	require.Equal(t, uint64(5), base.Uint64())
	require.Equal(t, uint64(7), quote.Uint64())

	for name, mutate := range map[string]func(*poolReads){
		"newest observation inside the window": func(pr *poolReads) { pr.newest.ActualWindow = anchorWindow - 1 },
		"segment before it at another price":   func(pr *poolReads) { pr.newest.Price = big.NewInt(2001e8) },
		"pool anchored elsewhere (seedPrice)":  func(pr *poolReads) { pr.anchor = big.NewInt(2306e8) },
		"a read failed":                        func(pr *poolReads) { pr.ok = false },
		"anchor read missing":                  func(pr *poolReads) { pr.anchor = nil },
	} {
		require.Nil(t, reads(mutate).bandPool(lmp, &base, &quote), name)
	}

	// checkCalls: every band below bandCount must have answered.
	pr := reads(nil)
	results := make([]bool, 4+3*maxBands)
	for i := range 4 + 3 {
		results[i] = true
	}
	require.True(t, pr.checkCalls(results))
	results[5] = false
	require.False(t, pr.checkCalls(results))
}

// TestPool_MsgpackRoundTrip: the router loads simulators from msgpack (pkg/msgpack includes
// unexported fields), so the band pool must survive it or pool quotes silently change.
func TestPool_MsgpackRoundTrip(t *testing.T) {
	sim := newSim(t, []Level{level(2000e8, order("1000000000"))}, []Level{level(2001e8, order("500000000000000000"))},
		withPool())
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.IncludeUnexported(true)
	enc.SetForceAsArray(true)
	require.NoError(t, enc.Encode(sim))
	dec := msgpack.NewDecoder(&buf)
	dec.IncludeUnexported(true)
	var decoded PoolSimulator
	require.NoError(t, dec.Decode(&decoded))
	require.NotNil(t, decoded.BandPool)
	for _, in := range []struct{ token, amount string }{{tBase, "3000000000000000000"}, {tQuote, "8000000000"}} {
		want, err := quote(t, sim, in.token, in.amount)
		require.NoError(t, err)
		got, err := quote(t, &decoded, in.token, in.amount)
		require.NoError(t, err)
		require.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
		require.Equal(t, want.RemainingTokenAmountIn.Amount, got.RemainingTokenAmountIn.Amount)
		require.NotEmpty(t, got.SwapInfo.(SwapInfo).Fills)
	}
}
