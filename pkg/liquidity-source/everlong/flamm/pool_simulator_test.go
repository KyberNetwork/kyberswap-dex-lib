package everlongflamm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// Simulator tests over states read exactly as the tracker reads them: the tracked entity at 51302915 and the
// core end-to-end fixture's scenario states (armed leverage, stale spread, paused, stale feed, sequencer grace,
// IRM outage, ...), whose previewSwap / previewLever answers and executed sequences come from the deployed pool.

// simEntity is a tracked entity for reads: the listing's StaticExtra and an attested Extra with the tracked
// entity's venue set.
func simEntity(t testing.TB, reads *flammReads, policy Policy) entity.Pool {
	stored := loadTracked(t)
	extra := Extra{Reads: reads, Venues: entityVenues(t, stored), Attested: true, Probes: 1, Policy: policy}
	raw, err := json.Marshal(&extra)
	require.NoError(t, err)
	stored.Extra = string(raw)
	stored.BlockNumber = reads.Block
	stored.Reserves = entity.PoolReserves{reads.Pool.Gross.Dec(), "0"}
	return stored
}

func simFor(t testing.TB, reads *flammReads, policy Policy) *PoolSimulator {
	t.Helper()
	sim, err := NewPoolSimulator(simEntity(t, reads, policy))
	require.NoError(t, err)
	ts := reads.Timestamp
	sim.nowFn = func() uint64 { return ts }
	return sim
}

func amountIn(sim *PoolSimulator, sell bool, a uint64) pool.CalcAmountOutParams {
	in, out := sim.Info.Tokens[0], sim.Info.Tokens[1]
	if !sell {
		in, out = out, in
	}
	return pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: new(big.Int).SetUint64(a)},
		TokenOut: out}
}

func stateJSON(t *testing.T, sim *PoolSimulator) string {
	t.Helper()
	raw, err := json.Marshal(sim.state)
	require.NoError(t, err)
	return string(raw)
}

func TestSimulatorRealSell(t *testing.T) {
	t.Parallel()
	sim, err := NewPoolSimulator(loadTracked(t))
	require.NoError(t, err)
	ts := sim.state.Timestamp
	sim.nowFn = func() uint64 { return ts }
	testutil.TestCalcAmountOut(t, sim, map[int]map[int]map[string]string{
		0: {1: {"15000": "11301759", "0": ErrInvalidAmount.Error()}},
		1: {0: {"1": ErrFillInvalid.Error()}},
	})
	require.Equal(t, PoolMeta{ApprovalAddress: sim.Info.Address, BlockNumber: 51302915}, sim.GetMetaInfo("", ""))
	require.Equal(t, sim.Info.Address, sim.GetApprovalAddress(sim.Info.Tokens[0], sim.Info.Tokens[1]))
	require.Equal(t, []string{sim.Info.Tokens[1]}, sim.CanSwapTo(sim.Info.Tokens[0]))
	// exact-input only: the dex must not be advertised to router-service's exact-out finder (AGENTS.md:110).
	require.NotContains(t, pool.CanCalcAmountIn, DexType)
	require.NotImplements(t, (*pool.IPoolExactOutSimulator)(nil), sim)
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: big.NewInt(1)}, TokenOut: sim.Info.Tokens[0]})
	require.ErrorIs(t, err, ErrInvalidToken)
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: new(big.Int).Lsh(big.NewInt(1), 256)},
		TokenOut:      sim.Info.Tokens[1]})
	require.ErrorIs(t, err, ErrInvalidAmount)
}

// TestSimulatorFactoryStaleCheck: the registered factory honours pool.FactoryOpts.StaleCheck, which route finding
// sets and indexing does not. The tracked fixture's snapshot is long past its MaxSnapshotAgeSec at the wall clock.
func TestSimulatorFactoryStaleCheck(t *testing.T) {
	t.Parallel()
	tracked := loadTracked(t)
	sim, err := NewPoolSimulatorFromParams(pool.FactoryParams{EntityPool: tracked})
	require.NoError(t, err) // indexing still gets a simulator
	require.Equal(t, uint64(defaultMaxSnapshotAgeSec), sim.Policy.MaxSnapshotAgeSec)
	_, err = sim.CalcAmountOut(amountIn(sim, true, 15_000))
	require.ErrorIs(t, err, ErrSnapshotStale) // which refuses every quote anyway

	_, err = NewPoolSimulatorFromParams(pool.FactoryParams{EntityPool: tracked,
		Opts: pool.FactoryOpts{StaleCheck: true}})
	require.ErrorIs(t, err, ErrSnapshotStale) // route finding gets none

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	extra.Reads.Timestamp = uint64(time.Now().Unix())
	_, err = NewPoolSimulatorFromParams(pool.FactoryParams{EntityPool: simEntity(t, extra.Reads, extra.Policy),
		Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)

	// An explicit 0 turns the age off on both paths.
	_, err = NewPoolSimulatorFromParams(pool.FactoryParams{EntityPool: simEntity(t, extra.Reads, Policy{}),
		Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
}

// TestSimulatorPurity: quoting never writes the state, succeeds or fails identically every time and is race free.
func TestSimulatorPurity(t *testing.T) {
	t.Parallel()
	sim := simFor(t, gridReads(t, "51313000", "armed"), Policy{LeverRouting: true})
	before := stateJSON(t, sim)
	for _, c := range []struct {
		sell bool
		a    uint64
	}{{true, 15000}, {false, 20_000_000}, {true, 10_000_000}, {false, 1}, {true, 5000}, {false, 5_000_000}} {
		first, err1 := sim.CalcAmountOut(amountIn(sim, c.sell, c.a))
		for i := 0; i < 3; i++ {
			again, err2 := sim.CalcAmountOut(amountIn(sim, c.sell, c.a))
			require.Equal(t, fmt.Sprint(err1), fmt.Sprint(err2))
			if err1 == nil {
				require.Equal(t, first.TokenAmountOut.Amount.String(), again.TokenAmountOut.Amount.String())
				require.Equal(t, first.RemainingTokenAmountIn.Amount.String(), again.RemainingTokenAmountIn.Amount.String())
				require.Equal(t, first.SwapInfo.(SwapInfo).Venue, again.SwapInfo.(SwapInfo).Venue)
			}
		}
		require.Equal(t, before, stateJSON(t, sim), "quote %+v wrote the state", c)
		_, _ = testutil.MustConcurrentSafe(t, func() (string, error) {
			r, err := sim.CalcAmountOut(amountIn(sim, c.sell, c.a))
			if err != nil {
				return "", err
			}
			return r.TokenAmountOut.Amount.String() + "/" + r.RemainingTokenAmountIn.Amount.String(), nil
		})
	}
	require.Equal(t, before, stateJSON(t, sim))
}

// TestSimulatorCloneAndUpdate: UpdateBalance adopts the quoted settlement, a clone is isolated, and a SwapInfo quoted
// on another state is refused.
func TestSimulatorCloneAndUpdate(t *testing.T) {
	t.Parallel()
	sim := simFor(t, gridReads(t, "51302915", "live"), Policy{})
	params := amountIn(sim, true, 15000)
	testutil.TestCloneState(t, sim, params, nil)

	res, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	si := res.SwapInfo.(SwapInfo)
	require.Equal(t, VenueSwap, si.Venue)
	clone := sim.CloneState().(*PoolSimulator)
	clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: si})
	require.NotEqual(t, stateJSON(t, sim), stateJSON(t, clone))
	require.Empty(t, e2eDiff(clone.state, si.next))
	require.NotSame(t, si.next, clone.state, "UpdateBalance adopts a copy")
	require.Equal(t, "234423", clone.Info.Reserves[0].String())

	// The same SwapInfo on the original (same seq) is fine; replaying it on the advanced clone is not.
	orig := sim.CloneState().(*PoolSimulator)
	orig.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: si})
	require.Equal(t, stateJSON(t, clone), stateJSON(t, orig))
	clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: si})
	_, err = clone.CalcAmountOut(params)
	require.ErrorIs(t, err, ErrSwapInfoMismatch)
	bad := sim.CloneState().(*PoolSimulator)
	bad.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: "not a swap info"})
	_, err = bad.CalcAmountOut(params)
	require.ErrorIs(t, err, ErrSwapInfoMismatch)
	// A SwapInfo from another refresh at the same seq (an older decoded copy of the pool) is not this state's.
	other := simFor(t, gridReads(t, "51313000", "live"), Policy{})
	oldRes, err := other.CalcAmountOut(amountIn(other, true, 15000))
	require.NoError(t, err)
	stale := sim.CloneState().(*PoolSimulator)
	stale.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: oldRes.SwapInfo})
	_, err = stale.CalcAmountOut(params)
	require.ErrorIs(t, err, ErrSwapInfoMismatch)

	// The second fill on the updated state is the pool's second fill (CoreE2ESeq basic step 1: 20 USDC -> 25653).
	next, err := orig.CalcAmountOut(amountIn(orig, false, 20_000_000))
	require.NoError(t, err)
	require.Equal(t, "25653", next.TokenAmountOut.Amount.String())

	// Two clones that adopted different fills have the same transition count and snapshot block but not the same
	// state: a quote made on one is refused by the other, and accepted by a copy of the state it was quoted on.
	a, b := sim.CloneState().(*PoolSimulator), sim.CloneState().(*PoolSimulator)
	for _, c := range []struct {
		s    *PoolSimulator
		sell bool
		amt  uint64
	}{{a, true, 100_000}, {b, false, 1_000_000}} {
		r, err := c.s.CalcAmountOut(amountIn(c.s, c.sell, c.amt))
		require.NoError(t, err)
		c.s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: r.SwapInfo})
	}
	require.Equal(t, a.seq, b.seq)
	fromA, err := a.CalcAmountOut(amountIn(a, true, 1_000))
	require.NoError(t, err)
	sibling := b.CloneState().(*PoolSimulator)
	sibling.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: fromA.SwapInfo})
	_, err = sibling.CalcAmountOut(amountIn(sibling, true, 30_000))
	require.ErrorIs(t, err, ErrSwapInfoMismatch, "a sibling's SwapInfo replaced this clone's own fill")
	copyA := a.CloneState().(*PoolSimulator)
	copyA.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: fromA.SwapInfo})
	_, err = copyA.CalcAmountOut(amountIn(copyA, true, 30_000))
	require.NoError(t, err)
	require.Empty(t, e2eDiff(copyA.state, fromA.SwapInfo.(SwapInfo).next))
}

