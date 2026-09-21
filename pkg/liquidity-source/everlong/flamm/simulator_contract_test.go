package everlongflamm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/msgpack/v5"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The simulator's contract with pool-service and router-service, offline over the core end-to-end scenario states
// (testdata/core_e2e_grid_*.jsonl.gz): purity under concurrency, UpdateBalance adopting the quoted post-state
// verbatim, deep clones, RemainingTokenAmountIn and metadata (with every margin and leverage routing on), the input
// contract, margins that only refuse, UpdateBalance across a msgpack hop, and how far an unseen feed round moves a
// quote.

// simQuote is everything one CalcAmountOut returns, rendered for exact comparison.
type simQuote struct {
	Err       string
	Out       string
	Remaining string
	Fee       string
	Gas       int64
	Info      SwapInfo
	Seq       uint64
	Next      string
}

func renderQuote(t testing.TB, r *pool.CalcAmountOutResult, err error) simQuote {
	if err != nil {
		return simQuote{Err: err.Error()}
	}
	si := r.SwapInfo.(SwapInfo)
	raw, e := json.Marshal(si.next)
	require.NoError(t, e)
	return simQuote{Out: r.TokenAmountOut.Amount.String(), Remaining: r.RemainingTokenAmountIn.Amount.String(),
		Fee: r.Fee.Amount.String(), Gas: r.Gas, Info: SwapInfo{Venue: si.Venue, PoolAssetIn: si.PoolAssetIn,
			AmountInUsed: si.AmountInUsed, AmountOut: si.AmountOut, SpreadPpm: si.SpreadPpm}, Seq: si.seq,
		Next: string(raw)}
}

type quoteCase struct {
	sell bool
	a    uint64
}

func quoteCases() []quoteCase {
	var out []quoteCase
	for _, a := range liveGrid(1, 3_000_000, 40) {
		out = append(out, quoteCase{true, a})
	}
	for _, a := range liveGrid(100, 20_000_000_000, 40) {
		out = append(out, quoteCase{false, a})
	}
	return out
}

// TestSimulatorPurityConcurrent: many goroutines quoting one simulator (routing and every margin on) while others clone
// it and adopt quotes on the clones; every answer equals the sequential one, and the shared state never moves.
func TestSimulatorPurityConcurrent(t *testing.T) {
	t.Parallel()
	margins := Policy{LeverRouting: true, PriceBandMarginBps: 1, PriceAgeMarginSec: 5, SpreadAgeMarginSec: 5,
		DebtDriftSec: 30}
	for _, c := range []struct {
		blk, tag string
		policy   Policy
	}{{"51313000", "armed", margins}, {"51313000", "armed", Policy{LeverRouting: true}},
		{"51324800", "armed_hi", Policy{LeverRouting: true}}, {"51302915", "capped", margins},
		{"51313000", "btc_down10", margins}} {
		t.Run(fmt.Sprintf("%s/%s/%+v", c.blk, c.tag, c.policy), func(t *testing.T) {
			sim := simFor(t, gridReads(t, c.blk, c.tag), c.policy)
			before := stateJSON(t, sim)
			// every other case under -race (fixture_sample_test.go): the goroutines still interleave on each
			var cases []quoteCase
			for i, k := range quoteCases() {
				if sampleCase(i, 0, 2) {
					cases = append(cases, k)
				}
			}
			want := make([]simQuote, len(cases))
			venues := map[uint8]int{}
			for i, k := range cases {
				r, err := sim.CalcAmountOut(amountIn(sim, k.sell, k.a))
				want[i] = renderQuote(t, r, err)
				if err == nil {
					venues[want[i].Info.Venue]++
				}
			}
			t.Logf("sequential: venues %v", venues)
			var wg sync.WaitGroup
			errs := make(chan string, 64)
			for g := 0; g < 12; g++ {
				wg.Add(1)
				go func(g int) {
					defer wg.Done()
					rng := rand.New(rand.NewSource(int64(g)))
					for _, i := range rng.Perm(len(cases)) {
						k := cases[i]
						r, err := sim.CalcAmountOut(amountIn(sim, k.sell, k.a))
						got := renderQuote(t, r, err)
						if fmt.Sprint(got) != fmt.Sprint(want[i]) {
							errs <- fmt.Sprintf("goroutine %d case %+v: got %+v want %+v", g, k, got, want[i])
							return
						}
						if err == nil && g%3 == 0 {
							clone := sim.CloneState().(*PoolSimulator)
							clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: r.SwapInfo})
							if clone.broken || clone.seq != 1 {
								errs <- fmt.Sprintf("clone did not adopt %+v", k)
								return
							}
							if _, err := clone.CalcAmountOut(amountIn(clone, !k.sell, 1_000)); err != nil &&
								err == ErrSwapInfoMismatch {
								errs <- "clone broken after adopting its own quote"
								return
							}
						}
					}
				}(g)
			}
			wg.Wait()
			close(errs)
			for e := range errs {
				t.Error(e)
			}
			require.Equal(t, before, stateJSON(t, sim), "concurrent quoting or clone updates wrote the shared state")
			require.Zero(t, sim.seq)
		})
	}
}

