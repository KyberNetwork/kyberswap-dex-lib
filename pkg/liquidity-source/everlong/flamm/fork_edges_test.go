package everlongflamm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Edge windows, a long sequence and a gas grid on an anvil fork of Base through the EverlongFlammAdapter artifact
// (same gating as fork_parity_test.go: EVERLONG_FLAMM_FORK_RPC, EVERLONG_ADAPTER_OUT; EVERLONG_FLAMM_FORK_BLOCK
// overrides the default block 51330060, a later head than fork_parity_test.go's). Every quote is compared with
// executeEverlongFlamm to the wei and by revert class; nothing is sampled away:
//
//   - edges: unit-stride windows around every edge the port finds (the band's upper edge, the dust floor, the
//     partial-fill boundary under notional caps, the leverage curve's edges) on the pool as deployed, armed at a
//     spread below the floor, mid-range and above the band ceiling, with a disabled staleness window, and with the
//     clock moved forward to and past the feed and spread deadlines (eth_call block time overrides);
//   - a long seeded sequence of routed and forced fills on both venues with irregular time gaps, adopted only by
//     UpdateBalance, including refusals mined as failing transactions, then a fresh refresh equal to the state.

const forkEdgesBlock = "51330060"

var forkEdgesABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(`[
 {"type":"function","name":"setBounds","stateMutability":"nonpayable","inputs":[{"name":"a","type":"uint24"},{"name":"b","type":"uint24"}],"outputs":[]},
 {"type":"function","name":"setMaxSpreadAge","stateMutability":"nonpayable","inputs":[{"name":"a","type":"uint32"}],"outputs":[]}
]`))
	if err != nil {
		panic(err)
	}
	return a
}()

// ---------------------------------------------------------------- mismatch report

type forkMismatch struct {
	Area  string `json:"area"`
	Input string `json:"input"`
	Go    string `json:"go"`
	Chain string `json:"onchain"`
}

type forkReport struct {
	t      *testing.T
	mu     sync.Mutex
	list   []forkMismatch
	counts map[string]int
	quiet  bool
}

func newForkReport(t *testing.T) *forkReport {
	r := &forkReport{t: t, counts: map[string]int{}}
	t.Cleanup(func() {
		t.Logf("counts %v; %d mismatches", r.counts, len(r.list))
	})
	return r
}

func (r *forkReport) mismatch(area, input, goS, chain string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.list = append(r.list, forkMismatch{area, input, goS, chain})
	if len(r.list) <= 60 && !r.quiet {
		r.t.Errorf("MISMATCH %s %s: go %s, chain %s", area, input, goS, chain)
	}
}

func (r *forkReport) count(k string) {
	r.mu.Lock()
	r.counts[k]++
	r.mu.Unlock()
}

// ---------------------------------------------------------------- adapter calls

type adapterCall struct {
	venue  uint8
	sell   bool
	amount *big.Int
}

// batchFills eth_calls executeEverlongFlamm for each call at the latest block (with its timestamp overridden to ts
// when non-zero), the adapter holding exactly amountIn.
func (e *forkEnv) batchFills(t *testing.T, calls []adapterCall, ts uint64) []forkFill {
	t.Helper()
	out := make([]forkFill, len(calls))
	for start := 0; start < len(calls); start += 40 {
		end := min(start+40, len(calls))
		elems := make([]rpc.BatchElem, end-start)
		results := make([]hexutil.Bytes, end-start)
		for i := start; i < end; i++ {
			c := calls[i]
			in, outTok := e.tokens(c.sell)
			calldata, err := forkABI.Pack("executeEverlongFlamm", adapterData(e.pool, c.venue), c.amount, in, outTok,
				forkRecipient)
			require.NoError(t, err)
			overrides := map[common.Address]map[string]any{
				forkAdapter: {"code": hexutil.Encode(e.code)},
				in:          {"stateDiff": map[common.Hash]common.Hash{balanceSlot(forkAdapter): common.BigToHash(c.amount)}},
			}
			args := []any{map[string]any{"from": forkDeployer, "to": forkAdapter, "data": hexutil.Encode(calldata),
				"gas": hexutil.Uint64(30_000_000)}, "latest", overrides}
			if ts != 0 {
				args = append(args, map[string]any{"time": hexutil.Uint64(ts)})
			}
			elems[i-start] = rpc.BatchElem{Method: "eth_call", Args: args, Result: &results[i-start]}
		}
		require.NoError(t, e.f.rc.BatchCallContext(context.Background(), elems))
		for i := range elems {
			if elems[i].Error != nil {
				data, ok := revertData(elems[i].Error)
				require.True(t, ok, "adapter call: %v", elems[i].Error)
				out[start+i] = forkFill{reverted: true, revert: data}
				continue
			}
			vals, err := forkABI.Methods["executeEverlongFlamm"].Outputs.Unpack(results[i])
			require.NoError(t, err)
			out[start+i] = forkFill{unused: vals[0].(*big.Int), out: vals[1].(*big.Int)}
		}
	}
	return out
}