// TestSimulatorPickVenue: the larger output wins, a tie keeps the swap (whose venue, gas and unused input the
// quote then reports), and a refusal loses to a fill; when both refuse the swap's refusal is reported.
func TestSimulatorPickVenue(t *testing.T) {
	t.Parallel()
	fillOf := func(venue uint8, out uint64) *fill {
		return &fill{venue: venue, out: *uint256.NewInt(out)}
	}
	errSwap, errLever := errors.New("swap refused"), errors.New("lever refused")
	for _, c := range []struct {
		name      string
		sw, lv    uint64
		swE, lvE  error
		edge      uint64
		wantVenue uint8
		wantErr   error
	}{
		{"tie", 100, 100, nil, nil, 0, VenueSwap, nil},
		{"lever larger", 100, 101, nil, nil, 0, VenueLever, nil},
		{"swap larger", 101, 100, nil, nil, 0, VenueSwap, nil},
		{"lever refuses", 100, 0, nil, errLever, 0, VenueSwap, nil},
		{"swap refuses", 0, 100, errSwap, nil, 0, VenueLever, nil},
		{"both refuse", 0, 0, errSwap, errLever, 0, 0, errSwap},
		// The edge is bps of the swap fill the leverage fill would replace: it must be beaten, not matched.
		{"edge unmet", 1_000_000, 1_000_050, nil, nil, 10, VenueSwap, nil},
		{"edge exactly met", 1_000_000, 1_001_000, nil, nil, 10, VenueSwap, nil},
		{"edge beaten", 1_000_000, 1_001_001, nil, nil, 10, VenueLever, nil},
		{"edge default keeps a thin gain on the swap", 21_307_000, 21_318_000, nil, nil,
			defaultLeverMinEdgeBps, VenueSwap, nil},
		// A swap that cannot settle at all still hands the route to the leverage venue, edge or not.
		{"edge with no swap fill", 0, 1, errSwap, nil, 10_000, VenueLever, nil},
	} {
		var sw, lv *fill
		if c.swE == nil {
			sw = fillOf(VenueSwap, c.sw)
		}
		if c.lvE == nil {
			lv = fillOf(VenueLever, c.lv)
		}
		got, err := pickVenue(sw, c.swE, lv, c.lvE, c.edge)
		if c.wantErr != nil {
			require.ErrorIs(t, err, c.wantErr, c.name)
			require.Nil(t, got, c.name)
			continue
		}
		require.NoError(t, err, c.name)
		require.Equal(t, c.wantVenue, got.venue, c.name)
		if c.wantVenue == VenueSwap {
			require.Same(t, sw, got, c.name)
		} else {
			require.Same(t, lv, got, c.name)
		}
	}
}

// TestSimulatorFreshness: a snapshot older than MaxSnapshotAgeSec is refused (measured from the refresh's block
// timestamp, which an adopted fill does not move), and so is every quote from scheduledChangeLeadSec before a
// scheduled implementation, hook-set, venue or loan-asset change becomes executable.
func TestSimulatorFreshness(t *testing.T) {
	t.Parallel()
	reads := gridReads(t, "51302915", "live")
	ts := reads.Timestamp
	at := func(s *PoolSimulator, clock uint64) (*pool.CalcAmountOutResult, error) {
		s.nowFn = func() uint64 { return clock }
		return s.CalcAmountOut(amountIn(s, true, 15_000))
	}
	sim := simFor(t, reads, Policy{MaxSnapshotAgeSec: 600})
	_, err := at(sim, ts+600)
	require.NoError(t, err)
	_, err = at(sim, ts+601)
	require.ErrorIs(t, err, ErrSnapshotStale)
	r, err := at(sim, ts+300)
	require.NoError(t, err)
	adopted := sim.CloneState().(*PoolSimulator)
	adopted.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: r.SwapInfo})
	require.Equal(t, ts+300, adopted.state.Timestamp)
	_, err = at(adopted, ts+600)
	require.NoError(t, err)
	_, err = at(adopted, ts+601)
	require.ErrorIs(t, err, ErrSnapshotStale, "an adopted fill does not refresh the snapshot")
	_, err = at(simFor(t, reads, Policy{}), ts+86_400)
	require.NotErrorIs(t, err, ErrSnapshotStale, "a zero maximum age is off")

	scheduled := func(executableAt uint64) *PoolSimulator {
		e := simEntity(t, reads, Policy{})
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(e.Extra), &extra))
		extra.ScheduledChangeAt = executableAt
		raw, err := json.Marshal(&extra)
		require.NoError(t, err)
		e.Extra = string(raw)
		s, err := NewPoolSimulator(e)
		require.NoError(t, err)
		return s
	}
	s := scheduled(ts + scheduledChangeLeadSec + 10)
	_, err = at(s, ts+9)
	require.NoError(t, err)
	_, err = at(s, ts+10)
	require.ErrorIs(t, err, ErrScheduledChange)
	_, err = at(scheduled(ts), ts)
	require.ErrorIs(t, err, ErrScheduledChange, "executable at the snapshot")
}

// TestSimulatorSequences replays every executed on-chain sequence (testdata/core_e2e_seq_*) through CalcAmountOut
// and UpdateBalance: each fill's amounts and the adopted post-state equal the chain's. Governance moves, warps and
// feed or IRM changes are a tracker refresh: the simulator is rebuilt from the chain's post-state.
func TestSimulatorSequences(t *testing.T) {
	t.Parallel()
	for _, blk := range e2eBlocks {
		t.Run(blk, func(t *testing.T) {
			var sim *PoolSimulator
			var simErr error
			var now uint64
			rebuild := func(raw json.RawMessage) {
				var reads flammReads
				require.NoError(t, json.Unmarshal(raw, &reads))
				sim, simErr = NewPoolSimulator(simEntity(t, &reads, Policy{LeverRouting: true}))
				if sim != nil {
					sim.nowFn = func() uint64 { return now }
				}
			}
			fills, refused := 0, 0
			for _, line := range e2eLines(t, e2eFixture("core_e2e_seq_"+blk+".jsonl.gz")) {
				var st e2eStep
				require.NoError(t, json.Unmarshal(line, &st))
				switch st.K {
				case "seq":
					var hdr struct {
						Timestamp uint64 `json:"timestamp"`
					}
					require.NoError(t, json.Unmarshal(st.S, &hdr))
					now = hdr.Timestamp
					rebuild(st.S)
					continue
				case "sw", "lv":
					continue
				}
				now = st.Ts
				where := fmt.Sprintf("%s step %d %s %v", st.Seq, st.I, st.Op, st.Args)
				var venue int
				var sell bool
				var amount uint256.Int
				switch st.Op {
				case "swap":
					if !st.Args[2].IsZero() || !st.Args[3].IsZero() || !st.Args[4].IsZero() {
						rebuild(st.Post)
						continue // a user limit, deadline or pair the adapter never sends
					}
					venue, sell, amount = int(VenueSwap), !st.Args[0].IsZero(), st.Args[1]
				case "leverUp", "leverDown":
					if !st.Args[1].IsZero() {
						rebuild(st.Post)
						continue
					}
					venue, sell, amount = int(VenueLever), st.Op == "leverUp", st.Args[0]
				default:
					rebuild(st.Post)
					continue
				}
				var res *pool.CalcAmountOutResult
				err := simErr
				if sim != nil {
					params := amountIn(sim, sell, amount.Uint64())
					params.TokenAmountIn.Amount = amount.ToBig()
					res, err = sim.calcAmountOut(params, venue)
				}
				switch {
				case st.Ok && err == nil:
					require.Equal(t, st.R[0].Dec(), new(big.Int).Sub(amount.ToBig(),
						res.RemainingTokenAmountIn.Amount).String(), "%s used", where)
					require.Equal(t, st.R[1].Dec(), res.TokenAmountOut.Amount.String(), "%s out", where)
					sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
					var want flammReads
					require.NoError(t, json.Unmarshal(st.Post, &want))
					ws, err := want.baseState()
					require.NoError(t, err)
					require.Empty(t, e2eDiff(sim.state, ws), "%s post-state", where)
					fills++
					continue
				case st.Ok:
					// Refused where the chain filled: only the simulator's own envelope and spread policy may do that.
					require.True(t, errors.Is(err, ErrSpreadNotLive) || errors.Is(err, ErrPoolRefused), "%s: %v", where,
						err)
					refused++
				case err == nil:
					t.Fatalf("%s: chain reverted %s, simulator quoted %s", where, *st.E, res.TokenAmountOut.Amount)
				default:
					want := e2eRevert(*st.E)
					require.NotNil(t, want, where)
					require.True(t, errors.Is(err, want) || errors.Is(err, ErrPoolRefused) || errors.Is(err, ErrSpreadNotLive),
						"%s: chain %v, simulator %v", where, want, err)
				}
				rebuild(st.Post)
			}
			t.Logf("block %s: %d fills replayed through CalcAmountOut/UpdateBalance, %d policy refusals", blk, fills, refused)
			require.GreaterOrEqual(t, fills, 30)
		})
	}
}

