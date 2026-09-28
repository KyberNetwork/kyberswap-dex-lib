package everlongflamm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Governance, keeper and third-party scenarios on an anvil fork of Base through the EverlongFlammAdapter artifact
// (EVERLONG_FLAMM_FORK_RPC, EVERLONG_ADAPTER_OUT; EVERLONG_FLAMM_FORK_BLOCK overrides the default block 51333700, a
// block right after a cbBTC round so sequences have feed life; EVERLONG_FLAMM_FORK_SCENARIOS, comma separated,
// runs a subset). Every scenario starts from the
// same armed snapshot (leverage unpaused, spread 13000 ppm, no staleness window), applies one change a curator,
// guardian, keeper, the protocol safe or any third party can make, refreshes through the real tracker, compares
// unit windows around every edge on both venues and the routed quote against executeEverlongFlamm, then mines a
// seeded mixed sequence adopted only through CloneState + UpdateBalance with a msgpack hop every few steps, and
// requires a fresh refresh to equal the simulator's state.

const forkScenariosBlock = "51333700"

var scenariosABI = func() abi.ABI {
	const mp = `{"name":"marketParams","type":"tuple","components":[{"name":"loanToken","type":"address"},{"name":"collateralToken","type":"address"},{"name":"oracle","type":"address"},{"name":"irm","type":"address"},{"name":"lltv","type":"uint256"}]}`
	a, err := abi.JSON(strings.NewReader(`[
 {"type":"function","name":"setPaused","stateMutability":"nonpayable","inputs":[{"name":"p","type":"bool"}],"outputs":[]},
 {"type":"function","name":"setDials","stateMutability":"nonpayable","inputs":[{"name":"phi","type":"uint64"},{"name":"ltv","type":"uint64"}],"outputs":[]},
 {"type":"function","name":"setFeeBounds","stateMutability":"nonpayable","inputs":[{"name":"f","type":"uint64"},{"name":"c","type":"uint64"}],"outputs":[]},
 {"type":"function","name":"setFeatures","stateMutability":"nonpayable","inputs":[{"name":"b","type":"uint256"}],"outputs":[]},
 {"type":"function","name":"setVenueFlags","stateMutability":"nonpayable","inputs":[{"name":"id","type":"uint16"},{"name":"b","type":"bool"},{"name":"s","type":"bool"}],"outputs":[]},
 {"type":"function","name":"setVenueCaps","stateMutability":"nonpayable","inputs":[{"name":"id","type":"uint16"},{"name":"d","type":"uint128"},{"name":"s","type":"uint128"},{"name":"r","type":"uint64"}],"outputs":[]},
 {"type":"function","name":"setRiskLimits","stateMutability":"nonpayable","inputs":[{"name":"dc","type":"uint256"},{"name":"mn","type":"uint256"},{"name":"band","type":"uint64"},{"name":"rt","type":"uint256"},{"name":"md","type":"uint8"}],"outputs":[]},
 {"type":"function","name":"retireVenue","stateMutability":"nonpayable","inputs":[{"name":"id","type":"uint16"}],"outputs":[]},
 {"type":"function","name":"maintainCollateral","stateMutability":"nonpayable","inputs":[],"outputs":[{"name":"","type":"uint16"},{"name":"","type":"uint256"}]},
 {"type":"function","name":"poke","stateMutability":"nonpayable","inputs":[],"outputs":[]},
 {"type":"function","name":"recenter","stateMutability":"nonpayable","inputs":[{"name":"a","type":"uint256"},{"name":"b","type":"uint256"},{"name":"d","type":"uint256"}],"outputs":[]},
 {"type":"function","name":"skimPerformanceFee","stateMutability":"nonpayable","inputs":[],"outputs":[]},
 {"type":"function","name":"setGlobalPaused","stateMutability":"nonpayable","inputs":[{"name":"p","type":"bool"}],"outputs":[]},
 {"type":"function","name":"protocolSafe","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
 {"type":"function","name":"approve","stateMutability":"nonpayable","inputs":[{"name":"s","type":"address"},{"name":"a","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]},
 {"type":"function","name":"supplyCollateral","stateMutability":"nonpayable","inputs":[` + mp + `,{"name":"assets","type":"uint256"},{"name":"onBehalf","type":"address"},{"name":"data","type":"bytes"}],"outputs":[]},
 {"type":"function","name":"supply","stateMutability":"nonpayable","inputs":[` + mp + `,{"name":"assets","type":"uint256"},{"name":"shares","type":"uint256"},{"name":"onBehalf","type":"address"},{"name":"data","type":"bytes"}],"outputs":[{"name":"","type":"uint256"},{"name":"","type":"uint256"}]},
 {"type":"function","name":"borrow","stateMutability":"nonpayable","inputs":[` + mp + `,{"name":"assets","type":"uint256"},{"name":"shares","type":"uint256"},{"name":"onBehalf","type":"address"},{"name":"receiver","type":"address"}],"outputs":[{"name":"","type":"uint256"},{"name":"","type":"uint256"}]},
 {"type":"function","name":"repay","stateMutability":"nonpayable","inputs":[` + mp + `,{"name":"assets","type":"uint256"},{"name":"shares","type":"uint256"},{"name":"onBehalf","type":"address"},{"name":"data","type":"bytes"}],"outputs":[{"name":"","type":"uint256"},{"name":"","type":"uint256"}]}
]`))
	if err != nil {
		panic(err)
	}
	return a
}()

type morphoMarketParams struct {
	LoanToken, CollateralToken, Oracle, Irm common.Address
	Lltv                                    *big.Int
}

