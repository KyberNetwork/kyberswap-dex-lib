package everlongflamm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Executor-path parity on an anvil fork of Base (skipped unless EVERLONG_FLAMM_FORK_RPC and EVERLONG_ADAPTER_OUT,
// the EverlongFlammAdapter forge artifact, are set; EVERLONG_FLAMM_FORK_BLOCK overrides the pinned block):
//
//   - single fills: the real lister -> tracker -> simulator quote against executeEverlongFlamm eth_calls at the
//     tracked block over grids in both directions, band edges included -- amountOut and amountUnused identical,
//     every refusal a revert with the same error -- on the live pool and with a notional cap that clips sells;
//   - sequential fills: mixed fills mined one after another at the timestamps the simulator quoted them at, the
//     simulator advanced only by UpdateBalance; after them a fresh refresh must equal the simulator's state;
//   - the leverage venue after arming it (curator unpause, keeper spread), through adapter venue 1, a lever-down's
//     amountUnused being amountIn - payNative;
//   - the receipt gas of every mined fill within the simulator's estimate (fork_gas_test.go measures the defaults).

const forkDefaultBlock = 51313000

type forkEnv struct {
	f        *anvilFork
	code     []byte
	pool     common.Address
	cbBTC    common.Address
	usdc     common.Address
	listed   entity.Pool
	tracked  entity.Pool // the last refresh track published
	gas      map[string]uint64
	matchedN int
}

func newForkEnv(t *testing.T) *forkEnv {
	t.Helper()
	upstream, artifact := os.Getenv("EVERLONG_FLAMM_FORK_RPC"), os.Getenv("EVERLONG_ADAPTER_OUT")
	if upstream == "" || artifact == "" {
		t.Skip("EVERLONG_FLAMM_FORK_RPC / EVERLONG_ADAPTER_OUT not set")
	}
	block := uint64(forkDefaultBlock)
	if b := os.Getenv("EVERLONG_FLAMM_FORK_BLOCK"); b != "" {
		v, err := strconv.ParseUint(b, 10, 64)
		require.NoError(t, err)
		block = v
	}
	f := startFork(t, upstream, block)
	e := &forkEnv{f: f, pool: c104.Pool, cbBTC: c104.PoolAsset,
		usdc: c104.LoanAsset, gas: map[string]uint64{}}
	e.code = f.placeAdapter(artifact)
	// Freeze the venue market oracles at the answer they give at the fork head, so a Chainlink SVR reveal upstream
	// cannot land inside a mined sequence (fork_harness_test.go pinOracles).
	f.pinOracles(t, &c104)
	pools, _, err := NewPoolsListUpdater(baseConfig(), f.client).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	e.listed = pools[0]
	return e
}

// newForkEnvAt is newForkEnv at defaultBlock, unless EVERLONG_FLAMM_FORK_BLOCK overrides it.
func newForkEnvAt(t *testing.T, defaultBlock string) *forkEnv {
	t.Helper()
	if os.Getenv("EVERLONG_FLAMM_FORK_BLOCK") == "" {
		t.Setenv("EVERLONG_FLAMM_FORK_BLOCK", defaultBlock)
	}
	return newForkEnv(t)
}

// track refreshes at the fork head and returns an attested simulator quoting at the head's timestamp.
func (e *forkEnv) track(t *testing.T, policy Policy) *PoolSimulator {
	t.Helper()
	cfg := parityConfig(policy)
	tracked, err := NewPoolTracker(cfg, e.f.client).GetNewPoolState(context.Background(), e.listed,
		pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.True(t, extra.Attested, "fork refresh: attest %q drift %q", extra.AttestFailure, extra.ProfileDrift)
	e.tracked = tracked
	sim, err := NewPoolSimulator(tracked)
	require.NoError(t, err)
	ts := sim.state.Timestamp
	sim.nowFn = func() uint64 { return ts }
	t.Logf("refresh at %d (ts %d): %d probes reproduced", tracked.BlockNumber, ts, extra.Probes)
	return sim
}

func (e *forkEnv) tokens(sell bool) (common.Address, common.Address) {
	if sell {
		return e.cbBTC, e.usdc
	}
	return e.usdc, e.cbBTC
}

// singleFill compares one quote on venue with the adapter's eth_call at the tracked block.
func (e *forkEnv) singleFill(t *testing.T, sim *PoolSimulator, venue uint8, sell bool, amount *big.Int) {
	t.Helper()
	params := amountIn(sim, sell, 1)
	params.TokenAmountIn.Amount = amount
	res, err := sim.calcAmountOut(params, int(venue))
	in, out := e.tokens(sell)
	fill := e.f.callFill(e.pool, venue, in, out, amount, e.code)
	where := fmt.Sprintf("venue %d sell=%v amount %s", venue, sell, amount)
	if fill.reverted {
		want := forkRevertError(fill.revert)
		require.NotNil(t, want, "%s: unmapped adapter revert %x (sim %v)", where, fill.revert, err)
		if !errors.Is(err, want) {
			require.ErrorIs(t, err, ErrSpreadNotLive, "%s: adapter reverted %v, simulator %v", where, want, err)
			require.False(t, chainSpreadLive(sim, sim.now()), "%s: spread refusal while the pool's spread is live",
				where)
		}
		return
	}
	require.NoError(t, err, "%s: adapter filled (unused %s out %s)", where, fill.unused, fill.out)
	require.Equal(t, fill.out.String(), res.TokenAmountOut.Amount.String(), "%s amountOut", where)
	require.Equal(t, fill.unused.String(), res.RemainingTokenAmountIn.Amount.String(), "%s amountUnused", where)
	require.Equal(t, venue, res.SwapInfo.(SwapInfo).Venue)
	e.matchedN++
}

// grid is the log grid plus each side of the port's own band edge.
func (e *forkEnv) grid(sim *PoolSimulator, venue uint8, sell bool) []*big.Int {
	lo, hi := uint64(1), uint64(2_000_000)
	if !sell {
		lo, hi = 100, 1_000_000_000
	}
	var out []*big.Int
	for _, a := range liveGrid(lo, hi, 14) {
		out = append(out, new(big.Int).SetUint64(a))
	}
	grid := probeSellGrid
	if !sell {
		grid = probeBuyGrid
	}
	now := sim.now()
	if l, h, ok := edge(grid, func(a *uint256.Int) bool {
		var err error
		if venue == VenueSwap {
			_, err = sim.quoteSwap(sell, a, now)
		} else {
			_, err = sim.quoteLever(sell, a, now)
		}
		return err == nil
	}); ok {
		out = append(out, l.ToBig(), h.ToBig())
	}
	return out
}

// TestForkRecordArmedTape records testdata/tracker_rpc_armed_51313004.json.gz (EVERLONG_FLAMM_RECORD=armed and the
// fork environment): the lister and a refresh of the pool at 51313000 with the leverage venue armed as
// TestForkEdges arms it (curator unpause, spread bounds widened, a 3600 s staleness window, a 13,000 ppm keeper
// post), so the refresh's probes include previewLever. TestTrackerReplayArmed replays it offline.
func TestForkRecordArmedTape(t *testing.T) {
	if os.Getenv("EVERLONG_FLAMM_RECORD") != "armed" {
		t.Skip("EVERLONG_FLAMM_RECORD=armed not set")
	}
	e := newForkEnv(t)
	require.Equal(t, uint64(forkDefaultBlock), e.f.head().Number.Uint64(), "record from the default fork block")
	e.armLeverage(t, 13_000, 3600)
	head := e.f.head().Number.Uint64()
	require.Equal(t, uint64(armedTapeBlock), head)
	tp := recordTape(t, armedTape, e.f.url)
	pools, _, err := NewPoolsListUpdater(tapeConfig(), tp.client()).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	tracked, err := NewPoolTracker(parityConfig(Policy{LeverRouting: true}), tp.client()).GetNewPoolStateAtBlock(
		context.Background(), pools[0], new(big.Int).SetUint64(head))
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.True(t, extra.Attested, "attest %q drift %q", extra.AttestFailure, extra.ProfileDrift)
	tp.save()
	t.Logf("recorded %s at %d: %d probes", armedTape, head, extra.Probes)
}

// seqFill quotes on sim at ts -- the first of the amounts the venue fills -- mines the fill through the adapter at ts
// and adopts the quote.
func (e *forkEnv) seqFill(t *testing.T, sim *PoolSimulator, venue int, sell bool, amounts []uint64, ts uint64) bool {
	t.Helper()
	sim.nowFn = func() uint64 { return ts }
	var res *pool.CalcAmountOutResult
	var err error
	var amount uint64
	var params pool.CalcAmountOutParams
	for _, amount = range amounts {
		params = amountIn(sim, sell, amount)
		if res, err = sim.calcAmountOut(params, venue); err == nil {
			break
		}
	}
	if err != nil && venue == int(VenueLever) {
		// The leverage venue refuses at this book (its taker band or curve); the next swap moves the book.
		t.Logf("sequential lever sell=%v at %d: not fillable at %v (%v)", sell, ts, amounts, err)
		return false
	}
	require.NoError(t, err, "sequential quote venue %d sell=%v %v", venue, sell, amounts)
	si := res.SwapInfo.(SwapInfo)
	in, out := e.tokens(sell)
	fill := e.f.execFill(e.pool, si.Venue, in, out, params.TokenAmountIn.Amount, ts)
	where := fmt.Sprintf("sequential venue %d sell=%v %d at %d", si.Venue, sell, amount, ts)
	require.Equal(t, fill.out.String(), res.TokenAmountOut.Amount.String(), "%s amountOut", where)
	require.Equal(t, fill.unused.String(), res.RemainingTokenAmountIn.Amount.String(), "%s amountUnused", where)
	require.LessOrEqual(t, fill.gas, uint64(res.Gas), "%s: gas above the estimate", where)
	key := fmt.Sprintf("venue%d/%s", si.Venue, map[bool]string{true: "sell", false: "buy"}[sell])
	e.gas[key] = max(e.gas[key], fill.gas)
	t.Logf("%s: out %s unused %s gas %d", where, fill.out, fill.unused, fill.gas)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: si})
	return true
}

