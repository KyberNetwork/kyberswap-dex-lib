package everlongflamm

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestForkGas measures the receipt gas behind the defaults in constant.go on an anvil fork (same gating as
// TestForkParity; EVERLONG_FLAMM_FORK_BLOCK picks the block). Gas is state-dependent in two independent ways:
//
//   - the leverage hook's anchor solves: a frame on the curve's half-law piece is closed-form, one on its Hermite
//     piece bisects ~60 steps of a quartic Bezier per solve (CollRebalancerMath._strictAnchor :190,
//     _cvRequiredOnAnchor :333), and a lever fill runs up to five solves in each of the plan and the execution;
//   - the Router legs a settlement takes through Morpho (liquid only, withdraw supply, post collateral and borrow;
//     repay, reclaim, supply).
//
// So the leverage venue is armed (spread 13000 ppm, no staleness window) and a seeded walk of mined fills on both
// venues moves the book from the deployed half-law frame onto the Hermite piece and back, every mined fill being a
// measurement; every third step, dust, mid-size and edge fills of each venue and direction are also mined from an
// anvil snapshot and reverted. Every receipt must be within the simulator's estimate.
func TestForkGas(t *testing.T) {
	e := newForkEnv(t)
	f := e.f
	core := f.view(e.pool, "core")[0].(common.Address)
	curator := f.view(core, "owner")[0].(common.Address)
	keeper := f.view(core, "keeper")[0].(common.Address)
	f.impersonate(curator)
	f.impersonate(keeper)
	ts := f.head().Time
	for _, c := range []struct {
		from, to common.Address
		method   string
		arg      any
	}{
		{curator, e.pool, "setLevPaused", false}, {curator, c104.SpreadHook, "setMaxSpreadAge", uint32(0)},
		{keeper, c104.SpreadHook, "setSpread", big.NewInt(13_000)},
	} {
		data, err := forkABI.Pack(c.method, c.arg)
		require.NoError(t, err)
		ts += 2
		f.send(c.from, c.to, data, ts)
	}

	maxima, worst := map[string]uint64{}, map[string]string{}
	var above []string
	record := func(where string, venue uint8, sell bool, gas uint64, estimate int64) {
		key := fmt.Sprintf("venue%d/%s", venue, map[bool]string{true: "sell", false: "buy"}[sell])
		if gas > maxima[key] {
			maxima[key], worst[key] = gas, where
		}
		if gas > uint64(estimate) {
			above = append(above, fmt.Sprintf("%s: receipt gas %d, estimate %d", where, gas, estimate))
		}
		t.Logf("%s: receipt gas %d (estimate %d)", where, gas, estimate)
	}
	// fill quotes amount on venue at now and mines it, from a snapshot when probe is set.
	fill := func(sim *PoolSimulator, phase string, venue uint8, sell bool, a uint64, now uint64, probe bool) bool {
		res, err := sim.calcAmountOut(amountIn(sim, sell, a), int(venue))
		if err != nil {
			return false
		}
		var snap hexutil.Big
		if probe {
			f.rpc(&snap, "evm_snapshot")
		}
		in, out := e.tokens(sell)
		fl := f.execFill(e.pool, venue, in, out, new(big.Int).SetUint64(a), now)
		if probe {
			var reverted bool
			f.rpc(&reverted, "evm_revert", snap)
			require.True(t, reverted)
		}
		next := sim.CloneState().(*PoolSimulator)
		next.state = res.SwapInfo.(SwapInfo).next.clone()
		where := fmt.Sprintf("%s (frame %s -> %s) venue %d sell=%v amountIn %d", phase, gasFrameBranch(t, sim),
			gasFrameBranch(t, next), venue, sell, a)
		require.Equal(t, res.TokenAmountOut.Amount.String(), fl.out.String(), "%s amountOut", where)
		require.Equal(t, res.RemainingTokenAmountIn.Amount.String(), fl.unused.String(), "%s amountUnused", where)
		record(where, venue, sell, fl.gas, res.Gas)
		return true
	}

	// fillable scans a log grid offline: every amount the simulator fills on venue, with the frame after the fill.
	type candidate struct {
		a    uint64
		post string
	}
	fillable := func(sim *PoolSimulator, venue uint8, sell bool) []candidate {
		grid := liveGrid(1, 3_000_000, 48)
		if !sell {
			grid = liveGrid(100, 3_000_000_000, 48)
		}
		var out []candidate
		for _, a := range grid {
			res, err := sim.calcAmountOut(amountIn(sim, sell, a), int(venue))
			if err != nil {
				continue
			}
			next := sim.CloneState().(*PoolSimulator)
			next.state = res.SwapInfo.(SwapInfo).next.clone()
			out = append(out, candidate{a, gasFrameBranch(t, next)})
		}
		return out
	}

	rng := rand.New(rand.NewSource(0x6A5))
	const steps = 40
	for i := 0; i < steps; i++ {
		sim := e.track(t, Policy{LeverRouting: true})
		now := sim.state.Timestamp + 2
		sim.nowFn = func() uint64 { return now }
		pre := gasFrameBranch(t, sim)
		cands := map[uint8]map[bool][]candidate{}
		for _, venue := range []uint8{VenueSwap, VenueLever} {
			cands[venue] = map[bool][]candidate{}
			for _, sell := range []bool{true, false} {
				c := fillable(sim, venue, sell)
				cands[venue][sell] = c
				if i%2 != 0 || len(c) == 0 {
					continue
				}
				// Probes: the smallest and largest fill, and for the leverage venue the smallest and largest whose
				// frame is on the Hermite piece before and after (every anchor solve of the fill bisects).
				pick := map[uint64]bool{c[0].a: true, c[len(c)-1].a: true}
				if venue == VenueLever && pre == "hermite" {
					var both []uint64
					for _, x := range c {
						if x.post == "hermite" {
							both = append(both, x.a)
						}
					}
					if len(both) != 0 {
						pick[both[0]], pick[both[len(both)-1]] = true, true
					}
				}
				for a := range pick {
					fill(sim, fmt.Sprintf("step %d probe", i), venue, sell, a, now, true)
				}
			}
		}
		// The walk keeps the frame moving across the pieces: from a half-law frame the largest lever-up; otherwise a
		// lever-up, the smallest lever-down, a sell or a buy, sized from the fillable grid.
		venue, sell, largest := VenueLever, true, true
		if pre == "hermite" || pre == "recovery" || len(cands[VenueLever][true]) == 0 {
			switch rng.Intn(5) {
			case 0, 1:
			case 2:
				sell, largest = false, false
			default:
				venue, sell = VenueSwap, rng.Intn(2) == 0
				largest = rng.Intn(2) == 0
			}
		}
		c := cands[venue][sell]
		if len(c) == 0 {
			continue
		}
		a := c[0].a
		if largest {
			a = c[len(c)-1].a
		} else if venue == VenueSwap {
			a = c[rng.Intn(len(c))].a
		}
		fill(sim, fmt.Sprintf("step %d walk", i), venue, sell, a, now, false)
	}

	keys := make([]string, 0, len(maxima))
	for k := range maxima {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("receipt gas maximum %s: %d at %s", k, maxima[k], worst[k])
	}
	require.Len(t, maxima, 4)
	require.Empty(t, above)
}

// gasFrameBranch names the piece of the leverage curve the pool's frame sits on at the simulator's clock: flat (no
// net debt, no anchor solve), half-law (closed form), hermite (bisection) or recovery (beyond the wall).
func gasFrameBranch(t *testing.T, sim *PoolSimulator) string {
	t.Helper()
	s := sim.state.clone()
	pool := s.Pool
	_, ctx, err := s.leverFrame(&pool, &leverPlan{}, sim.now())
	if err != nil {
		return "unpriced: " + err.Error()
	}
	hb, err := s.Hooks.Swap.EverlongSwap.bookFor(&ctx)
	require.NoError(t, err)
	rp := s.Hooks.Swap.EverlongSwap.ReservationPriceWad
	fr, err := levFrameFor(&ctx, &levBook{Rs: &hb.Rs, Is: &hb.Is, Rv: &hb.Rv, Iv: &hb.Iv, ReservationPriceWad: &rp})
	require.NoError(t, err)
	if fr.D.Sign() <= 0 {
		return "flat"
	}
	ok, _, _, halfLaw := levStrictAnchor(fr.Cv, (*uint256.Int)(fr.D), levLeverageRatioWad)
	switch {
	case !ok:
		return "recovery"
	case halfLaw:
		return "half-law"
	}
	return "hermite"
}

// gasClipABI: the calls TestForkGasSwapSellClip makes beyond forkABI (pool and Router admin, ERC-20, Morpho Blue).
var gasClipABI = func() abi.ABI {
	const mp = `{"name":"marketParams","type":"tuple","components":[{"name":"loanToken","type":"address"},{"name":"collateralToken","type":"address"},{"name":"oracle","type":"address"},{"name":"irm","type":"address"},{"name":"lltv","type":"uint256"}]}`
	a, err := abi.JSON(strings.NewReader(`[
 {"type":"function","name":"setVenueCaps","stateMutability":"nonpayable","inputs":[{"name":"id","type":"uint16"},{"name":"d","type":"uint128"},{"name":"s","type":"uint128"},{"name":"r","type":"uint64"}],"outputs":[]},
 {"type":"function","name":"setPin","stateMutability":"nonpayable","inputs":[{"name":"p","type":"uint64"}],"outputs":[]},
 {"type":"function","name":"approve","stateMutability":"nonpayable","inputs":[{"name":"s","type":"address"},{"name":"a","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]},
 {"type":"function","name":"supplyCollateral","stateMutability":"nonpayable","inputs":[` + mp + `,{"name":"assets","type":"uint256"},{"name":"onBehalf","type":"address"},{"name":"data","type":"bytes"}],"outputs":[]},
 {"type":"function","name":"borrow","stateMutability":"nonpayable","inputs":[` + mp + `,{"name":"assets","type":"uint256"},{"name":"shares","type":"uint256"},{"name":"onBehalf","type":"address"},{"name":"receiver","type":"address"}],"outputs":[{"name":"","type":"uint256"},{"name":"","type":"uint256"}]}
]`))
	if err != nil {
		panic(err)
	}
	return a
}()

// gasClipBlock is TestForkGasSwapSellClip's default fork block, right after a cbBTC round.
const gasClipBlock = "51333700"

// TestForkGasSwapSellClip measures the clipped swap sell behind defaultGasSwapSellCapEval and defaultGasSwapSellPass
// (same gating as TestForkParity; EVERLONG_FLAMM_FORK_BLOCK defaults to 51333700). A sell the pool clips runs
// EverlongHook._maxInForGrossCap, a bisection over [0, amountIn], in every previewExactIn of its plan and again in
// executeExactIn, and a sell whose funding ceiling binds re-plans up to four passes, so receipt gas grows with the
// input. Leverage is armed and the book moved off the deployed frame, then each clip is set up from the same anvil
// snapshot -- a notional cap, a venue debt cap, Morpho liquidity borrowed away by a third party, and (the pool
// holding loan asset after a buy that clears its debt) the Router's pin lowered below the gate's ltv, so the
// funding ceiling binds and depends on the collateral the sell posts -- and sells from 2^9 to 2^72 base units are
// mined from a snapshot. Every clipped fill must reproduce the chain's amounts and stay within its estimate; the
// least-squares fit of receipt gas on (solves, extra passes) is logged.
func TestForkGasSwapSellClip(t *testing.T) {
	e := newForkEnvAt(t, gasClipBlock)
	f := e.f
	se := StaticExtra{}
	require.NoError(t, json.Unmarshal([]byte(e.listed.StaticExtra), &se))
	core := f.view(e.pool, "core")[0].(common.Address)
	curator := f.view(core, "owner")[0].(common.Address)
	keeper := f.view(core, "keeper")[0].(common.Address)
	whale := common.HexToAddress("0x00000000000000000000000000000000000beef3")
	for _, who := range []common.Address{curator, keeper, whale, e.pool} {
		f.impersonate(who)
	}
	tx := func(from, to common.Address, a *abi.ABI, method string, args ...any) {
		t.Helper()
		data, err := a.Pack(method, args...)
		require.NoError(t, err)
		rcpt := f.send(from, to, data, f.head().Time+2)
		require.Equal(t, types.ReceiptStatusSuccessful, rcpt.Status, "%s", method)
	}
	tx(curator, e.pool, &forkABI, "setLevPaused", false)
	tx(curator, c104.SpreadHook, &forkABI, "setMaxSpreadAge", uint32(0))
	tx(keeper, c104.SpreadHook, &forkABI, "setSpread", big.NewInt(13_000))
	ts := f.head().Time
	for i, mv := range []struct {
		venue uint8
		sell  bool
		a     int64
	}{{VenueLever, true, 9_000}, {VenueSwap, true, 60_000}, {VenueLever, true, 5_000}, {VenueSwap, false, 30_000_000}} {
		in, out := e.tokens(mv.sell)
		fl := f.execFill(e.pool, mv.venue, in, out, big.NewInt(mv.a), ts+uint64(4*(i+1)))
		t.Logf("book move %d: venue %d sell=%v amountIn %d: out %s gas %d", i, mv.venue, mv.sell, mv.a, fl.out, fl.gas)
	}
	moved := e.track(t, Policy{})
	loan := moved.state.Pool.Loans[0]
	setLoan := func(cap, reserve *uint256.Int) {
		tx(curator, e.pool, &forkABI, "setLoanConfig", uint8(0), loan.SwapPriceBandWad.Uint64(), loan.FeeFloorWad.Uint64(),
			cap.ToBig(), reserve.ToBig())
	}
	v0 := entityVenues(t, e.listed)[0]
	vp, err := moved.state.Router.venuePosition(0, moved.state.Timestamp)
	require.NoError(t, err)
	scenarios := []struct {
		name  string
		setup func()
	}{
		{"notional cap 8 USDC", func() { setLoan(uint256.NewInt(8_000_000), &loan.ReserveTarget) }},
		{"venue debt cap +6 USDC", func() {
			tx(curator, e.pool, &gasClipABI, "setVenueCaps", uint16(0),
				new(big.Int).Add(vp[4].ToBig(), big.NewInt(6_000_000)), big.NewInt(0), uint64(0))
		}},
		{"Morpho liquidity 2.5 USDC", func() {
			coll := big.NewInt(3_000_000_000_000)
			f.setBalance(se.PoolAsset, whale, coll)
			params := struct {
				LoanToken, CollateralToken, Oracle, Irm common.Address
				Lltv                                    *big.Int
			}{se.LoanAsset, se.PoolAsset, v0.Oracle, v0.Irm, v0.Lltv.ToBig()}
			morpho := c104.Morpho
			tx(whale, se.PoolAsset, &gasClipABI, "approve", morpho, coll)
			tx(whale, morpho, &gasClipABI, "supplyCollateral", params, coll, whale, []byte{})
			data, err := morphoABI.Pack("market", v0.MarketID)
			require.NoError(t, err)
			var out hexutil.Bytes
			f.rpc(&out, "eth_call", map[string]any{"to": morpho, "data": hexutil.Encode(data)}, "latest")
			m, err := morphoABI.Methods["market"].Outputs.Unpack(out)
			require.NoError(t, err)
			liquidity := new(big.Int).Sub(m[0].(*big.Int), m[2].(*big.Int))
			tx(whale, morpho, &gasClipABI, "borrow", params, liquidity.Sub(liquidity, big.NewInt(2_500_000)),
				big.NewInt(0), whale, whale)
		}},
	}
	for _, pin := range []uint64{3e16, 1e16, 1e15, 1e14} {
		scenarios = append(scenarios, struct {
			name  string
			setup func()
		}{fmt.Sprintf("funding ceiling, pin %d", pin), func() {
			setLoan(&loan.MaxSwapNotional, uint256.NewInt(150_000_000))
			debt := e.track(t, Policy{})
			pos, err := debt.state.Router.venuePosition(0, debt.state.Timestamp)
			require.NoError(t, err)
			buy := new(big.Int).Add(pos[4].ToBig(), big.NewInt(60_000_000))
			fl := f.execFill(e.pool, VenueSwap, e.usdc, e.cbBTC, buy, f.head().Time+2)
			t.Logf("debt-clearing buy %s: out %s", buy, fl.out)
			tx(e.pool, se.Router, &gasClipABI, "setPin", pin)
		}})
	}

	type point struct {
		scenario            string
		gas, solves, passes float64
		estimate            float64
	}
	var points []point
	var above []string
	var base hexutil.Big
	f.rpc(&base, "evm_snapshot")
	for _, sc := range scenarios {
		var ok bool
		f.rpc(&ok, "evm_revert", base)
		require.True(t, ok)
		f.rpc(&base, "evm_snapshot")
		sc.setup()
		sim := e.track(t, Policy{})
		now := sim.state.Timestamp + 2
		sim.nowFn = func() uint64 { return now }
		for bits := 9; bits <= 72; bits++ {
			for _, m := range []int64{1, 3, 5, 7} {
				if m != 1 && bits%3 != 0 {
					continue
				}
				a := new(big.Int).Lsh(big.NewInt(m), uint(bits))
				params := amountIn(sim, true, 1)
				params.TokenAmountIn.Amount = a
				res, err := sim.calcAmountOut(params, int(VenueSwap))
				if err != nil || res.RemainingTokenAmountIn.Amount.Sign() == 0 {
					continue
				}
				var in uint256.Int
				in.SetFromBig(a)
				q, err := sim.quoteSwap(true, &in, now)
				require.NoError(t, err)
				var snap hexutil.Big
				f.rpc(&snap, "evm_snapshot")
				fl := f.execFill(e.pool, VenueSwap, e.cbBTC, e.usdc, a, now)
				f.rpc(&ok, "evm_revert", snap)
				require.True(t, ok)
				where := fmt.Sprintf("%s: amountIn %s (%d solves, %d passes)", sc.name, a, q.capEvals, q.passes)
				require.Equal(t, res.TokenAmountOut.Amount.String(), fl.out.String(), "%s amountOut", where)
				require.Equal(t, res.RemainingTokenAmountIn.Amount.String(), fl.unused.String(), "%s amountUnused", where)
				t.Logf("%s: receipt gas %d (estimate %d)", where, fl.gas, res.Gas)
				if fl.gas > uint64(res.Gas) {
					above = append(above, fmt.Sprintf("%s: receipt gas %d, estimate %d", where, fl.gas, res.Gas))
				}
				if q.capEvals != 0 {
					points = append(points, point{sc.name, float64(fl.gas), float64(q.capEvals), float64(q.passes),
						float64(res.Gas)})
				}
			}
		}
	}

	// The terms: the steepest per-solve slope of any (scenario, passes) group; the largest step between the costliest
	// fills of consecutive pass counts of one scenario, net of their solves; the largest intercept left.
	slope := func(g []point) (float64, bool) {
		var n, sx, sy, sxx, sxy float64
		for _, pt := range g {
			n, sx, sy, sxx, sxy = n+1, sx+pt.solves, sy+pt.gas, sxx+pt.solves*pt.solves, sxy+pt.solves*pt.gas
		}
		d := n*sxx - sx*sx
		return (n*sxy - sx*sy) / d, n > 1 && d > 0
	}
	groups := map[string]map[float64][]point{}
	for _, pt := range points {
		if groups[pt.scenario] == nil {
			groups[pt.scenario] = map[float64][]point{}
		}
		groups[pt.scenario][pt.passes] = append(groups[pt.scenario][pt.passes], pt)
	}
	var perSolve, perPass, intercept float64
	for _, byPasses := range groups {
		for _, g := range byPasses {
			if e, ok := slope(g); ok {
				perSolve = math.Max(perSolve, e)
			}
		}
	}
	for _, byPasses := range groups {
		top := map[float64]float64{}
		for k, g := range byPasses {
			top[k] = math.Inf(-1)
			for _, pt := range g {
				top[k] = math.Max(top[k], pt.gas-perSolve*pt.solves)
			}
		}
		for k := range top {
			if next, ok := top[k+1]; ok {
				perPass = math.Max(perPass, next-top[k])
			}
		}
	}
	lo, hi, maxSolves, maxPasses := math.Inf(1), 0.0, 0.0, 0.0
	for _, pt := range points {
		intercept = math.Max(intercept, pt.gas-perSolve*pt.solves-perPass*(pt.passes-1))
		lo, hi = math.Min(lo, pt.estimate/pt.gas), math.Max(hi, pt.estimate/pt.gas)
		maxSolves, maxPasses = math.Max(maxSolves, pt.solves), math.Max(maxPasses, pt.passes)
	}
	t.Logf("%d clipped sells, up to %.0f solves and %.0f passes: receipt gas <= %.0f + %.0f/solve + %.0f/extra pass; "+
		"estimate/receipt in [%.3f, %.3f]", len(points), maxSolves, maxPasses, intercept, perSolve, perPass, lo, hi)
	require.Greater(t, maxPasses, 2.0, "no multi-pass sell was measured")
	require.Empty(t, above)
	require.LessOrEqual(t, 1.25*perSolve, float64(defaultGasSwapSellCapEval))
	require.LessOrEqual(t, 1.25*perPass, float64(defaultGasSwapSellPass))
	require.LessOrEqual(t, 1.25*intercept, float64(defaultGasSwapSell))
}