// TestSimulatorUpdateBalanceVerbatim: UpdateBalance adopts SwapInfo.next as quoted -- a post-state substituted into a
// SwapInfo is taken as is (nothing is recomputed from the amounts), the SwapInfo's state is never aliased, and one
// SwapInfo applied to two clones leaves them identical and independent.
func TestSimulatorUpdateBalanceVerbatim(t *testing.T) {
	t.Parallel()
	sim := simFor(t, gridReads(t, "51313000", "armed"), Policy{LeverRouting: true})
	r, err := sim.CalcAmountOut(amountIn(sim, true, 5_000))
	require.NoError(t, err)
	si := r.SwapInfo.(SwapInfo)

	// A foreign post-state (another scenario's) with this simulator's seq: adopted verbatim.
	foreign, err := gridReads(t, "51313000", "btc_up10").baseState()
	require.NoError(t, err)
	fake := si
	fake.next = foreign
	a := sim.CloneState().(*PoolSimulator)
	a.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: fake, TokenAmountIn: pool.TokenAmount{Amount: big.NewInt(1)}})
	require.Empty(t, e2eDiff(a.state, foreign), "UpdateBalance recomputed instead of adopting SwapInfo.next")
	require.NotSame(t, foreign, a.state)

	// The adopted state is a copy: writing it does not reach the SwapInfo or a sibling clone.
	b := sim.CloneState().(*PoolSimulator)
	c := sim.CloneState().(*PoolSimulator)
	b.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: si})
	c.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: si})
	require.Empty(t, e2eDiff(b.state, c.state))
	nextJSON, err := json.Marshal(si.next)
	require.NoError(t, err)
	b.state.Pool.Loans[0].Liquid.AddUint64(&b.state.Pool.Loans[0].Liquid, 7)
	b.state.Router.Venues[0].Morpho.Market.TotalBorrowAssets.AddUint64(&b.state.Router.Venues[0].Morpho.Market.TotalBorrowAssets, 7)
	b.state.Feed.Loans[0].Round.Answer.AddUint64(&b.state.Feed.Loans[0].Round.Answer, 7)
	b.state.Hooks.Swap.EverlongSwap.XWad.AddUint64(&b.state.Hooks.Swap.EverlongSwap.XWad, 7)
	b.state.Hooks.Spread.EverlongSpread.Spread.AddUint64(&b.state.Hooks.Spread.EverlongSpread.Spread, 7)
	after, err := json.Marshal(si.next)
	require.NoError(t, err)
	require.Equal(t, string(nextJSON), string(after), "UpdateBalance aliased SwapInfo.next")
	require.NotEqual(t, stateJSON(t, b), stateJSON(t, c), "sibling clones share adopted state")

	// Quoting on the adopted state is quoting on SwapInfo.next directly.
	direct := simFor(t, gridReads(t, "51313000", "armed"), Policy{LeverRouting: true})
	direct.state = si.next.clone()
	for _, k := range quoteCases() {
		r1, e1 := c.CalcAmountOut(amountIn(c, k.sell, k.a))
		r2, e2 := direct.CalcAmountOut(amountIn(direct, k.sell, k.a))
		g1, g2 := renderQuote(t, r1, e1), renderQuote(t, r2, e2)
		g1.Seq, g2.Seq = 0, 0
		require.Equal(t, g2, g1, "%+v", k)
	}
	require.Equal(t, PoolMeta{ApprovalAddress: sim.Info.Address, BlockNumber: sim.Info.BlockNumber}, c.GetMetaInfo("", ""))
	require.Equal(t, uint64(51313000), c.GetMetaInfo("", "").(PoolMeta).BlockNumber)

	// The reserves UpdateBalance publishes are a function of the adopted state alone: the capacity is recomputed at
	// the clock the fill settled at, so two clones that adopt one SwapInfo publish the same capacity however long
	// apart they adopt it.
	d, e := sim.CloneState().(*PoolSimulator), sim.CloneState().(*PoolSimulator)
	d.nowFn = func() uint64 { return si.next.Timestamp }
	e.nowFn = func() uint64 { return si.next.Timestamp + 7*86_400 }
	d.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: si})
	e.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: si})
	require.Equal(t, reserveStrings(d), reserveStrings(e), "the published reserves follow the wall clock")
	require.NotEqual(t, reserveStrings(sim), reserveStrings(d), "the fill moved neither published reserve")

	// And they are what the refresh publishes for that state at that clock (pool_tracker.go reservesOf).
	var gross uint256.Int
	pos, err := d.state.Router.positions(si.next.Timestamp)
	require.NoError(t, err)
	_, overflow := gross.AddOverflow(&d.state.Pool.Physical, &pos.TotalColl)
	require.False(t, overflow)
	require.Equal(t, []string(reservesOf(&gross, d.state, si.next.Timestamp)), reserveStrings(d))
}

