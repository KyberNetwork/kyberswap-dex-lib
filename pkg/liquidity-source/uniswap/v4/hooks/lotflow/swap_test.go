package lotflow

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func marketOf(t *testing.T, c *Case) *Market {
	t.Helper()
	navCs, navP, ts := navInputsOf(c)
	m := &Market{Lot: c.Pre.Lot.Address, Venue: venueOf(c), NavParams: navP, Timestamp: ts,
		PmLotBalance: c.Pre.PM.LotBalance.Copy()}
	if st := c.Pre.Venue.Standing; st.Liquidity.Sign() != 0 {
		m.Venue.Standing = &StandingPosition{Liquidity: st.Liquidity.Copy(), Lower: st.Lower, Upper: st.Upper}
	}
	for i := range c.Pre.Constituents {
		k := &c.Pre.Constituents[i]
		ok, venues := bookOf(k)
		mc := MarketConstituent{Stock: k.Stock, Unit: k.Unit.Copy(), Nav: navCs[i], Registry: registryOf(k),
			VenuesOK: ok, Venues: venues, Pools: map[string]*LegPool{}, StableDecimals: k.Registry.StableDecimals}
		for j := range k.Candidates {
			mc.Pools[strings.ToLower(k.Candidates[j].SourceID)] = legPoolOf(&k.Candidates[j])
		}
		m.Constituents = append(m.Constituents, mc)
	}
	if q := c.Pre.Mq; q != nil {
		m.Mq = mqMarketOf(c)
		for i := range c.Pre.Constituents {
			k := &c.Pre.Constituents[i]
			if !k.Registry.IsWethToken {
				continue
			}
			cand := &k.Candidates[0]
			mc := &m.Constituents[i]
			mc.Mq = &MqToken{Pool: cand.Pool, Factory: cand.Factory, TokenIs0: addrLess(k.Stock, q.Weth),
				TokenDecimals: k.Registry.StockDecimals, Twap: twapFor(c, cand.Pool)}
		}
	}
	return m
}

func inputOf(c *Case) SwapInput {
	return SwapInput{BuyLot: c.Input.IsBuy(), AmountIn: c.Input.AmountIn.Copy(),
		SqrtPriceLimitX96: c.Input.SqrtPriceLimitX96.Copy(), Prefund: c.Input.Settlement == "prefund"}
}

// checkOutcome compares a simulated swap with the fixture's recorded outcome, raw-unit exact: on a
// fill amountOut, treasury, minted/redeemed LOT and every LegExecuted (constituent, side, source,
// paid, got, mode); on a revert the hook phase, innermost selector, name and args. "" = identical.
func checkOutcome(c *Case, res *SwapResult, err error) string {
	if c.IsFill() {
		if err != nil {
			return fmt.Sprintf("chain filled, simulator: %v", err)
		}
		if res.AmountOut.Cmp(c.Outcome.AmountOut.Copy()) != 0 {
			return fmt.Sprintf("amountOut %s != %s", res.AmountOut, c.Outcome.AmountOut.String())
		}
		if res.Treasury.Cmp(c.Outcome.TreasuryUsdgReceived.Copy()) != 0 {
			return fmt.Sprintf("treasury %s != %s", res.Treasury, c.Outcome.TreasuryUsdgReceived.String())
		}
		if len(res.Legs) != len(c.Derived.Legs) {
			return fmt.Sprintf("%d legs != %d", len(res.Legs), len(c.Derived.Legs))
		}
		for j, l := range c.Derived.Legs {
			g := res.Legs[j]
			if l.Buy != g.Buy || l.ConstituentIndex != g.ConstituentIndex ||
				!strings.EqualFold(l.Source, g.Source.ID) || l.Paid.Cmp(g.Paid) != 0 || l.Got.Cmp(g.Got) != 0 {
				return fmt.Sprintf("leg %d: %+v != sim %s %s/%s", j, l, g.Source.ID, g.Paid, g.Got)
			}
		}
		for _, pr := range c.Probes {
			want := "probed"
			if strings.HasPrefix(pr.Mode, "single") {
				want = "single"
			} else if strings.HasPrefix(pr.Mode, "memo") {
				want = "memo"
			}
			for _, l := range res.Legs {
				if l.ConstituentIndex == int(pr.ConstituentIndex) && l.Buy == pr.Buy && l.Mode != want {
					return fmt.Sprintf("constituent %d buy=%v mode %s != %s", l.ConstituentIndex, l.Buy, l.Mode, want)
				}
			}
		}
		if c.Input.IsBuy() && c.Derived.BasketMinted[0].Cmp(res.Inventory) != 0 {
			return "minted LOT differs"
		}
		if len(c.Derived.BasketRedeemed) > 0 && c.Derived.BasketRedeemed[0].Cmp(res.LotRedeemed) != 0 {
			return "redeemed LOT differs"
		}
		return ""
	}
	var se *SwapError
	if !errors.As(err, &se) {
		return fmt.Sprintf("chain reverted %s, simulator: %v", c.Derived.RevertInnermost.Name, err)
	}
	if w := c.Derived.Revert.Wrapped; w == nil || w.HookSelector != se.Phase {
		return "hook phase differs"
	}
	inner := c.Derived.RevertInnermost
	var named NamedError
	if Selector(err) != inner.Selector || !errors.As(err, &named) || named.ErrorName() != inner.Name ||
		fmtArgs(inner.Args) != fmtArgs(named.ErrorArgs()) {
		return fmt.Sprintf("revert %v != %s(%v)", err, inner.Name, inner.Args)
	}
	return ""
}

