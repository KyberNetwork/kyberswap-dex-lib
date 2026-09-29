package everlongflamm

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// One test per declared refusal of the integration layer, on a state that reaches it. Each is the only thing
// standing between the quote and the fill it refuses, so deleting the check makes exactly the test below fail --
// which is what these are for: coverage of a branch is not evidence that anything depends on it, and every one of
// them could be deleted with the rest of the suite green.
//
// Two of them no state of the deployed pool reaches, so they are pinned as units on a synthetic state instead: the
// spread's `ppm < PPM` bound (LeverageSpreadHook.sol:47 refuses a maxSpread >= FEE_DEN at construction and
// setBounds keeps it, so `spread` never reaches 1e6) and the swap sell's per-funding-pass gas term (no offline
// state plans a second pass; the fork gas grid mines them).

// TestPolicyEnvelopeVenueReadable: a venue whose market oracle does not answer, or whose IRM the account cannot
// read, refuses the whole pool -- not just the leverage venue, since the swap venue funds through the same Router
// legs. Both come from the state as the tracker read it (venueReads.OracleOk / IrmReadable).
func TestPolicyEnvelopeVenueReadable(t *testing.T) {
	t.Parallel()
	// The control: the untouched state quotes, so a deleted envelope check would serve these fills.
	served := 0
	sim := simFor(t, gridReads(t, "51302915", "live"), baseConfig().policy())
	for _, sell := range []bool{true, false} {
		for _, a := range []uint64{1_000, 10_000, 100_000, 1_000_000, 10_000_000} {
			if _, err := sim.CalcAmountOut(amountIn(sim, sell, a)); err == nil {
				served++
			}
		}
	}
	require.GreaterOrEqual(t, served, 5, "the control state has to quote")

	for _, c := range []struct {
		name   string
		mutate func(*flammReads)
	}{
		{"market oracle does not answer", func(r *flammReads) { r.Router.Venues[0].OracleOk = false }},
		{"market oracle answers zero", func(r *flammReads) {
			r.Router.Venues[0].OracleOk, r.Router.Venues[0].OracleZero = false, true
		}},
		{"the IRM is not readable", func(r *flammReads) { r.Router.Venues[0].IrmReadable = false }},
	} {
		reads := gridReads(t, "51302915", "live")
		c.mutate(reads)
		_, err := NewPoolSimulator(simEntity(t, reads, baseConfig().policy()))
		require.ErrorIs(t, err, ErrPoolRefused, c.name)
	}
	// The IRM half is reached by real recorded states too (the account's grace and quarantine windows).
	for _, tag := range []string{"irm_grace", "irm_quarantine"} {
		reads := gridReads(t, "51302915", tag)
		require.False(t, reads.Router.Venues[0].IrmReadable, tag)
		_, err := NewPoolSimulator(simEntity(t, reads, baseConfig().policy()))
		require.ErrorIs(t, err, ErrPoolRefused, tag)
	}
}

// TestPolicyLeverBandMargin: the taker band less PriceBandMarginBps refuses a leverage fill in both directions.
// The fills below settle to the wei margin-free and sit inside the margin of the pool's own band.
func TestPolicyLeverBandMargin(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		up   bool
		a    uint64
	}{{"lever up", true, 10_000}, {"lever down", false, 5_000}, {"lever down, larger", false, 10_000}} {
		off := simFor(t, gridReads(t, "51302915", "armed"), Policy{LeverRouting: true})
		res, err := off.calcAmountOut(amountIn(off, c.up, c.a), int(VenueLever))
		require.NoError(t, err, "%s: margin-free the pool fills it", c.name)
		require.Positive(t, res.TokenAmountOut.Amount.Sign())

		on := simFor(t, gridReads(t, "51302915", "armed"), Policy{LeverRouting: true,
			PriceBandMarginBps: defaultPriceBandMarginBps})
		_, err = on.calcAmountOut(amountIn(on, c.up, c.a), int(VenueLever))
		require.ErrorIs(t, err, ErrBandMargin, c.name)
	}
}