func describeQuote(res *pool.CalcAmountOutResult, err error) string {
	if err != nil {
		return "refused " + err.Error()
	}
	return fmt.Sprintf("venue %d out %s unused %s", res.SwapInfo.(SwapInfo).Venue, res.TokenAmountOut.Amount,
		res.RemainingTokenAmountIn.Amount)
}

func describeFill(f forkFill) string {
	if f.reverted {
		if e := forkRevertError(f.revert); e != nil {
			return "reverted " + e.Error()
		}
		return "reverted " + hexutil.Encode(f.revert)
	}
	return fmt.Sprintf("out %s unused %s", f.out, f.unused)
}

// chainSpreadLive reports whether a spread refusal by sim at ts would be a mismatch: the pool's spread hook
// answers at ts (LeverageSpreadHook.spreadPpm, LeverageSpreadHook.sol:74-77: live through lastSetTs + maxSpreadAge
// inclusive, or always with a zero maxSpreadAge) with a ppm the pool accepts, and no margin moves the deadline.
func chainSpreadLive(sim *PoolSimulator, ts uint64) bool {
	s := sim.state
	if !s.Hooks.hasSpread() {
		return false
	}
	sp := s.Hooks.Spread.EverlongSpread
	if !sp.Spread.Lt(uPpm) || sim.Policy.SpreadAgeMarginSec != 0 {
		return false
	}
	age := sp.MaxSpreadAge.Uint64()
	return age == 0 || ts <= sp.LastSetTs.Uint64()+age
}

// judgeFill compares one quote with the adapter's answer on the venue the quote names; spreadLive
// (chainSpreadLive) makes a spread refusal a mismatch.
func judgeFill(rep *forkReport, area string, c adapterCall, res *pool.CalcAmountOutResult, simErr error, f forkFill,
	spreadLive bool) {
	input := fmt.Sprintf("venue=%d sell=%v amountIn=%s", c.venue, c.sell, c.amount)
	switch {
	case f.reverted:
		want := forkRevertError(f.revert)
		switch {
		case want == nil:
			rep.mismatch(area, input, describeQuote(res, simErr), "unmapped revert "+hexutil.Encode(f.revert))
		case simErr == nil:
			rep.mismatch(area, input, describeQuote(res, simErr), describeFill(f))
		case errors.Is(simErr, want):
			rep.count(area + "/both-refused")
		case errors.Is(simErr, ErrSpreadNotLive) && !spreadLive:
			rep.count(area + "/policy-refused(chain reverts too)")
		default:
			rep.mismatch(area, input, describeQuote(res, simErr), describeFill(f))
		}
	case simErr != nil:
		if errors.Is(simErr, ErrSpreadNotLive) && !spreadLive {
			rep.count(area + "/policy-refused(chain fills)")
			return
		}
		rep.mismatch(area, input, describeQuote(res, simErr), describeFill(f))
	default:
		if res.SwapInfo.(SwapInfo).Venue != c.venue || res.TokenAmountOut.Amount.Cmp(f.out) != 0 ||
			res.RemainingTokenAmountIn.Amount.Cmp(f.unused) != 0 {
			rep.mismatch(area, input, describeQuote(res, simErr), describeFill(f))
			return
		}
		key := area + "/identical"
		if f.unused.Sign() > 0 {
			key += "-partial"
		}
		rep.count(key)
	}
}

