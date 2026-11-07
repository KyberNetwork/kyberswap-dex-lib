package rateob

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const (
	tBase  = "0x00000000000000000000000000000000000000b1"
	tQuote = "0x00000000000000000000000000000000000000c2"
	now    = 1_800_000_000
)

func order(deposit string, deadline ...uint64) Order {
	var o Order
	if err := o.Deposit.SetFromDecimal(deposit); err != nil {
		panic(err)
	}
	if len(deadline) > 0 {
		o.Deadline = deadline[0]
	}
	return o
}

func level(price uint64, orders ...Order) Level {
	l := Level{Orders: orders}
	l.Price.SetUint64(price)
	return l
}

// newSim builds an 18-decimal base / 6-decimal quote pair (decDiff 1e12, base has more decimals),
// lmp 2000 quote per base, a 0.1% taker fee and a 3% limit spread both ways.
func newSim(t *testing.T, bids, asks []Level, mutate ...func(*Extra)) *PoolSimulator {
	t.Helper()
	extra := Extra{SpreadBuy: 3_000_000, SpreadSell: 3_000_000, TakerFee: 100_000, MaxMatches: 20,
		Timestamp: now, Bids: bids, Asks: asks}
	extra.Lmp.SetUint64(2000e8)
	for _, m := range mutate {
		m(&extra)
	}
	extraBytes, err := json.Marshal(&extra)
	require.NoError(t, err)
	staticBytes, _ := json.Marshal(StaticExtra{Engine: "0x00000000000000000000000000000000000000e0", DecDiff: 1e12,
		BaseBQuote: true})
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{
		Address: "0x00000000000000000000000000000000000000a0", Exchange: DexType, Type: DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens:   []*entity.PoolToken{{Address: tBase}, {Address: tQuote}},
		Extra:    string(extraBytes), StaticExtra: string(staticBytes),
	}})
	require.NoError(t, err)
	return sim
}

func quote(t *testing.T, sim *PoolSimulator, tokenIn, amount string) (*pool.CalcAmountOutResult, error) {
	t.Helper()
	in, _ := new(big.Int).SetString(amount, 10)
	tokenOut := tQuote
	if tokenIn == tQuote {
		tokenOut = tBase
	}
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: in}, TokenOut: tokenOut})
}

func TestSell_ClearsThenPartial(t *testing.T) {
	// Two bids at 2000: 1000 USDC and 3000 USDC. Selling 2 base clears the first (0.5 base) and
	// partially fills the second with 1.5 base -> 3000 USDC, which is its whole deposit.
	sim := newSim(t, []Level{level(2000e8, order("1000000000"), order("3000000000"))}, nil)
	res, err := quote(t, sim, tBase, "2000000000000000000")
	require.NoError(t, err)
	// 1000 USDC - 0.1% and 3000 USDC - 0.1%, each fee floored per match.
	require.Equal(t, "3996000000", res.TokenAmountOut.Amount.String())
	require.Equal(t, "4000000", res.Fee.Amount.String())
	require.Equal(t, "0", res.RemainingTokenAmountIn.Amount.String())
	si := res.SwapInfo.(SwapInfo)
	require.Equal(t, "200000000000", si.LimitPrice)
	require.Equal(t, 2, si.Matches)
}

func TestSell_PartialFillKeepsHead(t *testing.T) {
	sim := newSim(t, []Level{level(2000e8, order("3000000000"))}, nil)
	res, err := quote(t, sim, tBase, "1000000000000000000") // 1 base -> 2000 USDC of the 3000
	require.NoError(t, err)
	require.Equal(t, "1998000000", res.TokenAmountOut.Amount.String())
	si := res.SwapInfo.(SwapInfo)
	require.NotNil(t, si.Head)
	require.Equal(t, "1000000000", si.Head.Dec())

	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
	res, err = quote(t, sim, tBase, "1000000000000000000") // only 0.5 base of bids is left
	require.NoError(t, err)
	require.Equal(t, "999000000", res.TokenAmountOut.Amount.String())
	require.Equal(t, "500000000000000000", res.RemainingTokenAmountIn.Amount.String())
}

