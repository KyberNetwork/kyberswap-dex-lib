package everlongflamm

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// Fixture shapes written by testdata/gen/MMFixtureBase.sol and the helpers that load them into the port's state.

type mmFxLoan struct {
	Decimals      uint8       `json:"decimals"`
	LoanScale     uint256.Int `json:"loanScale"`
	DebtCap       uint256.Int `json:"debtCap"`
	SupplyCap     uint256.Int `json:"supplyCap"`
	BorrowEnabled bool        `json:"borrowEnabled"`
	Retired       bool        `json:"retired"`
}

type mmFxVenue struct {
	Kind                uint8       `json:"kind"`
	LoanIndex           uint8       `json:"loanIndex"`
	LltvWad             uint256.Int `json:"lltvWad"`
	BorrowEnabled       bool        `json:"borrowEnabled"`
	SupplyEnabled       bool        `json:"supplyEnabled"`
	Retired             bool        `json:"retired"`
	DebtCap             uint256.Int `json:"debtCap"`
	SupplyCap           uint256.Int `json:"supplyCap"`
	MaxBorrowRateWad    uint256.Int `json:"maxBorrowRateWad"`
	ManagedCollateral   uint256.Int `json:"managedCollateral"`
	ManagedSupplyShares uint256.Int `json:"managedSupplyShares"`
	Market              mmMarket    `json:"market"`
	Position            mmPosition  `json:"position"`
	RateAtTarget        uint256.Int `json:"rateAtTarget"`
	HasIrm              bool        `json:"hasIrm"`
	IrmReadable         bool        `json:"irmReadable"`
	OracleOk            bool        `json:"oracleOk"`
	OraclePrice         uint256.Int `json:"oraclePrice"`
}

type mmFxRouter struct {
	GlobalPaused   bool        `json:"globalPaused"`
	PinLtvWad      uint256.Int `json:"pinLtvWad"`
	SafetyGapWad   uint256.Int `json:"safetyGapWad"`
	OracleBandWad  uint256.Int `json:"oracleBandWad"`
	MaxDrawnAssets uint8       `json:"maxDrawnAssets"`
	Loans          []mmFxLoan  `json:"loans"`
	BorrowOrder    []uint16    `json:"borrowOrder"`
	SupplyOrder    []uint16    `json:"supplyOrder"`
	WithdrawOrder  []uint16    `json:"withdrawOrder"`
	RepayOrder     []uint16    `json:"repayOrder"`
	Venues         []mmFxVenue `json:"venues"`
}

func (f *mmFxRouter) state() *mmRouter {
	r := &mmRouter{GlobalPaused: f.GlobalPaused, PinLtvWad: f.PinLtvWad, SafetyGapWad: f.SafetyGapWad,
		OracleBandWad: f.OracleBandWad, MaxDrawnAssets: f.MaxDrawnAssets, BorrowOrder: f.BorrowOrder,
		SupplyOrder: f.SupplyOrder, WithdrawOrder: f.WithdrawOrder, RepayOrder: f.RepayOrder}
	for _, l := range f.Loans {
		r.Loans = append(r.Loans, mmLoan(l))
	}
	for _, v := range f.Venues {
		r.Venues = append(r.Venues, mmVenue{
			Morpho: mmVenueMarket{Market: v.Market, Position: v.Position, Lltv: v.LltvWad, HasIrm: v.HasIrm,
				IrmReadable: v.IrmReadable, RateAtTarget: v.RateAtTarget, OracleOk: v.OracleOk, OraclePrice: v.OraclePrice},
			Kind: v.Kind, LoanIndex: v.LoanIndex, LltvWad: v.LltvWad, BorrowEnabled: v.BorrowEnabled,
			SupplyEnabled: v.SupplyEnabled, Retired: v.Retired, DebtCap: v.DebtCap, SupplyCap: v.SupplyCap,
			MaxBorrowRateWad: v.MaxBorrowRateWad, ManagedCollateral: v.ManagedCollateral,
			ManagedSupplyShares: v.ManagedSupplyShares})
	}
	return r
}