// reserveStrings is a simulator's published reserves as decimal strings.
func reserveStrings(p *PoolSimulator) []string {
	out := make([]string, len(p.Info.Reserves))
	for i, r := range p.Info.Reserves {
		out[i] = r.String()
	}
	return out
}

// TestSimulatorCloneDeep: every word an execution or UpdateBalance can write, written in place on a clone, leaves the
// original's state and quotes untouched; and UpdateBalance on the original leaves an earlier clone untouched.
func TestSimulatorCloneDeep(t *testing.T) {
	t.Parallel()
	sim := simFor(t, gridReads(t, "51313000", "armed"), Policy{LeverRouting: true})
	cases := quoteCases()
	baseline := make([]simQuote, len(cases))
	for i, k := range cases {
		r, err := sim.CalcAmountOut(amountIn(sim, k.sell, k.a))
		baseline[i] = renderQuote(t, r, err)
	}
	before := stateJSON(t, sim)
	cl := sim.CloneState().(*PoolSimulator)
	s := cl.state
	bump := func(x *uint256.Int) { x.AddUint64(x, 1) }
	bump(&s.Pool.Physical)
	bump(&s.Pool.Loans[0].Liquid)
	bump(&s.Pool.Loans[0].MaxSwapNotional)
	bump(&s.Router.Loans[0].DebtCap)
	v := &s.Router.Venues[0]
	bump(&v.ManagedCollateral)
	bump(&v.ManagedSupplyShares)
	bump(&v.Morpho.Market.TotalSupplyAssets)
	bump(&v.Morpho.Market.TotalBorrowShares)
	bump(&v.Morpho.Position.Collateral)
	bump(&v.Morpho.RateAtTarget)
	bump(&s.Feed.Loans[0].Round.UpdatedAt)
	bump(&s.Feed.Asset.Round.Answer)
	bump(&s.Hooks.Swap.EverlongSwap.ReserveStable)
	// The spread half of poolHooks.clone (hook_kinds.go): the armed scenario lists a spread hook, and every word of
	// its post moves a lever quote (FLAMMLeverLib._spread).
	require.NotNil(t, s.Hooks.Spread.EverlongSpread, "the armed scenario lists a spread hook")
	bump(&s.Hooks.Spread.EverlongSpread.Spread)
	bump(&s.Hooks.Spread.EverlongSpread.LastSetTs)
	bump(&s.Hooks.Spread.EverlongSpread.MaxSpreadAge)
	bump(&s.LastLeverSpreadPpm)
	if s.Pool.PriceWad != nil {
		bump(&s.Pool.PriceWad[0])
	}
	s.Router.TransientRepay = map[uint16]mmRepaySnapshot{0: {}}
	cl.Info.Reserves[0].SetInt64(-1)
	require.Equal(t, before, stateJSON(t, sim), "clone shares state with the original")
	require.NotEqual(t, "-1", sim.Info.Reserves[0].String())
	for i, k := range cases {
		r, err := sim.CalcAmountOut(amountIn(sim, k.sell, k.a))
		require.Equal(t, baseline[i], renderQuote(t, r, err), "%+v after clone writes", k)
	}

	// The reverse direction: an earlier clone survives the original adopting a fill.
	early := sim.CloneState().(*PoolSimulator)
	earlyJSON := stateJSON(t, early)
	r, err := sim.CalcAmountOut(amountIn(sim, false, 5_000_000))
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: r.SwapInfo})
	require.Equal(t, earlyJSON, stateJSON(t, early))
	require.NotEqual(t, earlyJSON, stateJSON(t, sim))
}