// TestPolicyLeverDebtDrift: a leverage fill the venue's own debt cap only just admits is refused, because thirty
// seconds of Morpho accrual on the same position puts it over -- the fill would be quoted now and revert in the
// block it lands in. The cap below is the smallest one the fill still settles under at the snapshot clock.
func TestPolicyLeverDebtDrift(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		blk string
		cap uint64
	}{{"51313000", 18_589_310}, {"51324800", 18_589_688}} {
		tight := func() *flammReads {
			r := gridReads(t, c.blk, "armed")
			r.Router.Venues[0].DebtCap.SetUint64(c.cap)
			return r
		}
		off := simFor(t, tight(), Policy{LeverRouting: true})
		res, err := off.calcAmountOut(amountIn(off, true, 10_000), int(VenueLever))
		require.NoError(t, err, "%s: the fill fits under the cap at the snapshot clock", c.blk)
		require.Positive(t, res.TokenAmountOut.Amount.Sign())

		on := simFor(t, tight(), Policy{LeverRouting: true, DebtDriftSec: defaultDebtDriftSec})
		_, err = on.calcAmountOut(amountIn(on, true, 10_000), int(VenueLever))
		require.ErrorIs(t, err, ErrDebtDrift, c.blk)
	}
}

// TestPolicyFeedAgeMarginLoanAsset: PriceAgeMarginSec covers both rounds a fill reads, not only the pool asset's.
// On the deployed pair the loan asset's heartbeat is 90,000 s against the pool asset's 3,600, so it is the pool
// asset's round that normally binds and a check that looked only at it would pass every test -- until the loan
// asset's round is the one about to expire, which is this state.
func TestPolicyFeedAgeMarginLoanAsset(t *testing.T) {
	t.Parallel()
	reads := gridReads(t, "51302915", "live")
	now := reads.Timestamp
	loan := &reads.Feed.Tokens[1]
	require.Equal(t, uint64(90_000), loan.Heartbeat.Uint64())
	// The round expires thirty seconds from now: inside the sixty-second margin, outside the fill's own check.
	loan.Round.UpdatedAt.SetUint64(now + 30 - loan.Heartbeat.Uint64())
	require.Greater(t, reads.Feed.Tokens[0].Round.UpdatedAt.Uint64()+reads.Feed.Tokens[0].Heartbeat.Uint64(),
		now+defaultPriceAgeMarginSec, "the pool asset's round is not the one binding")

	off := simFor(t, reads, Policy{})
	res, err := off.CalcAmountOut(amountIn(off, true, 15_000))
	require.NoError(t, err, "the pool itself still prices it")
	require.Positive(t, res.TokenAmountOut.Amount.Sign())

	on := simFor(t, reads, Policy{PriceAgeMarginSec: defaultPriceAgeMarginSec})
	_, err = on.CalcAmountOut(amountIn(on, true, 15_000))
	require.ErrorIs(t, err, ErrFeedAgeMargin)

	// The margin skips a round the fill itself refuses, so an unregistered token or an aggregator that reverts is
	// reported as the pool's own error and not as a margin refusal.
	for name, mutate := range map[string]func(*flammReads){
		"an unregistered token":      func(r *flammReads) { r.Feed.Tokens[1].Known = false },
		"an aggregator that reverts": func(r *flammReads) { r.Feed.Tokens[1].Round.Ok = false },
	} {
		r := gridReads(t, "51302915", "live")
		mutate(r)
		sim := simFor(t, r, Policy{PriceAgeMarginSec: defaultPriceAgeMarginSec})
		_, err := sim.CalcAmountOut(amountIn(sim, true, 15_000))
		require.Error(t, err, name)
		require.NotErrorIs(t, err, ErrFeedAgeMargin, name)
	}
}

