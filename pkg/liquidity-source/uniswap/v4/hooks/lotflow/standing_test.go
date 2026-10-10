package lotflow

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// Standing venue liquidity (NavJitHookV17 `_liftStanding` / `_restoreStanding` / `_addStanding`).
// No P2-FIX fixture has a standing position, so these are synthetic: a real fixture's pre-state plus
// a standing position, with the expected numbers computed by an independent Python implementation of
// the contract arithmetic (the CRITIC-P2 legsim tick maths, not this package). They pin:
//   - lift before quote: the user never trades standing liquidity, so amountOut and treasury are
//     the fixture's own;
//   - restore re-centred on this swap's NAV, half = (upper-lower)/2, one wei held back per side;
//   - lifted = re-added + paid to owner, per currency;
//   - a standing position whose re-add rounds to zero liquidity is deleted (all lifted -> owner);
//   - the LOT paid to owner leaves the PoolManager before `_unwind`'s float check.

func bigS(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(s)
	}
	return v
}

func withStanding(t *testing.T, id string, liq *big.Int, dLo, dHi int) (*Case, *Market) {
	t.Helper()
	c, err := LoadCase(fixtureDir + "/" + id + ".json")
	require.NoError(t, err)
	m := marketOf(t, c)
	t0 := c.Pre.Venue.Tick
	m.Venue.Standing = &StandingPosition{Liquidity: liq, Lower: t0 + dLo, Upper: t0 + dHi}
	return c, m
}

func TestStandingBuyRecentred(t *testing.T) {
	c, m := withStanding(t, "navjit2-buy-1kusd", bigS("1000000000000000000"), -100, 100)
	res, err := Quote(fixtureConfig(c, FinalisedRules), m, inputOf(c))
	require.NoError(t, err)
	// the user's fill and the treasury are untouched by standing capital
	eq(t, c.Outcome.AmountOut.Copy(), res.AmountOut)
	eq(t, c.Outcome.TreasuryUsdgReceived.Copy(), res.Treasury)

	s := res.Standing
	require.NotNil(t, s)
	eq(t, bigS("4967930924"), s.Lifted0) // USDG (currency0)
	eq(t, bigS("5006262828371821162516"), s.Lifted1)
	require.NotNil(t, s.After)
	eq(t, bigS("990756295300015895"), s.After.Liquidity)
	require.Equal(t, 276116, s.After.Lower)
	require.Equal(t, 276316, s.After.Upper)
	eq(t, bigS("4967930923"), s.Added0)
	eq(t, bigS("4914555618981265903070"), s.Added1)
	eq(t, bigS("1"), s.ToOwner0)
	eq(t, bigS("91707209390555259446"), s.ToOwner1)
	for _, x := range [][3]*big.Int{{s.Lifted0, s.Added0, s.ToOwner0}, {s.Lifted1, s.Added1, s.ToOwner1}} {
		eq(t, x[0], new(big.Int).Add(x[1], x[2]), "lifted = re-added + to owner")
	}
	// the venue ends at the new centre, not where the user's swap left it
	eq(t, bigS("78801400613099849516437902027549694"), res.PostSqrtPriceX96)
	require.Equal(t, 276216, res.PostTick)
}

func TestStandingSellRecentred(t *testing.T) {
	c, m := withStanding(t, "navjit2-sell-10usd-payafter", bigS("1000000000000000000"), -100, 100)
	res, err := Quote(fixtureConfig(c, FinalisedRules), m, inputOf(c))
	require.NoError(t, err)
	eq(t, c.Outcome.AmountOut.Copy(), res.AmountOut)
	eq(t, c.Outcome.TreasuryUsdgReceived.Copy(), res.Treasury)
	s := res.Standing
	eq(t, bigS("4989372938"), s.Lifted0)
	eq(t, bigS("4985037119214764864039"), s.Lifted1)
	eq(t, bigS("994783769461323929"), s.After.Liquidity)
	eq(t, bigS("4989372937"), s.Added0)
	eq(t, bigS("4933300117174278419771"), s.Added1)
	eq(t, bigS("1"), s.ToOwner0)
	eq(t, bigS("51737002040486444268"), s.ToOwner1)
}