func TestBuy_WalksLevelsWithinReach(t *testing.T) {
	// Asks at 2000, 2050 and 2100. The reach is ceil(2000 * 1.03) = 2060, so 2100 is out.
	sim := newSim(t, nil, []Level{
		level(2000e8, order("1000000000000000000")),
		level(2050e8, order("1000000000000000000")),
		level(2100e8, order("1000000000000000000")),
	})
	res, err := quote(t, sim, tQuote, "10000000000") // 10,000 USDC
	require.NoError(t, err)
	// 2000 + 2050 USDC buys 2 base; 0.1% fee on each.
	require.Equal(t, "1998000000000000000", res.TokenAmountOut.Amount.String())
	require.Equal(t, "5950000000", res.RemainingTokenAmountIn.Amount.String())
	require.Equal(t, "205000000000", res.SwapInfo.(SwapInfo).LimitPrice)
}

func TestDustAndExpiredAreEvictedNotFilled(t *testing.T) {
	// A bid of 1 raw USDC converts to 0 base at 2000: fpop evicts it. An expired order is evicted
	// too. Neither costs a match, and the order behind them fills.
	sim := newSim(t, []Level{level(2000e8, order("1"), order("5000000", now-1), order("2000000000"),
		order("1000000000"))}, nil)
	res, err := quote(t, sim, tBase, "1000000000000000000")
	require.NoError(t, err)
	require.Equal(t, "1998000000", res.TokenAmountOut.Amount.String())
	si := res.SwapInfo.(SwapInfo)
	require.Equal(t, 1, si.Matches)
	require.Equal(t, 3, si.Orders) // both evictions are applied with the fill
	require.Equal(t, int64(gasBase+gasMatch+2*gasEviction), res.Gas)
}

func TestNearExpiryStopsTheWalk(t *testing.T) {
	// An order that may expire before the swap lands is not quoted, nor is anything behind it.
	sim := newSim(t, []Level{level(2000e8, order("1000000000"), order("1000000000", now+60), order("9000000000"))}, nil)
	res, err := quote(t, sim, tBase, "2000000000000000000")
	require.NoError(t, err)
	require.Equal(t, "999000000", res.TokenAmountOut.Amount.String())
	require.Equal(t, "1500000000000000000", res.RemainingTokenAmountIn.Amount.String())
}

func TestEvictionOnlyPassEndsTheSweep(t *testing.T) {
	// MatchingLib sets the head to 0 after a pass with no match: the dust-only level at 1999 ends
	// the sweep even though 1998 holds a fillable bid within reach.
	sim := newSim(t, []Level{
		level(2000e8, order("1000000000")),
		level(1999e8, order("1")),
		level(1998e8, order("9000000000")),
	}, nil)
	res, err := quote(t, sim, tBase, "2000000000000000000")
	require.NoError(t, err)
	require.Equal(t, "999000000", res.TokenAmountOut.Amount.String())
	require.Equal(t, "1500000000000000000", res.RemainingTokenAmountIn.Amount.String())
	si := res.SwapInfo.(SwapInfo)
	require.Equal(t, "200000000000", si.LimitPrice) // the executor's price stops before 1999
	require.Equal(t, 1, si.Levels)                  // the dust level is not touched
}

func TestEvictionCapStartsANewPass(t *testing.T) {
	// Nine dust orders then a real one at one price: the first pass evicts 8 and yields with no
	// match, which ends the sweep -- so nothing fills.
	orders := make([]Order, 0, 10)
	for range 9 {
		orders = append(orders, order("1"))
	}
	orders = append(orders, order("2000000000"))
	sim := newSim(t, []Level{level(2000e8, orders...)}, nil)
	_, err := quote(t, sim, tBase, "1000000000000000000")
	require.ErrorIs(t, err, ErrInsufficientLiquidity)

	// Seven dust orders fit in one pass, so the real order still fills.
	sim = newSim(t, []Level{level(2000e8, append(orders[2:9:9], order("2000000000"))...)}, nil)
	res, err := quote(t, sim, tBase, "1000000000000000000")
	require.NoError(t, err)
	require.Equal(t, "1998000000", res.TokenAmountOut.Amount.String())
}

func TestPartialFillLeavingDustEvictsIt(t *testing.T) {
	// Selling 1 wei under 1 base into a 2000.000001 USDC bid leaves 2 raw USDC, which converts to
	// 0 base: _evictIfDust removes it in the same match, so the next quote starts at the next order.
	sim := newSim(t, []Level{level(2000e8, order("2000000001"), order("1000000000"))}, nil)
	res, err := quote(t, sim, tBase, "999999999999999999")
	require.NoError(t, err)
	si := res.SwapInfo.(SwapInfo)
	require.Nil(t, si.Head)
	require.Equal(t, 1, si.Orders)
}