// TestPolicySpreadPpmBound pins the two halves of spreadLive that no state reaches: a pool with no spread hook,
// and a posted spread at or above PPM. The deployed hook cannot post one (LeverageSpreadHook.sol:47, :92 keep
// maxSpread < FEE_DEN = 1e6, and setSpread clamps to it), so the bound is checked here rather than on a fixture.
func TestPolicySpreadPpmBound(t *testing.T) {
	t.Parallel()
	sim := simFor(t, gridReads(t, "51302915", "armed"), Policy{LeverRouting: true})
	now := sim.state.Timestamp
	require.NoError(t, sim.spreadLive(now, 0), "the armed state's post is live")

	s := sim.state.clone()
	s.Hooks.Spread.EverlongSpread.Spread.SetUint64(1_000_000)
	require.ErrorIs(t, (&PoolSimulator{state: s, Policy: sim.Policy}).spreadLive(now, 0), ErrSpreadNotLive,
		"a spread at PPM leaves the taker nothing")
	s = sim.state.clone()
	s.Hooks.Spread.EverlongSpread.Spread.SetUint64(999_999)
	require.NoError(t, (&PoolSimulator{state: s, Policy: sim.Policy}).spreadLive(now, 0))
	s = sim.state.clone()
	s.Hooks.Spread = spreadHookSlot{}
	require.ErrorIs(t, (&PoolSimulator{state: s, Policy: sim.Policy}).spreadLive(now, 0), ErrSpreadNotLive,
		"a pool with no spread hook has no answer")
}

// TestPolicySpreadAgeMargin: SpreadAgeMarginSec refuses a post that lapses inside the margin, and the hard
// liveness check refuses one that has already lapsed whatever the margin.
func TestPolicySpreadAgeMargin(t *testing.T) {
	t.Parallel()
	reads := gridReads(t, "51302915", "armed")
	now := reads.Timestamp
	// A post that stays live for another thirty seconds: inside the default margin, outside the pool's own check.
	reads.Spread.LastSetTs.SetUint64(now + 30 - reads.Spread.MaxSpreadAge.Uint64())

	off := simFor(t, reads, Policy{LeverRouting: true})
	_, err := off.calcAmountOut(amountIn(off, false, 5_000), int(VenueLever))
	require.NoError(t, err, "the pool prices it: the post answers at the quote's own clock")

	on := simFor(t, reads, Policy{LeverRouting: true, SpreadAgeMarginSec: defaultSpreadAgeMarginSec})
	_, err = on.calcAmountOut(amountIn(on, false, 5_000), int(VenueLever))
	require.ErrorIs(t, err, ErrSpreadNotLive)

	// And the margin does not move the routed quote: the venue is picked on the two settlements, which the margin
	// does not run, so the routed pick is the same with the margin on and off.
	require.Equal(t, routedVenue(t, off, false, 5_000), routedVenue(t, on, false, 5_000))

	// A post that has already lapsed is refused whatever the margin: that half is the settlement's own, so the
	// venue pick never sees a lever fill priced at the stored degrade value (the recorded stale_spread state).
	lapsed := simFor(t, gridReads(t, "51302915", "stale_spread"), Policy{LeverRouting: true})
	for _, up := range []bool{true, false} {
		_, err := lapsed.calcAmountOut(amountIn(lapsed, up, 5_000), int(VenueLever))
		require.ErrorIs(t, err, ErrSpreadNotLive, "up=%v", up)
	}
}

// routedVenue is the venue the routed quote picks, or -1 when it refuses.
func routedVenue(t *testing.T, sim *PoolSimulator, sell bool, a uint64) int {
	t.Helper()
	res, err := sim.CalcAmountOut(amountIn(sim, sell, a))
	if err != nil {
		return -1
	}
	return int(res.SwapInfo.(SwapInfo).Venue)
}