// TestSimulatorResultShape: over every scenario state of every block, every successful quote (routing on) has a
// positive output equal to SwapInfo.AmountOut, a positive used input no larger than amountIn, RemainingTokenAmountIn
// exactly amountIn - SwapInfo.AmountInUsed, the gas of its venue and direction, and a fee in the output token.
func TestSimulatorResultShape(t *testing.T) {
	t.Parallel()
	tags := []string{"live", "armed", "armed_hi", "capped", "band_tight", "fee_cap", "fee_floor", "fee_floor_loan",
		"btc_down10", "btc_down25", "btc_up10", "ltv_low", "pin_low", "accrued", "stale_spread", "no_sell", "no_buy",
		"paused", "peg_broken"}
	counts := map[string]int{}
	combo := 0
	for _, blk := range e2eBlocks {
		for _, tag := range tags {
			sim := simFor(t, gridReads(t, blk, tag), Policy{LeverRouting: true, GasLeverDown: 77, GasSwapBuy: 55})
			combo++
			for ci, k := range quoteCases() {
				if !sampleCase(ci, combo, 4) { // every case of some scenario under -race (fixture_sample_test.go)
					continue
				}
				params := amountIn(sim, k.sell, k.a)
				r, err := sim.CalcAmountOut(params)
				if err != nil {
					require.Nil(t, r)
					counts["refused"]++
					continue
				}
				si := r.SwapInfo.(SwapInfo)
				where := fmt.Sprintf("%s/%s %+v", blk, tag, k)
				require.Equal(t, params.TokenOut, r.TokenAmountOut.Token, where)
				require.Equal(t, params.TokenOut, r.Fee.Token, where)
				require.Equal(t, params.TokenAmountIn.Token, r.RemainingTokenAmountIn.Token, where)
				require.Positive(t, r.TokenAmountOut.Amount.Sign(), where)
				require.Equal(t, si.AmountOut.Dec(), r.TokenAmountOut.Amount.String(), where)
				require.False(t, si.AmountInUsed.IsZero(), where)
				require.LessOrEqual(t, si.AmountInUsed.Uint64(), k.a, where)
				require.Equal(t, new(big.Int).Sub(new(big.Int).SetUint64(k.a), si.AmountInUsed.ToBig()).String(),
					r.RemainingTokenAmountIn.Amount.String(), where)
				require.Equal(t, k.sell, si.PoolAssetIn, where)
				require.GreaterOrEqual(t, r.Fee.Amount.Sign(), 0, where)
				switch {
				case si.Venue == VenueSwap && k.sell:
					// A clipped sell adds its cap solves and extra funding passes (constant.go).
					q, err := sim.quoteSwap(true, uint256.NewInt(k.a), sim.now())
					require.NoError(t, err, where)
					require.Equal(t, defaultGasSwapSell+int64(q.capEvals)*defaultGasSwapSellCapEval+
						int64(q.passes-1)*defaultGasSwapSellPass, r.Gas, where)
					if r.RemainingTokenAmountIn.Amount.Sign() == 0 {
						require.Zero(t, q.capEvals, where)
					}
				case si.Venue == VenueSwap:
					require.Equal(t, int64(55), r.Gas, where)
				case k.sell:
					require.Equal(t, defaultGasLeverUp, r.Gas, where)
					require.Zero(t, r.Fee.Amount.Sign(), where)
				default:
					require.Equal(t, int64(77), r.Gas, where)
				}
				key := fmt.Sprintf("venue%d/sell=%v", si.Venue, k.sell)
				if r.RemainingTokenAmountIn.Amount.Sign() > 0 {
					key += "/partial"
				}
				counts[key]++
			}
		}
	}
	t.Logf("result shapes: %v", counts)
	for _, k := range []string{"venue0/sell=true", "venue0/sell=false", "venue1/sell=true", "venue1/sell=false/partial",
		"venue0/sell=true/partial"} {
		require.Positive(t, counts[k], k)
	}
}