// forkSequenceFixture is TestForkSequence's record (forkSequence) from its default fork block 51330060, armed at
// 51330064.
const forkSequenceFixture = "testdata/fork_sequence_51330064.json"

// TestSimulatorForkSequence replays, offline, the adapter fills TestForkSequence mined on an anvil fork of Base:
// one refresh, then every step quoted at the timestamp it was mined (irregular gaps, some of several minutes) and
// adopted only through UpdateBalance -- Morpho interest accrued from each adopted state to the next clock with no
// refresh in between -- with the adapter's amountOut and amountUnused to the wei, its reverts by class, and after
// the last step the state of a fresh refresh.
func TestSimulatorForkSequence(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(forkSequenceFixture)
	require.NoError(t, err)
	var rec forkSequence
	require.NoError(t, json.Unmarshal(raw, &rec))
	sim, err := NewPoolSimulator(rec.Start)
	require.NoError(t, err)
	start := sim.state.Timestamp
	fills, reverts, policy, gaps := 0, 0, 0, map[bool]int{}
	last := start
	for i, st := range rec.Steps {
		ts := st.Ts
		sim.nowFn = func() uint64 { return ts }
		res, err := sim.calcAmountOut(amountIn(sim, st.Sell, st.AmountIn), st.Venue)
		where := fmt.Sprintf("step %d venue=%d sell=%v amountIn=%d ts=%d", i, st.Venue, st.Sell, st.AmountIn, ts)
		switch {
		case st.Policy:
			require.ErrorIs(t, err, ErrSpreadNotLive, where)
			policy++
			continue
		case st.Revert != nil:
			want := forkRevertError(common.FromHex(*st.Revert))
			require.NotNil(t, want, "%s: unmapped revert %s", where, *st.Revert)
			require.ErrorIs(t, err, want, where)
			reverts++
			continue
		}
		require.NoError(t, err, where)
		require.Equal(t, st.Out.String(), res.TokenAmountOut.Amount.String(), "%s amountOut", where)
		require.Equal(t, st.Unused.String(), res.RemainingTokenAmountIn.Amount.String(), "%s amountUnused", where)
		sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
		gaps[ts-last >= 60]++
		last = ts
		fills++
	}
	fresh, err := NewPoolSimulator(rec.Refresh)
	require.NoError(t, err)
	got := sim.state.clone()
	got.Block, got.Timestamp = fresh.state.Block, fresh.state.Timestamp
	require.Empty(t, e2eDiff(got, fresh.state), "the adopted state after the sequence vs a refresh of the fork")
	t.Logf("block %d: %d fills adopted over %d s (%d after a gap of a minute or more), %d reverts, %d policy refusals",
		rec.Block, fills, last-start, gaps[true], reverts, policy)
	require.GreaterOrEqual(t, fills, 30)
	require.Positive(t, gaps[true])
}

// TestSimulatorPreviewGrid: every venue quote agrees with the pool's own preview on every scenario of the grid
// fixture (sampled): the same amounts when both answer and the same error when the pool reverts. The simulator runs
// the settlement the preview does not, yet never refuses a previewed fill; the one refusal allowed is its own
// spread policy on a leverage fill (a lapsed post re-prices a lever-down on chain).
func TestSimulatorPreviewGrid(t *testing.T) {
	t.Parallel()
	for _, blk := range e2eBlocks {
		t.Run(blk, func(t *testing.T) {
			sims := map[string]*PoolSimulator{}
			refusals := map[string]error{}
			counts := map[string]int{}
			lines := e2eLines(t, e2eFixture("core_e2e_grid_"+blk+".jsonl.gz"))
			rows := make([]e2eRow, len(lines))
			for i, line := range lines {
				require.NoError(t, json.Unmarshal(line, &rows[i]))
			}
			// Every third row -- each row here is a full settlement, not a preview, and the three grids hold
			// 82,545 of them -- under -race a sample of those, and every row under EVERLONG_FLAMM_FIXTURES=full
			// (fixture_sample_test.go).
			stride := 3
			if fixturesSampled() {
				stride = 64
			}
			keep := sampleRows(len(rows), rowSampling{stride: stride, edges: true}, func(i int) (string, string) {
				return e2eRowKey(&rows[i])
			})
			if !fixturesSampled() && !fixturesFull() {
				n := 0
				for i := range rows {
					if rows[i].K != "state" {
						n++
						keep[i] = n%3 == 0
					}
				}
			}
			for i := range rows {
				row := rows[i]
				if row.K == "state" {
					var reads flammReads
					require.NoError(t, json.Unmarshal(row.S, &reads))
					sim, err := NewPoolSimulator(simEntity(t, &reads, Policy{LeverRouting: true}))
					if err != nil {
						refusals[row.Tag] = err
						require.ErrorIs(t, err, ErrPoolRefused, row.Tag) // IRM outage scenarios
						continue
					}
					ts := reads.Timestamp
					sim.nowFn = func() uint64 { return ts }
					sims[row.Tag] = sim
					continue
				}
				if !keep[i] {
					continue
				}
				sim := sims[row.Tag]
				if sim == nil {
					counts["pool refused"]++
					continue
				}
				var f *fill
				var err error
				if row.K == "sw" {
					f, err = sim.quoteSwap(row.D == 1, &row.A, sim.now())
				} else {
					f, err = sim.quoteLever(row.D == 1, &row.A, sim.now())
				}
				where := fmt.Sprintf("%s %s d=%d a=%s", row.Tag, row.K, row.D, row.A.Dec())
				switch {
				case row.E != nil:
					want := e2eRevert(*row.E)
					require.NotNil(t, want, where)
					if !errors.Is(err, want) {
						require.ErrorIs(t, err, ErrSpreadNotLive, "%s: chain %v simulator %v", where, want, err)
					}
					counts["revert"]++
				case err != nil:
					require.True(t, row.K == "lv" && errors.Is(err, ErrSpreadNotLive), "%s: chain %v, simulator %v", where,
						row.R, err)
					counts["spread policy refusal"]++
				default:
					require.Equal(t, row.R[0].Dec(), f.used.Dec(), where)
					require.Equal(t, row.R[1].Dec(), f.out.Dec(), where)
					counts["identical"]++
				}
			}
			t.Logf("block %s: %v (%d of %d rows kept); pools refused %v", blk, counts, kept(keep), len(rows), refusals)
			if !fixturesSampled() {
				require.Greater(t, counts["identical"], 1000)
			}
		})
	}
}

// TestSimulatorVenueSelection: routing off quotes only the swap; routing on keeps the larger output (the swap on a
// tie, and when the leverage venue refuses) and reports the venue the adapter must be told.
func TestSimulatorVenueSelection(t *testing.T) {
	t.Parallel()
	seen := map[uint8]int{}
	for _, tag := range []string{"armed", "no_sell", "no_buy", "armed_hi"} {
		reads := gridReads(t, "51313000", tag)
		off := simFor(t, reads, Policy{})
		on := simFor(t, reads, Policy{LeverRouting: true})
		for _, sell := range []bool{true, false} {
			grid := liveGrid(1_000, 200_000, 30)
			if !sell {
				grid = liveGrid(1_000_000, 200_000_000, 30)
			}
			for _, a := range grid {
				params := amountIn(on, sell, a)
				rOff, errOff := off.CalcAmountOut(params)
				if errOff == nil {
					require.Equal(t, VenueSwap, rOff.SwapInfo.(SwapInfo).Venue)
				}
				rOn, errOn := on.CalcAmountOut(params)
				now := on.now()
				sw, swErr := on.quoteSwap(sell, uint256.NewInt(a), now)
				lv, lvErr := on.quoteLever(sell, uint256.NewInt(a), now)
				where := fmt.Sprintf("%s sell=%v a=%d", tag, sell, a)
				switch {
				case swErr != nil && lvErr != nil:
					require.Error(t, errOn, where)
					require.Equal(t, swErr.Error(), errOn.Error(), where)
					continue
				case lvErr == nil && (swErr != nil || lv.out.Gt(&sw.out)):
					require.NoError(t, errOn, where)
					si := rOn.SwapInfo.(SwapInfo)
					require.Equal(t, VenueLever, si.Venue, where)
					require.Equal(t, lv.out.Dec(), rOn.TokenAmountOut.Amount.String(), where)
					require.Equal(t, new(big.Int).Sub(new(big.Int).SetUint64(a), lv.used.ToBig()).String(),
						rOn.RemainingTokenAmountIn.Amount.String(), "%s: the unused input (a lever-down's amountIn - payNative)",
						where)
					want := on.Policy.GasLeverDown
					if sell {
						want = on.Policy.GasLeverUp
					}
					require.Equal(t, want, rOn.Gas, where)
				default:
					require.NoError(t, errOn, where)
					require.Equal(t, VenueSwap, rOn.SwapInfo.(SwapInfo).Venue, where)
					require.Equal(t, sw.out.Dec(), rOn.TokenAmountOut.Amount.String(), where)
				}
				seen[rOn.SwapInfo.(SwapInfo).Venue]++
			}
		}
	}
	t.Logf("venues chosen: %v", seen)
	require.Positive(t, seen[VenueSwap])
	require.Positive(t, seen[VenueLever])
}