// compareFills quotes every amount on venue (-1: routed, compared on the venue the simulator picked; a double
// refusal is compared on the swap venue) at ts and checks the adapter at the same clock.
func (e *forkEnv) compareFills(t *testing.T, rep *forkReport, area string, sim *PoolSimulator, venue int, sell bool,
	amounts []uint64, ts uint64) {
	t.Helper()
	sim.nowFn = func() uint64 { return ts }
	calls := make([]adapterCall, len(amounts))
	res := make([]*pool.CalcAmountOutResult, len(amounts))
	errs := make([]error, len(amounts))
	for i, a := range amounts {
		res[i], errs[i] = sim.calcAmountOut(amountIn(sim, sell, a), venue)
		v := uint8(max(venue, 0))
		if errs[i] == nil {
			v = res[i].SwapInfo.(SwapInfo).Venue
		}
		calls[i] = adapterCall{venue: v, sell: sell, amount: new(big.Int).SetUint64(a)}
	}
	fills := e.batchFills(t, calls, ts)
	for i := range amounts {
		judgeFill(rep, area, calls[i], res[i], errs[i], fills[i], chainSpreadLive(sim, ts))
	}
}

// ---------------------------------------------------------------- the port's own edges

func simAccepts(sim *PoolSimulator, venue int, sell bool, a uint64) (bool, bool) {
	r, err := sim.calcAmountOut(amountIn(sim, sell, a), venue)
	if err != nil {
		return false, false
	}
	return true, r.RemainingTokenAmountIn.Amount.Sign() > 0
}

func addWindow(set map[uint64]bool, center uint64, w uint64) {
	lo := uint64(1)
	if center > w {
		lo = center - w
	}
	for a := lo; a <= center+w; a++ {
		set[a] = true
	}
}

// edgeAmounts: a log grid, unit windows around the upper acceptance edge, the smallest accepted amount (with a
// strided sweep of the non-interval dust region below 4x it), and the first partial fill.
func edgeAmounts(sim *PoolSimulator, venue int, sell bool, w uint64) []uint64 {
	set := map[uint64]bool{}
	hiEnd := uint64(20_000_000)
	if !sell {
		hiEnd = 40_000_000_000
	}
	for _, a := range liveGrid(1, hiEnd, 36) {
		set[a] = true
	}
	var firstOk uint64
	for a := uint64(1); a <= hiEnd; a = a + a/8 + 1 {
		if ok, _ := simAccepts(sim, venue, sell, a); ok {
			firstOk = a
			break
		}
	}
	if firstOk == 0 {
		return sortedAmounts(set)
	}
	// the smallest accepted amount at unit resolution below firstOk
	low := firstOk
	for a := firstOk - min(firstOk-1, firstOk/8+1, 400); a < firstOk; a++ {
		if ok, _ := simAccepts(sim, venue, sell, a); ok {
			low = a
			break
		}
	}
	addWindow(set, low, w)
	stride := max(uint64(1), 3*low/200)
	for a := low; a <= 4*low; a += stride {
		set[a] = true
	}
	// the largest accepted amount from a doubling walk, then bisection to a unit edge
	lastOk := firstOk
	a := firstOk
	for a < hiEnd*8 {
		a *= 2
		if ok, _ := simAccepts(sim, venue, sell, a); !ok {
			break
		}
		lastOk = a
	}
	hi := a
	for hi-lastOk > 1 {
		mid := (hi + lastOk) / 2
		if ok, _ := simAccepts(sim, venue, sell, mid); ok {
			lastOk = mid
		} else {
			hi = mid
		}
	}
	addWindow(set, lastOk, w)
	// the first partial fill between firstOk and lastOk, if any
	var full, part uint64
	for _, g := range liveGrid(firstOk, max(lastOk, firstOk+1), 60) {
		ok, partial := simAccepts(sim, venue, sell, g)
		if ok && !partial {
			full = g
		}
		if ok && partial {
			part = g
			break
		}
	}
	if part != 0 && full != 0 && full < part {
		for part-full > 1 {
			mid := (part + full) / 2
			if ok, partial := simAccepts(sim, venue, sell, mid); ok && partial {
				part = mid
			} else {
				full = mid
			}
		}
		addWindow(set, part, w)
	}
	return sortedAmounts(set)
}

