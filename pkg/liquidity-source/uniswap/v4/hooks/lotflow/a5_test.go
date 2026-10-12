package lotflow

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const a5Dir = "testdata/a5"

func loadA5(t *testing.T) (singles, multis []*Case) {
	t.Helper()
	idx, err := LoadIndex(filepath.Join(a5Dir, "index.json"))
	require.NoError(t, err)
	for _, e := range idx.Cases {
		c, err := LoadCase(filepath.Join(a5Dir, filepath.Base(e.File)))
		require.NoError(t, err, e.ID)
		require.Equal(t, e.ID, c.ID)
		singles = append(singles, c)
	}
	for _, e := range idx.MultiCases {
		c, err := LoadCase(filepath.Join(a5Dir, filepath.Base(strings.Split(e.File, "#")[0])))
		require.NoError(t, err, e.ID)
		require.Equal(t, "multi", c.Shape, e.ID)
		require.Len(t, c.Swaps, e.SwapCount, e.ID)
		multis = append(multis, c)
	}
	return singles, multis
}

// quoteChecked quotes a venue fixture and requires it to equal the chain: exact under the default
// policy, or (Inexact only) exact under the near-stipend-skip enumeration. Returns the matching result.
func quoteChecked(t *testing.T, c *Case, m *Market) (*SwapResult, error) {
	t.Helper()
	cfg := fixtureConfig(c, FinalisedRules)
	res, err := Quote(cfg, m, inputOf(c))
	if mm := checkOutcome(c, res, err); mm != "" {
		require.True(t, res != nil && res.Inexact, "%s: not Inexact and not exact: %s", c.ID, mm)
		skip := *cfg
		skip.NearStipend = NearStipendSkip
		res, err = Quote(&skip, m, inputOf(c))
		require.Empty(t, checkOutcome(c, res, err), "%s: neither enumerated outcome is the chain's", c.ID)
	}
	cq, cerr := QuoteConservative(cfg, m, inputOf(c))
	require.False(t, overQuotes(c, cq, cerr), "%s: conservative over-quote", c.ID)
	return res, err
}

// checkStanding: the owner's receipt and the standing position after the swap (on a revert: the
// pre-swap position, since the revert rolls the lift back).
func checkStanding(t *testing.T, c *Case, res *SwapResult, err error) bool {
	t.Helper()
	pre := c.Pre.Venue.Standing
	if pre.Liquidity.Sign() == 0 {
		require.Nil(t, res.Standing, c.ID)
		return false
	}
	sa := c.Outcome.StandingAfter
	require.NotNil(t, sa, c.ID)
	if err != nil {
		require.Equal(t, pre.Liquidity.String(), sa.Liquidity.String(), c.ID)
		require.Equal(t, pre.Lower, sa.Lower, c.ID)
		return true
	}
	usdgTo, lotTo := res.Standing.ToOwner0, res.Standing.ToOwner1
	if !c.Pre.Venue.UsdgIsCurrency0 {
		usdgTo, lotTo = lotTo, usdgTo
	}
	eq(t, c.Outcome.Owner.UsdgDelta.Copy(), usdgTo, c.ID, "owner usdg")
	eq(t, c.Outcome.Owner.LotDelta.Copy(), lotTo, c.ID, "owner lot")
	if sa.Liquidity.Sign() == 0 {
		require.Nil(t, res.Standing.After, c.ID, "standing deleted")
	} else {
		require.NotNil(t, res.Standing.After, c.ID)
		eq(t, sa.Liquidity.Copy(), res.Standing.After.Liquidity, c.ID, "standing liquidity")
		require.Equal(t, sa.Lower, res.Standing.After.Lower, c.ID)
		require.Equal(t, sa.Upper, res.Standing.After.Upper, c.ID)
	}
	eq(t, c.Post.Venue.SqrtPriceX96.Copy(), res.PostSqrtPriceX96, c.ID, "venue post sqrtPrice")
	require.Equal(t, c.Post.Venue.Tick, res.PostTick, c.ID, "venue post tick")
	return true
}