// TestPolicyGasTerms: the swap sell estimate charges its own settlement's cap solves and every funding pass past
// the first (FLAMMSwapLib.sol:87, :163, :197). No offline state reaches a second funding pass -- the fork gas grid
// does -- so the terms are pinned here against the fill counts the settlement reports.
func TestPolicyGasTerms(t *testing.T) {
	t.Parallel()
	sim := simFor(t, gridReads(t, "51302915", "live"), baseConfig().policy())
	g := sim.Policy.withDefaults()
	for _, c := range []struct {
		name             string
		venue            uint8
		sell             bool
		capEvals, passes uint64
		want             int64
	}{
		{"buy", VenueSwap, false, 0, 1, g.GasSwapBuy},
		{"sell, one pass, no solve", VenueSwap, true, 0, 1, g.GasSwapSell},
		{"sell, 24 solves", VenueSwap, true, 24, 1, g.GasSwapSell + 24*g.GasSwapSellCapEval},
		{"sell, two passes", VenueSwap, true, 0, 2, g.GasSwapSell + g.GasSwapSellPass},
		{"sell, four passes and 256 solves", VenueSwap, true, 256, 4,
			g.GasSwapSell + 256*g.GasSwapSellCapEval + 3*g.GasSwapSellPass},
		{"lever up", VenueLever, true, 0, 0, g.GasLeverUp},
		{"lever down", VenueLever, false, 0, 0, g.GasLeverDown},
	} {
		f := &fill{venue: c.venue, capEvals: c.capEvals, passes: c.passes}
		require.Equal(t, c.want, sim.gas(f, c.sell), c.name)
	}

	// And a real clipped sell reports the counts the estimate is built from.
	capped := simFor(t, gridReads(t, "51302915", "capped"), Policy{})
	res, err := capped.CalcAmountOut(amountIn(capped, true, 10_000_000))
	require.NoError(t, err)
	require.Positive(t, res.RemainingTokenAmountIn.Amount.Sign(), "the sell is clipped")
	require.Greater(t, res.Gas, g.GasSwapSell, "a clipped sell is charged for its cap solves")
}

// TestPolicyMarginBandExhausted: a margin at or above the loan asset's whole band leaves nothing to quote inside,
// and every fill on either venue is refused rather than quoted against a band of zero.
func TestPolicyMarginBandExhausted(t *testing.T) {
	t.Parallel()
	reads := gridReads(t, "51302915", "armed")
	band := reads.Pool.Loans[0].SwapPriceBandWad.Uint64() // 8e16 on the deployed pair
	require.Positive(t, band)
	bps := band / 1e14
	sim := simFor(t, reads, Policy{LeverRouting: true, PriceBandMarginBps: bps})
	_, ok := sim.marginBand(uint256.NewInt(band))
	require.False(t, ok, "a margin of the whole band leaves nothing")
	for _, venue := range []int{int(VenueSwap), int(VenueLever)} {
		for _, sell := range []bool{true, false} {
			_, err := sim.calcAmountOut(amountIn(sim, sell, 10_000), venue)
			require.ErrorIs(t, err, ErrBandMargin, "venue %d sell %v", venue, sell)
		}
	}
	// One basis point below it still quotes, so the refusal is the margin and not the state.
	looser := simFor(t, gridReads(t, "51302915", "armed"), Policy{LeverRouting: true, PriceBandMarginBps: bps - 1})
	_, ok = looser.marginBand(uint256.NewInt(band))
	require.True(t, ok)
}

