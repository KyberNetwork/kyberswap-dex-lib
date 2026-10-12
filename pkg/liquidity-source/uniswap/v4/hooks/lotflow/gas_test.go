package lotflow

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// gasCase is one fork measurement (testdata/gas/measured.json): a swap on a deployed LOTFLOW v1 venue
// through Kyber's real router and executor on an anvil fork of chain 4663, with the adapter's own gas
// features of that swap at the fork block.
type gasCase struct {
	ID          string `json:"id"`
	Session     string `json:"session"`
	Block       uint64 `json:"block"`
	Side        string `json:"side"`
	Usd         int64  `json:"usd"`
	MinGasLimit int64  `json:"minGasLimit"`
	GasUsed     int64  `json:"gasUsed"`
	// StipendProbesHi: the most stipend probes the case may hold when the table cannot pin the count
	// (GasFeatures.StipendProbes is the least); 0 = exact.
	StipendProbesHi int `json:"stipendProbesHi"`
	GasFeatures
}

func loadGasCases(t *testing.T) []gasCase {
	b, err := os.ReadFile("testdata/gas/measured.json")
	require.NoError(t, err)
	var d struct {
		Cases []gasCase `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(b, &d))
	require.NotEmpty(t, d.Cases)
	return d.Cases
}

// ADV2-1: the reported gas covers the minimum passing gas limit of every measured swap, and its fitted
// part is within 1.3x of it (a route cost, not a blanket cap). D4-2: a probe that may exhaust its
// stipend adds gasPerStipendProbe on top. The runs did not record whether the chain filled or burned
// those probes, and the fit already covers what happened, so the charge is checked as the only addition
// (it can push the reported total to ~1.4x, the price of not knowing a near-stipend verdict).
func TestSwapGasCoversForkMeasured(t *testing.T) {
	require.Equal(t, 1_369_653, probeCallerGas) // NavJitLegs.PROBE_CALLER_GAS evaluated in uint256
	under, over, maxR := 0, 0, 0.0
	for _, c := range loadGasCases(t) {
		require.Equal(t, c.Side == "buy", c.Buy, c.ID)
		g := SwapGasFor(c.GasFeatures)
		fit := c.GasFeatures
		fit.StipendProbes = 0
		gFit := SwapGasFor(fit)
		hi := c.GasFeatures
		hi.StipendProbes = max(c.StipendProbes, c.StipendProbesHi)
		gHi := SwapGasFor(hi)
		r, rFit := float64(gHi)/float64(c.MinGasLimit), float64(gFit)/float64(c.MinGasLimit)
		maxR = max(maxR, r)
		t.Logf("%-22s %-5s blk %d: min %9d used %9d reported %9d (%.3fx; fitted part %.3fx; stipend probes %d-%d)",
			c.ID, c.Session, c.Block, c.MinGasLimit, c.GasUsed, g, float64(g)/float64(c.MinGasLimit), rFit,
			c.StipendProbes, hi.StipendProbes)
		if g < c.MinGasLimit {
			under++
			t.Errorf("%s @%d: reported %d < minimum passing limit %d", c.ID, c.Block, g, c.MinGasLimit)
		}
		if rFit > 1.3 {
			over++
			t.Errorf("%s @%d: fitted %d > 1.3x minimum passing limit %d", c.ID, c.Block, gFit, c.MinGasLimit)
		}
		require.LessOrEqual(t, gHi, gFit+int64(hi.StipendProbes)*gasPerStipendProbe+gasRound,
			"%s: the stipend charge is the only addition", c.ID)
		require.GreaterOrEqual(t, g, SwapGas(c.Constituents, c.Mq), "%s: the flat value is the floor", c.ID)
	}
	t.Logf("under %d, fitted part over 1.3x %d; reported incl. stipend charge up to %.3fx", under, over, maxR)
}

// The adapter's own features on the recorded live state of the 2-constituent venue (rpc tape) match what
// the fork measurements recorded for the same venue: a small swap probes 3 candidates per constituent
// at the base leg gas and crosses no tick; a $100k sell crosses ticks and probes at more gas. So the
// table's features are what BeforeSwap computes, and BeforeSwap reports SwapGasFor of them.
func TestGasFeaturesOnRecordedVenue(t *testing.T) {
	h, _, err, _ := tapeTrack(t, liveVenues[0], replayFaults{})
	require.NoError(t, err)
	require.NotNil(t, h.Market)
	nav, err := checkedNavOf(h.Market)
	require.NoError(t, err)
	feat := func(buy bool, usd int64) (GasFeatures, int64) {
		amt := new(big.Int).Mul(big.NewInt(usd), pow10(h.Market.Venue.UsdgDecimals))
		if !buy {
			amt = new(big.Int).Quo(new(big.Int).Mul(big.NewInt(usd), new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)), nav)
		}
		r, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: h.Market.Venue.ZeroForOne(buy),
			AmountSpecified: amt})
		require.NoError(t, err, "buy=%v $%d", buy, usd)
		return gasFeatures(h.Market, buy, r.SwapInfo.(*SwapInfo).res), r.Gas + uniswapv4.DefaultGas.BaseGas
	}
	f, g := feat(true, 1)
	require.Equal(t, GasFeatures{Constituents: 2, Buy: true, Legs: 4, Probes: 6, ProbeGasHi: 930_000,
		LegGasHi: 780_000}, f)
	require.Equal(t, int64(3_900_000), g) // the floor
	f, g = feat(false, 1)
	require.Equal(t, GasFeatures{Constituents: 2, Legs: 2, Probes: 6, ProbeGasHi: 930_000, LegGasHi: 390_000}, f)
	require.Equal(t, int64(4_400_000), g)
	f, g = feat(false, 100_000)
	t.Logf("$100k sell: %+v -> %d", f, g)
	require.Positive(t, f.Ticks)
	require.Greater(t, f.ProbeGasHi, uint64(930_000))
	require.Equal(t, SwapGasFor(f), g)
	require.Greater(t, g, int64(4_992_287), "above the smallest $100k-sell minimum measured on a fork")
}

// D4-3: exact features of tick-crossing swaps on the recorded venue, per leg and summed, so the
// plumbing is pinned where it matters: ticks on BOTH constituents' legs, probes whose bound exceeds
// half the stipend (and one capped at it), and a near-stipend probe (StipendProbes, D4-2).
func TestGasFeaturesPinnedOnRecordedVenue(t *testing.T) {
	h, _, err, _ := tapeTrack(t, liveVenues[0], replayFaults{})
	require.NoError(t, err)
	nav, err := checkedNavOf(h.Market)
	require.NoError(t, err)
	type leg struct {
		Probes, Stipend, Ticks int
		ProbeGasHi, GasHi      uint64
	}
	for _, c := range []struct {
		buy  bool
		usd  int64
		legs []leg
		f    GasFeatures
		gas  int64
	}{
		{true, 100_000, []leg{{3, 0, 5, 1_495_000, 545_000}, {3, 0, 2, 1_075_000, 335_000}, {0, 0, 0, 0, 195_000},
			{0, 0, 0, 0, 195_000}},
			GasFeatures{Constituents: 2, Buy: true, Legs: 4, Probes: 6, Ticks: 7, ProbeGasHi: 2_570_000,
				LegGasHi: 1_270_000}, 4_900_000},
		{false, 100_000, []leg{{3, 0, 8, 1_705_000, 755_000}, {3, 0, 3, 1_215_000, 405_000}},
			GasFeatures{Constituents: 2, Legs: 2, Probes: 6, Ticks: 11, ProbeGasHi: 2_920_000, LegGasHi: 1_160_000},
			5_900_000},
		{false, 150_000, []leg{{3, 1, 3, 2_210_000, 255_000}, {3, 0, 5, 1_415_000, 545_000}},
			GasFeatures{Constituents: 2, Legs: 2, Probes: 6, Ticks: 8, ProbeGasHi: 3_625_000, LegGasHi: 800_000,
				StipendProbes: 1}, 6_900_000},
	} {
		amt := new(big.Int).Mul(big.NewInt(c.usd), pow10(h.Market.Venue.UsdgDecimals))
		if !c.buy {
			amt = new(big.Int).Quo(new(big.Int).Mul(big.NewInt(c.usd), new(big.Int).Exp(big.NewInt(10), big.NewInt(36),
				nil)), nav)
		}
		r, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: h.Market.Venue.ZeroForOne(c.buy),
			AmountSpecified: amt})
		require.NoError(t, err, "buy=%v $%d", c.buy, c.usd)
		res := r.SwapInfo.(*SwapInfo).res
		var legs []leg
		for _, l := range res.Legs {
			legs = append(legs, leg{l.Probes, l.StipendProbes, l.CrossedTicks, l.ProbeGasHi, l.GasHi})
		}
		require.Equal(t, c.legs, legs, "buy=%v $%d: per-leg features", c.buy, c.usd)
		require.Equal(t, c.f, gasFeatures(h.Market, c.buy, res), "buy=%v $%d", c.buy, c.usd)
		require.Equal(t, c.gas, r.Gas+uniswapv4.DefaultGas.BaseGas, "buy=%v $%d", c.buy, c.usd)
	}
}

// D4-3: gasFeatures sums every leg (both directions: a buy's leftover sells count too) and SwapGasFor
// is the documented formula. Expected values worked by hand from the README's model:
// intercept + 45% x ProbeGasHi + 50k x Ticks + 874,653 x StipendProbes, rounded up to 0.1M, floored.
func TestGasFeaturesAndModelTable(t *testing.T) {
	r := &SwapResult{Legs: []ExecutedLeg{
		{Buy: true, Probes: 3, ProbeGasHi: 1_000_000, StipendProbes: 0, CrossedTicks: 4, GasHi: 300_000},
		{Buy: true, Probes: 3, ProbeGasHi: 2_300_000, StipendProbes: 1, CrossedTicks: 6, GasHi: 400_000},
		{Buy: false, Probes: 0, CrossedTicks: 2, GasHi: 195_000},
	}}
	m := &Market{Constituents: make([]MarketConstituent, 2)}
	require.Equal(t, GasFeatures{Constituents: 2, Buy: true, Legs: 3, Probes: 6, Ticks: 12, ProbeGasHi: 3_300_000,
		LegGasHi: 895_000, StipendProbes: 1}, gasFeatures(m, true, r))
	m.Mq = &MqMarket{}
	require.Equal(t, GasFeatures{Constituents: 2, Mq: true, Legs: 3, Probes: 6, Ticks: 12, ProbeGasHi: 3_300_000,
		LegGasHi: 895_000, StipendProbes: 1}, gasFeatures(m, false, r))

	for _, c := range []struct {
		f    GasFeatures
		want int64
	}{
		// 3,970,000 + 1,314,000 + 550,000 = 5,834,000
		{GasFeatures{Constituents: 2, ProbeGasHi: 2_920_000, Ticks: 11}, 5_900_000},
		// 3,380,000 + 1,156,500 + 350,000 = 4,886,500
		{GasFeatures{Constituents: 2, Buy: true, ProbeGasHi: 2_570_000, Ticks: 7}, 4_900_000},
		// 3,620,000 + 456,750 + 250,000 = 4,326,750
		{GasFeatures{Constituents: 3, Mq: true, Buy: true, ProbeGasHi: 1_015_000, Ticks: 5}, 4_400_000},
		// 4,000,000 + 456,750 + 250,000 = 4,706,750
		{GasFeatures{Constituents: 3, Mq: true, ProbeGasHi: 1_015_000, Ticks: 5}, 4_800_000},
		// 3,970,000 + 1,631,250 + 400,000 + 874,653 = 6,875,903
		{GasFeatures{Constituents: 2, ProbeGasHi: 3_625_000, Ticks: 8, StipendProbes: 1}, 6_900_000},
		// 3,380,000 + 418,500 = 3,798,500 -> floor 3,900,000 (2 constituents)
		{GasFeatures{Constituents: 2, Buy: true, ProbeGasHi: 930_000}, 3_900_000},
		// 3,620,000 + 209,250 = 3,829,250 -> floor 4,200,000 (MQ)
		{GasFeatures{Constituents: 3, Mq: true, Buy: true, ProbeGasHi: 465_000}, 4_200_000},
	} {
		require.Equal(t, c.want, SwapGasFor(c.f), "%+v", c.f)
	}
}

// D4-2: a probe that exhausts its stipend burns all of PROBE_GAS on chain (NavJitLegs.sol:87), plus the
// frames around it; the reported gas must carry that in full, not the fitted 45% share of a capped
// bound. The same swap with and without one such probe (features as JSON, so the case reads like a
// row of testdata/gas/measured.json).
func TestSwapGasCountsExhaustedProbeInFull(t *testing.T) {
	const probeGas = 1_200_000 // NavJitLegs.PROBE_GAS
	for _, c := range []struct{ name, without, with string }{
		{"standing buy",
			`{"constituents":2,"buy":true,"legs":4,"probes":5,"ticks":4,"probeGasHi":1500000,"legGasHi":900000}`,
			`{"constituents":2,"buy":true,"legs":4,"probes":6,"ticks":4,"probeGasHi":2700000,"legGasHi":900000,"stipendProbes":1}`},
		{"standing sell, a fully dusted leg",
			`{"constituents":2,"legs":2,"probes":3,"ticks":6,"probeGasHi":1200000,"legGasHi":700000}`,
			`{"constituents":2,"legs":2,"probes":6,"ticks":6,"probeGasHi":4800000,"legGasHi":700000,"stipendProbes":3}`},
		{"MQ sell, near-stipend (Inexact) probe",
			`{"constituents":3,"mq":true,"legs":3,"probes":2,"ticks":5,"probeGasHi":700000,"legGasHi":800000}`,
			`{"constituents":3,"mq":true,"legs":3,"probes":3,"ticks":5,"probeGasHi":1800000,"legGasHi":800000,"stipendProbes":1}`},
	} {
		var without, with GasFeatures
		require.NoError(t, json.Unmarshal([]byte(c.without), &without))
		require.NoError(t, json.Unmarshal([]byte(c.with), &with))
		n := int64(with.Probes - without.Probes)
		g0, g1 := SwapGasFor(without), SwapGasFor(with)
		t.Logf("%s: without %d, with %d exhausted: %d (+%d)", c.name, g0, n, g1, g1-g0)
		require.Greater(t, g0, SwapGas(without.Constituents, without.Mq), "%s: above the floor", c.name)
		require.GreaterOrEqual(t, g1, g0+n*probeGas, "%s: each exhausted probe adds at least PROBE_GAS", c.name)
	}
}

// D4-2 on the chain's own record: wherever a fixture's probe table shows a candidate whose leg needs at
// least PROBE_GAS (re-measured at the pre-state: that probe exhausts its stipend in the swap), the
// adapter's gas features of the same swap count it as a stipend probe.
func TestGasFeaturesCountStipendExhaustingProbes(t *testing.T) {
	const probeGas = 1_200_000
	_, base := loadAll(t)
	singles, _ := loadA5(t)
	exhausted, counted := 0, 0
	for _, c := range append(append(base, singles...), loadMq(t)...) {
		want := 0
		for _, p := range c.Probes {
			for _, k := range p.Candidates {
				if k.Exec != nil && k.Exec.LegGas.Copy().Int64() >= probeGas {
					want++
				}
			}
		}
		if !c.IsFill() {
			continue
		}
		m := marketOf(t, c)
		res, err := Quote(fixtureConfig(c, FinalisedRules), m, inputOf(c))
		require.NoError(t, err, c.ID)
		gf := gasFeatures(m, c.Input.ZeroForOne == m.Venue.UsdgIs0, res)
		// gasUsed of the fixture's own router call (not Kyber's executor, and gasUsed, not the minimum
		// passing limit): a floor the reported gas must clear on every fill.
		used := c.Outcome.GasUsedRouterCall.Copy().Int64()
		require.GreaterOrEqual(t, SwapGasFor(gf), used, "%s: reported below the fixture's gasUsed", c.ID)
		if want == 0 {
			continue
		}
		b, err := json.Marshal(gf)
		require.NoError(t, err)
		var f map[string]any
		require.NoError(t, json.Unmarshal(b, &f))
		got, _ := f["stipendProbes"].(float64)
		t.Logf("%s: %d candidate(s) need >= PROBE_GAS on chain; features %s; reported %d, gasUsed %d", c.ID,
			want, b, SwapGasFor(gf), used)
		exhausted += want
		counted += min(int(got), want)
		require.GreaterOrEqual(t, int(got), want, "%s: stipend-exhausting probes counted", c.ID)
	}
	require.Positive(t, exhausted, "the fixtures hold stipend-exhausting probes (instrument check)")
	t.Logf("stipend-exhausting probes on chain %d, counted %d", exhausted, counted)
}

// D4-2: the stipend-probe rule's boundary. A probe counts when the simulator skips it, or when its
// leg-gas upper bound reaches PROBE_GAS - InexactGasMargin (1.1M by default), whether the maths fills
// it or not (in the navjit1/navjit2-buy-maxfill fixtures the counted probe is not a fill by the maths:
// those quotes are not Inexact, yet each holds a candidate whose leg needs >= PROBE_GAS on chain).
func TestMayExhaustStipendBoundary(t *testing.T) {
	cfg := DefaultConfig("", "", FinalisedRules)
	require.Equal(t, uint64(1_200_000), cfg.ProbeGas)
	require.Equal(t, uint64(100_000), cfg.InexactGasMargin)
	for _, c := range []struct {
		r    LegResult
		want bool
	}{
		{LegResult{Fill: true, GasHi: 1_099_999}, false},
		{LegResult{Fill: true, GasHi: 1_100_000}, true},
		{LegResult{Fill: true, GasHi: 1_150_000, Inexact: true}, true},
		{LegResult{Fill: false, GasHi: 1_150_000}, true},
		{LegResult{Fill: false, GasHi: 900_000}, false},
		{LegResult{Fill: true, GasHi: 2_000_000, GasSkip: true}, true},
		{LegResult{Fill: true, GasHi: 300_000, GasSkip: true}, true}, // NearStipendSkip enumeration
	} {
		require.Equal(t, c.want, cfg.mayExhaustStipend(&c.r), "%+v", c.r)
	}
}