func sortedAmounts(set map[uint64]bool) []uint64 {
	out := make([]uint64, 0, len(set))
	for a := range set {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// armLeverage unpauses leverage, widens the spread bounds, posts spread and sets the staleness window (0 disables).
func (e *forkEnv) armLeverage(t *testing.T, spread uint64, maxAge uint32) {
	t.Helper()
	f := e.f
	core := f.view(e.pool, "core")[0].(common.Address)
	curator := f.view(core, "owner")[0].(common.Address)
	keeper := f.view(core, "keeper")[0].(common.Address)
	f.impersonate(curator)
	f.impersonate(keeper)
	sh := c104.SpreadHook
	ts := f.head().Time
	if lp, _ := e.levPaused(t); lp {
		data, err := forkABI.Pack("setLevPaused", false)
		require.NoError(t, err)
		ts += 2
		f.send(curator, e.pool, data, ts)
	}
	bounds, err := forkEdgesABI.Pack("setBounds", big.NewInt(0), big.NewInt(150_000))
	require.NoError(t, err)
	ts += 2
	f.send(curator, sh, bounds, ts)
	age, err := forkEdgesABI.Pack("setMaxSpreadAge", maxAge)
	require.NoError(t, err)
	ts += 2
	f.send(curator, sh, age, ts)
	post, err := forkABI.Pack("setSpread", new(big.Int).SetUint64(spread))
	require.NoError(t, err)
	ts += 2
	f.send(keeper, sh, post, ts)
}

func (e *forkEnv) levPaused(t *testing.T) (bool, error) {
	data, err := flammABI.Pack("switches")
	require.NoError(t, err)
	var out hexutil.Bytes
	e.f.rpc(&out, "eth_call", map[string]any{"to": e.pool, "data": hexutil.Encode(data)}, "latest")
	vals, err := flammABI.Methods["switches"].Outputs.Unpack(out)
	require.NoError(t, err)
	return vals[2].(bool), nil
}

func (e *forkEnv) poolCurator(t *testing.T) common.Address {
	core := e.f.view(e.pool, "core")[0].(common.Address)
	curator := e.f.view(core, "owner")[0].(common.Address)
	e.f.impersonate(curator)
	return curator
}

// edgeSuite compares every venue and direction the simulator can quote at its snapshot.
func (e *forkEnv) edgeSuite(t *testing.T, rep *forkReport, area string, sim *PoolSimulator, lever bool, w uint64) {
	t.Helper()
	ts := sim.state.Timestamp
	venues := []int{int(VenueSwap)}
	if lever {
		venues = append(venues, int(VenueLever), -1)
	}
	for _, venue := range venues {
		for _, sell := range []bool{true, false} {
			v := venue
			if v < 0 {
				v = int(VenueSwap)
			}
			amounts := edgeAmounts(sim, v, sell, w)
			if venue < 0 {
				amounts = liveGrid(1, 20_000_000, 48)
				if !sell {
					amounts = liveGrid(500, 40_000_000_000, 48)
				}
			}
			t.Logf("%s venue %d sell=%v: %d amounts", area, venue, sell, len(amounts))
			e.compareFills(t, rep, fmt.Sprintf("%s/venue%d/sell=%v", area, venue, sell), sim, venue, sell, amounts, ts)
		}
	}
}

// ---------------------------------------------------------------- tests

func TestForkEdges(t *testing.T) {
	e := newForkEnvAt(t, forkEdgesBlock)
	rep := newForkReport(t)
	const w = 24

	// ---- as deployed (leverage paused at the pinned head)
	sim := e.track(t, Policy{})
	t.Logf("deployed: block %d ts %d paused %v levPaused %v", sim.state.Block, sim.state.Timestamp, sim.state.Paused,
		sim.state.LevPaused)
	e.edgeSuite(t, rep, "deployed", sim, false, w)
	e.compareFills(t, rep, "deployed/lever-paused", sim, int(VenueLever), true, []uint64{1, 5_000, 50_000}, sim.state.Timestamp)
	e.compareFills(t, rep, "deployed/lever-paused", sim, int(VenueLever), false, []uint64{1, 5_000_000}, sim.state.Timestamp)

	// ---- armed at a spread below the floor (clamped to 2,500), mid-range, above the band ceiling (clamped to the
	// band), and with the staleness window disabled
	for _, arm := range []struct {
		name   string
		spread uint64
		age    uint32
	}{{"spread1000", 1_000, 3600}, {"spread13000", 13_000, 3600}, {"spread150000", 150_000, 3600},
		{"spread17500-noage", 17_500, 0}} {
		e.armLeverage(t, arm.spread, arm.age)
		armed := e.track(t, Policy{LeverRouting: true})
		require.False(t, armed.state.LevPaused)
		e.edgeSuite(t, rep, arm.name, armed, true, w)
	}

	// ---- the clock moved forward without a refresh: accrual, then the spread and feed deadlines
	e.armLeverage(t, 13_000, 3600)
	armed := e.track(t, Policy{LeverRouting: true})
	s := armed.state
	sp := s.Hooks.Spread.EverlongSpread
	spreadDeadline := sp.LastSetTs.Uint64() + sp.MaxSpreadAge.Uint64()
	feedDeadline := min(s.Feed.Asset.Round.UpdatedAt.Uint64()+s.Feed.Asset.Heartbeat.Uint64(),
		s.Feed.Loans[0].Round.UpdatedAt.Uint64()+s.Feed.Loans[0].Heartbeat.Uint64())
	t.Logf("drift: snapshot %d, spread deadline %d, feed deadline %d", s.Timestamp, spreadDeadline, feedDeadline)
	clocks := []uint64{s.Timestamp + 1, s.Timestamp + 600}
	for _, d := range []uint64{spreadDeadline, feedDeadline} {
		if d > s.Timestamp+1 {
			clocks = append(clocks, d-1, d, d+1)
		}
	}
	clocks = append(clocks, s.Timestamp+86_400*3)
	for _, ts := range clocks {
		for _, venue := range []int{int(VenueSwap), int(VenueLever), -1} {
			for _, sell := range []bool{true, false} {
				amounts := liveGrid(1, 400_000, 20)
				if !sell {
					amounts = liveGrid(1_000, 400_000_000, 20)
				}
				e.compareFills(t, rep, fmt.Sprintf("drift+%d/venue%d/sell=%v", ts-s.Timestamp, venue, sell), armed, venue,
					sell, amounts, ts)
			}
		}
	}

	// ---- a short staleness window: the spread deadline before the feed deadline
	e.armLeverage(t, 13_000, 600)
	short := e.track(t, Policy{LeverRouting: true})
	shortSpread := short.state.Hooks.Spread.EverlongSpread
	sd := shortSpread.LastSetTs.Uint64() + shortSpread.MaxSpreadAge.Uint64()
	t.Logf("short spread window: snapshot %d, spread deadline %d", short.state.Timestamp, sd)
	for _, ts := range []uint64{sd - 1, sd, sd + 1} {
		for _, venue := range []int{int(VenueLever), -1} {
			for _, sell := range []bool{true, false} {
				amounts := liveGrid(1, 400_000, 20)
				if !sell {
					amounts = liveGrid(1_000, 400_000_000, 20)
				}
				e.compareFills(t, rep, fmt.Sprintf("spread-deadline%+d/venue%d/sell=%v", int64(ts)-int64(sd), venue, sell),
					short, venue, sell, amounts, ts)
			}
		}
	}

	// ---- notional caps: the sell's partial-fill boundary and the buy's NotionalCap boundary
	curator := e.poolCurator(t)
	loan := s.Pool.Loans[0]
	for _, capUSDC := range []int64{8_000_000, 50_000_000, 1} {
		data, err := forkABI.Pack("setLoanConfig", uint8(0), loan.SwapPriceBandWad.Uint64(), loan.FeeFloorWad.Uint64(),
			big.NewInt(capUSDC), loan.ReserveTarget.ToBig())
		require.NoError(t, err)
		e.f.send(curator, e.pool, data, e.f.head().Time+2)
		capped := e.track(t, Policy{LeverRouting: true})
		e.edgeSuite(t, rep, fmt.Sprintf("cap%d", capUSDC), capped, true, w)
	}
	require.Empty(t, rep.list)
}

// forkSeqStep is one fill of the long sequence.
type forkSeqStep struct {
	venue int
	sell  bool
	a     uint64
}

// forkSequence is the offline record of TestForkSequence (testdata/fork_sequence_<block>.json, written with
// EVERLONG_FLAMM_RECORD=sequence): the refresh the sequence starts from, every step as quoted and as the chain
// settled it, and the refresh after the last step. TestSimulatorForkSequence replays it.
type forkSequence struct {
	Block   uint64             `json:"block"`
	Start   entity.Pool        `json:"start"`
	Steps   []forkSequenceStep `json:"steps"`
	Refresh entity.Pool        `json:"refresh"`
}

// forkSequenceStep is one step: the forced venue (-1 routed), direction, input and clock; the adapter's fill (out,
// unused) or its revert, or the simulator's own spread-policy refusal (no transaction).
type forkSequenceStep struct {
	Venue    int      `json:"venue"`
	Sell     bool     `json:"sell"`
	AmountIn uint64   `json:"amountIn"`
	Ts       uint64   `json:"ts"`
	Out      *big.Int `json:"out,omitempty"`
	Unused   *big.Int `json:"unused,omitempty"`
	Revert   *string  `json:"revert,omitempty"`
	Policy   bool     `json:"policy,omitempty"`
}

func TestForkSequence(t *testing.T) {
	e := newForkEnvAt(t, forkEdgesBlock)
	rep := newForkReport(t)
	e.armLeverage(t, 13_000, 0) // no spread staleness: the sequence may run long
	sim := e.track(t, Policy{LeverRouting: true})
	record := forkSequence{Block: sim.state.Block, Start: e.tracked}
	s := sim.state
	feedDeadline := min(s.Feed.Asset.Round.UpdatedAt.Uint64()+s.Feed.Asset.Heartbeat.Uint64(),
		s.Feed.Loans[0].Round.UpdatedAt.Uint64()+s.Feed.Loans[0].Heartbeat.Uint64())
	budget := feedDeadline - s.Timestamp
	t.Logf("sequence from %d, %d s of feed life", s.Timestamp, budget)
	require.Greater(t, budget, uint64(600), "the fork block's feed round is too old for a sequence")

	rng := rand.New(rand.NewSource(0xF1A33))
	ts := s.Timestamp
	adopted, refusedMined, policy := map[string]int{}, 0, 0
	gasMax := map[string]uint64{}
	const steps = 90
	for i := 0; i < steps; i++ {
		st := forkSeqStep{venue: rng.Intn(3) - 1, sell: rng.Intn(2) == 0}
		gap := uint64(2 + rng.Intn(20))
		if rng.Intn(10) == 0 {
			gap = uint64(60 + rng.Intn(400))
		}
		if ts+gap+uint64(steps-i)*3 >= feedDeadline {
			gap = 2
		}
		ts += gap
		sim.nowFn = func() uint64 { return ts }
		venue := st.venue
		if venue < 0 {
			venue = int(VenueSwap)
		}
		// sizes: log-uniform, or exactly at / one past the port's current edge
		lo, hi := 1.0, 400_000.0
		if !st.sell {
			lo, hi = 1_000.0, 400_000_000.0
		}
		st.a = uint64(lo * math.Pow(hi/lo, rng.Float64()))
		if rng.Intn(4) == 0 {
			amts := edgeAmounts(sim, venue, st.sell, 0)
			st.a = amts[rng.Intn(len(amts))]
		}
		params := amountIn(sim, st.sell, st.a)
		res, err := sim.calcAmountOut(params, st.venue)
		where := fmt.Sprintf("step %d venue=%d sell=%v amountIn=%d ts=%d", i, st.venue, st.sell, st.a, ts)
		in, out := e.tokens(st.sell)
		if err != nil && errors.Is(err, ErrSpreadNotLive) {
			if chainSpreadLive(sim, ts) {
				rep.mismatch("sequence", where, describeQuote(res, err), "the pool's spread is live")
			}
			policy++
			record.Steps = append(record.Steps, forkSequenceStep{Venue: st.venue, Sell: st.sell, AmountIn: st.a, Ts: ts,
				Policy: true})
			ts -= gap
			continue
		}
		v := uint8(venue)
		if err == nil {
			v = res.SwapInfo.(SwapInfo).Venue
		}
		fill := e.mineFill(t, v, in, out, params.TokenAmountIn.Amount, ts)
		step := forkSequenceStep{Venue: st.venue, Sell: st.sell, AmountIn: st.a, Ts: ts, Out: fill.out,
			Unused: fill.unused}
		if fill.reverted {
			r := hexutil.Encode(fill.revert)
			step.Revert, step.Out, step.Unused = &r, nil, nil
		}
		record.Steps = append(record.Steps, step)
		switch {
		case err == nil && fill.reverted:
			rep.mismatch("sequence", where, describeQuote(res, err), describeFill(fill))
			t.FailNow()
		case err == nil:
			if res.TokenAmountOut.Amount.Cmp(fill.out) != 0 || res.RemainingTokenAmountIn.Amount.Cmp(fill.unused) != 0 {
				rep.mismatch("sequence", where, describeQuote(res, err), describeFill(fill))
				t.FailNow()
			}
			if fill.gas > uint64(res.Gas) {
				rep.mismatch("sequence/gas", where, fmt.Sprintf("gas %d", res.Gas), fmt.Sprintf("receipt gas %d", fill.gas))
				e.traceFailures(t, common.Hash{})
			}
			sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
			key := fmt.Sprintf("venue%d/sell=%v", v, st.sell)
			adopted[key]++
			gasMax[key] = max(gasMax[key], fill.gas)
			t.Logf("%s: %s, receipt gas %d (estimate %d)", where, describeQuote(res, err), fill.gas, res.Gas)
			rep.count("sequence/identical")
		case !fill.reverted:
			rep.mismatch("sequence", where, describeQuote(res, err), describeFill(fill))
			t.FailNow()
		default:
			want := forkRevertError(fill.revert)
			if want == nil || !errors.Is(err, want) {
				rep.mismatch("sequence", where, describeQuote(res, err), describeFill(fill))
			}
			refusedMined++
			sim.nowFn = func() uint64 { return ts }
		}
	}
	t.Logf("sequence: adopted %v, refusals mined %d, policy refusals %d, last ts %d; receipt gas maxima %v", adopted,
		refusedMined, policy, ts, gasMax)
	fresh := e.track(t, Policy{LeverRouting: true})
	got := sim.state.clone()
	got.Block, got.Timestamp = fresh.state.Block, fresh.state.Timestamp
	if d := e2eDiff(got, fresh.state); len(d) != 0 {
		rep.mismatch("sequence/post-state", "after the sequence", "simulator state", strings.Join(d, "; "))
	}
	require.Empty(t, rep.list)
	if os.Getenv("EVERLONG_FLAMM_RECORD") == "sequence" {
		record.Refresh = e.tracked
		record.Start.Timestamp, record.Refresh.Timestamp = 0, 0
		raw, err := json.MarshalIndent(&record, "", " ")
		require.NoError(t, err)
		path := fmt.Sprintf("testdata/fork_sequence_%d.json", record.Block)
		require.NoError(t, os.WriteFile(path, append(raw, '\n'), 0o644))
		t.Logf("wrote %s: %d steps", path, len(record.Steps))
	}
}

// mineFill mines executeEverlongFlamm at ts from a funded adapter; a failing transaction is mined too and its revert
// data read back with an eth_call at the same block (same timestamp, and the state the failed transaction saw).
func (e *forkEnv) mineFill(t *testing.T, venue uint8, tokenIn, tokenOut common.Address, amountIn *big.Int,
	ts uint64) forkFill {
	t.Helper()
	f := e.f
	f.setBalance(tokenIn, forkAdapter, amountIn)
	before := f.balance(tokenOut, forkRecipient)
	calldata, err := forkABI.Pack("executeEverlongFlamm", adapterData(e.pool, venue), amountIn, tokenIn, tokenOut,
		forkRecipient)
	require.NoError(t, err)
	rcpt := f.send(forkDeployer, forkAdapter, calldata, ts)
	if rcpt.Status != types.ReceiptStatusSuccessful {
		var out hexutil.Bytes
		callErr := f.rc.CallContext(context.Background(), &out, "eth_call", map[string]any{"from": forkDeployer,
			"to": forkAdapter, "data": hexutil.Encode(calldata), "gas": hexutil.Uint64(8_000_000)}, "latest")
		require.Error(t, callErr, "a mined revert did not revert in replay")
		data, ok := revertData(callErr)
		require.True(t, ok)
		f.setBalance(tokenIn, forkAdapter, new(big.Int))
		return forkFill{reverted: true, revert: data}
	}
	out := new(big.Int).Sub(f.balance(tokenOut, forkRecipient), before)
	unused := f.balance(tokenIn, forkAdapter)
	f.setBalance(tokenIn, forkAdapter, new(big.Int))
	traceHashes[len(traceHashes)-1] = rcpt.TxHash
	return forkFill{unused: unused, out: out, gas: rcpt.GasUsed}
}

var traceHashes = make([]common.Hash, 1)

// traceFailures logs every reverted sub-call of a mined transaction with its gas.
func (e *forkEnv) traceFailures(t *testing.T, _ common.Hash) {
	h := traceHashes[len(traceHashes)-1]
	var trace map[string]any
	if err := e.f.rc.CallContext(context.Background(), &trace, "debug_traceTransaction", h,
		map[string]any{"tracer": "callTracer"}); err != nil {
		t.Logf("trace %s: %v", h, err)
		return
	}
	var walk func(depth int, c map[string]any)
	walk = func(depth int, c map[string]any) {
		in, _ := c["input"].(string)
		sel := in
		if len(sel) > 10 {
			sel = sel[:10]
		}
		if c["error"] != nil || depth <= 2 {
			t.Logf("  %s%s %v -> %v sel %s gas %v used %v error %v revert %v", strings.Repeat("  ", depth), c["type"],
				c["from"], c["to"], sel, c["gas"], c["gasUsed"], c["error"], c["revertReason"])
		}
		calls, _ := c["calls"].([]any)
		for _, sub := range calls {
			if m, ok := sub.(map[string]any); ok {
				walk(depth+1, m)
			}
		}
	}
	walk(0, trace)
}

// TestForkGasGrid: every venue and direction over a size grid, each fill mined from an anvil snapshot and reverted,
// on the armed head and after a few fills: the receipt gas must not exceed the simulator's estimate.
func TestForkGasGrid(t *testing.T) {
	e := newForkEnvAt(t, forkEdgesBlock)
	rep := newForkReport(t)
	e.armLeverage(t, 13_000, 0)
	maxima := map[string]uint64{}
	worst := map[string]string{}
	measure := func(phase string) {
		sim := e.track(t, Policy{LeverRouting: true})
		ts := sim.state.Timestamp
		for _, venue := range []int{int(VenueSwap), int(VenueLever)} {
			for _, sell := range []bool{true, false} {
				amounts := liveGrid(1, 3_000_000, 26)
				if !sell {
					amounts = liveGrid(100, 3_000_000_000, 26)
				}
				for _, a := range amounts {
					sim.nowFn = func() uint64 { return ts + 2 }
					res, err := sim.calcAmountOut(amountIn(sim, sell, a), venue)
					if err != nil {
						continue
					}
					var snap hexutil.Big
					e.f.rpc(&snap, "evm_snapshot")
					in, out := e.tokens(sell)
					fl := e.mineFill(t, uint8(venue), in, out, new(big.Int).SetUint64(a), ts+2)
					var ok bool
					e.f.rpc(&ok, "evm_revert", snap)
					require.True(t, ok)
					where := fmt.Sprintf("%s venue=%d sell=%v amountIn=%d", phase, venue, sell, a)
					if fl.reverted || fl.out.Cmp(res.TokenAmountOut.Amount) != 0 {
						rep.mismatch("gas/fill", where, describeQuote(res, err), describeFill(fl))
						continue
					}
					key := fmt.Sprintf("venue%d/sell=%v", venue, sell)
					if fl.gas > maxima[key] {
						maxima[key], worst[key] = fl.gas, where
					}
					if fl.gas > uint64(res.Gas) {
						rep.mismatch("gas", where, fmt.Sprintf("estimate %d", res.Gas), fmt.Sprintf("receipt gas %d", fl.gas))
					}
				}
			}
		}
	}
	measure("armed")
	// move the book: lever-ups and sells build debt, then measure again
	sim := e.track(t, Policy{LeverRouting: true})
	ts := sim.state.Timestamp
	for i, st := range []struct {
		venue uint8
		sell  bool
		a     uint64
	}{{1, true, 9_000}, {0, true, 60_000}, {1, true, 5_000}, {0, false, 30_000_000}, {1, false, 10_000_000}} {
		in, out := e.tokens(st.sell)
		fl := e.mineFill(t, st.venue, in, out, new(big.Int).SetUint64(st.a), ts+uint64(4*(i+1)))
		t.Logf("book move %d: %s", i, describeFill(fl))
	}
	measure("moved")
	t.Logf("receipt gas maxima %v at %v", maxima, worst)
	require.Empty(t, rep.list)
}