// TestPolicyBrokenSimulator: which SwapInfo a simulator adopts. The lineage token is a value derived from the
// entity and folded with each adopted fill (pool_simulator.go lineageOf), not an identity drawn per construction,
// so a second simulator built from the same entity is the same state and its quote is adopted -- while a state
// that has moved on, a different state at the same block and anything that is not a SwapInfo are refused, and the
// refusal sticks to every later quote rather than quoting liquidity the route already spent.
func TestPolicyBrokenSimulator(t *testing.T) {
	t.Parallel()
	sim := simFor(t, gridReads(t, "51302915", "live"), Policy{})
	twin := simFor(t, gridReads(t, "51302915", "live"), Policy{})
	require.Equal(t, sim.lineage, twin.lineage, "two simulators of one entity are the same state")
	res, err := twin.CalcAmountOut(amountIn(twin, true, 15_000))
	require.NoError(t, err)

	// The twin's quote is this state's quote: adopted, and the two stay in step.
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
	twin.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
	require.Equal(t, sim.lineage, twin.lineage)
	after, err := sim.CalcAmountOut(amountIn(sim, true, 15_000))
	require.NoError(t, err)
	againOnTwin, err := twin.CalcAmountOut(amountIn(twin, true, 15_000))
	require.NoError(t, err)
	require.Equal(t, againOnTwin.TokenAmountOut.Amount, after.TokenAmountOut.Amount)

	// A SwapInfo from a state that has adopted a fill since is not this one's.
	stale := simFor(t, gridReads(t, "51302915", "live"), Policy{})
	moved := simFor(t, gridReads(t, "51302915", "live"), Policy{})
	first, err := moved.CalcAmountOut(amountIn(moved, true, 1_000))
	require.NoError(t, err)
	moved.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
	second, err := moved.CalcAmountOut(amountIn(moved, true, 15_000))
	require.NoError(t, err)
	require.NotEqual(t, stale.lineage, moved.lineage)
	stale.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: second.SwapInfo})
	_, err = stale.CalcAmountOut(amountIn(stale, true, 15_000))
	require.ErrorIs(t, err, ErrSwapInfoMismatch)

	// Another state of the same pool at the same block: a different token, refused.
	sibling := simFor(t, gridReads(t, "51302915", "ltv_low"), Policy{})
	require.NotEqual(t, sim.lineage, sibling.lineage)
	other, err := sibling.CalcAmountOut(amountIn(sibling, true, 15_000))
	require.NoError(t, err)
	fresh := simFor(t, gridReads(t, "51302915", "live"), Policy{})
	fresh.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: other.SwapInfo})
	_, err = fresh.CalcAmountOut(amountIn(fresh, true, 15_000))
	require.ErrorIs(t, err, ErrSwapInfoMismatch)

	// The fold covers every input of the settlement, not only its amounts: a fill quoted for another input (a
	// lever-down is sized on the whole input) or in the other direction leaves another state.
	si := res.SwapInfo.(SwapInfo)
	base := adoptLineage(sim.lineage, &si)
	otherIn := si
	otherIn.amountIn.AddUint64(&si.amountIn, 1)
	otherDir := si
	otherDir.PoolAssetIn = !si.PoolAssetIn
	otherVenue := si
	otherVenue.Venue = VenueLever
	otherClock := si
	otherClock.next = si.next.clone()
	otherClock.next.Timestamp++
	for name, v := range map[string]*SwapInfo{"input": &otherIn, "direction": &otherDir, "venue": &otherVenue,
		"clock": &otherClock} {
		require.NotEqual(t, base, adoptLineage(sim.lineage, v), name)
	}
	require.Equal(t, base, adoptLineage(sim.lineage, &si))

	// And anything that is not a SwapInfo at all.
	notInfo := simFor(t, gridReads(t, "51302915", "live"), Policy{})
	notInfo.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: struct{}{}})
	_, err = notInfo.CalcAmountOut(amountIn(notInfo, true, 15_000))
	require.ErrorIs(t, err, ErrSwapInfoMismatch)
}

// TestPolicyInvalidInputs: the parameters CalcAmountOut refuses before it reaches the state.
func TestPolicyInvalidInputs(t *testing.T) {
	t.Parallel()
	sim := simFor(t, gridReads(t, "51302915", "live"), Policy{})
	in, out := sim.Info.Tokens[0], sim.Info.Tokens[1]
	for _, c := range []struct {
		name   string
		params pool.CalcAmountOutParams
		want   error
	}{
		{"unknown token in", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: "0xdead",
			Amount: big.NewInt(1)}, TokenOut: out}, ErrInvalidToken},
		{"same token", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: big.NewInt(1)},
			TokenOut: in}, ErrInvalidToken},
		{"nil amount", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in}, TokenOut: out},
			ErrInvalidAmount},
		{"zero amount", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: big.NewInt(0)},
			TokenOut: out}, ErrInvalidAmount},
		{"negative amount", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in,
			Amount: big.NewInt(-1)}, TokenOut: out}, ErrInvalidAmount},
	} {
		_, err := sim.CalcAmountOut(c.params)
		require.ErrorIs(t, err, c.want, c.name)
	}
}