type mmFxPoolLoan struct {
	Scale         uint256.Int `json:"scale"`
	Liquid        uint256.Int `json:"liquid"`
	ReserveTarget uint256.Int `json:"reserveTarget"`
	PriceWad      uint256.Int `json:"priceWad"`
	CrossWad      uint256.Int `json:"crossWad"`
}

type mmFxResult struct {
	V   *uint256.Int `json:"v"`
	Err string       `json:"err"`
}

type mmFxPool struct {
	Physical       uint256.Int    `json:"physical"`
	Gross          uint256.Int    `json:"gross"`
	PhiWad         uint256.Int    `json:"phiWad"`
	LtvWad         uint256.Int    `json:"ltvWad"`
	RoomEpsilonWad uint256.Int    `json:"roomEpsilonWad"`
	Features       uint256.Int    `json:"features"`
	TotalSupply    uint256.Int    `json:"totalSupply"`
	Loans          []mmFxPoolLoan `json:"loans"`
	LoanPosition   []uint256.Int  `json:"loanPosition"`
	TotalAssets    mmFxResult     `json:"totalAssets"`
}

func (f *mmFxPool) state() *gatePool {
	p := &gatePool{Physical: f.Physical, LtvWad: f.LtvWad, PhiWad: f.PhiWad, RoomEpsilonWad: f.RoomEpsilonWad,
		Features: f.Features}
	for _, l := range f.Loans {
		p.Loans = append(p.Loans, gateLoanCfg{Scale: l.Scale, Liquid: l.Liquid, ReserveTarget: l.ReserveTarget})
		p.PriceWad = append(p.PriceWad, l.PriceWad)
		p.CrossWad = append(p.CrossWad, l.CrossWad)
	}
	return p
}

type mmFxRate struct {
	DBorrow     uint256.Int `json:"dBorrow"`
	DSupplyDown uint256.Int `json:"dSupplyDown"`
	Ok          bool        `json:"ok"`
	Rate        uint256.Int `json:"rate"`
}

type mmFxShares struct {
	Shares uint256.Int  `json:"shares"`
	V      *uint256.Int `json:"v"`
	Err    string       `json:"err"`
}

type mmFxAccount struct {
	TryPosition struct {
		Readable     bool        `json:"readable"`
		Collateral   uint256.Int `json:"collateral"`
		SupplyShares uint256.Int `json:"supplyShares"`
		Supplied     uint256.Int `json:"supplied"`
		Debt         uint256.Int `json:"debt"`
	} `json:"tryPosition"`
	DebtOf               mmFxResult   `json:"debtOf"`
	SuppliedOf           mmFxResult   `json:"suppliedOf"`
	CollateralOf         uint256.Int  `json:"collateralOf"`
	FreeLiquidity        uint256.Int  `json:"freeLiquidity"`
	SupplySharesToAssets []mmFxShares `json:"supplySharesToAssets"`
	BorrowRateAfter      []mmFxRate   `json:"borrowRateAfter"`
}

type mmFxCeiling struct {
	CollIn   uint256.Int  `json:"collIn"`
	PriceWad uint256.Int  `json:"priceWad"`
	V        *uint256.Int `json:"v"`
	Err      string       `json:"err"`
}

type mmFxViews struct {
	Positions struct {
		Coll      []uint256.Int `json:"coll"`
		Sup       []uint256.Int `json:"sup"`
		Debt      []uint256.Int `json:"debt"`
		TotalColl uint256.Int   `json:"totalColl"`
	} `json:"positions"`
	Drawn               []uint8     `json:"drawn"`
	MinLltv             uint256.Int `json:"minLltv"`
	Reclaimable         mmFxResult  `json:"reclaimable"`
	ReclaimableUnpriced mmFxResult  `json:"reclaimableUnpriced"`
	Loans               []struct {
		Position   []uint256.Int `json:"position"`
		Quarantine struct {
			Any        bool        `json:"any"`
			FrozenDebt uint256.Int `json:"frozenDebt"`
			FrozenColl uint256.Int `json:"frozenColl"`
		} `json:"quarantine"`
		FundingCeiling []mmFxCeiling `json:"fundingCeiling"`
	} `json:"loans"`
	Venues []struct {
		VenuePosition []uint256.Int `json:"venuePosition"`
		Readable      bool          `json:"readable"`
		Health        mmFxResult    `json:"health"`
		Account       mmFxAccount   `json:"account"`
	} `json:"venues"`
}