// TestSimulatorLeverGates: the leverage venue is quoted only when the pool would fill it and the spread is live
// through the margin.
func TestSimulatorLeverGates(t *testing.T) {
	t.Parallel()
	type gate struct {
		tag    string
		policy Policy
		up     error
		down   error
		// fills: an open gate must quote both sizes; otherwise it may still refuse on the fill's own value (the
		// band, the value-leak guard), never on a gate or the spread policy.
		fills bool
	}
	for _, g := range []gate{
		{"armed", Policy{}, nil, nil, true},
		{"live", Policy{}, ErrLevPaused, ErrLevPaused, false},
		{"no_lev", Policy{}, ErrFeatureDisabled, ErrFeatureDisabled, false},
		{"paused", Policy{}, ErrPaused, nil, false},
		{"peg_broken", Policy{}, ErrPegBroken, nil, false},
		{"stale_spread", Policy{}, ErrSpreadNotLive, ErrSpreadNotLive, false},
		// The spread is live through lastSetTs + maxSpreadAge inclusive: this scenario's snapshot is 3600 s before
		// that deadline, so a 3600 s margin lands exactly on it and 3601 one past it.
		{"armed", Policy{SpreadAgeMarginSec: 3600}, nil, nil, true},
		{"armed", Policy{SpreadAgeMarginSec: 3601}, ErrSpreadNotLive, ErrSpreadNotLive, false},
	} {
		sim := simFor(t, gridReads(t, "51313000", g.tag), g.policy)
		now := sim.now()
		_, errUp := sim.quoteLever(true, uint256.NewInt(5_000), now)
		_, errDown := sim.quoteLever(false, uint256.NewInt(5_000_000), now)
		for _, c := range []struct {
			got, want error
			dir       string
		}{{errUp, g.up, "up"}, {errDown, g.down, "down"}} {
			switch {
			case c.want != nil:
				require.ErrorIs(t, c.got, c.want, "%s %+v %s", g.tag, g.policy, c.dir)
			case g.fills:
				require.NoError(t, c.got, "%s %+v %s", g.tag, g.policy, c.dir)
			case c.got != nil:
				require.True(t, errors.Is(c.got, ErrPriceBand) || errors.Is(c.got, ErrLevValueLeak), "%s %+v %s: %v",
					g.tag, g.policy, c.dir, c.got)
			}
		}
	}
	// Without the spread hook nothing is live.
	sim := simFor(t, gridReads(t, "51313000", "armed"), Policy{})
	sim.state.Hooks.Spread = spreadHookSlot{}
	require.ErrorIs(t, sim.spreadLive(sim.now(), 0), ErrSpreadNotLive)
	// Routing off never quotes the leverage venue through CalcAmountOut.
	sim = simFor(t, gridReads(t, "51313000", "armed"), Policy{})
	sim.state.Paused = true // the swap sell refuses; the lever-up would too
	_, err := sim.CalcAmountOut(amountIn(sim, true, 5_000))
	require.ErrorIs(t, err, ErrPaused)
}

// TestSimulatorClockAndMargins: feed staleness and the sequencer are evaluated at the quote clock; the feed-age,
// band and drift margins only refuse and never change an amount.
func TestSimulatorClockAndMargins(t *testing.T) {
	t.Parallel()
	reads := gridReads(t, "51302915", "live")
	sim := simFor(t, reads, Policy{})
	cb, usdc := &sim.state.Feed.Asset, &sim.state.Feed.Loans[0]
	deadline := min(cb.Round.UpdatedAt.Uint64()+cb.Heartbeat.Uint64(), usdc.Round.UpdatedAt.Uint64()+usdc.Heartbeat.Uint64())
	params := amountIn(sim, true, 15000)
	at := func(s *PoolSimulator, ts uint64) (*pool.CalcAmountOutResult, error) {
		s.nowFn = func() uint64 { return ts }
		return s.CalcAmountOut(params)
	}
	base, err := at(sim, reads.Timestamp)
	require.NoError(t, err)
	_, err = at(sim, deadline)
	require.NoError(t, err)
	_, err = at(sim, deadline+1)
	require.ErrorIs(t, err, ErrStalePrice)
	// A clock behind the snapshot quotes at the snapshot.
	early, err := at(sim, reads.Timestamp-100)
	require.NoError(t, err)
	require.Equal(t, base.TokenAmountOut.Amount.String(), early.TokenAmountOut.Amount.String())

	margin := deadline - reads.Timestamp
	m := simFor(t, reads, Policy{PriceAgeMarginSec: margin})
	r, err := at(m, reads.Timestamp)
	require.NoError(t, err)
	require.Equal(t, base.TokenAmountOut.Amount.String(), r.TokenAmountOut.Amount.String())
	m = simFor(t, reads, Policy{PriceAgeMarginSec: margin + 1})
	_, err = at(m, reads.Timestamp)
	require.ErrorIs(t, err, ErrFeedAgeMargin)

	for tag, want := range map[string]error{"seq_grace": ErrSequencerGrace, "seq_down": ErrSequencerDown,
		"stale_feed": ErrStalePrice, "feed_invalid": ErrInvalidPrice, "peg_broken": ErrPegBroken,
		"paused": ErrPaused, "no_sell": ErrFeatureDisabled} {
		s := simFor(t, gridReads(t, "51302915", tag), Policy{})
		_, err := s.CalcAmountOut(amountIn(s, true, 15000))
		require.ErrorIs(t, err, want, tag)
	}
	// The grace ends at the clock: through startedAt + grace the sequencer refuses, one second later the next check
	// decides (this scenario's cbBTC round is then past its heartbeat).
	g := simFor(t, gridReads(t, "51302915", "seq_grace"), Policy{})
	graceEnd := g.state.Feed.Sequencer.StartedAt.Uint64() + g.state.Feed.SequencerGrace.Uint64()
	_, err = at(g, graceEnd)
	require.ErrorIs(t, err, ErrSequencerGrace)
	_, err = at(g, graceEnd+1)
	require.ErrorIs(t, err, ErrStalePrice)

	// Band margin: a fill accepted near its band edge is refused inside the margin, with its amount unchanged
	// whenever it is accepted.
	lo, hi, found := edge(probeSellGrid, func(a *uint256.Int) bool {
		_, err := sim.state.previewSwap(true, a, reads.Timestamp)
		return err == nil
	})
	require.True(t, found)
	require.True(t, hi.Eq(new(uint256.Int).AddUint64(lo, 1)))
	edgeParams := amountIn(sim, true, lo.Uint64())
	sim.nowFn = func() uint64 { return reads.Timestamp }
	plain, err := sim.CalcAmountOut(edgeParams)
	require.NoError(t, err)
	tight := simFor(t, reads, Policy{PriceBandMarginBps: 50})
	_, err = tight.CalcAmountOut(edgeParams)
	require.ErrorIs(t, err, ErrBandMargin)
	loose := simFor(t, reads, Policy{PriceBandMarginBps: 1})
	small, err := loose.CalcAmountOut(amountIn(loose, true, 15000))
	require.NoError(t, err)
	require.Equal(t, base.TokenAmountOut.Amount.String(), small.TokenAmountOut.Amount.String())
	require.NotEqual(t, "0", plain.TokenAmountOut.Amount.String())
	all := simFor(t, reads, Policy{PriceBandMarginBps: 800})
	_, err = all.CalcAmountOut(amountIn(all, true, 15000))
	require.ErrorIs(t, err, ErrBandMargin, "a margin covering the whole band refuses everything")

	// Drift: a second of accrual leaves this fill unchanged; a drift past the feed heartbeat refuses it.
	d := simFor(t, reads, Policy{DebtDriftSec: 1})
	r, err = at(d, reads.Timestamp)
	require.NoError(t, err)
	require.Equal(t, base.TokenAmountOut.Amount.String(), r.TokenAmountOut.Amount.String())
	d = simFor(t, reads, Policy{DebtDriftSec: margin + 1})
	_, err = at(d, reads.Timestamp)
	require.ErrorIs(t, err, ErrDebtDrift)
}

