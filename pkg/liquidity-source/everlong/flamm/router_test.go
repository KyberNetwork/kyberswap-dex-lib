package everlongflamm

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestMMLiveViews: the deployed c104 router / account / Morpho views at block 51317000 and warped to +1s, +1h,
// +1d, +30d, +365d (accrual and IRM adaptation), with managed figures below the position, a binding rate
// ceiling (the non-monotone fundingCeiling branch) and an unreadable IRM inside and beyond the grace.
func TestMMLiveViews(t *testing.T) {
	t.Parallel()
	var fx struct {
		Snaps []struct {
			Name string   `json:"name"`
			Snap mmFxSnap `json:"snap"`
		} `json:"snaps"`
	}
	mmReadFixture(t, "mm_live_views.json.gz", &fx)
	require.Equal(t, 11, len(fx.Snaps))
	for _, s := range fx.Snaps {
		r := s.Snap.Router.state()
		p := s.Snap.Pool.state()
		mmCheckViews(t, s.Name, r, s.Snap.Prices, s.Snap.Timestamp, s.Snap.Views)
		mmCheckPoolViews(t, s.Name, r, p, s.Snap.Timestamp, s.Snap.Pool)
		mmCheckLens(t, s.Name, &s.Snap)
	}
}

// TestMMRealSell replays the only live swap (block 51302916, tx 0x46c3cd72…: 15000 sats in, 11301759 USDC out)
// through the settlement port from the in-block pre-state and matches the chain's post-state and views.
func TestMMRealSell(t *testing.T) {
	t.Parallel()
	var fx struct {
		PreBlock mmFxSnap `json:"preBlock"`
		Pre      mmFxSnap `json:"pre"`
		Post     mmFxSnap `json:"post"`
	}
	mmReadFixture(t, "mm_real_sell.json.gz", &fx)
	for name, s := range map[string]*mmFxSnap{"preBlock": &fx.PreBlock, "pre": &fx.Pre, "post": &fx.Post} {
		mmCheckViews(t, name, s.Router.state(), s.Prices, s.Timestamp, s.Views)
		mmCheckPoolViews(t, name, s.Router.state(), s.Pool.state(), s.Timestamp, s.Pool)
		mmCheckLens(t, name, s)
	}
	r, p := fx.Pre.Router.state(), fx.Pre.Pool.state()
	used, net := uint256FromU64(15000), uint256FromU64(11301759)
	require.NoError(t, mmSettleSell(p, r, 0, &p.PriceWad[0], used, net, fx.Pre.Timestamp))
	mmCheckState(t, "realSell", r, p, &fx.Post)
}

// TestMMLiveSettle executes the fork's settlement sequence on the live pool: sells that borrow and post, a buy
// that repays and reclaims while indebted, a buy that repays everything and lends the surplus, a sell that
// withdraws the supply before borrowing, a debt-free reclaim, each from the chain's pre-state.
func TestMMLiveSettle(t *testing.T) {
	t.Parallel()
	var fx struct {
		Steps []struct {
			Name string     `json:"name"`
			Sell bool       `json:"sell"`
			Ok   bool       `json:"ok"`
			Used uint256Dec `json:"used"`
			Out  uint256Dec `json:"out"`
			Pre  mmFxSnap   `json:"pre"`
			Post mmFxSnap   `json:"post"`
		} `json:"steps"`
	}
	mmReadFixture(t, "mm_live_settle.json.gz", &fx)
	settled := 0
	for _, s := range fx.Steps {
		r, p := s.Pre.Router.state(), s.Pre.Pool.state()
		now := s.Pre.Timestamp
		if s.Ok {
			var err error
			if s.Sell {
				err = mmSettleSell(p, r, 0, &p.PriceWad[0], &s.Used.Int, &s.Out.Int, now)
			} else {
				err = mmSettleBuy(p, r, 0, &s.Used.Int, &s.Out.Int, now)
			}
			require.NoError(t, err, s.Name)
			settled++
		}
		mmCheckState(t, s.Name, r, p, &s.Post)
		mmCheckViews(t, s.Name, s.Post.Router.state(), s.Post.Prices, s.Post.Timestamp, s.Post.Views)
		mmCheckPoolViews(t, s.Name, s.Post.Router.state(), s.Post.Pool.state(), s.Post.Timestamp, s.Post.Pool)
		mmCheckLens(t, s.Name, &s.Post)
	}
	require.GreaterOrEqual(t, settled, 6)
}

type mmFxOp struct {
	Op               string        `json:"op"`
	Idx              uint8         `json:"idx"`
	Assets           uint256.Int   `json:"assets"`
	CollateralIn     uint256.Int   `json:"collateralIn"`
	PriceWad         uint256.Int   `json:"priceWad"`
	PriceWads        []uint256.Int `json:"priceWads"`
	N                uint8         `json:"n"`
	ID               uint16        `json:"id"`
	DebtCap          uint256.Int   `json:"debtCap"`
	SupplyCap        uint256.Int   `json:"supplyCap"`
	MaxBorrowRateWad uint256.Int   `json:"maxBorrowRateWad"`
	Proportional     bool          `json:"proportional"`
}