// A5 single-swap cases (38): tick boundary, fee tiers, TradeTooSmall / OutsideBand boundaries,
// NavGuard (fresh/stale/paused, refused NAV), standing (sizes, NAV drift, dust, rangeLeft, reverts,
// float boundary, LOT-eli live shape): every outcome raw-unit exact; standing cases also match the
// owner's receipt, the standing position after, and the venue's post slot0.
func TestA5Singles(t *testing.T) {
	singles, _ := loadA5(t)
	require.Len(t, singles, 38)
	standing, refusals := 0, 0
	for _, c := range singles {
		res, err := quoteChecked(t, c, marketOf(t, c))
		if checkStanding(t, c, res, err) {
			standing++
		}
		if err != nil {
			refusals++
		}
	}
	require.Equal(t, 12, refusals)
	t.Logf("A5 singles: 38/38 exact (%d refusals with selector+args; %d standing cases with owner take, "+
		"standing after and post slot0)", refusals, standing)
}

// A5 multi-swap cases (5 fixtures, 13 swaps): swap 0 runs on its fixture pre-state; every later swap
// runs on the state THE SIMULATOR produced (pools, venue slot0, standing, PoolManager LOT float), which
// must first equal the fixture's own pre-state for that swap. Plain v3/v4 swaps on constituent pools
// are simulated too. The one-transaction route must equal the sequential outputs.
func TestA5MultiThreaded(t *testing.T) {
	_, multis := loadA5(t)
	require.Len(t, multis, 5)
	swaps, threadedPools := 0, 0
	for _, mc := range multis {
		w := NewWorld()
		var outs []string
		// spot NAV inputs as first seen per basket: a later swap must get its NAV from the price pools
		// the simulator moved, not from the fixture's (already moved) spot values
		firstSpot := map[string][]string{}
		for k, sc := range mc.Swaps {
			id := fmt.Sprintf("%s#%d", mc.ID, k)
			m := marketOf(t, sc)
			lot := strings.ToLower(m.Lot)
			if fs, ok := firstSpot[lot]; ok {
				for i := range m.Constituents {
					m.Constituents[i].Nav.SpotUsd18 = bigS(fs[i])
				}
			} else {
				for _, c := range m.Constituents {
					firstSpot[lot] = append(firstSpot[lot], c.Nav.SpotUsd18.String())
				}
			}
			if k > 0 {
				// the simulator's state must BE the chain's state before this swap
				n, diffs := w.Overlay(m)
				require.Empty(t, diffs, id)
				threadedPools += n
			}
			switch sc.Input.Type {
			case "", "venue":
				res, err := quoteChecked(t, sc, m)
				checkStanding(t, sc, res, err)
				if err == nil {
					w.Absorb(m, res)
					outs = append(outs, res.AmountOut.String())
				}
			case "plainV4", "plainV3":
				cfg := fixtureConfig(sc, FinalisedRules)
				r, err := PlainSwap(cfg, m, sc.Input.Stock, sc.Input.SourceID, sc.Input.ZeroForOne,
					sc.Input.AmountIn.Copy(), sc.Input.SqrtPriceLimitX96.Copy())
				require.NoError(t, err, id)
				eq(t, sc.Outcome.AmountOut.Copy(), r.Got, id, "plain amountOut")
				eq(t, sc.Input.AmountIn.Copy(), r.Paid, id, "plain amountIn")
				w.AbsorbPool(sc.Input.SourceID, m, r)
				outs = append(outs, r.Got.String())
			default:
				t.Fatalf("%s: input type %q", id, sc.Input.Type)
			}
			swaps++
		}
		if mc.Tx != nil {
			require.True(t, mc.Tx.Outcome.MatchesSequential, mc.ID)
			var tx []string
			for _, o := range mc.Tx.Outcome.LegOutputs {
				tx = append(tx, o.String())
			}
			require.Equal(t, tx, outs, mc.ID, "one-tx route == simulated sequential outputs")
		}
	}
	require.Equal(t, 13, swaps)
	t.Logf("A5 multi: %d/13 swaps exact on threaded state (%d pool states carried and equal to the "+
		"fixture's next pre-state)", swaps, threadedPools)
}