// TestSimulatorClockMonotone: a quote adopts the quote timestamp; a later quote with an earlier clock quotes at the
// adopted timestamp, never before it, and matches a simulator whose clock is that timestamp.
func TestSimulatorClockMonotone(t *testing.T) {
	t.Parallel()
	reads := gridReads(t, "51313000", "armed")
	sim := simFor(t, reads, Policy{LeverRouting: true})
	t1 := reads.Timestamp + 900
	sim.nowFn = func() uint64 { return t1 }
	r, err := sim.CalcAmountOut(amountIn(sim, true, 20_000))
	require.NoError(t, err)
	require.Equal(t, t1, r.SwapInfo.(SwapInfo).next.Timestamp)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: r.SwapInfo})
	require.Equal(t, t1, sim.state.Timestamp)
	ref := sim.CloneState().(*PoolSimulator)
	sim.nowFn = func() uint64 { return reads.Timestamp }
	ref.nowFn = func() uint64 { return t1 }
	for _, k := range quoteCases() {
		r1, e1 := sim.CalcAmountOut(amountIn(sim, k.sell, k.a))
		r2, e2 := ref.CalcAmountOut(amountIn(ref, k.sell, k.a))
		require.Equal(t, renderQuote(t, r2, e2), renderQuote(t, r1, e1), "%+v", k)
	}
	require.Equal(t, uint64(51313000), sim.GetMetaInfo("", "").(PoolMeta).BlockNumber)
}

// TestSimulatorInputContract: malformed inputs refuse without touching the state, the smallest amounts either
// refuse or pay consistently, and every SwapInfo shape UpdateBalance may be handed is adopted or leaves a refusing
// clone.
func TestSimulatorInputContract(t *testing.T) {
	t.Parallel()
	sim := simFor(t, gridReads(t, "51313000", "armed"), Policy{LeverRouting: true})
	before := stateJSON(t, sim)
	tokens := sim.Info.Tokens
	huge := new(big.Int).Lsh(big.NewInt(1), 256)
	for _, c := range []struct {
		name    string
		in, out string
		amount  *big.Int
	}{
		{"nil amount", tokens[0], tokens[1], nil},
		{"zero", tokens[0], tokens[1], big.NewInt(0)},
		{"negative", tokens[0], tokens[1], big.NewInt(-5)},
		{"2^256", tokens[0], tokens[1], huge},
		{"2^256 buy", tokens[1], tokens[0], huge},
		{"same token", tokens[0], tokens[0], big.NewInt(1000)},
		{"unknown in", "0x0000000000000000000000000000000000000001", tokens[1], big.NewInt(1000)},
		{"unknown out", tokens[1], "0x0000000000000000000000000000000000000001", big.NewInt(1000)},
		{"checksummed in", "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf", tokens[1], big.NewInt(1000)},
		{"2^256-1", tokens[0], tokens[1], new(big.Int).Sub(huge, big.NewInt(1))},
		{"2^255 buy", tokens[1], tokens[0], new(big.Int).Lsh(big.NewInt(1), 255)},
	} {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: c.in,
			Amount: c.amount}, TokenOut: c.out})
		t.Logf("%s: %v", c.name, describeQuote(res, err))
		require.Error(t, err, c.name)
	}
	require.Equal(t, before, stateJSON(t, sim))

	// The smallest amounts on every venue either refuse or pay a positive amount with a consistent remainder.
	for _, sell := range []bool{true, false} {
		for a := uint64(1); a <= 3000; a += 1 + a/7 {
			for _, venue := range []int{-1, int(VenueSwap), int(VenueLever)} {
				params := amountIn(sim, sell, a)
				res, err := sim.calcAmountOut(params, venue)
				if err != nil {
					continue
				}
				si := res.SwapInfo.(SwapInfo)
				require.Positive(t, res.TokenAmountOut.Amount.Sign())
				require.Zero(t, new(big.Int).Add(res.RemainingTokenAmountIn.Amount, si.AmountInUsed.ToBig()).
					Cmp(params.TokenAmountIn.Amount))
				require.Equal(t, si.AmountOut.ToBig().String(), res.TokenAmountOut.Amount.String())
			}
		}
	}

	// A pointer SwapInfo, a zero SwapInfo and a foreign type each leave a clone refusing, never half-applied.
	res, err := sim.CalcAmountOut(amountIn(sim, true, 15_000))
	require.NoError(t, err)
	si := res.SwapInfo.(SwapInfo)
	for name, info := range map[string]any{"pointer": &si, "zero": SwapInfo{}, "foreign": 7, "nil": nil} {
		c := sim.CloneState().(*PoolSimulator)
		c.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: info})
		_, err := c.CalcAmountOut(amountIn(c, true, 15_000))
		require.ErrorIs(t, err, ErrSwapInfoMismatch, name)
		require.Equal(t, before, stateJSON(t, c), name)
	}
	require.Equal(t, before, stateJSON(t, sim))
}