// sameState requires the simulator's state to equal a fresh refresh's, field for field, and the reserves
// UpdateBalance published to be the ones the refresh itself publishes for the same chain state: the fresh
// simulator's own Info.Reserves, which are the entity's (pool_tracker.go publish / reservesOf), not a
// recomputation of them here. The refresh reads the block the last fill was mined in, so the two are the same
// clock, and sameState asserts that rather than assuming it -- a later refresh would re-price the capacity at its
// own clock and the comparison would be against a different number.
func sameState(t *testing.T, sim, fresh *PoolSimulator) {
	t.Helper()
	got := sim.state.clone()
	got.Block, got.Timestamp = fresh.state.Block, fresh.state.Timestamp
	require.Empty(t, e2eDiff(got, fresh.state), "simulator state after UpdateBalance vs a refresh of the fork")
	require.Len(t, sim.Info.Reserves, 2, "the simulator publishes both reserves")
	require.Len(t, fresh.Info.Reserves, 2, "the refresh publishes both reserves")
	require.Equal(t, fresh.state.Timestamp, sim.state.Timestamp,
		"the refresh read the block the last fill settled in")
	require.Equal(t, reserveStrings(fresh), reserveStrings(sim),
		"reserves published after UpdateBalance vs the refresh's own of the fork's post-fill state")
}

