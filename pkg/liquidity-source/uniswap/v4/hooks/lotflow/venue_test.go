package lotflow

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------- fixture -> model adapters

func navInputsOf(c *Case) ([]NavConstituent, NavGuardParams, uint64) {
	cs := make([]NavConstituent, len(c.Pre.Constituents))
	for i, k := range c.Pre.Constituents {
		cs[i] = NavConstituent{
			Unit:          k.Unit.Copy(),
			SpotUsd18:     k.Nav.SpotUsd18.Copy(),
			StockDecimals: k.Registry.StockDecimals,
			OraclePaused:  k.Nav.OraclePaused == "true",
		}
		if f, ok := new(big.Int).SetString(strings.TrimPrefix(k.Nav.Feed, "0x"), 16); ok && f.Sign() != 0 {
			cs[i].Feed = &FeedState{
				Answer:    k.Nav.FeedAnswer.Copy(),
				Decimals:  k.Nav.FeedDecimals,
				UpdatedAt: k.Nav.FeedUpdatedAt.Uint64(),
			}
		}
	}
	p := c.Pre.Nav.Params
	return cs, NavGuardParams{
		NavTolBps:      uint64(p.NavTolBps),
		StaleTolBps:    uint64(p.StaleTolBps),
		MaxFeedAge:     uint64(p.MaxFeedAge),
		MinCoverageBps: uint64(p.MinCoverageBps),
	}, c.Pin.BlockTimestamp.Uint64()
}

func venueOf(c *Case) *Venue {
	hp := c.Pre.Hook.Params
	return &Venue{
		UsdgIs0:      c.Pre.Venue.UsdgIsCurrency0,
		UsdgDecimals: c.Pre.Hook.UsdgDecimals,
		SqrtPriceX96: c.Pre.Venue.SqrtPriceX96.Copy(),
		Tick:         c.Pre.Venue.Tick,
		Params: Params{
			BuySpreadBps:  uint64(hp.BuySpreadBps),
			SellSpreadBps: uint64(hp.SellSpreadBps),
			BandBps:       uint64(hp.BandBps),
			SizeBufferBps: uint64(hp.SizeBufferBps),
			WidthTicks:    hp.WidthTicks,
		},
	}
}

func unitsOf(c *Case) []*big.Int {
	u := make([]*big.Int, len(c.Pre.Constituents))
	for i, k := range c.Pre.Constituents {
		u[i] = k.Unit.Copy()
	}
	return u
}

func sumLegs(c *Case, buy bool, field func(*DerivedLeg) *big.Int) *big.Int {
	s := new(big.Int)
	for i := range c.Derived.Legs {
		if c.Derived.Legs[i].Buy == buy {
			s.Add(s, field(&c.Derived.Legs[i]))
		}
	}
	return s
}

func paidOf(l *DerivedLeg) *big.Int { return l.Paid.Copy() }
func gotOf(l *DerivedLeg) *big.Int  { return l.Got.Copy() }

// legAmounts returns, per constituent index, the stock amount of the case's legs in one direction
// (buy: got, sell: paid), in leg order.
func legAmounts(c *Case, buy bool) []*big.Int {
	var out []*big.Int
	for i := range c.Derived.Legs {
		l := &c.Derived.Legs[i]
		if l.Buy != buy {
			continue
		}
		if buy {
			out = append(out, l.Got.Copy())
		} else {
			out = append(out, l.Paid.Copy())
		}
	}
	return out
}

func nonZero(xs []*big.Int) []*big.Int {
	var out []*big.Int
	for _, x := range xs {
		if x.Sign() != 0 {
			out = append(out, x)
		}
	}
	return out
}

func eq(t *testing.T, want *big.Int, got *big.Int, msg ...any) {
	t.Helper()
	require.Equal(t, want.String(), got.String(), msg...)
}

// ---------------------------------------------------------------- NAV