// sameQuote describes how two quotes differ (amounts, venue, post-state, and gas when asked); empty when they agree.
func sameQuote(t testing.TB, a, b *pool.CalcAmountOutResult, gas bool) string {
	sa, sb := a.SwapInfo.(SwapInfo), b.SwapInfo.(SwapInfo)
	ja, _ := json.Marshal(sa.next)
	jb, _ := json.Marshal(sb.next)
	switch {
	case a.TokenAmountOut.Amount.Cmp(b.TokenAmountOut.Amount) != 0:
		return fmt.Sprintf("out %s vs %s", a.TokenAmountOut.Amount, b.TokenAmountOut.Amount)
	case a.RemainingTokenAmountIn.Amount.Cmp(b.RemainingTokenAmountIn.Amount) != 0:
		return fmt.Sprintf("remaining %s vs %s", a.RemainingTokenAmountIn.Amount, b.RemainingTokenAmountIn.Amount)
	case sa.Venue != sb.Venue:
		return fmt.Sprintf("venue %d vs %d", sa.Venue, sb.Venue)
	case !bytes.Equal(ja, jb):
		return "post-state differs"
	case gas && a.Gas != b.Gas:
		return fmt.Sprintf("gas %d vs %d", a.Gas, b.Gas)
	}
	return ""
}