// mmApplyOp runs one pool-scoped Router call on a clone, committing only on success, and returns its words.
func mmApplyOp(r *mmRouter, op *mmFxOp, now uint64) ([]uint256.Int, error) {
	c := r.clone()
	var out []uint256.Int
	var err error
	switch op.Op {
	case "fund":
		var w, b, p uint256.Int
		w, b, p, err = c.fund(op.Idx, &op.Assets, &op.CollateralIn, &op.PriceWad, now)
		out = []uint256.Int{w, b, p}
	case "repayCascade":
		var x uint256.Int
		x, err = c.repayCascade(op.Idx, &op.Assets, now)
		out = []uint256.Int{x}
	case "supplyCascade":
		var x uint256.Int
		x, err = c.supplyCascade(op.Idx, &op.Assets, now)
		out = []uint256.Int{x}
	case "reclaim", "reclaimBestEffort":
		var x uint256.Int
		x, err = c.reclaim(&op.Assets, op.PriceWads, op.Op == "reclaim", now)
		out = []uint256.Int{x}
	case "supply":
		var x uint256.Int
		x, err = c.supply(op.ID, &op.Assets, now)
		out = []uint256.Int{x}
	case "withdrawSupplied":
		var x uint256.Int
		x, err = c.withdrawSuppliedEntry(op.ID, &op.Assets, now)
		out = []uint256.Int{x}
	case "repay":
		var x uint256.Int
		x, err = c.repay(op.ID, &op.Assets, now)
		out = []uint256.Int{x}
	case "postCollateral":
		err = c.postCollateral(op.ID, &op.Assets)
	case "borrow":
		err = c.borrow(op.ID, &op.Assets, &op.PriceWad, now)
	case "withdrawCollateral":
		err = c.withdrawCollateral(op.ID, &op.Assets, &op.PriceWad, op.Proportional, now)
	case "setMaxDrawnAssets":
		if op.N == 0 || int(op.N) > len(c.Loans) {
			return nil, ErrInvalidConfig
		}
		c.MaxDrawnAssets = op.N
	case "setVenueCaps":
		v := &c.Venues[op.ID]
		v.DebtCap, v.SupplyCap, v.MaxBorrowRateWad = op.DebtCap, op.SupplyCap, op.MaxBorrowRateWad
	default:
		return nil, fmt.Errorf("unknown op %s", op.Op)
	}
	if err != nil {
		return nil, err
	}
	*r = *c
	return out, nil
}

// TestMMMultiVenue drives the DEPLOYED router bytecode (etched on a fork, fresh storage) over three venues: two
// USDC markets (the live 86% market and a seeded 77% market sharing its oracle and IRM) and a WETH market behind
// a constant oracle, with venue debt / supply caps, distinct borrow / supply / withdraw / repay orders, the drawn
// set, a binding rate ceiling on the small market, and an IRM outage on one venue inside and beyond the grace.
// Every call is replayed from the chain's pre-state; post-state and every view must match, reverts by selector
// (and InsufficientLiquidity's shortfall by the plan's remainder).
func TestMMMultiVenue(t *testing.T) {
	t.Parallel()
	var fx struct {
		Setup mmFxSnap `json:"setup"`
		Steps []struct {
			Name string   `json:"name"`
			Args *mmFxOp  `json:"args"`
			Ok   bool     `json:"ok"`
			Ret  string   `json:"ret"`
			Err  string   `json:"err"`
			Pre  mmFxSnap `json:"pre"`
			Post mmFxSnap `json:"post"`
		} `json:"steps"`
	}
	mmReadFixture(t, "mm_multi_venue.json.gz", &fx)
	mmCheckViews(t, "setup", fx.Setup.Router.state(), mmMultiPrices(&fx.Setup), fx.Setup.Timestamp, fx.Setup.Views)
	ops, fails := 0, 0
	// transient storage outlives each call: the whole generator test is one transaction
	var transient map[uint16]mmRepaySnapshot
	for _, s := range fx.Steps {
		if s.Args == nil {
			continue
		}
		r := s.Pre.Router.state()
		r.TransientRepay = transient
		now := s.Pre.Timestamp
		out, err := mmApplyOp(r, s.Args, now)
		if s.Ok {
			require.NoError(t, err, s.Name)
			raw, derr := hex.DecodeString(strings.TrimPrefix(s.Ret, "0x"))
			require.NoError(t, derr)
			require.Equal(t, len(out)*32, len(raw), s.Name)
			for i := range out {
				want := new(uint256.Int).SetBytes(raw[i*32 : i*32+32])
				mmEq(t, want, &out[i], "%s ret[%d]", s.Name, i)
			}
			ops++
		} else {
			require.ErrorIs(t, err, mmRevertError(t, s.Err), s.Name)
			if strings.HasPrefix(s.Err, "0xc730333f") && s.Args.Op == "fund" {
				plan, perr := s.Pre.Router.state().buildPlan(s.Args.Idx, &s.Args.Assets, &s.Args.CollateralIn, &s.Args.PriceWad, now)
				require.NoError(t, perr)
				raw, _ := hex.DecodeString(strings.TrimPrefix(s.Err, "0x"))
				mmEq(t, new(uint256.Int).SetBytes(raw[4:36]), &plan.Remaining, "%s shortfall", s.Name)
			}
			fails++
		}
		transient = r.TransientRepay
		mmCheckState(t, s.Name, r, nil, &s.Post)
		mmCheckViews(t, s.Name, s.Post.Router.state(), mmMultiPrices(&s.Post), s.Post.Timestamp, s.Post.Views)
	}
	require.GreaterOrEqual(t, ops, 25)
	require.GreaterOrEqual(t, fails, 5)
}