// TestSimulatorPartialFills: the unused input is returned for a notional-capped sell and for a lever-down, which
// pulls only payNative.
func TestSimulatorPartialFills(t *testing.T) {
	t.Parallel()
	sim := simFor(t, gridReads(t, "51302915", "capped"), Policy{})
	for _, a := range liveGrid(10_000, 400_000, 30) {
		r, err := sim.CalcAmountOut(amountIn(sim, true, a))
		if err != nil {
			continue
		}
		used := new(big.Int).Sub(new(big.Int).SetUint64(a), r.RemainingTokenAmountIn.Amount)
		si := r.SwapInfo.(SwapInfo)
		require.Equal(t, used.String(), si.AmountInUsed.Dec())
		if r.RemainingTokenAmountIn.Amount.Sign() > 0 {
			t.Logf("capped sell %d: used %s, remaining %s, out %s", a, used, r.RemainingTokenAmountIn.Amount,
				r.TokenAmountOut.Amount)
			return
		}
	}
	t.Fatal("no partial sell on the capped scenario")
}

// TestSimulatorClippedSellGas: a clipped sell's estimate grows with its input by the solves of the hook's cap
// bisection. On a one-pass clip the plan's previewExactIn and the execution's executeExactIn each bisect [0, 2^k] in
// exactly min(k, 64) solves (the interval halves exactly; EverlongHook.sol:588-594), an unclipped sell and every buy
// (hook.go capFill) solve none, and the per-solve and per-pass terms follow the policy.
func TestSimulatorClippedSellGas(t *testing.T) {
	t.Parallel()
	policy := Policy{GasSwapSell: 1_000_000, GasSwapSellCapEval: 3, GasSwapSellPass: 1_000}
	sim := simFor(t, gridReads(t, "51302915", "capped"), policy)
	now := sim.now()
	clipped := 0
	for k := 1; k <= 80; k++ {
		a := new(big.Int).Lsh(big.NewInt(1), uint(k))
		params := amountIn(sim, true, 1)
		params.TokenAmountIn.Amount = a
		r, err := sim.CalcAmountOut(params)
		if err != nil {
			continue
		}
		var in uint256.Int
		in.SetFromBig(a)
		q, err := sim.quoteSwap(true, &in, now)
		require.NoError(t, err)
		require.Equal(t, 1_000_000+3*int64(q.capEvals)+1_000*int64(q.passes-1), r.Gas, "2^%d", k)
		if r.RemainingTokenAmountIn.Amount.Sign() == 0 {
			require.Zero(t, q.capEvals, "2^%d", k)
			continue
		}
		require.Equal(t, uint64(1), q.passes, "2^%d", k)
		require.Equal(t, uint64(2*min(k, 64)), q.capEvals, "2^%d", k)
		clipped++
	}
	require.Positive(t, clipped)
	for _, blk := range e2eBlocks {
		for _, tag := range []string{"live", "armed", "capped", "fee_cap", "btc_down25", "btc_up10", "ltv_low", "pin_low"} {
			sim := simFor(t, gridReads(t, blk, tag), Policy{})
			for _, a := range liveGrid(100, 1<<62, 40) {
				if q, err := sim.quoteSwap(false, uint256.NewInt(a), sim.now()); err == nil {
					require.Zero(t, q.capEvals, "%s/%s buy %d", blk, tag, a)
					require.Zero(t, q.passes, "%s/%s buy %d", blk, tag, a)
				}
			}
		}
	}
}

// TestSimulatorRefusals: a refresh that is not attested, drifted, a decoded simulator whose verdict or listing was
// altered, and states outside the envelope are all refused.
func TestSimulatorRefusals(t *testing.T) {
	t.Parallel()
	reads := gridReads(t, "51302915", "live")
	e := simEntity(t, reads, Policy{})
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(e.Extra), &extra))
	for name, c := range map[string]struct {
		mutate func(x *Extra, p *entity.Pool)
		want   error
	}{
		"not attested":   {func(x *Extra, _ *entity.Pool) { x.Attested = false }, ErrNotAttested},
		"attest failure": {func(x *Extra, _ *entity.Pool) { x.AttestFailure = "probe" }, ErrNotAttested},
		"drift":          {func(x *Extra, _ *entity.Pool) { x.ProfileDrift = "hook set" }, ErrProfileDrift},
		"no reads":       {func(x *Extra, _ *entity.Pool) { x.Reads = nil }, ErrNotAttested},
		"block":          {func(_ *Extra, p *entity.Pool) { p.BlockNumber++ }, ErrNotAttested},
		"type":           {func(_ *Extra, p *entity.Pool) { p.Type = "other" }, ErrInvalidProfile},
		"token order": {func(_ *Extra, p *entity.Pool) {
			p.Tokens[0], p.Tokens[1] = p.Tokens[1], p.Tokens[0]
		}, ErrInvalidProfile},
		"donated collateral": {func(x *Extra, _ *entity.Pool) {
			r := *x.Reads
			r.Router.Venues = append([]venueReads(nil), r.Router.Venues...)
			v := &r.Router.Venues[0]
			v.Position.Collateral.AddUint64(&v.ManagedCollateral, 1)
			x.Reads = &r
		}, ErrPoolRefused},
		"donated supply shares": {func(x *Extra, _ *entity.Pool) {
			r := *x.Reads
			r.Router.Venues = append([]venueReads(nil), r.Router.Venues...)
			v := &r.Router.Venues[0]
			v.Position.SupplyShares.AddUint64(&v.ManagedSupplyShares, 1)
			x.Reads = &r
		}, ErrPoolRefused},
		"irm down": {func(x *Extra, _ *entity.Pool) {
			r := *x.Reads
			r.Router.Venues = append([]venueReads(nil), r.Router.Venues...)
			r.Router.Venues[0].IrmReadable = false
			x.Reads = &r
		}, ErrPoolRefused},
		// The two halves of the Extra split: every venue the settlement runs on (Extra.Reads) has to be one of the
		// venues that were checked against the registry (Extra.Venues, validVenues), so a reads set that carries a
		// venue the published set does not is refused whole rather than settled through unchecked.
		"venue reads": {func(x *Extra, _ *entity.Pool) {
			r := *x.Reads
			r.Router.Venues = append(append([]venueReads(nil), r.Router.Venues...), r.Router.Venues[0])
			x.Reads = &r
		}, ErrPoolRefused},
	} {
		x := extra
		p := e
		p.Tokens = []*entity.PoolToken{{Address: e.Tokens[0].Address, Swappable: true},
			{Address: e.Tokens[1].Address, Swappable: true}}
		c.mutate(&x, &p)
		raw, err := json.Marshal(&x)
		require.NoError(t, err)
		p.Extra = string(raw)
		_, err = NewPoolSimulator(p)
		require.ErrorIs(t, err, c.want, name)
	}
	for _, tag := range []string{"irm_grace", "irm_quarantine"} {
		_, err := NewPoolSimulator(simEntity(t, gridReads(t, "51302915", tag), Policy{}))
		require.ErrorIs(t, err, ErrPoolRefused, tag)
	}

	sim := simFor(t, reads, Policy{})
	params := amountIn(sim, true, 15000)
	for name, c := range map[string]struct {
		mutate func(s *PoolSimulator)
		want   error
	}{
		"attested flag": {func(s *PoolSimulator) { s.Attested = false }, ErrNotAttested},
		"listing":       {func(s *PoolSimulator) { s.StaticExtra.Hooks[5] = s.StaticExtra.Hooks[4] }, ErrInvalidProfile},
		"tokens":        {func(s *PoolSimulator) { s.Info.Tokens = []string{s.Info.Tokens[1], s.Info.Tokens[0]} }, ErrInvalidProfile},
		"address":       {func(s *PoolSimulator) { s.Info.Address = strings.ToUpper(s.Info.Address) }, ErrInvalidProfile},
		"state":         {func(s *PoolSimulator) { s.state = nil }, ErrNotAttested},
		"no venues":     {func(s *PoolSimulator) { s.Venues = nil }, ErrInvalidProfile},
		"venue account": {func(s *PoolSimulator) { s.Venues[0].Account = common.Address{} }, ErrInvalidProfile},
		"venue irm": {func(s *PoolSimulator) {
			s.Venues[0].Irm = common.HexToAddress("0x0000000000000000000000000000000000000bad")
		}, ErrInvalidProfile},
		"loan set": {func(s *PoolSimulator) {
			s.state.Pool.Loans = append(s.state.Pool.Loans, s.state.Pool.Loans[0])
		}, ErrPoolRefused},
	} {
		c2 := sim.CloneState().(*PoolSimulator)
		c2.Venues = append([]StaticVenue(nil), sim.Venues...)
		c.mutate(c2)
		_, err := c2.CalcAmountOut(params)
		require.ErrorIs(t, err, c.want, name)
	}
	_, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
}