// Every fixture: SpotLib.price18 from the registry price pool reproduces nav.spotUsd18, and
// NavGuard.checkedNavPerUnit18 over the constituents reproduces pre.nav.checkedNavPerUnit18, exactly.
func TestNavMatchesFixtures(t *testing.T) {
	_, cases := loadAll(t)
	spots, navs := 0, 0
	for _, c := range cases {
		for _, k := range c.Pre.Constituents {
			r := k.Registry
			sp, err := SpotUsd18(r.Pools[r.PriceIndex].SqrtPriceX96.Copy(), r.StockIsCurrency0, r.StockDecimals,
				r.StableDecimals)
			require.NoError(t, err, c.ID)
			eq(t, k.Nav.SpotUsd18.Copy(), sp, c.ID, k.Symbol)
			spots++
		}
		cs, p, ts := navInputsOf(c)
		nav, err := CheckedNavPerUnit18(cs, p, ts)
		require.NoError(t, err, c.ID)
		eq(t, c.Pre.Nav.CheckedNavPerUnit18.Copy(), nav, c.ID)
		navs++
	}
	require.Equal(t, 85, navs)
	t.Logf("NAV: %d/%d spotUsd18 exact, %d/85 checkedNavPerUnit18 exact", spots, spots, navs)
}

// NavGuard branches the fixtures never reach (no stale, paused, missing or bad feed in any of them).
func TestNavGuardBranches(t *testing.T) {
	e18 := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	mk := func(spot, feed int64, updated uint64, paused bool) NavConstituent {
		c := NavConstituent{Unit: new(big.Int).Set(e18), SpotUsd18: new(big.Int).Mul(big.NewInt(spot), e18),
			StockDecimals: 18, OraclePaused: paused}
		if feed != 0 {
			c.Feed = &FeedState{Answer: big.NewInt(feed * 1e8), Decimals: 8, UpdatedAt: updated}
		}
		return c
	}
	p := NavGuardParams{NavTolBps: 100, StaleTolBps: 300, MaxFeedAge: 100, MinCoverageBps: 0}

	// 2% apart: outside navTol, inside staleTol only when a feed is stale or the stock is paused
	_, err := CheckedNavPerUnit18([]NavConstituent{mk(102, 100, 1000, false)}, p, 1000)
	var nm *NavMismatchError
	require.ErrorAs(t, err, &nm)
	nav, err := CheckedNavPerUnit18([]NavConstituent{mk(102, 100, 1000, false)}, p, 1101) // updatedAt+age < ts
	require.NoError(t, err)
	eq(t, new(big.Int).Mul(big.NewInt(102), e18), nav)
	_, err = CheckedNavPerUnit18([]NavConstituent{mk(102, 100, 1000, false)}, p, 1100) // not yet stale
	require.ErrorAs(t, err, &nm)
	_, err = CheckedNavPerUnit18([]NavConstituent{mk(102, 100, 1000, true)}, p, 1000) // paused = stale
	require.NoError(t, err)

	// no feed: the spot value stands in for Chainlink and counts as uncovered
	_, err = CheckedNavPerUnit18([]NavConstituent{mk(50, 0, 0, false), mk(50, 50, 1000, false)},
		NavGuardParams{NavTolBps: 100, StaleTolBps: 300, MaxFeedAge: 100, MinCoverageBps: 5001}, 1000)
	var cov *FeedCoverageTooLowError
	require.ErrorAs(t, err, &cov)
	require.Equal(t, uint64(5000), cov.CoveredBps)

	// bad answer
	bad := mk(1, 1, 1000, false)
	bad.Feed.Answer = big.NewInt(0)
	_, err = CheckedNavPerUnit18([]NavConstituent{bad}, p, 1000)
	require.True(t, errors.Is(err, ErrBadFeedAnswer))

	// zero NAV
	_, err = CheckedNavPerUnit18([]NavConstituent{mk(0, 0, 0, false)}, p, 1000)
	require.True(t, errors.Is(err, ErrZeroNav))
}