// TestPolicyLeverOracleWindow: the leverage venue's half of the oracle-window gate. Every fill is re-run at the
// market oracle answer the refresh read for the end of the snapshot window (Extra.OracleAhead) and refused unless
// it settles identically; the swap venue's half is pinned by TestSimulatorOracleWindow, which quotes only venue 0
// because the deployed pool has leverage paused. Without this case the leverage branch of guardLever's
// oracleShifted re-run can be deleted with the whole offline suite green.
//
// A lever-up is the direction the price gates: it draws on the venue, so the Router's band around the PriceFeed
// cross (MMRouterLib.bandOk) and Morpho's health both read the market oracle. A lever-down repays, so neither
// binds it -- measured below rather than assumed, so that the asymmetry is on the record.
func TestPolicyLeverOracleWindow(t *testing.T) {
	t.Parallel()
	// ahead is the venues' own oracle answers moved by one part in `over` (negative: down); over == 0 leaves them.
	ahead := func(s *flammState, over int64) []OracleAnswer {
		out := make([]OracleAnswer, len(s.Router.Venues))
		for i := range s.Router.Venues {
			m := &s.Router.Venues[i].Morpho
			var p, step uint256.Int
			p.Set(&m.OraclePrice)
			if over > 0 {
				p.Add(&p, step.Div(&p, uint256.NewInt(uint64(over))))
			} else if over < 0 {
				p.Sub(&p, step.Div(&p, uint256.NewInt(uint64(-over))))
			}
			out[i] = OracleAnswer{Ok: m.OracleOk, Price: p}
		}
		return out
	}
	armed := func(t *testing.T) *PoolSimulator {
		return simFor(t, gridReads(t, "51302915", "armed"), Policy{LeverRouting: true})
	}
	const up, a = true, uint64(10_000)

	base := armed(t)
	want, err := base.calcAmountOut(amountIn(base, up, a), int(VenueLever))
	require.NoError(t, err, "the pool fills the lever-up with no window declared")

	// An unmoved answer costs nothing: the same amounts, to the wei.
	still := armed(t)
	still.OracleAhead = ahead(still.state, 0)
	got, err := still.calcAmountOut(amountIn(still, up, a), int(VenueLever))
	require.NoError(t, err)
	require.Zero(t, got.TokenAmountOut.Amount.Cmp(want.TokenAmountOut.Amount))
	require.Zero(t, got.RemainingTokenAmountIn.Amount.Cmp(want.RemainingTokenAmountIn.Amount))

	// A round inside the window that moves the answer enough to stop the fill settling refuses the quote.
	for _, over := range []int64{50, 33, 20, -20, -33} {
		moved := armed(t)
		moved.OracleAhead = ahead(moved.state, over)
		_, err := moved.calcAmountOut(amountIn(moved, up, a), int(VenueLever))
		require.ErrorIs(t, err, ErrOracleDrift, "lever up at one part in %d", over)
	}
	// An oracle that stops answering inside the window is refused too.
	dead := armed(t)
	dead.OracleAhead = make([]OracleAnswer, len(dead.state.Router.Venues))
	_, err = dead.calcAmountOut(amountIn(dead, up, a), int(VenueLever))
	require.ErrorIs(t, err, ErrOracleDrift, "the oracle stops answering")

	// A lever-down on the same state is not priced by this oracle at all: it repays, so neither the Router band nor
	// Morpho health reads the answer on its amounts. Recorded, not assumed.
	down := armed(t)
	downWant, err := down.calcAmountOut(amountIn(down, false, 5_000), int(VenueLever))
	require.NoError(t, err)
	shifted := armed(t)
	shifted.OracleAhead = ahead(shifted.state, 5) // a 20% move
	downGot, err := shifted.calcAmountOut(amountIn(shifted, false, 5_000), int(VenueLever))
	require.NoError(t, err, "a lever-down is not gated by the market oracle answer")
	require.Zero(t, downGot.TokenAmountOut.Amount.Cmp(downWant.TokenAmountOut.Amount))
}