// overQuotes reports whether a quote promises more than the chain delivered: a fill the chain
// reverted, or a larger amountOut.
func overQuotes(c *Case, res *SwapResult, err error) bool {
	if err != nil {
		return false
	}
	return !c.IsFill() || res.AmountOut.Cmp(c.Outcome.AmountOut.Copy()) > 0
}

// A4 acceptance, on the D-B base set: the whole swap from pre-state alone. Every case whose quote is
// exact (not Inexact) equals the chain raw-unit exactly (checkOutcome). A case the simulator flags
// Inexact (a probe within the gas envelope of PROBE_GAS, whose verdict depends on the transaction's
// warm/cold access set, F-FIXTURES §1) must equal EXACTLY one of the two outcomes it enumerates:
// near-stipend probes fill (NearStipendFill) or skip (NearStipendSkip). QuoteConservative never
// over-quotes the chain.
func TestWholeSwap(t *testing.T) {
	_, cases := loadAll(t)
	fills, reverts := 0, 0
	var inexact, bySkip []string
	for _, c := range cases {
		cfg := fixtureConfig(c, FinalisedRules)
		res, err := Quote(cfg, marketOf(t, c), inputOf(c))
		mm := checkOutcome(c, res, err)
		if res != nil && res.Inexact {
			inexact = append(inexact, c.ID)
		}
		if mm != "" {
			require.True(t, res != nil && res.Inexact, "%s: not Inexact and not exact: %s", c.ID, mm)
			skip := *cfg
			skip.NearStipend = NearStipendSkip
			alt, altErr := Quote(&skip, marketOf(t, c), inputOf(c))
			require.Empty(t, checkOutcome(c, alt, altErr), "%s: neither enumerated outcome is the chain's "+
				"(fill policy: %s)", c.ID, mm)
			bySkip = append(bySkip, c.ID)
		}
		cq, cerr := QuoteConservative(cfg, marketOf(t, c), inputOf(c))
		require.False(t, overQuotes(c, cq, cerr), "%s: conservative quote over-quotes", c.ID)
		if c.IsFill() {
			fills++
		} else {
			reverts++
		}
	}
	require.Equal(t, 55, fills)
	require.Equal(t, 30, reverts)
	t.Logf("whole swap: %d fills + %d reverts; exact under the default policy %d/85; Inexact-flagged %v; "+
		"matched by the near-stipend-skip outcome %v; conservative over-quotes 0", fills, reverts,
		85-len(bySkip), inexact, bySkip)
}