// ---------------------------------------------------------------- venue arithmetic (P2-FIX §5.3)

// A2 acceptance. For every filling fixture, driven from pre-state plus the executed legs' amounts
// (the legs themselves are A3/A4): inventory, basket mint/redeem amounts, edge price, JIT range,
// liquidity, the reposition, the user's venue swap (amountOut, end price/tick), the band check and
// the treasury surplus are all raw-unit equal to the fixture. Counted per side: 27 buys, 28 sells (D-B set).
func TestVenueArithmeticFills(t *testing.T) {
	_, cases := loadAll(t)
	buys, sells := 0, 0
	for _, c := range cases {
		if !c.IsFill() {
			continue
		}
		v := venueOf(c)
		nav := c.Pre.Nav.CheckedNavPerUnit18.Copy()
		a := c.Input.AmountIn.Copy()
		buy := c.Input.IsBuy()
		units := unitsOf(c)
		require.Len(t, c.Derived.VenueModifyLiquidity, 2, c.ID)
		ml := c.Derived.VenueModifyLiquidity[0]
		require.Len(t, c.Derived.VenueSwaps, 2, c.ID)
		repo, user := c.Derived.VenueSwaps[0], c.Derived.VenueSwaps[1]

		var price18, inventory *big.Int
		var err error
		if buy {
			inventory, err = v.BuyInventory(a, nav)
			require.NoError(t, err, c.ID)
			eq(t, c.Derived.BasketMinted[0].Copy(), inventory, c.ID, "inventory")
			// the buy legs deliver exactly ceil(unit*inventory/1e18) of each constituent
			require.Equal(t, fmtList(nonZero(MintAmounts(units, inventory))), fmtList(legAmounts(c, true)), c.ID)
			price18 = v.BuyPrice18(sumLegs(c, true, paidOf), inventory)
		} else {
			price18 = v.SellPrice18(nav)
			inventory, err = v.SellInventory(a, price18)
			require.NoError(t, err, c.ID)
		}

		pos, err := v.Place(price18, inventory, buy)
		require.NoError(t, err, c.ID)
		require.Equal(t, ml.TickLower, pos.Lower, c.ID)
		require.Equal(t, ml.TickUpper, pos.Upper, c.ID)
		eq(t, ml.LiquidityDelta.Copy(), pos.Liquidity, c.ID, "liquidity")
		eq(t, repo.SqrtPriceX96.Copy(), pos.SqrtPriceX96, c.ID, "reposition sqrtPrice")
		require.Equal(t, repo.Tick, pos.Tick, c.ID, "reposition tick")

		sw, err := v.Swap(pos, buy, a, c.Input.SqrtPriceLimitX96.Copy())
		require.NoError(t, err, c.ID)
		eq(t, c.Outcome.AmountOut.Copy(), sw.AmountOut, c.ID, "amountOut")
		eq(t, a, sw.AmountIn, c.ID, "amountIn consumed")
		eq(t, user.SqrtPriceX96.Copy(), sw.SqrtPriceX96, c.ID, "venue end sqrtPrice")
		require.Equal(t, user.Tick, sw.Tick, c.ID, "venue end tick")
		eq(t, c.Post.Venue.SqrtPriceX96.Copy(), sw.SqrtPriceX96, c.ID)
		require.NoError(t, v.RequireInBand(sw.SqrtPriceX96, nav), c.ID)

		// unwind: the LOT the swap left on the hook is redeemed; the redeem pays floor(unit*x/1e18)
		lotLeft := sw.LotLeftOnHook(buy, inventory)
		eq(t, c.Derived.BasketRedeemed[0].Copy(), lotLeft, c.ID, "redeemed")
		require.Equal(t, fmtList(nonZero(RedeemAmounts(units, lotLeft))), fmtList(legAmounts(c, false)), c.ID)

		// treasury = the hook's USDG delta: USDG the JIT position returned net of what it was funded
		// with, minus buy-leg cost, plus sell-leg proceeds
		treasury := sw.HookUsdgDelta(buy)
		treasury.Sub(treasury, sumLegs(c, true, paidOf))
		treasury.Add(treasury, sumLegs(c, false, gotOf))
		eq(t, c.Outcome.TreasuryUsdgReceived.Copy(), treasury, c.ID, "treasury")

		if buy {
			buys++
		} else {
			sells++
		}
	}
	require.Equal(t, 27, buys)
	require.Equal(t, 28, sells)
	t.Logf("venue arithmetic exact: buys %d/27, sells %d/28", buys, sells)
}

