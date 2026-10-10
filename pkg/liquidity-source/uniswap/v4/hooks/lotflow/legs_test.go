package lotflow

import (
	"math/big"
	"strings"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

// Hook-set fees of the two dynamic-fee (hooked) registry price pools in the fixtures, MEASURED on
// fork in every observation (P2-FIX.md §6): GLD's Fables pool 2500 pips, CRCL's 2200. The simulator
// takes a dynamic fee as an input (Config.DynamicFee); these are the values the fixtures saw.
var fixtureDynamicFees = map[string]uint32{"GLD": 2500, "CRCL": 2200}

// ---------------------------------------------------------------- fixture -> model adapters

func fixtureConfig(c *Case, rules Rules) *Config {
	dyn := map[string]uint32{}
	for _, k := range c.Pre.Constituents {
		for _, cand := range k.Candidates {
			if cand.DynamicFee {
				if f, ok := fixtureDynamicFees[k.Symbol]; ok {
					dyn[strings.ToLower(cand.SourceID)] = f
				}
			}
		}
	}
	cfg := DefaultConfig(c.Pre.Hook.Usdg, c.Pre.Hook.V3Factory, rules)
	cfg.MaxCandidates = int(c.Pre.LegsConst.MaxCandidates.Int64()) // the fixture's own constant (3)
	cfg.DynamicFee = func(id string) (uint32, bool) { f, ok := dyn[strings.ToLower(id)]; return f, ok }
	return cfg
}

func registryOf(k *Constituent) RegistryEntry {
	r := k.Registry
	e := RegistryEntry{Stable: r.Stable, PriceIndex: r.PriceIndex, StockIsCurrency0: r.StockIsCurrency0}
	for _, p := range r.Pools {
		e.Pools = append(e.Pools, RegistryPoolView{PoolID: p.PoolID, Hooks: p.Key.Hooks,
			TickSpacing: p.Key.TickSpacing, SqrtPriceX96: p.SqrtPriceX96.Copy(), Liquidity: p.Liquidity.Copy()})
	}
	return e
}

func readWord(raw []byte) (*big.Int, bool) {
	s := strings.Trim(string(raw), `"`)
	v, ok := new(big.Int).SetString(s, 10)
	return v, ok
}

func bookOf(k *Constituent) (bool, []BookVenueView) {
	var out []BookVenueView
	if k.Book.VenuesOf == "" && k.Book.Venues == nil {
		// The MQ set records no VenueBook block for registry stocks, only the admitted candidate list
		// (asserted by the harness against NavJitLegsMq.sources): the v3 candidates stand in for the
		// book, so admission on that set is the contract's, not independently re-derived.
		for _, cand := range k.Candidates {
			if cand.Kind != KindV3 {
				continue
			}
			out = append(out, BookVenueView{Pool: cand.Pool, Family: 1, Factory: cand.Factory, FactoryOK: true,
				SqrtPriceX96: cand.SqrtPriceX96.Copy(), SqrtOK: true, Liquidity: cand.Liquidity.Copy(), LiqOK: true,
				TickSpacing: cand.TickSpacing, TickSpacingOK: true})
		}
		return true, out
	}
	for _, v := range k.Book.Venues {
		bv := BookVenueView{Pool: v.Pool, Family: v.Family, Factory: v.ReadFactory,
			FactoryOK: strings.HasPrefix(v.ReadFactory, "0x")}
		bv.SqrtPriceX96, bv.SqrtOK = readWord(v.ReadSqrtPriceX96)
		bv.Liquidity, bv.LiqOK = readWord(v.ReadLiquidity)
		if ts, ok := readWord(v.ReadTickSpacing); ok {
			bv.TickSpacing, bv.TickSpacingOK = int(ts.Int64()), true
		}
		out = append(out, bv)
	}
	return k.Book.VenuesOf == "ok", out
}

func legPoolOf(cand *Candidate) *LegPool {
	p := &LegPool{
		Kind: cand.Kind, SqrtPriceX96: cand.SqrtPriceX96.Copy(), Tick: cand.Tick,
		Liquidity: cand.Liquidity.Copy(), TickSpacing: cand.TickSpacing,
		Fee: cand.Fee, LpFee: cand.LpFee, ProtocolFee: cand.ProtocolFee, DynamicFee: cand.DynamicFee,
	}
	for _, t := range cand.Ticks {
		p.Ticks = append(p.Ticks, uniswapv3.TickU256{Index: t.Tick,
			LiquidityGross: uint256.MustFromBig(t.LiquidityGross.Copy()),
			LiquidityNet:   int256.MustFromBig(t.LiquidityNet.Copy())})
	}
	return p
}

// sourcesOf runs the admission rules on a fixture constituent and returns the sources plus one
// LegPool per source taken from the fixture's candidate state.
func sourcesOf(t *testing.T, cfg *Config, k *Constituent) ([]Source, []*LegPool) {
	t.Helper()
	ok, venues := bookOf(k)
	srcs, _, _ := Sources(cfg, k.Stock, registryOf(k), ok, venues)
	require.Len(t, srcs, len(k.Candidates))
	pools := make([]*LegPool, len(srcs))
	for i := range srcs {
		require.Equal(t, strings.ToLower(k.Candidates[i].SourceID), strings.ToLower(srcs[i].ID))
		pools[i] = legPoolOf(&k.Candidates[i])
	}
	return srcs, pools
}

// ---------------------------------------------------------------- admission

// NavJitLegs.sources() mirror: every constituent of every fixture gets the exact candidate list, in
// order, and every registry pool / VenueBook venue gets the fixture's admission verdict.
func TestAdmissionMatchesFixtures(t *testing.T) {
	_, cases := loadAll(t)
	consts, labels := 0, 0
	for _, c := range cases {
		cfg := fixtureConfig(c, FinalisedRules)
		for i := range c.Pre.Constituents {
			k := &c.Pre.Constituents[i]
			ok, venues := bookOf(k)
			srcs, regAdm, bookAdm := Sources(cfg, k.Stock, registryOf(k), ok, venues)
			require.Len(t, srcs, len(k.Candidates), c.ID, k.Symbol)
			for j, s := range srcs {
				cand := k.Candidates[j]
				require.Equal(t, cand.Kind, s.Kind, c.ID)
				require.Equal(t, strings.ToLower(cand.SourceID), strings.ToLower(s.ID), c.ID)
				require.Equal(t, cand.StockIs0, s.StockIs0, c.ID)
			}
			for j, p := range k.Registry.Pools {
				require.Equal(t, p.Admission, string(regAdm[j]), c.ID, k.Symbol, j)
				labels++
			}
			for j, v := range k.Book.Venues {
				require.Equal(t, v.Admission, string(bookAdm[j]), c.ID, k.Symbol, j)
				labels++
			}
			consts++
		}
	}
	require.Equal(t, 218, consts)
	t.Logf("admission: %d/218 candidate lists exact (ids, order, kind, stockIs0), %d/%d labels", consts, labels,
		labels)
}

// Branches the fixtures never reach, and the parameterised finalisation rules (DESIGN D-B).
func TestAdmissionRules(t *testing.T) {
	usdg, fac := "0x5fc5360D0400a0Fd4f2af552ADD042D716F1d168", "0x1f7d7550B1b028f7571E69A784071F0205FD2EfA"
	one := big.NewInt(1)
	price := new(big.Int).Lsh(big.NewInt(1), 96)
	stock := "0x0000000000000000000000000000000000000001"
	reg := RegistryEntry{Stable: usdg, PriceIndex: 1, Pools: []RegistryPoolView{
		{PoolID: "0xa1", Hooks: "0x00000000000000000000000000000000000000aa", TickSpacing: 60, SqrtPriceX96: price, Liquidity: one}, // hooked, not price pool
		{PoolID: "0xa2", Hooks: "0x00000000000000000000000000000000000000bb", TickSpacing: 60, SqrtPriceX96: price, Liquidity: one}, // hooked price pool
		{PoolID: "0xa3", Hooks: zeroAddr, TickSpacing: 60, SqrtPriceX96: new(big.Int), Liquidity: one},                              // no price
		{PoolID: "0xa4", Hooks: zeroAddr, TickSpacing: 1, SqrtPriceX96: price, Liquidity: one},                                      // narrow
		{PoolID: "0xa5", Hooks: zeroAddr, TickSpacing: 10, SqrtPriceX96: price, Liquidity: one},
	}}
	venue := func(pool string, fam int, f string, sp *big.Int, ts int) BookVenueView {
		return BookVenueView{Pool: pool, Family: fam, Factory: f, FactoryOK: true, SqrtPriceX96: sp, SqrtOK: true,
			Liquidity: one, LiqOK: true, TickSpacing: ts, TickSpacingOK: true}
	}
	book := []BookVenueView{
		venue("0x00000000000000000000000000000000000000b1", 0, fac, price, 10),
		venue("0x00000000000000000000000000000000000000b2", 1, fac, new(big.Int).Set(MaxSqrtPrice), 10),
		venue("0x00000000000000000000000000000000000000b3", 1, fac, price, 10),
		venue("0x00000000000000000000000000000000000000b4", 1, fac, price, 1),
		venue("0x00000000000000000000000000000000000000b5", 1, fac, price, 60),
		venue("0x00000000000000000000000000000000000000b6", 1, fac, price, 60),
		venue("0x00000000000000000000000000000000000000b7", 1, fac, price, 60),
	}
	ids := func(s []Source) string {
		var o []string
		for _, x := range s {
			o = append(o, x.ID[len(x.ID)-2:])
		}
		return strings.Join(o, ",")
	}
	cur := DefaultConfig(usdg, fac, PreFinalisationRules)
	s, ra, ba := Sources(cur, stock, reg, true, book)
	require.Equal(t, "a2,a4,a5,b3,b4,b5,b6", ids(s))
	require.Equal(t, []Admission{AdmHookedNotPricePool, AdmAdmitted, AdmNoPrice, AdmAdmitted, AdmAdmitted}, ra)
	require.Equal(t, []Admission{AdmFamily0, AdmPriceOutOfRange, AdmAdmitted, AdmAdmitted, AdmAdmitted, AdmAdmitted,
		AdmOverMaxBook}, ba)

	fin := DefaultConfig(usdg, fac, FinalisedRules)
	s, ra, ba = Sources(fin, stock, reg, true, book)
	require.Equal(t, "a5,b3,b5", ids(s))
	require.Equal(t, []Admission{AdmHooked, AdmHooked, AdmNoPrice, AdmTickSpacing, AdmAdmitted}, ra)
	require.Equal(t, AdmTickSpacing, ba[3])
	require.Equal(t, AdmOverMaxCandidates, ba[5])

	// stable not USDG / no pools / venuesOf reverted
	bad := reg
	bad.Stable = fac
	s, ra, _ = Sources(cur, stock, bad, true, book)
	require.Empty(t, s)
	require.Equal(t, AdmStableNotUsdg, ra[0])
	s, _, ba = Sources(cur, stock, reg, false, book)
	require.Equal(t, "a2,a4,a5", ids(s))
	require.Empty(t, ba)
}

// ---------------------------------------------------------------- legs

// legLimit (NavJitLegs._limit) reproduces every candidate's recorded buy and sell limit.
func TestLegLimitsMatchFixtures(t *testing.T) {
	_, cases := loadAll(t)
	n := 0
	for _, c := range cases {
		cfg := fixtureConfig(c, FinalisedRules)
		for _, k := range c.Pre.Constituents {
			for _, cand := range k.Candidates {
				sp := cand.SqrtPriceX96.Copy()
				eq(t, cand.LegLimit.Buy.Copy(), cfg.LegLimit(sp, true != cand.StockIs0), c.ID)
				eq(t, cand.LegLimit.Sell.Copy(), cfg.LegLimit(sp, false != cand.StockIs0), c.ID)
				n++
			}
		}
	}
	t.Logf("leg limits: %d/%d candidates exact (buy and sell)", n, n)
}

// A3 acceptance: every probe table of every filling fixture is reproduced from the semantic
// pre-state: each candidate's verdict, and for fills the exact USDG result and post-swap
// sqrtPrice/tick/liquidity; the pick equals the candidate the swap executed. Probes whose leg gas
// the model puts near PROBE_GAS are flagged Inexact (never silently decided).
func TestProbesReproduced(t *testing.T) {
	_, cases := loadAll(t)
	fills, nofills, legs, picks, inexact := 0, 0, 0, 0, 0
	var maxFillGas uint64
	for _, c := range cases {
		cfg := fixtureConfig(c, FinalisedRules)
		for _, pr := range c.Probes {
			if pr.Mode != ModeProbed {
				continue
			}
			k := &c.Pre.Constituents[pr.ConstituentIndex]
			require.Equal(t, strings.ToLower(k.Stock), strings.ToLower(pr.Stock), c.ID)
			srcs, pools := sourcesOf(t, cfg, k)
			require.Len(t, pr.Candidates, len(srcs), c.ID)
			amt := pr.Amount.Copy()
			pick, err := Pick(cfg, k.Stock, srcs, pools, pr.Buy, amt)
			require.NoError(t, err, c.ID)
			require.True(t, pick.Probed, c.ID)
			for j, pc := range pr.Candidates {
				require.Equal(t, strings.ToLower(pc.SourceID), strings.ToLower(srcs[j].ID), c.ID)
				r := pick.Results[j]
				if r.Inexact {
					inexact++
				}
				switch pc.Verdict {
				case VerdictFill:
					require.True(t, r.Filled(), "%s %s cand %d: %v", c.ID, k.Symbol, j, r.Err)
					eq(t, pc.ProbeResult.Copy(), r.Result, c.ID, k.Symbol, j)
					eq(t, pc.Exec.PostSqrtPriceX96.Copy(), r.SqrtPriceX96, c.ID, k.Symbol, j)
					require.Equal(t, int(*pc.Exec.PostTick), r.Tick, c.ID, k.Symbol, j)
					eq(t, pc.Exec.PostLiquidity.Copy(), r.Liquidity, c.ID, k.Symbol, j)
					g := pc.Exec.LegGas.Uint64()
					if g > maxFillGas {
						maxFillGas = g
					}
					require.True(t, r.GasLo <= g && g <= r.GasHi, "%s %s cand %d: gas %d outside [%d, %d]",
						c.ID, k.Symbol, j, g, r.GasLo, r.GasHi)
					if g >= cfg.ProbeGas-cfg.InexactGasMargin {
						require.True(t, r.Inexact, "%s: leg gas %d near PROBE_GAS must be Inexact", c.ID, g)
					}
					fills++
				case VerdictNoFill:
					require.False(t, r.Filled(), "%s %s cand %d", c.ID, k.Symbol, j)
					nofills++
				default:
					t.Fatalf("%s: verdict %q", c.ID, pc.Verdict)
				}
				if pc.ExecutedInSwap {
					require.Equal(t, j, pick.Index, "%s %s: picked %d, swap executed %d", c.ID, k.Symbol,
						pick.Index, j)
					picks++
				}
			}
			legs++
		}
	}
	require.Equal(t, 352, fills)
	require.Equal(t, 46, nofills)
	require.Equal(t, 140, picks)
	t.Logf("probes: %d legs, fills %d/352 exact (result + post state), nofills %d/46, picks %d/140, "+
		"Inexact-flagged %d (gas envelope reaches PROBE_GAS-margin); max filling leg gas %d, inside the "+
		"envelope on all %d fills", legs, fills, nofills, picks, inexact, maxFillGas, fills)
}

// Every executed leg that runs at the pre-state (probed and single-candidate legs; memo sells run
// after the buy leg on the same pool and belong to the whole-swap stage) reproduces its LegExecuted
// paid/got exactly, and single-candidate legs execute unprobed.
func TestExecutedLegsAtPreState(t *testing.T) {
	_, cases := loadAll(t)
	n, single := 0, 0
	for _, c := range cases {
		if !c.IsFill() {
			continue
		}
		cfg := fixtureConfig(c, FinalisedRules)
		mode := map[int]map[bool]string{}
		for _, pr := range c.Probes {
			ci := int(pr.ConstituentIndex)
			if mode[ci] == nil {
				mode[ci] = map[bool]string{}
			}
			mode[ci][pr.Buy] = pr.Mode
		}
		for _, l := range c.Derived.Legs {
			m := mode[l.ConstituentIndex][l.Buy]
			if m != ModeProbed && !strings.HasPrefix(m, "single") {
				continue
			}
			k := &c.Pre.Constituents[l.ConstituentIndex]
			srcs, pools := sourcesOf(t, cfg, k)
			amt := l.Got.Copy()
			if !l.Buy {
				amt = l.Paid.Copy()
			}
			pick, err := Pick(cfg, k.Stock, srcs, pools, l.Buy, amt)
			require.NoError(t, err, c.ID)
			require.Equal(t, l.CandidateIndex, pick.Index, c.ID)
			r := pick.Results[pick.Index]
			eq(t, l.Paid.Copy(), r.Paid, c.ID, k.Symbol)
			eq(t, l.Got.Copy(), r.Got, c.ID, k.Symbol)
			if strings.HasPrefix(m, "single") {
				require.False(t, pick.Probed, c.ID)
				single++
			}
			n++
		}
	}
	require.Equal(t, 0, single)
	require.Equal(t, 140, n)
	t.Logf("executed legs at pre-state: %d/140 exact paid/got (incl. %d single-candidate unprobed)", n, single)
}

// The four over-max buys that revert NoExecutableSource: at the constituent amount the venue sizes
// (ceil(unit*inventory/1e18)), every candidate of the named stock fails to fill.
func TestNoExecutableSourceReverts(t *testing.T) {
	_, cases := loadAll(t)
	n := 0
	for _, c := range cases {
		if c.IsFill() || c.Derived.RevertInnermost.Name != "NoExecutableSource" {
			continue
		}
		cfg := fixtureConfig(c, FinalisedRules)
		v := venueOf(c)
		var amts []*big.Int
		if c.Input.IsBuy() {
			inv, err := v.BuyInventory(c.Input.AmountIn.Copy(), c.Pre.Nav.CheckedNavPerUnit18.Copy())
			require.NoError(t, err, c.ID)
			amts = MintAmounts(unitsOf(c), inv)
		} else {
			amts = RedeemAmounts(unitsOf(c), c.Input.AmountIn.Copy()) // any size: the stock has no source
		}
		stock := c.Derived.RevertInnermost.Args[0]
		found := false
		for i := range c.Pre.Constituents {
			k := &c.Pre.Constituents[i]
			if strings.ToLower(k.Stock) != stock {
				continue
			}
			srcs, pools := sourcesOf(t, cfg, k)
			_, err := Pick(cfg, k.Stock, srcs, pools, c.Input.IsBuy(), amts[i])
			var nes *NoExecutableSourceError
			require.ErrorAs(t, err, &nes, c.ID)
			require.Equal(t, stock, nes.ErrorArgs()[0], c.ID)
			found = true
		}
		require.True(t, found, c.ID)
		n++
	}
	require.Equal(t, 5, n)
	t.Logf("NoExecutableSource reverts: %d/5 reproduced (every candidate of the named stock fails)", n)
}

// Ties go to the lowest index (strict comparison); no fixture has a tie, so this duplicates a real
// candidate: identical state gives identical results and the first copy must win, buy and sell.
func TestPickTieGoesToLowestIndex(t *testing.T) {
	c, err := LoadCase(fixtureDir + "/navjit2-buy-1kusd.json")
	require.NoError(t, err)
	cfg := fixtureConfig(c, FinalisedRules)
	k := &c.Pre.Constituents[0]
	srcs, pools := sourcesOf(t, cfg, k)
	amt := c.Probes[0].Amount.Copy()
	for _, buy := range []bool{true, false} {
		// [cand1, cand0, copy of cand1]: index 2 ties index 0 exactly
		s := []Source{srcs[1], srcs[0], srcs[1]}
		p := []*LegPool{pools[1], pools[0], pools[1]}
		pick, err := Pick(cfg, k.Stock, s, p, buy, amt)
		require.NoError(t, err)
		eq(t, pick.Results[0].Result, pick.Results[2].Result)
		require.NotEqual(t, 2, pick.Index, "buy=%v: a tie must not move the pick to a later index", buy)
	}
}