func TestForkParity(t *testing.T) {
	e := newForkEnv(t)
	f := e.f
	prof := &c104

	// ---- single fills on the pool as deployed (leverage paused), both directions
	sim := e.track(t, Policy{})
	for _, sell := range []bool{true, false} {
		for _, a := range e.grid(sim, VenueSwap, sell) {
			e.singleFill(t, sim, VenueSwap, sell, a)
		}
	}
	t.Logf("live single fills identical: %d", e.matchedN)
	require.Positive(t, e.matchedN)
	for _, up := range []bool{true, false} {
		e.singleFill(t, sim, VenueLever, up, big.NewInt(5_000)) // LevPaused through the adapter
	}

	// ---- arm the leverage venue (curator unpause, keeper spread) and fill venue 1
	core := f.view(e.pool, "core")[0].(common.Address)
	curator := f.view(core, "owner")[0].(common.Address)
	keeper := f.view(core, "keeper")[0].(common.Address)
	f.impersonate(curator)
	f.impersonate(keeper)
	head := f.head().Time
	unpause, err := forkABI.Pack("setLevPaused", false)
	require.NoError(t, err)
	f.send(curator, e.pool, unpause, head+2)
	spread, err := forkABI.Pack("setSpread", big.NewInt(17_500))
	require.NoError(t, err)
	f.send(keeper, prof.SpreadHook, spread, head+4)
	armed := e.track(t, Policy{LeverRouting: true})
	require.False(t, armed.state.LevPaused)
	levMatched := map[bool]int{}
	for _, up := range []bool{true, false} {
		for _, a := range e.grid(armed, VenueLever, up) {
			before := e.matchedN
			e.singleFill(t, armed, VenueLever, up, a)
			levMatched[up] += e.matchedN - before
		}
	}
	t.Logf("leverage single fills identical: up %d, down %d", levMatched[true], levMatched[false])
	require.Positive(t, levMatched[true])
	require.Positive(t, levMatched[false])
	// A lever-down pulls only payNative.
	down, err := armed.calcAmountOut(amountIn(armed, false, 10_000_000), int(VenueLever))
	require.NoError(t, err)
	require.Positive(t, down.RemainingTokenAmountIn.Amount.Sign())
	e.singleFill(t, armed, VenueLever, false, big.NewInt(10_000_000))

	// ---- sequential mixed fills on both venues, the simulator advanced only by UpdateBalance, then a refresh
	ups := []uint64{9_000, 6_000, 4_000, 2_500, 1_500, 1_000, 600}
	downs := []uint64{20_000_000, 10_000_000, 5_000_000, 2_500_000, 1_200_000, 600_000, 300_000, 150_000}
	steps := []struct {
		venue   int
		sell    bool
		amounts []uint64
	}{
		{int(VenueLever), true, ups}, {int(VenueSwap), true, []uint64{15_000}}, {int(VenueLever), false, downs},
		{int(VenueSwap), false, []uint64{20_000_000}}, {int(VenueSwap), true, []uint64{60_000}},
		{int(VenueLever), true, ups}, {-1, false, []uint64{8_000_000}}, {int(VenueSwap), false, []uint64{150_000_000}},
		{int(VenueLever), false, downs}, {-1, true, []uint64{4_000}}, {int(VenueSwap), true, []uint64{90_000}},
		{int(VenueLever), false, downs}, {int(VenueSwap), false, []uint64{45_000_000}}, {int(VenueLever), true, ups},
		{int(VenueSwap), true, []uint64{30_000}}, {int(VenueLever), false, downs}, {int(VenueLever), true, ups},
	}
	base := armed.state.Timestamp
	levFills := map[bool]int{}
	for i, s := range steps {
		if e.seqFill(t, armed, s.venue, s.sell, s.amounts, base+uint64(6*(i+1))) && s.venue == int(VenueLever) {
			levFills[s.sell]++
		}
	}
	require.GreaterOrEqual(t, levFills[true], 2, "sequential lever-ups")
	require.GreaterOrEqual(t, levFills[false], 2, "sequential lever-downs")
	fresh := e.track(t, Policy{LeverRouting: true})
	sameState(t, armed, fresh)

	// ---- a notional cap that clips sells: partial single fills, a sequential partial fill, then a refresh
	loan := fresh.state.Pool.Loans[0]
	capData, err := forkABI.Pack("setLoanConfig", uint8(0), loan.SwapPriceBandWad.Uint64(), loan.FeeFloorWad.Uint64(),
		big.NewInt(8_000_000), loan.ReserveTarget.ToBig())
	require.NoError(t, err)
	f.send(curator, e.pool, capData, fresh.state.Timestamp+5)
	capped := e.track(t, Policy{})
	partial := 0
	for _, a := range []uint64{5_000, 10_000, 11_000, 20_000, 50_000, 100_000} {
		before := e.matchedN
		e.singleFill(t, capped, VenueSwap, true, new(big.Int).SetUint64(a))
		if e.matchedN > before {
			r, _ := capped.CalcAmountOut(amountIn(capped, true, a))
			if r != nil && r.RemainingTokenAmountIn.Amount.Sign() > 0 {
				partial++
			}
		}
	}
	require.Positive(t, partial, "the cap must clip a sell")
	e.seqFill(t, capped, int(VenueSwap), true, []uint64{50_000}, capped.state.Timestamp+3)
	e.seqFill(t, capped, int(VenueSwap), false, []uint64{7_000_000}, capped.state.Timestamp+6)
	sameState(t, capped, e.track(t, Policy{}))

	t.Logf("fork parity: %d single fills identical; receipt gas maxima %v", e.matchedN, e.gas)
	for _, k := range []string{"venue0/sell", "venue0/buy", "venue1/sell", "venue1/buy"} {
		require.Contains(t, e.gas, k)
	}
}