func TestMaxMatchesCapsTheSweep(t *testing.T) {
	sim := newSim(t, []Level{level(2000e8, order("1000000000"), order("1000000000"), order("1000000000"))}, nil,
		func(e *Extra) { e.MaxMatches = 2 })
	res, err := quote(t, sim, tBase, "2000000000000000000")
	require.NoError(t, err)
	require.Equal(t, "1998000000", res.TokenAmountOut.Amount.String())
	require.Equal(t, "1000000000000000000", res.RemainingTokenAmountIn.Amount.String())
	require.Equal(t, uint32(2), res.SwapInfo.(SwapInfo).N)
}

func TestSellReachAndEmptyBook(t *testing.T) {
	// floor(2000 * 0.97) = 1940: a bid at 1900 is out of reach.
	sim := newSim(t, []Level{level(1900e8, order("1000000000"))}, nil)
	_, err := quote(t, sim, tBase, "1000000000000000000")
	require.ErrorIs(t, err, ErrInsufficientLiquidity)

	sim = newSim(t, nil, nil)
	_, err = quote(t, sim, tQuote, "1000000")
	require.ErrorIs(t, err, ErrInsufficientLiquidity)

	sim = newSim(t, []Level{level(2000e8, order("1000000000"))}, nil, func(e *Extra) { e.Lmp.Clear() })
	_, err = quote(t, sim, tBase, "1000000000000000000")
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func TestOrderSizeTooSmall(t *testing.T) {
	// 1e8 wei of base converts to 0 raw USDC: nothing can fill and the engine's size guard
	// (converted <= convert(1 wei)) would refuse it too.
	sim := newSim(t, []Level{level(2000e8, order("1000000000"))}, nil)
	_, err := quote(t, sim, tBase, "100000000")
	require.Error(t, err)
}

func TestConvertMatchesOrderbook(t *testing.T) {
	var z uint256.Int
	p := uint256.NewInt(2000e8)
	// base -> quote, base has more decimals: floor(1e18 * 2000e8 / 1e8) / 1e12 = 2000e6
	require.NoError(t, convert(&z, p, uint256.NewInt(1e18), true, 1e12, true))
	require.Equal(t, "2000000000", z.Dec())
	// quote -> base: floor(2000e6 * 1e8 / 2000e8) * 1e12 = 1e18
	require.NoError(t, convert(&z, p, uint256.NewInt(2000e6), false, 1e12, true))
	require.Equal(t, "1000000000000000000", z.Dec())
	// quote has more decimals: base -> quote scales up instead.
	require.NoError(t, convert(&z, p, uint256.NewInt(1e6), true, 1e12, false))
	require.Equal(t, "2000000000000000000000", z.Dec())
}

func TestCloneStateAndDeterminism(t *testing.T) {
	sim := newSim(t, []Level{level(2000e8, order("1000000000"), order("3000000000"))},
		[]Level{level(2001e8, order("1000000000000000000"), order("2000000000000000000"))})
	for _, in := range []struct{ token, amount string }{{tBase, "700000000000000000"}, {tQuote, "1500000000"}} {
		amount, _ := new(big.Int).SetString(in.amount, 10)
		out := tQuote
		if in.token == tQuote {
			out = tBase
		}
		testutil.TestCloneState(t, sim, pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in.token, Amount: amount}, TokenOut: out}, nil)
	}
	testutil.TestCalcAmountOutWithUpdateBalance(t, sim, map[int]map[int][][][2]string{
		0: {1: {{{"700000000000000000", "1398600000"}, {"700000000000000000", "1398600000"},
			{"700000000000000000", "1198800000"}}}},
	})
}

func TestNativeOut(t *testing.T) {
	sim := newSim(t, nil, nil)
	sim.NativeOut = tBase
	require.True(t, sim.SwapReturnNativeOut(tQuote, tBase, 0))
	require.False(t, sim.SwapReturnNativeOut(tBase, tQuote, 0))
	require.False(t, sim.SwapReceiveNativeIn(tBase, tQuote, 0))
}

func TestDecDiffProbe(t *testing.T) {
	for probe, want := range map[string]struct {
		d  uint64
		bq bool
	}{"1000000000000000000": {1, false}, "1000000": {1e12, true}, "1000000000000000000000000000000": {1e12, false}} {
		p, _ := new(big.Int).SetString(probe, 10)
		d, bq, err := decDiffOf(p)
		require.NoError(t, err)
		require.Equal(t, want.d, d, probe)
		require.Equal(t, want.bq, bq, probe)
	}
	_, _, err := decDiffOf(big.NewInt(3))
	require.Error(t, err)
}