// TestSimulatorMarginsOnlyRefuse: per forced venue, a quote a margined simulator accepts is the unmargined quote
// (amounts, venue, post-state, and gas unless DebtDriftSec charges a re-plan); a routed margined quote is the
// unmargined quote of the venue it picked; a DebtDriftSec-accepted swap is unchanged at every intermediate second,
// not only at the drift's end.
func TestSimulatorMarginsOnlyRefuse(t *testing.T) {
	t.Parallel()
	counts := map[string]int{}
	var fails []string
	combo := 0
	for _, blk := range []string{"51302915", "51313000", "51324800"} {
		for _, tag := range []string{"live", "armed", "armed_hi", "capped", "band_tight", "btc_down10", "btc_up10",
			"fee_cap", "fee_floor", "ltv_low", "pin_low", "accrued", "stale_spread"} {
			reads := func() (r *flammReads) {
				defer func() {
					if recover() != nil {
						r = nil
					}
				}()
				return gridReads(t, blk, tag)
			}()
			if reads == nil {
				continue
			}
			base := simFor(t, reads, Policy{LeverRouting: true})
			for _, m := range []Policy{
				{LeverRouting: true, PriceBandMarginBps: 3},
				{LeverRouting: true, PriceBandMarginBps: 25, SpreadAgeMarginSec: 120},
				{LeverRouting: true, PriceAgeMarginSec: 300, SpreadAgeMarginSec: 5},
				{LeverRouting: true, DebtDriftSec: 400},
				{LeverRouting: true, DebtDriftSec: 60, PriceBandMarginBps: 1, PriceAgeMarginSec: 30},
				{LeverRouting: true, LeverMinEdgeBps: defaultLeverMinEdgeBps,
					PriceBandMarginBps: defaultPriceBandMarginBps, PriceAgeMarginSec: defaultPriceAgeMarginSec,
					SpreadAgeMarginSec: defaultSpreadAgeMarginSec, DebtDriftSec: defaultDebtDriftSec,
					MaxSnapshotAgeSec: defaultMaxSnapshotAgeSec}, // the shipped defaults
			} {
				ms := simFor(t, reads, m)
				now := reads.Timestamp
				combo++
				for ci, k := range quoteCases() {
					if !sampleCase(ci, combo, 16) { // every case of some scenario under -race (fixture_sample_test.go)
						continue
					}
					params := amountIn(base, k.sell, k.a)
					for _, venue := range []int{int(VenueSwap), int(VenueLever)} {
						rb, eb := base.calcAmountOut(params, venue)
						rm, em := ms.calcAmountOut(params, venue)
						if em != nil {
							counts["margin-refused"]++
							continue
						}
						where := fmt.Sprintf("%s/%s %+v venue %d sell=%v a=%d", blk, tag, m, venue, k.sell, k.a)
						if eb != nil {
							fails = append(fails, where+": margined accepts, plain refuses "+eb.Error())
							continue
						}
						if d := sameQuote(t, rm, rb, m.DebtDriftSec == 0); d != "" {
							fails = append(fails, where+": "+d)
							continue
						}
						if m.DebtDriftSec != 0 && rm.Gas < rb.Gas {
							fails = append(fails, fmt.Sprintf("%s: drift gas %d below plain %d", where, rm.Gas, rb.Gas))
						}
						counts["identical"]++
						// A drift-accepted swap: identical at every intermediate second (sampled), not only at the end.
						if m.DebtDriftSec != 0 && venue == int(VenueSwap) {
							for _, dt := range []uint64{1, m.DebtDriftSec / 3, m.DebtDriftSec / 2, m.DebtDriftSec - 1} {
								at := now + dt
								base.nowFn = func() uint64 { return at }
								rl, el := base.calcAmountOut(params, venue)
								base.nowFn = func() uint64 { return now }
								if el != nil {
									fails = append(fails, fmt.Sprintf("%s: drift-accepted swap refused at +%d: %v", where, dt, el))
									continue
								}
								if rl.TokenAmountOut.Amount.Cmp(rb.TokenAmountOut.Amount) != 0 ||
									rl.RemainingTokenAmountIn.Amount.Cmp(rb.RemainingTokenAmountIn.Amount) != 0 {
									fails = append(fails, fmt.Sprintf("%s: drift-accepted swap at +%d pays %s unused %s, quoted %s unused %s",
										where, dt, rl.TokenAmountOut.Amount, rl.RemainingTokenAmountIn.Amount,
										rb.TokenAmountOut.Amount, rb.RemainingTokenAmountIn.Amount))
									continue
								}
								counts["drift-intermediate-identical"]++
							}
						}
					}
					// routed: the margined pick is the plain quote of the venue the *plain* routed quote picked -- a
					// margin may refuse the routed quote, but moving it to the other venue would be a margin changing
					// an amount (pool_simulator.go quote).
					rm, em := ms.CalcAmountOut(params)
					if em == nil {
						v := int(rm.SwapInfo.(SwapInfo).Venue)
						rp, ep := base.CalcAmountOut(params)
						switch {
						case ep != nil:
							fails = append(fails, fmt.Sprintf("%s/%s %+v routed a=%d: plain routing refuses %v", blk, tag,
								m, k.a, ep))
						case int(rp.SwapInfo.(SwapInfo).Venue) != v:
							fails = append(fails, fmt.Sprintf("%s/%s %+v routed a=%d sell=%v: the margin moved the pick "+
								"from venue %d to venue %d", blk, tag, m, k.a, k.sell, rp.SwapInfo.(SwapInfo).Venue, v))
						default:
							rb, eb := base.calcAmountOut(params, v)
							if eb != nil {
								fails = append(fails, fmt.Sprintf("%s/%s %+v routed a=%d: plain refuses venue %d", blk, tag, m, k.a, v))
							} else if d := sameQuote(t, rm, rb, m.DebtDriftSec == 0); d != "" {
								fails = append(fails, fmt.Sprintf("%s/%s %+v routed a=%d sell=%v: %s", blk, tag, m, k.a, k.sell, d))
							} else {
								counts["routed-identical"]++
							}
						}
					}
				}
			}
		}
	}
	t.Logf("counts %v; %d failures", counts, len(fails))
	for i, f := range fails {
		if i < 40 {
			t.Error(f)
		}
	}
}