func (e *forkEnv) viewCall(t *testing.T, target common.Address, a *abi.ABI, method string, args ...any) []any {
	t.Helper()
	data, err := a.Pack(method, args...)
	require.NoError(t, err)
	var out hexutil.Bytes
	e.f.rpc(&out, "eth_call", map[string]any{"to": target, "data": hexutil.Encode(data)}, "latest")
	vals, err := a.Methods[method].Outputs.Unpack(out)
	require.NoError(t, err)
	return vals
}

// sendTx mines one call from an impersonated account two seconds after the head; false when it reverted (logged).
func (e *forkEnv) sendTx(t *testing.T, from, to common.Address, a *abi.ABI, method string, args ...any) bool {
	t.Helper()
	data, err := a.Pack(method, args...)
	require.NoError(t, err)
	e.f.impersonate(from)
	rcpt := e.f.send(from, to, data, e.f.head().Time+2)
	if rcpt.Status != 1 {
		var out hexutil.Bytes
		callErr := e.f.rc.CallContext(context.Background(), &out, "eth_call", map[string]any{"from": from, "to": to,
			"data": hexutil.Encode(data), "gas": hexutil.Uint64(8_000_000)}, "latest")
		rd, _ := revertData(callErr)
		t.Logf("tx %s from %s reverted: %v %x (%v)", method, from, callErr, rd, revertError(rd))
		return false
	}
	return true
}

// tryTrack refreshes at the head; a refresh that does not produce a quotable simulator is returned as an error.
func (e *forkEnv) tryTrack(t *testing.T, policy Policy) (*PoolSimulator, *Extra, error) {
	t.Helper()
	cfg := parityConfig(policy)
	tracked, err := NewPoolTracker(cfg, e.f.client).GetNewPoolState(context.Background(), e.listed,
		pool.GetNewPoolStateParams{})
	if err != nil {
		return nil, nil, fmt.Errorf("refresh: %w", err)
	}
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	sim, err := NewPoolSimulator(tracked)
	if err != nil {
		return nil, &extra, fmt.Errorf("simulator: %w (attest %q drift %q)", err, extra.AttestFailure,
			extra.ProfileDrift)
	}
	ts := sim.state.Timestamp
	sim.nowFn = func() uint64 { return ts }
	t.Logf("refresh at %d (ts %d): %d probes reproduced", tracked.BlockNumber, ts, extra.Probes)
	return sim, &extra, nil
}

func msgpackHop(t *testing.T, sim *PoolSimulator) *PoolSimulator {
	t.Helper()
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.IncludeUnexported(true)
	enc.SetForceAsArray(true)
	require.NoError(t, enc.Encode(sim))
	dec := msgpack.NewDecoder(&buf)
	dec.IncludeUnexported(true)
	dec.SetForceAsArray(true)
	var decoded PoolSimulator
	require.NoError(t, dec.Decode(&decoded))
	return &decoded
}

func sameResult(a, b *pool.CalcAmountOutResult, errA, errB error) bool {
	if (errA == nil) != (errB == nil) {
		return false
	}
	if errA != nil {
		return errA.Error() == errB.Error()
	}
	sa, sb := a.SwapInfo.(SwapInfo), b.SwapInfo.(SwapInfo)
	return a.TokenAmountOut.Amount.Cmp(b.TokenAmountOut.Amount) == 0 &&
		a.RemainingTokenAmountIn.Amount.Cmp(b.RemainingTokenAmountIn.Amount) == 0 && a.Gas == b.Gas &&
		sa.Venue == sb.Venue && sa.AmountInUsed.Eq(&sb.AmountInUsed) && sa.AmountOut.Eq(&sb.AmountOut) &&
		len(e2eDiff(sa.next, sb.next)) == 0
}