// TestSimulatorDonation: collateral and loan-asset supply donated to the venue account on Morpho (the "donation"
// sequences of testdata/edges/core_edge_seq_*.jsonl.gz, generated by testdata/gen/CoreEdgeSeq.t.sol
// _seqDonation: 50,000 sats of supplyCollateral, then 7 USDC of supply, on behalf of the account) refuse the pool
// by default, and with QuoteDonatedVenues every executed swap and leverage fill the adapter could send (minimum out
// at most 1, deadline now) is quoted through NewPoolSimulator and CalcAmountOut on the donated state with the
// chain's amounts or revert class, and UpdateBalance adopts the chain's post-state.
func TestSimulatorDonation(t *testing.T) {
	t.Parallel()
	blocks := make([]string, 0, len(coreEdgeSeqFixtures))
	for b := range coreEdgeSeqFixtures {
		blocks = append(blocks, b)
	}
	sort.Strings(blocks)
	for _, blk := range blocks {
		t.Run(blk, func(t *testing.T) {
			var reads *flammReads
			var sim *PoolSimulator
			fills, reverts, policy := 0, 0, 0
			rebuild := func(raw json.RawMessage) {
				_, reads = coreEdgeState(t, raw)
				sim = nil
			}
			for _, line := range coreEdgeLoad(t, coreEdgeFixturePath("seq", blk), coreEdgeSeqFixtures[blk]) {
				if !bytes.Contains(line, []byte(`"seq":"donation"`)) {
					continue
				}
				var row coreEdgeSeqRow
				require.NoError(t, json.Unmarshal(line, &row))
				switch row.K {
				case "begin":
					rebuild(row.S)
					continue
				case "re":
					rebuild(row.Post)
					continue
				case "x":
				default:
					continue
				}
				v := &reads.Router.Venues[0]
				donated := v.Position.Collateral.Gt(&v.ManagedCollateral) || v.Position.SupplyShares.Gt(&v.ManagedSupplyShares)
				if !donated || row.Late != 0 || row.M.GtUint64(1) {
					rebuild(row.Post)
					continue
				}
				if sim == nil {
					_, err := NewPoolSimulator(simEntity(t, reads, Policy{LeverRouting: true}))
					require.ErrorIs(t, err, ErrPoolRefused, "a donated state is refused by default")
					sim, err = NewPoolSimulator(simEntity(t, reads, Policy{LeverRouting: true, QuoteDonatedVenues: true}))
					require.NoError(t, err, "a donated state is quotable with QuoteDonatedVenues")
					refusing := sim.CloneState().(*PoolSimulator)
					refusing.Policy.QuoteDonatedVenues = false
					_, err = refusing.CalcAmountOut(amountIn(refusing, true, 1_000))
					require.ErrorIs(t, err, ErrPoolRefused, "a decoded simulator re-checks the donation on every quote")
				}
				ts := row.Ts
				sim.nowFn = func() uint64 { return ts }
				sell := row.D == 1
				venue := int(VenueSwap)
				if row.Lever {
					venue = int(VenueLever)
				}
				res, err := sim.calcAmountOut(amountIn(sim, sell, row.A.Uint64()), venue)
				where := fmt.Sprintf("%s#%d lever=%v d=%d a=%s", blk, row.I, row.Lever, row.D, row.A.Dec())
				if row.Lever && errors.Is(err, ErrSpreadNotLive) {
					policy++ // the simulator's own spread policy (a lapsed post re-prices a lever-down on chain)
					rebuild(row.Post)
					continue
				}
				if row.R == nil {
					want := coreEdgeRevert(t, coreEdgeStr(row.E))
					require.NotNil(t, want, where)
					require.ErrorIs(t, err, want, where)
					reverts++
					rebuild(row.Post)
					continue
				}
				w := coreEdgeWords(*row.R)
				if w[1].IsZero() {
					require.ErrorIs(t, err, ErrZeroAmountOut, where) // minimum out 0: the adapter's 1 reverts
					rebuild(row.Post)
					continue
				}
				require.NoError(t, err, where)
				require.Equal(t, w[0].Dec(), new(big.Int).Sub(row.A.ToBig(), res.RemainingTokenAmountIn.Amount).String(),
					"%s used", where)
				require.Equal(t, w[1].Dec(), res.TokenAmountOut.Amount.String(), "%s out", where)
				sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
				want, wantReads := coreEdgeState(t, row.Post)
				require.Empty(t, e2eDiff(sim.state, want), "%s post-state", where)
				reads = wantReads
				fills++
			}
			t.Logf("block %s: %d fills and %d reverts quoted on donated states, %d spread-policy refusals", blk, fills,
				reverts, policy)
			require.Positive(t, fills)
		})
	}
}

// BenchmarkCalcAmountOut: one quote runs the whole settlement (the hook fill, up to four funding passes over the
// Router's Morpho accrual, the Router legs and the gate) on a copy of the state. It is measured both ways round:
// under the shipped configuration (Config.policy(), what pool-service runs) and with every margin off. The difference is not noise -- the margins that re-run a
// settlement do the whole plan again: DebtDriftSec re-runs it with accrual, PriceBandMarginBps re-runs a sell and
// every leverage fill at the feed moved either way, and the refresh's end-of-window market oracle answer
// (Extra.OracleAhead, published here as the snapshot's own so the gate runs and accepts) re-runs it once more.
func BenchmarkCalcAmountOut(b *testing.B) {
	defaults := func(lever bool) Policy { return (&Config{LeverRouting: lever}).policy() }
	for _, c := range []struct {
		name   string
		tag    string
		policy Policy
		sell   bool
		amount uint64
	}{
		{"swap-sell", "live", defaults(false), true, 15_000},
		{"swap-buy", "live", defaults(false), false, 20_000_000},
		{"routed-sell", "armed", defaults(true), true, 5_000},
		{"routed-buy", "armed", defaults(true), false, 5_000_000},
		{"swap-sell-no-margins", "live", Policy{}, true, 15_000},
		{"swap-buy-no-margins", "live", Policy{}, false, 20_000_000},
		{"routed-sell-no-margins", "armed", Policy{LeverRouting: true}, true, 5_000},
		{"routed-buy-no-margins", "armed", Policy{LeverRouting: true}, false, 5_000_000},
	} {
		sim := simFor(b, gridReads(b, "51313000", c.tag), c.policy)
		if c.policy.MaxSnapshotAgeSec != 0 {
			for i := range sim.state.Router.Venues {
				m := &sim.state.Router.Venues[i].Morpho
				sim.OracleAhead = append(sim.OracleAhead, OracleAnswer{Ok: m.OracleOk, Price: m.OraclePrice})
			}
		}
		params := amountIn(sim, c.sell, c.amount)
		if _, err := sim.CalcAmountOut(params); err != nil {
			b.Fatalf("%s: %v", c.name, err)
		}
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := sim.CalcAmountOut(params); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestSimulatorOracleWindow: the market oracle answer the refresh read at the far end of the snapshot window
// (Extra.OracleAhead) is a second price every fill has to settle at. A withheld Chainlink SVR round the window
// reveals moves that answer with no transaction, and the two predicates that read it -- MMRouterLib.bandOk and
// Morpho's own health check -- then refuse fills the snapshot's own answer accepts. The gate is a settlement
// comparison, not an equality check on the price: a move that changes no fill changes no quote.
func TestSimulatorOracleWindow(t *testing.T) {
	t.Parallel()
	stored := loadTracked(t)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(stored.Extra), &extra))
	require.Len(t, extra.OracleAhead, 1)
	snapshot := extra.Reads.Router.Venues[0].OraclePrice
	require.True(t, extra.OracleAhead[0].Ok)
	require.True(t, extra.OracleAhead[0].Price.Eq(&snapshot), "no round is withheld at the recorded block")

	sim := func(t *testing.T, ahead []OracleAnswer) *PoolSimulator {
		t.Helper()
		e := extra
		e.OracleAhead = ahead
		raw, err := json.Marshal(&e)
		require.NoError(t, err)
		p := stored
		p.Extra = string(raw)
		s, err := NewPoolSimulator(p)
		require.NoError(t, err)
		ts := s.state.Timestamp
		s.nowFn = func() uint64 { return ts }
		return s
	}
	// answer is the snapshot price moved by one part in `over` (negative: down), as an ahead answer.
	answer := func(over int64) []OracleAnswer {
		var p, step uint256.Int
		p.Set(&snapshot)
		if over < 0 {
			p.Sub(&p, step.Div(&p, uint256.NewInt(uint64(-over))))
		} else {
			p.Add(&p, step.Div(&p, uint256.NewInt(uint64(over))))
		}
		return []OracleAnswer{{Ok: true, Price: p}}
	}
	wei := func(up bool) []OracleAnswer {
		var p uint256.Int
		p.Set(&snapshot)
		if up {
			p.AddUint64(&p, 1)
		} else {
			p.SubUint64(&p, 1)
		}
		return []OracleAnswer{{Ok: true, Price: p}}
	}
	// The Router's band is 2% of the PriceFeed cross (Extra.Reads.Router.OracleBandWad), so a 3% move refuses.
	band := extra.Reads.Router.OracleBandWad
	require.Equal(t, "20000000000000000", band.Dec())
	for name, c := range map[string]struct {
		ahead []OracleAnswer
		want  error
	}{
		"published, unmoved":     {extra.OracleAhead, nil},
		"none published":         {nil, nil},
		"one wei up":             {wei(true), nil},
		"one wei down":           {wei(false), nil},
		"ten bps up":             {answer(1000), nil},
		"ten bps down":           {answer(-1000), nil},
		"three percent up":       {answer(33), ErrOracleDrift},
		"three percent down":     {answer(-33), ErrOracleDrift},
		"oracle stops answering": {[]OracleAnswer{{}}, ErrOracleDrift},
	} {
		s := sim(t, c.ahead)
		// The guard is a second settlement on a copy of the state, so quoting through it writes nothing and leaves
		// it armed for the next quote: one that consumed itself would find nothing moved from then on, which is
		// F02 re-opened on the second quote rather than on the first.
		armed, before := s.oracleShifted() != nil, stateJSON(t, s)
		res, err := s.CalcAmountOut(amountIn(s, true, 15_000))
		require.Equal(t, before, stateJSON(t, s), "%s: the end-of-window guard wrote the quoted state", name)
		require.Equal(t, armed, s.oracleShifted() != nil, "%s: the end-of-window guard consumed itself", name)
		if c.want != nil {
			require.ErrorIs(t, err, c.want, name)
			continue
		}
		require.NoError(t, err, name)
		require.Equal(t, "11301759", res.TokenAmountOut.Amount.String(), name)
	}

	// A set that does not match the venues is an Extra no refresh wrote (readOracleAhead reads one answer per
	// venue): the pool is refused rather than quoted with the guard skipped, at construction and on every quote of
	// a simulator msgpack restored without one. Having no set at all is the no-window case above, and it quotes.
	mismatched := extra
	mismatched.OracleAhead = []OracleAnswer{{Ok: true, Price: snapshot}, {Ok: true, Price: snapshot}}
	raw, err := json.Marshal(&mismatched)
	require.NoError(t, err)
	refused := stored
	refused.Extra = string(raw)
	_, err = NewPoolSimulator(refused)
	require.ErrorIs(t, err, ErrPoolRefused)

	restored := sim(t, extra.OracleAhead)
	restored.OracleAhead = mismatched.OracleAhead
	require.Nil(t, restored.oracleShifted())
	_, err = restored.CalcAmountOut(amountIn(restored, true, 15_000))
	require.ErrorIs(t, err, ErrPoolRefused)
}