// TestSimulatorUpdateAcrossMsgpack: quote, encode, decode, then adopt the pre-encoding SwapInfo on the decoded
// simulator and on the original: both keep quoting identically over a mixed sequence.
func TestSimulatorUpdateAcrossMsgpack(t *testing.T) {
	t.Parallel()
	enc := func(s *PoolSimulator) *PoolSimulator {
		var buf bytes.Buffer
		e := msgpack.NewEncoder(&buf)
		e.IncludeUnexported(true)
		e.SetForceAsArray(true)
		require.NoError(t, e.Encode(s))
		d := msgpack.NewDecoder(&buf)
		d.IncludeUnexported(true)
		d.SetForceAsArray(true)
		var out PoolSimulator
		require.NoError(t, d.Decode(&out))
		return &out
	}
	sim := simFor(t, gridReads(t, "51313000", "armed"), Policy{LeverRouting: true})
	now := sim.state.Timestamp
	steps := []struct {
		venue int
		sell  bool
		a     uint64
	}{{1, true, 6000}, {0, false, 20_000_000}, {-1, true, 30_000}, {1, false, 15_000_000}, {0, true, 90_000},
		{-1, false, 5_000_000}, {1, true, 2_000}, {0, false, 70_000_000}}
	a, b := sim, sim.CloneState().(*PoolSimulator)
	adopted := 0
	for i, s := range steps {
		at := now + uint64(3*i)
		a.nowFn = func() uint64 { return at }
		ra, ea := a.calcAmountOut(amountIn(a, s.sell, s.a), s.venue)
		decoded := enc(b)
		decoded.nowFn = func() uint64 { return at }
		b.nowFn = func() uint64 { return at }
		rb, eb := b.calcAmountOut(amountIn(b, s.sell, s.a), s.venue)
		require.Equal(t, ea == nil, eb == nil, "step %d", i)
		if ea != nil {
			require.Equal(t, ea.Error(), eb.Error())
			continue
		}
		require.Empty(t, sameQuote(t, ra, rb, true), "step %d", i)
		a.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: ra.SwapInfo})
		decoded.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: rb.SwapInfo}) // quoted on b before encoding
		require.False(t, decoded.broken, "step %d: the decoded simulator refused a SwapInfo quoted before the hop", i)
		require.Equal(t, a.seq, decoded.seq)
		b = decoded
		adopted++
		require.Equal(t, stateJSON(t, a), stateJSON(t, b), "step %d", i)
	}
	require.GreaterOrEqual(t, adopted, 5)
}

// TestSimulatorFeedRoundMovesQuotes measures how much one cbBTC/USD round the tracker has not seen moves a quote: the
// pool prices both venues at the feed cross (FLAMMSwapLib toN18/fromN18 at the checked cross, the band, the
// ceilings), so a refresh that is not triggered by the aggregator's round keeps quoting the old price.
func TestSimulatorFeedRoundMovesQuotes(t *testing.T) {
	t.Parallel()
	reads := gridReads(t, "51313000", "armed")
	base := simFor(t, reads, Policy{LeverRouting: true})
	moved := *reads
	moved.Feed.Tokens = append([]feedTokenReads(nil), reads.Feed.Tokens...)
	ans := moved.Feed.Tokens[0].Round.Answer.ToBig()
	for _, bps := range []int64{10, 50, -50} {
		a := new(big.Int).Div(new(big.Int).Mul(ans, big.NewInt(10_000+bps)), big.NewInt(10_000))
		var w int256.Int
		require.False(t, w.SetFromBig(a))
		moved.Feed.Tokens[0].Round.Answer = w
		sim := simFor(t, &moved, Policy{LeverRouting: true})
		diff, same, flips := 0, 0, 0
		var example string
		for _, k := range quoteCases() {
			for _, venue := range []int{int(VenueSwap), int(VenueLever)} {
				rb, eb := base.calcAmountOut(amountIn(base, k.sell, k.a), venue)
				rm, em := sim.calcAmountOut(amountIn(sim, k.sell, k.a), venue)
				switch {
				case (eb == nil) != (em == nil):
					flips++
				case eb != nil:
				case rb.TokenAmountOut.Amount.Cmp(rm.TokenAmountOut.Amount) != 0:
					diff++
					if example == "" {
						example = fmt.Sprintf("venue %d sell=%v amountIn %d: snapshot price pays %s, the new round pays %s",
							venue, k.sell, k.a, rb.TokenAmountOut.Amount, rm.TokenAmountOut.Amount)
					}
				default:
					same++
				}
			}
		}
		t.Logf("cbBTC answer %+d bps: %d quotes change amount, %d unchanged, %d flip accept/refuse; e.g. %s", bps, diff,
			same, flips, example)
	}
}