// The venue-level refusals among the reverting fixtures, driven from pre-state plus the legs the
// fixture recorded before the revert: *-sell-1wei TradeTooSmall at sizing; tiny buys and the metal
// over-max buy OutsideBand(px, nav) after the user's swap; *-sell-overmax NotProfitable(delta) at
// unwind; pay-after sells above the PoolManager float SellExceedsFloat(float, needed). Only the four
// NoExecutableSource cases need the leg layer (A3/A4).
func TestVenueArithmeticRefusals(t *testing.T) {
	_, cases := loadAll(t)
	counts := map[string]int{}
	for _, c := range cases {
		if c.IsFill() {
			continue
		}
		v := venueOf(c)
		nav := c.Pre.Nav.CheckedNavPerUnit18.Copy()
		a := c.Input.AmountIn.Copy()
		buy := c.Input.IsBuy()
		want := c.Derived.RevertInnermost
		if want.Name == "NoExecutableSource" {
			continue
		}
		var got error
		func() {
			var price18, inv *big.Int
			var err error
			if buy {
				if inv, got = v.BuyInventory(a, nav); got != nil {
					return
				}
				price18 = v.BuyPrice18(sumLegs(c, true, paidOf), inv)
			} else {
				price18 = v.SellPrice18(nav)
				if inv, got = v.SellInventory(a, price18); got != nil {
					return
				}
			}
			pos, err := v.Place(price18, inv, buy)
			if got = err; got != nil {
				return
			}
			sw, err := v.Swap(pos, buy, a, c.Input.SqrtPriceLimitX96.Copy())
			require.NoError(t, err, c.ID)
			if got = v.RequireInBand(sw.SqrtPriceX96, nav); got != nil {
				return
			}
			if !buy {
				float := c.Pre.PM.LotBalance.Copy()
				if c.Input.Settlement == "prefund" {
					float.Add(float, a) // the router settled the seller's LOT before the swap
				}
				if got = RequireFloat(float, sw.LotLeftOnHook(false, inv)); got != nil {
					return
				}
			}
			delta := sw.HookUsdgDelta(buy)
			delta.Sub(delta, sumLegs(c, true, paidOf))
			delta.Add(delta, sumLegs(c, false, gotOf))
			got = RequireProfitable(delta)
		}()
		require.Error(t, got, c.ID)
		var named NamedError
		require.True(t, errors.As(got, &named), c.ID)
		require.Equal(t, want.Name, named.ErrorName(), c.ID)
		require.Equal(t, fmtArgs(want.Args), fmtArgs(named.ErrorArgs()), c.ID)
		counts[want.Name]++
	}
	require.Equal(t, map[string]int{"TradeTooSmall": 5, "OutsideBand": 13, "NotProfitable": 5,
		"SellExceedsFloat": 2}, counts)
	t.Logf("venue refusals exact (name + args): %v = 25/30 (the other 5 are NoExecutableSource, leg layer)", counts)
}

func fmtList(xs []*big.Int) string {
	s := make([]string, len(xs))
	for i, x := range xs {
		s[i] = x.String()
	}
	return strings.Join(s, ",")
}

func fmtArgs(a []string) string { return strings.Join(a, ",") }