// scenarioSeq mines steps seeded fills starting at the simulator's snapshot. Each quote runs on the current simulator
// and on a clone; the clone adopts the fill, the original must still quote as before; every 7th step the simulator
// crosses a msgpack hop. The final state must equal a fresh refresh. Returns the gas maxima.
func (e *forkEnv) scenarioSeq(t *testing.T, rep *forkReport, area string, sim *PoolSimulator, steps int, seed int64,
	leverBias bool) map[string]uint64 {
	t.Helper()
	s := sim.state
	feedDeadline := min(s.Feed.Asset.Round.UpdatedAt.Uint64()+s.Feed.Asset.Heartbeat.Uint64(),
		s.Feed.Loans[0].Round.UpdatedAt.Uint64()+s.Feed.Loans[0].Heartbeat.Uint64())
	rng := rand.New(rand.NewSource(seed))
	ts := s.Timestamp
	gasMax := map[string]uint64{}
	adopted, refused := 0, 0
	for i := 0; i < steps; i++ {
		venue := rng.Intn(3) - 1
		if leverBias && rng.Intn(2) == 0 {
			venue = int(VenueLever)
		}
		sell := rng.Intn(2) == 0
		gap := uint64(2 + rng.Intn(15))
		if rng.Intn(12) == 0 {
			gap = uint64(100 + rng.Intn(300))
		}
		if ts+gap+uint64(steps-i)*3 >= feedDeadline {
			gap = 2
		}
		ts += gap
		cur := sim
		cur.nowFn = func() uint64 { return ts }
		v := venue
		if v < 0 {
			v = int(VenueSwap)
		}
		lo, hi := 1.0, 600_000.0
		if !sell {
			lo, hi = 500.0, 600_000_000.0
		}
		a := uint64(lo * math.Pow(hi/lo, rng.Float64()))
		if rng.Intn(3) == 0 {
			amts := edgeAmounts(cur, v, sell, 1)
			a = amts[rng.Intn(len(amts))]
		}
		params := amountIn(cur, sell, a)
		res, err := cur.calcAmountOut(params, venue)
		clone := cur.CloneState().(*PoolSimulator)
		clone.nowFn = func() uint64 { return ts }
		resC, errC := clone.calcAmountOut(params, venue)
		where := fmt.Sprintf("%s step %d venue=%d sell=%v amountIn=%d ts=%d", area, i, venue, sell, a, ts)
		if !sameResult(res, resC, err, errC) {
			rep.mismatch(area+"/clone-quote", where, describeQuote(res, err), "clone quoted "+describeQuote(resC, errC))
		}
		if err != nil && errors.Is(err, ErrSpreadNotLive) {
			if chainSpreadLive(cur, ts) {
				rep.mismatch(area+"/spread", where, describeQuote(res, err), "the pool's spread is live")
			}
			ts -= gap
			continue
		}
		fv := uint8(v)
		if err == nil {
			fv = res.SwapInfo.(SwapInfo).Venue
		}
		in, out := e.tokens(sell)
		fill := e.mineFill(t, fv, in, out, params.TokenAmountIn.Amount, ts)
		switch {
		case err == nil && fill.reverted:
			rep.mismatch(area+"/sequence", where, describeQuote(res, err), describeFill(fill))
			return gasMax
		case err == nil:
			if res.TokenAmountOut.Amount.Cmp(fill.out) != 0 || res.RemainingTokenAmountIn.Amount.Cmp(fill.unused) != 0 {
				rep.mismatch(area+"/sequence", where, describeQuote(res, err), describeFill(fill))
				return gasMax
			}
			key := fmt.Sprintf("venue%d/sell=%v", fv, sell)
			gasMax[key] = max(gasMax[key], fill.gas)
			if fill.gas > uint64(res.Gas) {
				rep.mismatch(area+"/gas", where, fmt.Sprintf("estimate %d", res.Gas), fmt.Sprintf("receipt gas %d", fill.gas))
			}
			clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
			// the original is untouched: it quotes the same amount exactly as before
			again, errAgain := cur.calcAmountOut(params, venue)
			if !sameResult(res, again, err, errAgain) {
				rep.mismatch(area+"/purity", where, describeQuote(res, err), "re-quote "+describeQuote(again, errAgain))
			}
			if bn := clone.GetMetaInfo("", "").(PoolMeta).BlockNumber; bn != s.Block {
				rep.mismatch(area+"/meta", where, fmt.Sprintf("meta block %d", bn), fmt.Sprintf("snapshot %d", s.Block))
			}
			sim = clone
			adopted++
			rep.count(area + "/sequence-identical")
		case !fill.reverted:
			rep.mismatch(area+"/sequence", where, describeQuote(res, err), describeFill(fill))
			return gasMax
		default:
			want := forkRevertError(fill.revert)
			if want == nil || !errors.Is(err, want) {
				rep.mismatch(area+"/sequence-refusal", where, describeQuote(res, err), describeFill(fill))
			}
			refused++
		}
		if i%7 == 6 {
			decoded := msgpackHop(t, sim)
			decoded.nowFn = func() uint64 { return ts }
			if d := e2eDiff(sim.state, decoded.state); len(d) != 0 || decoded.seq != sim.seq {
				rep.mismatch(area+"/msgpack", where, "state after msgpack", strings.Join(d, "; "))
			}
			sim = decoded
		}
	}
	fresh, _, err := e.tryTrack(t, sim.Policy)
	if err != nil {
		rep.mismatch(area+"/post-refresh", "after the sequence", "simulator state", err.Error())
		return gasMax
	}
	got := sim.state.clone()
	got.Block, got.Timestamp = fresh.state.Block, fresh.state.Timestamp
	if d := e2eDiff(got, fresh.state); len(d) != 0 {
		rep.mismatch(area+"/post-state", "after the sequence", "simulator state", strings.Join(d, "; "))
	}
	t.Logf("%s: sequence adopted %d, refusals mined %d, gas maxima %v", area, adopted, refused, gasMax)
	return gasMax
}

// trackDonationAware refreshes with routing on; a pool the production policy refuses for a donated venue position
// is counted and refreshed again with QuoteDonatedVenues, so the donated state's quotes are still compared.
func (e *forkEnv) trackDonationAware(t *testing.T, rep *forkReport, name string) (*PoolSimulator, *Extra, error) {
	t.Helper()
	sim, extra, err := e.tryTrack(t, Policy{LeverRouting: true})
	if !errors.Is(err, ErrPoolRefused) || extra == nil || extra.Reads == nil {
		return sim, extra, err
	}
	for i := range extra.Reads.Router.Venues {
		v := &extra.Reads.Router.Venues[i]
		if v.Position.Collateral.Gt(&v.ManagedCollateral) || v.Position.SupplyShares.Gt(&v.ManagedSupplyShares) {
			rep.count(name + "/donation-refused")
			return e.tryTrack(t, Policy{LeverRouting: true, QuoteDonatedVenues: true})
		}
	}
	return sim, extra, err
}

// chainFills mines a small grid on each venue at the head when the simulator refuses the whole pool: a pool the
// chain still fills is a coverage gap worth reporting.
func (e *forkEnv) chainFills(t *testing.T, ts uint64) map[string]int {
	t.Helper()
	fills := map[string]int{}
	for _, venue := range []uint8{VenueSwap, VenueLever} {
		for _, sell := range []bool{true, false} {
			amounts := liveGrid(10, 400_000, 8)
			if !sell {
				amounts = liveGrid(10_000, 400_000_000, 8)
			}
			calls := make([]adapterCall, len(amounts))
			for i, a := range amounts {
				calls[i] = adapterCall{venue: venue, sell: sell, amount: new(big.Int).SetUint64(a)}
			}
			for _, f := range e.batchFills(t, calls, ts) {
				if !f.reverted {
					fills[fmt.Sprintf("venue%d/sell=%v", venue, sell)]++
				}
			}
		}
	}
	return fills
}