type mmFxSnap struct {
	Block     uint64        `json:"block"`
	Timestamp uint64        `json:"timestamp"`
	Router    mmFxRouter    `json:"router"`
	Pool      *mmFxPool     `json:"pool"`
	Prices    []uint256.Int `json:"prices"`
	Views     *mmFxViews    `json:"views"`
	Lens      *struct {
		Facts  []uint256.Int   `json:"facts"`
		Venues [][]uint256.Int `json:"venues"`
	} `json:"lens"`
}

// mmRevertError maps fixture revert data onto the port's sentinels: custom errors by selector, Morpho's require
// strings and Solidity panics by payload.
func mmRevertError(t *testing.T, data string) error {
	t.Helper()
	raw, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	require.NoError(t, err)
	if len(raw) < 4 {
		return errMulDivOverflow // a bare revert: Math.mulDiv's require
	}
	var sel [4]byte
	copy(sel[:], raw[:4])
	if e, ok := revertSelectors[sel]; ok {
		return e
	}
	switch hex.EncodeToString(sel[:]) {
	case "4e487b71":
		code := new(uint256.Int).SetBytes(raw[4:36]).Uint64()
		switch code {
		case 0x11:
			return errPanicArithmetic
		case 0x12:
			return errPanicDivZero
		case 0x32:
			return errPanicIndex
		}
	case "08c379a0":
		n := new(uint256.Int).SetBytes(raw[36:68]).Uint64()
		msg := string(raw[68 : 68+n])
		switch msg {
		case "insufficient liquidity":
			return errMMInsufficientLiquidity
		case "insufficient collateral":
			return errMMInsufficientCollateral
		case "max uint128 exceeded":
			return errMMMaxUint128Exceeded
		}
		return fmt.Errorf("unmapped revert string %q", msg)
	}
	return fmt.Errorf("unmapped revert %s", data)
}

func mmRequireResult(t *testing.T, want mmFxResult, got uint256.Int, err error, msg string) {
	t.Helper()
	if want.Err != "" {
		require.ErrorIs(t, err, mmRevertError(t, want.Err), msg)
		return
	}
	require.NoError(t, err, msg)
	require.NotNil(t, want.V, msg)
	require.Equal(t, want.V.Dec(), got.Dec(), msg)
}

func mmEq(t *testing.T, want, got *uint256.Int, msg string, args ...any) {
	t.Helper()
	require.Equal(t, want.Dec(), got.Dec(), append([]any{msg}, args...)...)
}