// mmMultiPrices recovers the per-asset crosses the generator passed to the views (the live grid's fifth price
// column is the unscaled cross).
func mmMultiPrices(s *mmFxSnap) []uint256.Int {
	px := make([]uint256.Int, len(s.Views.Loans))
	for i, l := range s.Views.Loans {
		px[i] = l.FundingCeiling[4].PriceWad
	}
	return px
}

// TestMMTransientRepayScope: the repay snapshot is EIP-1153 transient storage, so it survives within one
// transaction and is gone in the next. From the live buy that repays an indebted venue and then reclaims its
// collateral (still indebted after): a proportional withdrawCollateral in the same transaction is judged against
// the cascade's pre-repay snapshot (85209356 debt / 196499 coll, now 64209356 / 148072: Unhealthy even for one
// wei), while in a later transaction it reverts NoRepaySnapshot (MMRouterLib.sol:200-201; confirmed on the
// deployed router with forge --isolate). A raw repay leaves its snapshot for the caller's transaction until
// endTransaction.
func TestMMTransientRepayScope(t *testing.T) {
	t.Parallel()
	var fx struct {
		Steps []struct {
			Name string     `json:"name"`
			Sell bool       `json:"sell"`
			Used uint256Dec `json:"used"`
			Out  uint256Dec `json:"out"`
			Pre  mmFxSnap   `json:"pre"`
		} `json:"steps"`
	}
	mmReadFixture(t, "mm_live_settle.json.gz", &fx)
	var found bool
	for _, s := range fx.Steps {
		if s.Name != "buyReclaimIndebted" {
			continue
		}
		found = true
		now := s.Pre.Timestamp
		withdraw := uint256FromU64(1000)

		// the buy's own transaction: settleBuy's repayCascade snapshot is still readable
		r, p := s.Pre.Router.state(), s.Pre.Pool.state()
		pc, rc := p.clone(), r.clone()
		require.NoError(t, mmSettleBuyLegs(pc, rc, 0, &s.Used.Int, &s.Out.Int, now))
		debt, err := rc.Venues[0].Morpho.debtOf(now)
		require.NoError(t, err)
		require.False(t, debt.IsZero())
		require.ErrorIs(t, rc.clone().withdrawCollateral(0, uOne, uZero, true, now), ErrUnhealthy)

		// a later transaction: the committed settlement leaves no snapshot behind
		r, p = s.Pre.Router.state(), s.Pre.Pool.state()
		require.NoError(t, mmSettleBuy(p, r, 0, &s.Used.Int, &s.Out.Int, now))
		require.Nil(t, r.TransientRepay)
		require.ErrorIs(t, r.clone().withdrawCollateral(0, withdraw, uZero, true, now), ErrNoRepaySnapshot)

		// raw entries share the caller's transaction until it ends
		c := r.clone()
		_, err = c.repay(0, uint256FromU64(5e6), now)
		require.NoError(t, err)
		require.NoError(t, c.clone().withdrawCollateral(0, withdraw, uZero, true, now))
		c.endTransaction()
		require.ErrorIs(t, c.withdrawCollateral(0, withdraw, uZero, true, now), ErrNoRepaySnapshot)
	}
	require.True(t, found)
}

// TestMMRepaySnapshotPacking: `(debt << 128) | coll` read back as (packed >> 128, packed & type(uint128).max)
// keeps uint128 figures and drops a debt's bits at and above 2^128 (MMRouterLib.sol:808, 820-821).
func TestMMRepaySnapshotPacking(t *testing.T) {
	t.Parallel()
	two128 := new(uint256.Int).Lsh(uOne, 128)
	max128 := new(uint256.Int).Sub(two128, uOne)
	for _, c := range []struct{ debt, coll, wantDebt, wantColl *uint256.Int }{
		{uint256FromU64(85209356), uint256FromU64(196499), uint256FromU64(85209356), uint256FromU64(196499)},
		{max128, max128, max128, max128},
		{two128, uint256FromU64(7), uZero, uint256FromU64(7)},
		{new(uint256.Int).Add(two128, uint256FromU64(3)), uOne, uint256FromU64(3), uOne},
	} {
		s := mmPackRepaySnapshot(c.debt, c.coll)
		mmEq(t, c.wantDebt, &s.Debt, "debt %s", c.debt.Dec())
		mmEq(t, c.wantColl, &s.Coll, "coll %s", c.coll.Dec())
	}
}