type forkScenario struct {
	name  string
	apply func(t *testing.T, e *forkEnv, sim *PoolSimulator) bool
	// drift re-compares at these clock offsets (seconds after the refresh) without a new refresh
	drift []uint64
}

func TestForkScenarios(t *testing.T) {
	e := newForkEnvAt(t, forkScenariosBlock)
	rep := newForkReport(t)
	se := StaticExtra{}
	require.NoError(t, json.Unmarshal([]byte(e.listed.StaticExtra), &se))
	prof := &c104
	v0 := entityVenues(t, e.listed)[0]
	params := morphoMarketParams{se.LoanAsset, se.PoolAsset, v0.Oracle, v0.Irm, v0.Lltv.ToBig()}
	morpho := prof.Morpho
	core := e.f.view(e.pool, "core")[0].(common.Address)
	curator := e.f.view(core, "owner")[0].(common.Address)
	keeper := e.f.view(core, "keeper")[0].(common.Address)
	anyone := common.HexToAddress("0x00000000000000000000000000000000000beef2")
	whale := common.HexToAddress("0x00000000000000000000000000000000000beef3")
	e.f.impersonate(curator)
	e.f.impersonate(keeper)
	e.f.impersonate(anyone)
	e.f.impersonate(whale)

	e.armLeverage(t, 13_000, 0)
	// build some debt first so repay / withdraw / reclaim branches exist
	base, _, err := e.tryTrack(t, Policy{LeverRouting: true})
	require.NoError(t, err)
	ts := base.state.Timestamp
	for i, st := range []struct {
		venue uint8
		sell  bool
		a     uint64
	}{{1, true, 9_000}, {0, true, 40_000}, {1, true, 4_000}} {
		in, out := e.tokens(st.sell)
		fl := e.mineFill(t, st.venue, in, out, new(big.Int).SetUint64(st.a), ts+uint64(4*(i+1)))
		t.Logf("setup fill %d: %s gas %d", i, describeFill(fl), fl.gas)
	}
	base, _, err = e.tryTrack(t, Policy{LeverRouting: true})
	require.NoError(t, err)
	vp, err := base.state.Router.venuePosition(0, base.state.Timestamp)
	require.NoError(t, err)
	rateOk, rate, err := base.state.Router.Venues[0].Morpho.tryBorrowRate(uZero, uZero, base.state.Timestamp)
	require.NoError(t, err)
	loan := base.state.Pool.Loans[0]
	t.Logf("base: collateral %s recognized %s supplied %s recognized %s debt %s; rate ok %v %s; liquid %s band %s",
		vp[0].Dec(), vp[1].Dec(), vp[2].Dec(), vp[3].Dec(), vp[4].Dec(), rateOk, rate.Dec(), loan.Liquid.Dec(),
		loan.SwapPriceBandWad.Dec())
	limits := e.viewCall(t, e.pool, &flammABI, "limits")
	lim, err := tupleOf[abiLimits](limits[0])
	require.NoError(t, err)
	dialsV := e.viewCall(t, e.pool, &flammABI, "dials")
	dials, err := tupleOf[abiDials](dialsV[0])
	require.NoError(t, err)
	t.Logf("dials %+v limits depositCap %s feeFloor %d feeCap %d", dials, lim.DepositCapPoolAsset, lim.FeeFloorWad,
		lim.FeeCapWad)
	router := se.Router
	safe := e.viewCall(t, router, &scenariosABI, "protocolSafe")[0].(common.Address)
	e.f.impersonate(safe)

	morphoTx := func(t *testing.T, e *forkEnv, from common.Address, token common.Address, amount *big.Int,
		method string, args ...any) bool {
		if token != (common.Address{}) {
			cur := e.f.balance(token, from)
			e.f.setBalance(token, from, new(big.Int).Add(cur, amount))
			if !e.sendTx(t, from, token, &scenariosABI, "approve", morpho, amount) {
				return false
			}
		}
		return e.sendTx(t, from, morpho, &scenariosABI, method, args...)
	}
	u64 := func(x *uint256.Int) uint64 { return x.Uint64() }

	scenarios := []forkScenario{
		{name: "armed", apply: func(*testing.T, *forkEnv, *PoolSimulator) bool { return true }, drift: []uint64{900}},
		{name: "paused", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setPaused", true)
		}},
		{name: "global-paused", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, safe, router, &scenariosABI, "setGlobalPaused", true)
		}},
		{name: "features-no-sell", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setFeatures", big.NewInt(63&^2))
		}},
		{name: "features-no-buy", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setFeatures", big.NewInt(63&^4))
		}},
		{name: "features-no-lending", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setFeatures", big.NewInt(63&^8))
		}},
		{name: "features-no-leverage", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setFeatures", big.NewInt(63&^32))
		}},
		{name: "features-no-deposit-maint", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setFeatures", big.NewInt(63&^17))
		}},
		{name: "venue-borrow-off", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setVenueFlags", uint16(0), false, true)
		}},
		{name: "venue-supply-off", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setVenueFlags", uint16(0), true, false)
		}},
		{name: "venue-debtcap-tight", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			c := new(big.Int).Add(vp[4].ToBig(), big.NewInt(6_000_000))
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setVenueCaps", uint16(0), c, big.NewInt(0), uint64(0))
		}},
		{name: "venue-supplycap-tight", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			c := new(big.Int).Add(vp[2].ToBig(), big.NewInt(4_000_000))
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setVenueCaps", uint16(0), big.NewInt(0), c, uint64(0))
		}},
		{name: "venue-rate-ceiling", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setVenueCaps", uint16(0), big.NewInt(0), big.NewInt(0),
				max(u64(&rate)/2, 1))
		}, drift: []uint64{1200}},
		{name: "band-tight", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &forkABI, "setLoanConfig", uint8(0), u64(&loan.SwapPriceBandWad)/6,
				u64(&loan.FeeFloorWad), loan.MaxSwapNotional.ToBig(), loan.ReserveTarget.ToBig())
		}},
		{name: "loan-fee-floor-high", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &forkABI, "setLoanConfig", uint8(0), u64(&loan.SwapPriceBandWad),
				uint64(8e15), loan.MaxSwapNotional.ToBig(), loan.ReserveTarget.ToBig())
		}},
		{name: "reserve-target-big", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &forkABI, "setLoanConfig", uint8(0), u64(&loan.SwapPriceBandWad),
				u64(&loan.FeeFloorWad), loan.MaxSwapNotional.ToBig(), big.NewInt(5_000_000_000))
		}},
		{name: "reserve-target-zero", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &forkABI, "setLoanConfig", uint8(0), u64(&loan.SwapPriceBandWad),
				u64(&loan.FeeFloorWad), loan.MaxSwapNotional.ToBig(), big.NewInt(0))
		}},
		{name: "fee-bounds-narrow", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setFeeBounds", uint64(4e15), uint64(6e15))
		}},
		// No maxDrawnAssets scenario: MMRouter.setMaxDrawnAssets requires 1 <= n <= loans.length (MMRouter.sol:111),
		// so a one-loan pool keeps 1; the layout check binds that Router word (TestTrackerLayoutBindsRouterConfig).
		{name: "dials", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			phi := (dials.PhiMinWad + dials.PhiMaxWad) / 2
			ok := false
			for _, ltv := range []uint64{dials.LtvWad - min(dials.LtvMaxStepWad, dials.LtvWad-dials.LtvMinWad),
				dials.LtvWad + min(dials.LtvMaxStepWad, dials.LtvMaxWad-dials.LtvWad), dials.LtvWad} {
				if e.sendTx(t, keeper, e.pool, &scenariosABI, "setDials", phi, ltv) {
					ok = true
					break
				}
			}
			return ok
		}},
		{name: "maintain-collateral", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			// The setup fills left venue 0 posted at the ltv dial's posting law. Lowering the dial by its largest step
			// (0.55 -> 0.45 at the fork block) puts the venue's health below target = lltv / ltv * (1 - 10% slack)
			// (FLAMMOpsLib.sol:350-352), so anyone's maintainCollateral posts the shortfall.
			ltv := dials.LtvWad - min(dials.LtvMaxStepWad, dials.LtvWad-dials.LtvMinWad)
			return e.sendTx(t, keeper, e.pool, &scenariosABI, "setDials", dials.PhiWad, ltv) &&
				e.sendTx(t, anyone, e.pool, &scenariosABI, "maintainCollateral")
		}},
		{name: "poke", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, anyone, e.pool, &scenariosABI, "poke")
		}},
		{name: "recenter", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, keeper, e.pool, &scenariosABI, "recenter", big.NewInt(0),
				new(big.Int).Lsh(big.NewInt(1), 200), big.NewInt(1<<40))
		}},
		{name: "skim", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			return e.sendTx(t, anyone, e.pool, &scenariosABI, "skimPerformanceFee")
		}},
		{name: "donate-collateral-1btc", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			a := big.NewInt(100_000_000)
			return morphoTx(t, e, anyone, se.PoolAsset, a, "supplyCollateral", params, a, v0.Account, []byte{})
		}},
		{name: "donate-supply-5000usdc", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			a := big.NewInt(5_000_000_000)
			return morphoTx(t, e, anyone, se.LoanAsset, a, "supply", params, a, big.NewInt(0), v0.Account, []byte{})
		}},
		{name: "repay-on-behalf-half", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			if vp[4].IsZero() {
				t.Log("no debt to repay")
				return false
			}
			a := new(big.Int).Rsh(vp[4].ToBig(), 1)
			return morphoTx(t, e, anyone, se.LoanAsset, a, "repay", params, a, big.NewInt(0), v0.Account, []byte{})
		}},
		{name: "token-transfers", apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
			for _, who := range []common.Address{e.pool, router, v0.Account, prof.Hook, prof.LeverageHook,
				prof.SpreadHook, prof.PriceFeed} {
				for _, tok := range []common.Address{se.PoolAsset, se.LoanAsset} {
					cur := e.f.balance(tok, who)
					e.f.setBalance(tok, who, new(big.Int).Add(cur, big.NewInt(123_456_789)))
				}
			}
			// mine a block so the refresh reads at a new head
			return e.sendTx(t, anyone, anyone, &scenariosABI, "protocolSafe")
		}},
	}
	for _, leftover := range []int64{0, 2_500_000, 300_000_000} {
		leftover := leftover
		scenarios = append(scenarios, forkScenario{name: fmt.Sprintf("liquidity-squeeze-%d", leftover),
			apply: func(t *testing.T, e *forkEnv, _ *PoolSimulator) bool {
				coll := big.NewInt(3_000_000_000_000) // 30,000 cbBTC
				if !morphoTx(t, e, whale, se.PoolAsset, coll, "supplyCollateral", params, coll, whale, []byte{}) {
					return false
				}
				// accrue first so the liquidity read is the one borrow sees
				one := big.NewInt(1)
				if !morphoTx(t, e, whale, se.LoanAsset, one, "supply", params, one, big.NewInt(0), whale, []byte{}) {
					return false
				}
				m := e.viewCall(t, morpho, &morphoABI, "market", v0.MarketID)
				liq := new(big.Int).Sub(m[0].(*big.Int), m[2].(*big.Int))
				borrow := new(big.Int).Sub(liq, big.NewInt(leftover))
				t.Logf("market liquidity %s, whale borrows %s", liq, borrow)
				return borrow.Sign() > 0 && e.sendTx(t, whale, morpho, &scenariosABI, "borrow", params, borrow, big.NewInt(0),
					whale, whale)
			}, drift: []uint64{1500}})
	}

	var snap hexutil.Big
	e.f.rpc(&snap, "evm_snapshot")
	only := os.Getenv("EVERLONG_FLAMM_FORK_SCENARIOS")
	seqSteps := 36
	for si, sc := range scenarios {
		if only != "" && !strings.Contains(","+only+",", ","+sc.name+",") {
			continue
		}
		var ok bool
		e.f.rpc(&ok, "evm_revert", snap)
		require.True(t, ok)
		e.f.rpc(&snap, "evm_snapshot")
		if !sc.apply(t, e, base) {
			rep.mismatch(sc.name+"/not-applied", "scenario setup", "could not be applied", "every scenario applies")
			continue
		}
		sim, extra, err := e.trackDonationAware(t, rep, sc.name)
		if err != nil {
			fills := e.chainFills(t, e.f.head().Time)
			t.Logf("scenario %s: simulator refuses the pool: %v; chain fills on a small grid: %v", sc.name, err, fills)
			if len(fills) > 0 {
				rep.mismatch(sc.name+"/pool-refused", "refresh at head", err.Error(), fmt.Sprintf("chain fills %v", fills))
			} else {
				rep.count(sc.name + "/pool-refused-chain-refuses")
			}
			_ = extra
			continue
		}
		e.edgeSuite(t, rep, sc.name, sim, true, 6)
		for _, d := range sc.drift {
			at := sim.state.Timestamp + d
			for _, venue := range []int{int(VenueSwap), int(VenueLever), -1} {
				for _, sell := range []bool{true, false} {
					amounts := liveGrid(1, 600_000, 24)
					if !sell {
						amounts = liveGrid(500, 600_000_000, 24)
					}
					e.compareFills(t, rep, fmt.Sprintf("%s/drift+%d/venue%d/sell=%v", sc.name, d, venue, sell), sim, venue,
						sell, amounts, at)
				}
			}
		}
		e.scenarioSeq(t, rep, sc.name, sim, seqSteps, int64(0xF1A2)+int64(si), si%2 == 0)
	}
	require.Empty(t, rep.list)
}