// mmCheckViews recomputes every recorded router / account view from the loaded state at `now`.
func mmCheckViews(t *testing.T, name string, r *mmRouter, prices []uint256.Int, now uint64, w *mmFxViews) {
	t.Helper()
	pos, err := r.positions(now)
	require.NoError(t, err, name)
	for i := range pos.Coll {
		mmEq(t, &w.Positions.Coll[i], &pos.Coll[i], "%s positions.coll[%d]", name, i)
		mmEq(t, &w.Positions.Sup[i], &pos.Sup[i], "%s positions.sup[%d]", name, i)
		mmEq(t, &w.Positions.Debt[i], &pos.Debt[i], "%s positions.debt[%d]", name, i)
	}
	mmEq(t, &w.Positions.TotalColl, &pos.TotalColl, "%s positions.totalColl", name)
	count, mask, err := r.drawn(now)
	require.NoError(t, err)
	require.Equal(t, w.Drawn, []uint8{count, mask}, name+" drawn")
	lltv, err := r.minLltv(now)
	require.NoError(t, err)
	mmEq(t, &w.MinLltv, &lltv, "%s minLltv", name)
	rc, err := r.reclaimable(prices, now)
	mmRequireResult(t, w.Reclaimable, rc, err, name+" reclaimable")
	rc, err = r.reclaimable(make([]uint256.Int, len(prices)), now)
	mmRequireResult(t, w.ReclaimableUnpriced, rc, err, name+" reclaimableUnpriced")
	for i, lw := range w.Loans {
		c, s, d, err := r.position(uint8(i), now)
		require.NoError(t, err)
		mmEq(t, &lw.Position[0], &c, "%s loan %d position coll", name, i)
		mmEq(t, &lw.Position[1], &s, "%s loan %d position sup", name, i)
		mmEq(t, &lw.Position[2], &d, "%s loan %d position debt", name, i)
		q, err := r.quarantine(uint8(i), now)
		require.NoError(t, err)
		require.Equal(t, lw.Quarantine.Any, q.Any, name)
		mmEq(t, &lw.Quarantine.FrozenDebt, &q.FrozenDebt, "%s frozenDebt", name)
		mmEq(t, &lw.Quarantine.FrozenColl, &q.FrozenColl, "%s frozenColl", name)
		for _, row := range lw.FundingCeiling {
			got, err := r.fundingCeiling(uint8(i), &row.CollIn, &row.PriceWad, now)
			msg := fmt.Sprintf("%s loan %d fundingCeiling(coll %s, price %s)", name, i, row.CollIn.Dec(), row.PriceWad.Dec())
			mmRequireResult(t, mmFxResult{V: row.V, Err: row.Err}, got, err, msg)
		}
	}
	for i, vw := range w.Venues {
		v := &r.Venues[i]
		vp, err := r.venuePosition(uint16(i), now)
		require.NoError(t, err)
		for k := range vp {
			mmEq(t, &vw.VenuePosition[k], &vp[k], "%s venue %d venuePosition[%d]", name, i, k)
		}
		rd, err := r.read(v, now)
		require.NoError(t, err)
		require.Equal(t, vw.Readable, rd.Readable, name)
		h, err := r.venueHealth(uint16(i), &prices[v.LoanIndex], now)
		mmRequireResult(t, vw.Health, h, err, fmt.Sprintf("%s venue %d health", name, i))
		mmCheckAccount(t, fmt.Sprintf("%s venue %d", name, i), &v.Morpho, now, &vw.Account)
	}
}

func mmCheckAccount(t *testing.T, name string, m *mmVenueMarket, now uint64, w *mmFxAccount) {
	t.Helper()
	tp, err := m.tryPosition(now)
	require.NoError(t, err)
	require.Equal(t, w.TryPosition.Readable, tp.Readable, name)
	mmEq(t, &w.TryPosition.Collateral, &tp.Collateral, "%s tryPosition.collateral", name)
	mmEq(t, &w.TryPosition.SupplyShares, &tp.SupplyShares, "%s tryPosition.supplyShares", name)
	mmEq(t, &w.TryPosition.Supplied, &tp.Supplied, "%s tryPosition.supplied", name)
	mmEq(t, &w.TryPosition.Debt, &tp.Debt, "%s tryPosition.debt", name)
	d, err := m.debtOf(now)
	mmRequireResult(t, w.DebtOf, d, err, name+" debtOf")
	s, err := m.suppliedOf(now)
	mmRequireResult(t, w.SuppliedOf, s, err, name+" suppliedOf")
	mmEq(t, &w.CollateralOf, &m.Position.Collateral, "%s collateralOf", name)
	fl := m.freeLiquidity()
	mmEq(t, &w.FreeLiquidity, &fl, "%s freeLiquidity", name)
	for _, row := range w.SupplySharesToAssets {
		v, err := m.supplySharesToAssets(&row.Shares, now)
		mmRequireResult(t, mmFxResult{V: row.V, Err: row.Err}, v, err, name+" supplySharesToAssets "+row.Shares.Dec())
	}
	for _, row := range w.BorrowRateAfter {
		ok, rate, err := m.borrowRateAfter(&row.DBorrow, &row.DSupplyDown, now)
		require.NoError(t, err)
		msg := fmt.Sprintf("%s borrowRateAfter(%s, %s)", name, row.DBorrow.Dec(), row.DSupplyDown.Dec())
		require.Equal(t, row.Ok, ok, msg)
		mmEq(t, &row.Rate, &rate, msg)
	}
}