// Re-add rounds to zero liquidity -> the standing record is deleted and everything lifted goes to the
// owner: a dust position (liq 1 lifts 0 USDG; fork-proven by A5 amznaapl-buy-1kusd-standing-dust) and a
// one-sided one (the price outside its range lifts only USDG, so the re-add gets min(l0, l1(0)) = 0).
// The one-sided case is a contract-branch unit test only: the fixture generation showed it is not
// reachable through the hook, because every swap ends with the venue re-centred inside the standing
// range and seedStanding centres on NAV; A5 `rangeLeft` proves the reachable behaviour (re-centre).
func TestStandingDeletedWhenReaddIsZero(t *testing.T) {
	for _, tc := range []struct {
		name         string
		liq          string
		dLo, dHi     int
		lift0, lift1 string
	}{
		{"dust", "1", -100, 100, "0", "5006"},
		{"one-sided", "1000000000000000000", 150, 350, "9928813040", "0"},
	} {
		c, m := withStanding(t, "navjit2-buy-1kusd", bigS(tc.liq), tc.dLo, tc.dHi)
		res, err := Quote(fixtureConfig(c, FinalisedRules), m, inputOf(c))
		require.NoError(t, err, tc.name)
		eq(t, c.Outcome.AmountOut.Copy(), res.AmountOut, tc.name)
		s := res.Standing
		eq(t, bigS(tc.lift0), s.Lifted0, tc.name)
		eq(t, bigS(tc.lift1), s.Lifted1, tc.name)
		require.Nil(t, s.After, tc.name)
		eq(t, s.Lifted0, s.ToOwner0, tc.name)
		eq(t, s.Lifted1, s.ToOwner1, tc.name)
	}
}

// The standing leftover LOT is taken from the PoolManager to the owner before `_unwind` checks the
// float: a pay-after sell that fits the float without standing fails by exactly that amount with it.
func TestStandingLeftoverReducesSellFloat(t *testing.T) {
	c, m := withStanding(t, "navjit2-sell-10usd-payafter", bigS("1000000000000000000"), -100, 100)
	cfg := fixtureConfig(c, FinalisedRules)
	res, err := Quote(cfg, m, inputOf(c))
	require.NoError(t, err)
	needed := res.Venue.LotLeftOnHook(false, res.Inventory)
	leftLot := res.Standing.ToOwner1 // LOT is currency1

	// float = needed + leftover: passes; one wei less: SellExceedsFloat(needed-1, needed)
	m.PmLotBalance = new(big.Int).Add(needed, leftLot)
	_, err = Quote(cfg, m, inputOf(c))
	require.NoError(t, err)
	m.PmLotBalance.Sub(m.PmLotBalance, big.NewInt(1))
	_, err = Quote(cfg, m, inputOf(c))
	var sef *SellExceedsFloatError
	require.ErrorAs(t, err, &sef)
	eq(t, new(big.Int).Sub(needed, big.NewInt(1)), sef.Float)
	eq(t, needed, sef.Needed)
	var se *SwapError
	require.True(t, errors.As(err, &se))
	require.Equal(t, PhaseAfterSwap, se.Phase)
}

// A swap that reverts (band) never restores: the revert rolls the lift back, so the standing
// position is unchanged; the simulator reports no restore.
func TestStandingUntouchedOnRevert(t *testing.T) {
	c, m := withStanding(t, "navjit2-buy-1raw", bigS("1000000000000000000"), -100, 100)
	res, err := Quote(fixtureConfig(c, FinalisedRules), m, inputOf(c))
	var ob *OutsideBandError
	require.ErrorAs(t, err, &ob)
	require.Nil(t, res.Standing.After)
	require.Nil(t, res.Standing.ToOwner0)
}