// feedShiftedReads is reads with the pool asset's Chainlink answer moved by bps (signed): the adverse half of one
// cbBTC/USD round of that size. A real round also writes a new roundId and updatedAt, which only make the round
// fresher and so can only relax PriceFeed._read.
func feedShiftedReads(reads *flammReads, bps int64) *flammReads {
	c := *reads
	c.Feed.Tokens = append([]feedTokenReads(nil), reads.Feed.Tokens...)
	answer := uint256.Int(c.Feed.Tokens[0].Round.Answer)
	size := uint64(bps)
	if bps < 0 {
		size = uint64(-bps)
	}
	var d uint256.Int
	d.Mul(&answer, uint256.NewInt(size))
	d.Div(&d, uint256.NewInt(10_000))
	var moved uint256.Int
	if bps >= 0 {
		moved.Add(&answer, &d)
	} else {
		moved.Sub(&answer, &d)
	}
	c.Feed.Tokens[0].Round.Answer = int256.Int(moved)
	return &c
}

// TestSimulatorFeedMoveMargin: PriceBandMarginBps is how far the pool asset's feed may move before a quote is
// refused, and it is enforced on everything the feed prices, not only the price band.
//
//   - Over the whole grid, every quote the default policy accepts is still accepted, with identical amounts, at the
//     answer moved by the margin in both directions. That is the property the margin promises and the reason its
//     default is sized on the feed's own round sizes (constant.go): over 567 cbBTC/USD rounds in 167.6 h on Base,
//     the p99 consecutive round moved 52.1 bps.
//   - A sell the pool clips is re-priced by a feed move, because its ceiling is the gate room and the Router's
//     funding at the cross (FLAMMSwapLib.sol:164): those quotes are refused, where the price band alone accepted
//     them and an unseen round then paid the user less than quoted.
//   - A leverage fill sitting on one of the three gates that are priced at the feed and not at the curve's
//     reservation price -- value leak (FLAMMLeverLib.sol:108), CR floor (:112), concession (:138) -- is refused,
//     where the band margin alone accepted it and a 10 bps round reverted it on chain.
func TestSimulatorFeedMoveMargin(t *testing.T) {
	t.Parallel()
	prod := Policy{LeverRouting: true, PriceBandMarginBps: defaultPriceBandMarginBps,
		PriceAgeMarginSec: defaultPriceAgeMarginSec, SpreadAgeMarginSec: defaultSpreadAgeMarginSec,
		DebtDriftSec: defaultDebtDriftSec, MaxSnapshotAgeSec: defaultMaxSnapshotAgeSec}
	// round is the size the default has to cover, measured and not derived from the default: the p99 consecutive
	// cbBTC/USD round over Base blocks 51066959-51369359 moved 52.1 bps and the p95 37.8 (constant.go).
	const round = int64(50)

	t.Run("an accepted quote survives a round of that size", func(t *testing.T) {
		checked, combo := 0, 0
		for _, blk := range []string{"51302915", "51313000", "51324800"} {
			for _, tag := range []string{"live", "armed", "armed_hi", "capped", "band_tight", "btc_down10",
				"btc_up10", "fee_cap", "fee_floor", "ltv_low", "pin_low", "accrued", "stale_spread"} {
				raw, ok := gridStateJSON(t, blk, tag)
				if !ok || raw == nil {
					continue
				}
				reads := gridReads(t, blk, tag)
				sim := simFor(t, reads, prod)
				moved := map[int64]*PoolSimulator{}
				for _, b := range []int64{-round, round} {
					m := simFor(t, feedShiftedReads(reads, b), Policy{})
					m.nowFn = sim.nowFn
					moved[b] = m
				}
				combo++
				for ci, k := range quoteCases() {
					if !sampleCase(ci, combo, 8) {
						continue
					}
					for _, venue := range []int{int(VenueSwap), int(VenueLever)} {
						want, err := sim.calcAmountOut(amountIn(sim, k.sell, k.a), venue)
						if err != nil {
							continue
						}
						checked++
						for _, b := range []int64{-round, round} {
							got, err := moved[b].calcAmountOut(amountIn(moved[b], k.sell, k.a), venue)
							where := fmt.Sprintf("%s/%s venue %d sell=%v a=%d at %+d bps", blk, tag, venue, k.sell,
								k.a, b)
							require.NoError(t, err, where)
							require.Zero(t, got.TokenAmountOut.Amount.Cmp(want.TokenAmountOut.Amount),
								"%s: pays %s, quoted %s", where, got.TokenAmountOut.Amount, want.TokenAmountOut.Amount)
							require.Zero(t, got.RemainingTokenAmountIn.Amount.Cmp(want.RemainingTokenAmountIn.Amount),
								"%s: unused %s, quoted %s", where, got.RemainingTokenAmountIn.Amount,
								want.RemainingTokenAmountIn.Amount)
						}
					}
				}
			}
		}
		require.Greater(t, checked, 200, "quotes checked against a round of that size")
	})

	// The band binds hardest when the feed has drifted from the book, which is the state the pool is cheapest in
	// and the one the keeper has never recentered out of: the same promise has to hold there.
	t.Run("and survives it from a band-pinned state", func(t *testing.T) {
		accepted := 0
		for _, tag := range []string{"live", "armed", "ltv_low"} {
			for _, pin := range []int64{0, 200, 400, 600} {
				pinned := feedShiftedReads(gridReads(t, "51302915", tag), pin)
				sim := simFor(t, pinned, prod)
				moved := map[int64]*PoolSimulator{}
				for _, b := range []int64{-round, -35, 35, round} {
					m := simFor(t, feedShiftedReads(pinned, b), Policy{})
					m.nowFn = sim.nowFn
					moved[b] = m
				}
				for _, k := range quoteCases() {
					want, err := sim.calcAmountOut(amountIn(sim, k.sell, k.a), int(VenueSwap))
					if err != nil {
						continue
					}
					accepted++
					for b, m := range moved {
						got, err := m.calcAmountOut(amountIn(m, k.sell, k.a), int(VenueSwap))
						where := fmt.Sprintf("51302915/%s pinned %+d bps, round %+d bps, sell=%v a=%d", tag, pin, b,
							k.sell, k.a)
						require.NoError(t, err, where)
						require.Zero(t, got.TokenAmountOut.Amount.Cmp(want.TokenAmountOut.Amount), where)
						require.Zero(t, got.RemainingTokenAmountIn.Amount.Cmp(want.RemainingTokenAmountIn.Amount), where)
					}
				}
			}
		}
		require.Greater(t, accepted, 300, "accepted quotes across the pinned drifts")
	})

	t.Run("a clipped sell the feed re-prices is refused", func(t *testing.T) {
		reads := gridReads(t, "51302915", "ltv_low")
		const sell = uint64(14_190)
		off := simFor(t, reads, Policy{})
		clipped, err := off.calcAmountOut(amountIn(off, true, sell), int(VenueSwap))
		require.NoError(t, err)
		require.Equal(t, "9049312", clipped.TokenAmountOut.Amount.String())
		require.Equal(t, "2184", clipped.RemainingTokenAmountIn.Amount.String(), "the ceiling clips this sell")
		// One ordinary round pays the user 45,156 less on the same input, and nothing refuses it.
		down := simFor(t, feedShiftedReads(reads, -round), Policy{})
		after, err := down.calcAmountOut(amountIn(down, true, sell), int(VenueSwap))
		require.NoError(t, err)
		require.Equal(t, "9004156", after.TokenAmountOut.Amount.String())
		require.Equal(t, "2244", after.RemainingTokenAmountIn.Amount.String())
		// Under the default policy the quote is not published at all.
		on := simFor(t, reads, prod)
		_, err = on.calcAmountOut(amountIn(on, true, sell), int(VenueSwap))
		require.ErrorIs(t, err, ErrFeedMoveMargin)
		// An unclipped sell on the same state is unaffected.
		small, err := on.calcAmountOut(amountIn(on, true, 1_000), int(VenueSwap))
		require.NoError(t, err)
		require.Zero(t, small.RemainingTokenAmountIn.Amount.Sign())
	})

	t.Run("a feed-priced leverage gate is refused", func(t *testing.T) {
		// The snapshot's feed 3% below the hook's reservation price, which is more than the keeper's spread: the
		// lever-down then prices at the curve and settles against gates valued at the feed.
		reads := feedShiftedReads(gridReads(t, "51302915", "btc_up10"), -300)
		const buy = uint64(35_823)
		off := simFor(t, reads, Policy{LeverRouting: true, PriceAgeMarginSec: defaultPriceAgeMarginSec,
			SpreadAgeMarginSec: defaultSpreadAgeMarginSec, DebtDriftSec: defaultDebtDriftSec,
			MaxSnapshotAgeSec: defaultMaxSnapshotAgeSec})
		quoted, err := off.calcAmountOut(amountIn(off, false, buy), int(VenueLever))
		require.NoError(t, err, "the band margin alone accepts this fill")
		require.Positive(t, quoted.TokenAmountOut.Amount.Sign())
		// A 10 bps round -- well inside one ordinary cbBTC round -- and the pool reverts it.
		up := simFor(t, feedShiftedReads(reads, 10), Policy{LeverRouting: true})
		_, err = up.calcAmountOut(amountIn(up, false, buy), int(VenueLever))
		require.ErrorIs(t, err, ErrLevValueLeak)
		// The default policy refuses it instead of quoting it.
		on := simFor(t, reads, prod)
		_, err = on.calcAmountOut(amountIn(on, false, buy), int(VenueLever))
		require.ErrorIs(t, err, ErrFeedMoveMargin)
	})
}