// mmCheckState compares the port's router (and pool) after a transition with the chain's.
func mmCheckState(t *testing.T, name string, r *mmRouter, p *gatePool, w *mmFxSnap) {
	t.Helper()
	want := w.Router.state()
	require.Equal(t, len(want.Venues), len(r.Venues), name)
	require.Equal(t, want.MaxDrawnAssets, r.MaxDrawnAssets, name)
	for i := range want.Venues {
		a, b := &want.Venues[i], &r.Venues[i]
		msg := fmt.Sprintf("%s venue %d", name, i)
		require.Equal(t, a.Morpho.Market, b.Morpho.Market, msg+" market")
		require.Equal(t, a.Morpho.Position, b.Morpho.Position, msg+" position")
		mmEq(t, &a.Morpho.RateAtTarget, &b.Morpho.RateAtTarget, msg+" rateAtTarget")
		mmEq(t, &a.ManagedCollateral, &b.ManagedCollateral, msg+" managedCollateral")
		mmEq(t, &a.ManagedSupplyShares, &b.ManagedSupplyShares, msg+" managedSupplyShares")
		mmEq(t, &a.MaxBorrowRateWad, &b.MaxBorrowRateWad, msg+" maxBorrowRateWad")
	}
	if p != nil && w.Pool != nil {
		mmEq(t, &w.Pool.Physical, &p.Physical, "%s physical", name)
		for i := range p.Loans {
			mmEq(t, &w.Pool.Loans[i].Liquid, &p.Loans[i].Liquid, "%s liquid[%d]", name, i)
		}
	}
}

// mmCheckLens recomputes FLAMMLens.facts and the numeric part of FLAMMLens.venues (the Lens prices health at its
// own feed peek, so health is compared only where the Lens could price it).
func mmCheckLens(t *testing.T, name string, s *mmFxSnap) {
	t.Helper()
	r, p, now := s.Router.state(), s.Pool.state(), s.Timestamp
	b, err := gateBookOf(p, r, now)
	require.NoError(t, err, name)
	gross, _ := gateGross(&b)
	f := s.Lens.Facts
	mmEq(t, &f[0], &b.Physical, "%s lens physical", name)
	mmEq(t, &f[1], &b.Posted, "%s lens posted", name)
	mmEq(t, &f[2], &gross, "%s lens gross", name)
	mmEq(t, &f[3], &b.Legs[0].Liquid, "%s lens liquid", name)
	mmEq(t, &f[4], &b.Legs[0].Supplied, "%s lens supplied", name)
	mmEq(t, &f[5], &b.Legs[0].Debt, "%s lens debt", name)
	for i, lv := range s.Lens.Venues {
		vp, err := r.venuePosition(uint16(i), now)
		require.NoError(t, err)
		for k := 0; k < 5; k++ {
			mmEq(t, &lv[k], &vp[k], "%s lens venue %d [%d]", name, i, k)
		}
		fl := r.Venues[i].Morpho.freeLiquidity()
		mmEq(t, &lv[5], &fl, "%s lens venue %d freeLiquidity", name, i)
		if !lv[6].IsZero() {
			h, err := r.venueHealth(uint16(i), &s.Prices[r.Venues[i].LoanIndex], now)
			require.NoError(t, err)
			mmEq(t, &lv[6], &h, "%s lens venue %d health", name, i)
		}
	}
}

// mmCheckPoolViews recomputes the pool-level gate reads recorded with a live snapshot.
func mmCheckPoolViews(t *testing.T, name string, r *mmRouter, p *gatePool, now uint64, w *mmFxPool) {
	t.Helper()
	b, err := gateBookOf(p, r, now)
	require.NoError(t, err, name)
	gross, err := gateGross(&b)
	require.NoError(t, err)
	mmEq(t, &w.Gross, &gross, "%s gross", name)
	mmEq(t, &w.LoanPosition[0], &b.Legs[0].Liquid, "%s loanPosition.liquid", name)
	mmEq(t, &w.LoanPosition[1], &b.Legs[0].Supplied, "%s loanPosition.supplied", name)
	mmEq(t, &w.LoanPosition[2], &b.Legs[0].Debt, "%s loanPosition.debt", name)
	ta, err := gateTotalAssets(p, r, now)
	mmRequireResult(t, w.TotalAssets, ta, err, name+" totalAssets")
}

// uint256Dec decodes a quoted decimal.
type uint256Dec struct{ uint256.Int }

func uint256FromU64(x uint64) *uint256.Int { return uint256.NewInt(x) }