// TestForkGasHunt looks for receipt gas above the simulator's per-venue estimate: the book is moved onto the
// leverage curve's Hermite piece, then under each Router / pool configuration every venue and direction is mined
// from an anvil snapshot over a size grid plus the port's edges, and pairs of a lever fill followed by a dust
// opposite lever fill are mined too.
func TestForkGasHunt(t *testing.T) {
	e := newForkEnvAt(t, forkScenariosBlock)
	rep := newForkReport(t)
	se := StaticExtra{}
	require.NoError(t, json.Unmarshal([]byte(e.listed.StaticExtra), &se))
	core := e.f.view(e.pool, "core")[0].(common.Address)
	curator := e.f.view(core, "owner")[0].(common.Address)
	e.armLeverage(t, 13_000, 0)
	sim, _, err := e.tryTrack(t, Policy{LeverRouting: true})
	require.NoError(t, err)
	ts := sim.state.Timestamp
	for i, st := range []struct {
		venue uint8
		sell  bool
		a     uint64
	}{{1, true, 9_000}, {0, true, 60_000}, {1, true, 5_000}, {0, false, 30_000_000}} {
		in, out := e.tokens(st.sell)
		fl := e.mineFill(t, st.venue, in, out, new(big.Int).SetUint64(st.a), ts+uint64(4*(i+1)))
		t.Logf("book move %d: %s gas %d", i, describeFill(fl), fl.gas)
	}
	base, _, err := e.tryTrack(t, Policy{LeverRouting: true})
	require.NoError(t, err)
	loan := base.state.Pool.Loans[0]
	vp, err := base.state.Router.venuePosition(0, base.state.Timestamp)
	require.NoError(t, err)
	maxima, worst := map[string]uint64{}, map[string]string{}
	record := func(where string, venue uint8, sell bool, gas uint64, estimate int64) {
		key := fmt.Sprintf("venue%d/sell=%v", venue, sell)
		if gas > maxima[key] {
			maxima[key], worst[key] = gas, where
		}
		if gas > uint64(estimate) {
			rep.mismatch("gas", where, fmt.Sprintf("estimate %d", estimate), fmt.Sprintf("receipt gas %d", gas))
		}
	}
	type change struct {
		name  string
		apply func() bool
	}
	changes := []change{
		{"none", func() bool { return true }},
		{"borrow-off", func() bool {
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setVenueFlags", uint16(0), false, true)
		}},
		{"supply-off", func() bool {
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setVenueFlags", uint16(0), true, false)
		}},
		{"reserve-big", func() bool {
			return e.sendTx(t, curator, e.pool, &forkABI, "setLoanConfig", uint8(0), loan.SwapPriceBandWad.Uint64(),
				loan.FeeFloorWad.Uint64(), loan.MaxSwapNotional.ToBig(), big.NewInt(5_000_000_000))
		}},
		{"debtcap-tight", func() bool {
			c := new(big.Int).Add(vp[4].ToBig(), big.NewInt(6_000_000))
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setVenueCaps", uint16(0), c, big.NewInt(0), uint64(0))
		}},
		{"cap-8usdc", func() bool {
			return e.sendTx(t, curator, e.pool, &forkABI, "setLoanConfig", uint8(0), loan.SwapPriceBandWad.Uint64(),
				loan.FeeFloorWad.Uint64(), big.NewInt(8_000_000), loan.ReserveTarget.ToBig())
		}},
	}
	var snap hexutil.Big
	e.f.rpc(&snap, "evm_snapshot")
	for _, ch := range changes {
		var ok bool
		e.f.rpc(&ok, "evm_revert", snap)
		require.True(t, ok)
		e.f.rpc(&snap, "evm_snapshot")
		if !ch.apply() {
			continue
		}
		sim, _, err := e.tryTrack(t, Policy{LeverRouting: true})
		if err != nil {
			t.Logf("%s: %v", ch.name, err)
			continue
		}
		now := sim.state.Timestamp + 2
		sim.nowFn = func() uint64 { return now }
		mine := func(where string, venue uint8, sell bool, a uint64) {
			res, err := sim.calcAmountOut(amountIn(sim, sell, a), int(venue))
			if err != nil {
				return
			}
			var s hexutil.Big
			e.f.rpc(&s, "evm_snapshot")
			in, out := e.tokens(sell)
			fl := e.mineFill(t, venue, in, out, new(big.Int).SetUint64(a), now)
			if fl.reverted || fl.out.Cmp(res.TokenAmountOut.Amount) != 0 || fl.unused.Cmp(res.RemainingTokenAmountIn.Amount) != 0 {
				rep.mismatch("gas/fill", where, describeQuote(res, err), describeFill(fl))
			} else {
				record(where, venue, sell, fl.gas, res.Gas)
				// a dust fill of the opposite lever direction right after, quoted on the adopted state
				if venue == VenueLever {
					c := sim.CloneState().(*PoolSimulator)
					c.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
					then := now + 2
					c.nowFn = func() uint64 { return then }
					for _, d := range []uint64{1, 3, 50, 1441, 40_000} {
						da := d
						if sell && d < 500 {
							da = d * 1000
						}
						r2, err2 := c.calcAmountOut(amountIn(c, !sell, da), int(VenueLever))
						if err2 != nil {
							continue
						}
						var s2 hexutil.Big
						e.f.rpc(&s2, "evm_snapshot")
						in2, out2 := e.tokens(!sell)
						f2 := e.mineFill(t, VenueLever, in2, out2, new(big.Int).SetUint64(da), then)
						w2 := fmt.Sprintf("%s then lever sell=%v %d", where, !sell, da)
						if f2.reverted || f2.out.Cmp(r2.TokenAmountOut.Amount) != 0 {
							rep.mismatch("gas/fill", w2, describeQuote(r2, err2), describeFill(f2))
						} else {
							record(w2, VenueLever, !sell, f2.gas, r2.Gas)
						}
						var ok2 bool
						e.f.rpc(&ok2, "evm_revert", s2)
						require.True(t, ok2)
						break
					}
				}
			}
			var ok bool
			e.f.rpc(&ok, "evm_revert", s)
			require.True(t, ok)
		}
		for _, venue := range []uint8{VenueSwap, VenueLever} {
			for _, sell := range []bool{true, false} {
				amounts := liveGrid(1, 3_000_000, 16)
				if !sell {
					amounts = liveGrid(100, 3_000_000_000, 16)
				}
				edges := edgeAmounts(sim, int(venue), sell, 0)
				for i := 0; i < len(edges); i += max(1, len(edges)/12) {
					amounts = append(amounts, edges[i])
				}
				for _, a := range amounts {
					mine(fmt.Sprintf("%s venue=%d sell=%v amountIn=%d", ch.name, venue, sell, a), venue, sell, a)
				}
			}
		}
		t.Logf("%s: maxima so far %v", ch.name, maxima)
	}
	t.Logf("receipt gas maxima %v at %v", maxima, worst)
	require.Empty(t, rep.list)
}