// TestSimulatorMarginsKeepTheVenue: with LeverRouting on, the venue is picked on the two settlements alone, so a
// margin can refuse the routed quote but never hand it to the other venue at a different price. Before, a margin
// refusal of the swap venue fell through to the leverage venue: on 51302915/band_tight the router was handed 23
// sats for a buy the swap venue prices at 46.
func TestSimulatorMarginsKeepTheVenue(t *testing.T) {
	t.Parallel()
	prod := Policy{LeverRouting: true, PriceBandMarginBps: defaultPriceBandMarginBps,
		PriceAgeMarginSec: defaultPriceAgeMarginSec, SpreadAgeMarginSec: defaultSpreadAgeMarginSec,
		DebtDriftSec: defaultDebtDriftSec, MaxSnapshotAgeSec: defaultMaxSnapshotAgeSec}

	reads := gridReads(t, "51302915", "band_tight")
	const buy = uint64(35_823)
	plain := simFor(t, reads, Policy{LeverRouting: true})
	free, err := plain.CalcAmountOut(amountIn(plain, false, buy))
	require.NoError(t, err)
	require.Equal(t, VenueSwap, free.SwapInfo.(SwapInfo).Venue)
	require.Equal(t, "46", free.TokenAmountOut.Amount.String())

	sim := simFor(t, reads, prod)
	_, err = sim.calcAmountOut(amountIn(sim, false, buy), int(VenueSwap))
	require.ErrorIs(t, err, ErrBandMargin, "the margin refuses the venue the pick lands on")
	lever, err := sim.calcAmountOut(amountIn(sim, false, buy), int(VenueLever))
	require.NoError(t, err, "and the leverage venue would fill")
	require.Equal(t, "23", lever.TokenAmountOut.Amount.String())
	_, err = sim.CalcAmountOut(amountIn(sim, false, buy))
	require.ErrorIs(t, err, ErrBandMargin, "so the routed quote is refused, not re-priced onto venue 1")

	// The general property over the grid: a margined routed quote is on the venue the margin-free pick chose.
	checked, combo := 0, 0
	for _, blk := range []string{"51302915", "51313000", "51324800"} {
		for _, tag := range []string{"live", "armed", "armed_hi", "capped", "band_tight", "btc_down10", "btc_up10",
			"fee_cap", "fee_floor", "ltv_low", "pin_low", "accrued", "stale_spread"} {
			raw, ok := gridStateJSON(t, blk, tag)
			if !ok || raw == nil {
				continue
			}
			r := gridReads(t, blk, tag)
			base, margined := simFor(t, r, Policy{LeverRouting: true}), simFor(t, r, prod)
			combo++
			for ci, k := range quoteCases() {
				if !sampleCase(ci, combo, 8) {
					continue
				}
				rm, em := margined.CalcAmountOut(amountIn(margined, k.sell, k.a))
				if em != nil {
					continue
				}
				rb, eb := base.CalcAmountOut(amountIn(base, k.sell, k.a))
				where := fmt.Sprintf("%s/%s sell=%v a=%d", blk, tag, k.sell, k.a)
				require.NoError(t, eb, where)
				require.Equal(t, rb.SwapInfo.(SwapInfo).Venue, rm.SwapInfo.(SwapInfo).Venue, where)
				require.Zero(t, rb.TokenAmountOut.Amount.Cmp(rm.TokenAmountOut.Amount), where)
				checked++
			}
		}
	}
	require.Greater(t, checked, 50, "routed quotes compared with the margin-free pick")
}

func swapInfoUsed(r *pool.CalcAmountOutResult) string {
	si := r.SwapInfo.(SwapInfo)
	return si.AmountInUsed.Dec()
}

// TestSimulatorLeverDownInput: a lever-down's RemainingTokenAmountIn is headroom the fill needs, not input the
// pool could not use. FLAMMLeverLib._planDown sizes the hook fill on the whole loanIn (FLAMMLeverLib.sol:124-141)
// and then charges only payNative = ceilDiv(amountInUsed - virtualLeg, scale) (:140), so re-quoting at the used
// amount is a smaller trade that pays proportionally less and leaves a remainder of its own. The swap venue, and
// every other source in the library, returns the same output when re-quoted at what it used. A lever-down hop can
// therefore not be trimmed to its used amount: its residual has to be re-routed or accepted, which is one of the
// conditions on enabling LeverRouting (README).
func TestSimulatorLeverDownInput(t *testing.T) {
	t.Parallel()
	sim := simFor(t, gridReads(t, "51313000", "btc_up10"), Policy{LeverRouting: true})
	const buy = uint64(125_814_386)
	down, err := sim.calcAmountOut(amountIn(sim, false, buy), int(VenueLever))
	require.NoError(t, err)
	require.Equal(t, "77857", down.TokenAmountOut.Amount.String())
	require.Equal(t, "52237196", down.RemainingTokenAmountIn.Amount.String())
	used := new(big.Int).Sub(new(big.Int).SetUint64(buy), down.RemainingTokenAmountIn.Amount)
	require.Equal(t, "73577190", used.String())
	require.Equal(t, used.String(), swapInfoUsed(down), "the pool pulls payNative")

	trimmed, err := sim.calcAmountOut(amountIn(sim, false, used.Uint64()), int(VenueLever))
	require.NoError(t, err)
	require.Equal(t, "47474", trimmed.TokenAmountOut.Amount.String(), "the remainder is not spare input")
	require.Less(t, trimmed.TokenAmountOut.Amount.Cmp(down.TokenAmountOut.Amount), 0)
	require.Equal(t, "31852243", trimmed.RemainingTokenAmountIn.Amount.String(), "and it leaves one of its own")

	// The swap venue's control: re-quoting at what it used pays exactly the same.
	ctl := simFor(t, gridReads(t, "51302915", "armed"), Policy{LeverRouting: true})
	const ctlBuy = uint64(5_000_000)
	full, err := ctl.calcAmountOut(amountIn(ctl, false, ctlBuy), int(VenueSwap))
	require.NoError(t, err)
	ctlUsed := new(big.Int).Sub(new(big.Int).SetUint64(ctlBuy), full.RemainingTokenAmountIn.Amount)
	again, err := ctl.calcAmountOut(amountIn(ctl, false, ctlUsed.Uint64()), int(VenueSwap))
	require.NoError(t, err)
	require.Zero(t, again.TokenAmountOut.Amount.Cmp(full.TokenAmountOut.Amount))

	// Routing off, the leverage venue is never the one quoted, so the semantics never reach a route.
	closed := simFor(t, gridReads(t, "51313000", "btc_up10"), Policy{})
	routed, err := closed.CalcAmountOut(amountIn(closed, false, buy))
	if err == nil {
		require.Equal(t, VenueSwap, routed.SwapInfo.(SwapInfo).Venue)
	}
}