// TestForkGasLarge mines fills far above the usual grids -- a sell clipped by a notional cap, a debt cap or
// Morpho liquidity, and lever fills whose input the curve clips -- where the pool bisects over the input, so gas
// can grow with the input's magnitude.
func TestForkGasLarge(t *testing.T) {
	e := newForkEnvAt(t, forkScenariosBlock)
	rep := newForkReport(t)
	se := StaticExtra{}
	require.NoError(t, json.Unmarshal([]byte(e.listed.StaticExtra), &se))
	core := e.f.view(e.pool, "core")[0].(common.Address)
	curator := e.f.view(core, "owner")[0].(common.Address)
	whale := common.HexToAddress("0x00000000000000000000000000000000000beef3")
	v0 := entityVenues(t, e.listed)[0]
	params := morphoMarketParams{se.LoanAsset, se.PoolAsset, v0.Oracle, v0.Irm, v0.Lltv.ToBig()}
	morpho := c104.Morpho
	e.armLeverage(t, 13_000, 0)
	sim, _, err := e.tryTrack(t, Policy{LeverRouting: true})
	require.NoError(t, err)
	ts := sim.state.Timestamp
	for i, st := range []struct {
		venue uint8
		sell  bool
		a     uint64
	}{{1, true, 9_000}, {0, true, 60_000}, {1, true, 5_000}, {0, false, 30_000_000}} {
		in, out := e.tokens(st.sell)
		fl := e.mineFill(t, st.venue, in, out, new(big.Int).SetUint64(st.a), ts+uint64(4*(i+1)))
		t.Logf("book move %d: %s gas %d", i, describeFill(fl), fl.gas)
	}
	base, _, err := e.tryTrack(t, Policy{LeverRouting: true})
	require.NoError(t, err)
	loan := base.state.Pool.Loans[0]
	vp, err := base.state.Router.venuePosition(0, base.state.Timestamp)
	require.NoError(t, err)
	changes := []struct {
		name  string
		apply func() bool
	}{
		{"none", func() bool { return true }},
		{"cap-8usdc", func() bool {
			return e.sendTx(t, curator, e.pool, &forkABI, "setLoanConfig", uint8(0), loan.SwapPriceBandWad.Uint64(),
				loan.FeeFloorWad.Uint64(), big.NewInt(8_000_000), loan.ReserveTarget.ToBig())
		}},
		{"cap-60usdc", func() bool {
			return e.sendTx(t, curator, e.pool, &forkABI, "setLoanConfig", uint8(0), loan.SwapPriceBandWad.Uint64(),
				loan.FeeFloorWad.Uint64(), big.NewInt(60_000_000), loan.ReserveTarget.ToBig())
		}},
		{"debtcap-tight", func() bool {
			c := new(big.Int).Add(vp[4].ToBig(), big.NewInt(6_000_000))
			return e.sendTx(t, curator, e.pool, &scenariosABI, "setVenueCaps", uint16(0), c, big.NewInt(0), uint64(0))
		}},
		{"squeeze-2.5usdc", func() bool {
			coll := big.NewInt(3_000_000_000_000)
			cur := e.f.balance(se.PoolAsset, whale)
			e.f.setBalance(se.PoolAsset, whale, new(big.Int).Add(cur, coll))
			if !e.sendTx(t, whale, se.PoolAsset, &scenariosABI, "approve", morpho, coll) ||
				!e.sendTx(t, whale, morpho, &scenariosABI, "supplyCollateral", params, coll, whale, []byte{}) {
				return false
			}
			m := e.viewCall(t, morpho, &morphoABI, "market", v0.MarketID)
			liq := new(big.Int).Sub(m[0].(*big.Int), m[2].(*big.Int))
			return e.sendTx(t, whale, morpho, &scenariosABI, "borrow", params, new(big.Int).Sub(liq, big.NewInt(2_500_000)),
				big.NewInt(0), whale, whale)
		}},
	}
	maxima, worst := map[string]uint64{}, map[string]string{}
	var snap hexutil.Big
	e.f.rpc(&snap, "evm_snapshot")
	for _, ch := range changes {
		var ok bool
		e.f.rpc(&ok, "evm_revert", snap)
		require.True(t, ok)
		e.f.rpc(&snap, "evm_snapshot")
		if !ch.apply() {
			continue
		}
		sim, _, err := e.tryTrack(t, Policy{LeverRouting: true})
		if err != nil {
			t.Logf("%s: %v", ch.name, err)
			continue
		}
		now := sim.state.Timestamp + 2
		sim.nowFn = func() uint64 { return now }
		for _, venue := range []uint8{VenueSwap, VenueLever} {
			for _, sell := range []bool{true, false} {
				var amounts []*big.Int
				for exp := 5; exp <= 30; exp++ {
					amounts = append(amounts, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil))
					amounts = append(amounts, new(big.Int).Mul(big.NewInt(3), new(big.Int).Exp(big.NewInt(10),
						big.NewInt(int64(exp)), nil)))
				}
				amounts = append(amounts, new(big.Int).Lsh(big.NewInt(1), 128), new(big.Int).Lsh(big.NewInt(1), 200))
				for _, a := range amounts {
					params := amountIn(sim, sell, 1)
					params.TokenAmountIn.Amount = a
					res, err := sim.calcAmountOut(params, int(venue))
					where := fmt.Sprintf("%s venue=%d sell=%v amountIn=%s", ch.name, venue, sell, a)
					in, out := e.tokens(sell)
					fl := e.batchFills(t, []adapterCall{{venue: venue, sell: sell, amount: a}}, now)[0]
					judgeFill(rep, "large/"+ch.name, adapterCall{venue: venue, sell: sell, amount: a}, res, err, fl,
						chainSpreadLive(sim, now))
					if err != nil || fl.reverted {
						continue
					}
					var s hexutil.Big
					e.f.rpc(&s, "evm_snapshot")
					mined := e.mineFill(t, venue, in, out, a, now)
					key := fmt.Sprintf("venue%d/sell=%v", venue, sell)
					if !mined.reverted {
						if mined.gas > maxima[key] {
							maxima[key], worst[key] = mined.gas, where
						}
						if mined.gas > uint64(res.Gas) {
							rep.mismatch("gas", where, fmt.Sprintf("estimate %d (out %s unused %s)", res.Gas,
								res.TokenAmountOut.Amount, res.RemainingTokenAmountIn.Amount),
								fmt.Sprintf("receipt gas %d", mined.gas))
						}
						t.Logf("%s: out %s unused %s gas %d (estimate %d)", where, mined.out, mined.unused, mined.gas, res.Gas)
					} else {
						rep.mismatch("gas/mined", where, describeQuote(res, err), describeFill(mined))
					}
					var ok bool
					e.f.rpc(&ok, "evm_revert", s)
					require.True(t, ok)
				}
			}
		}
	}
	t.Logf("receipt gas maxima %v at %v", maxima, worst)
	require.Empty(t, rep.list)
}
