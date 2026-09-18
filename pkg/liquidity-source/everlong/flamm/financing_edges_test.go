package everlongflamm

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// Edge replays for the financing module (morpho.go, irm.go, account.go, router.go, gate.go). The fixtures under
// testdata/edges/ come from testdata/gen/{Morpho,Account,Router,Gate,Settle}Edges.t.sol: threshold edges and
// seeded grids replayed against the DEPLOYED Morpho Blue, AdaptiveCurveIrm, MorphoBlueAccount and MMRouter on a Base
// fork (block 51317000), plus FLAMMGateLib compiled from 80abd43 behind a harness. Every array opens with a header
// element. A row compares the full result: revert class or returned values and the written post-state.

// loadGzFixture decodes one gzipped fixture under testdata/.
func loadGzFixture(t *testing.T, name string, v any) {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(zr).Decode(v))
	require.NoError(t, zr.Close())
}

// signedDec decodes a quoted signed decimal as sign + magnitude.
type signedDec struct {
	Neg bool
	Abs uint256.Int
}

func (x *signedDec) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	x.Neg = strings.HasPrefix(s, "-")
	if err := x.Abs.SetFromDecimal(strings.TrimPrefix(s, "-")); err != nil {
		return err
	}
	if x.Abs.IsZero() {
		x.Neg = false
	}
	return nil
}

// finRevert maps raw revert data onto the port's error vocabulary.
func finRevert(t *testing.T, h string) error {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(h, "0x"))
	require.NoError(t, err)
	if len(b) == 0 {
		return errMulDivOverflow // OpenZeppelin v4 Math.mulDiv's bare require
	}
	require.GreaterOrEqual(t, len(b), 4)
	var sel [4]byte
	copy(sel[:], b[:4])
	switch hex.EncodeToString(sel[:]) {
	case "4e487b71":
		switch b[len(b)-1] {
		case 0x11:
			return errPanicArithmetic
		case 0x12:
			return errPanicDivZero
		case 0x32:
			return errPanicIndex
		}
		t.Fatalf("unexpected panic %x", b)
	case "08c379a0":
		n := new(uint256.Int).SetBytes(b[36:68]).Uint64()
		msg := string(b[68 : 68+n])
		switch msg {
		case "max uint128 exceeded":
			return errMMMaxUint128Exceeded
		case "insufficient liquidity":
			return errMMInsufficientLiquidity
		case "insufficient collateral":
			return errMMInsufficientCollateral
		case "inconsistent input":
			return errMMInconsistentInput
		case "zero assets":
			return errMMZeroAssets
		case "dead":
			return errMMOracleReverted
		case "irm dead":
			return errMMIrmReverted
		}
		t.Fatalf("unexpected require string %q", msg)
	}
	if e, ok := revertSelectors[sel]; ok {
		return e
	}
	t.Fatalf("unknown revert %s", h)
	return nil
}

// finMismatches collects divergences so one run reports all of them.
type finMismatches struct {
	t    *testing.T
	area string
	n    int
}

func (m *finMismatches) add(format string, args ...any) {
	m.t.Helper()
	m.n++
	if m.n <= 40 {
		m.t.Errorf("[%s] "+format, append([]any{m.area}, args...)...)
	}
}

func errClass(err error) string {
	if err == nil {
		return "ok"
	}
	return err.Error()
}

func finWantErr(got, want error) bool {
	if want == nil {
		return got == nil
	}
	return errors.Is(got, want)
}

// ------------------------------------------------------------------ Morpho transitions

type finMarket struct {
	Tsa        uint256.Int `json:"tsa"`
	Tss        uint256.Int `json:"tss"`
	Tba        uint256.Int `json:"tba"`
	Tbs        uint256.Int `json:"tbs"`
	LastUpdate uint256.Int `json:"lastUpdate"`
	Fee        uint256.Int `json:"fee"`
}

func (m *finMarket) state() mmMarket {
	return mmMarket{TotalSupplyAssets: m.Tsa, TotalSupplyShares: m.Tss, TotalBorrowAssets: m.Tba,
		TotalBorrowShares: m.Tbs, LastUpdate: m.LastUpdate, Fee: m.Fee}
}

type finPosition struct {
	SupplyShares uint256.Int `json:"supplyShares"`
	BorrowShares uint256.Int `json:"borrowShares"`
	Collateral   uint256.Int `json:"collateral"`
}

func (p *finPosition) state() mmPosition {
	return mmPosition{SupplyShares: p.SupplyShares, BorrowShares: p.BorrowShares, Collateral: p.Collateral}
}

type finMorphoRow struct {
	Header       bool        `json:"header"`
	Tag          string      `json:"tag"`
	Now          uint64      `json:"now,string"`
	NoIrm        bool        `json:"noIrm"`
	Lltv         uint256.Int `json:"lltv"`
	Rat          signedDec   `json:"rat"`
	Price        uint256.Int `json:"price"`
	Dead         bool        `json:"dead"`
	Op           uint64      `json:"op,string"`
	A            uint256.Int `json:"a"`
	S            uint256.Int `json:"s"`
	Market       finMarket   `json:"market"`
	Position     finPosition `json:"position"`
	Ok           bool        `json:"ok"`
	Revert       string      `json:"revert"`
	R0           uint256.Int `json:"r0"`
	R1           uint256.Int `json:"r1"`
	PostMarket   finMarket   `json:"postMarket"`
	PostPosition finPosition `json:"postPosition"`
	PostRat      signedDec   `json:"postRat"`
}

// TestMorphoEdges replays every Blue transition row against morpho.go / irm.go.
func TestMorphoEdges(t *testing.T) {
	t.Parallel()
	var rows []finMorphoRow
	loadGzFixture(t, "edges/mm_morpho_edges.json.gz", &rows)
	require.Greater(t, len(rows), 500)
	mm := &finMismatches{t: t, area: "morpho"}
	for i, r := range rows[1:] {
		v := mmVenueMarket{Market: r.Market.state(), Position: r.Position.state(), Lltv: r.Lltv, HasIrm: !r.NoIrm,
			IrmReadable: true, RateAtTarget: r.Rat.Abs, OracleOk: !r.Dead && !r.Price.IsZero(), OraclePrice: r.Price,
			OracleZero: !r.Dead && r.Price.IsZero()}
		var r0, r1 uint256.Int
		var err error
		switch r.Op {
		case 0:
			err = v.morphoAccrue(r.Now)
		case 1:
			r0 = r.A
			r1, err = v.morphoSupply(&r.A, r.Now)
		case 2:
			r0, r1, err = v.morphoWithdraw(&r.A, &r.S, r.Now)
		case 3:
			r0 = r.A
			r1, err = v.morphoBorrow(&r.A, r.Now)
		case 4:
			r0, r1, err = v.morphoRepay(&r.A, &r.S, r.Now)
		case 5:
			err = v.morphoSupplyCollateral(&r.A)
		case 6:
			err = v.morphoWithdrawCollateral(&r.A, r.Now)
		}
		ctx := fmt.Sprintf("row %d tag=%s op=%d a=%s s=%s market=%+v pos=%+v rat=%s price=%s dead=%v noIrm=%v", i, r.Tag,
			r.Op, r.A.Dec(), r.S.Dec(), r.Market, r.Position, r.Rat.Abs.Dec(), r.Price.Dec(), r.Dead, r.NoIrm)
		if !r.Ok {
			want := finRevert(t, r.Revert)
			if !finWantErr(err, want) {
				mm.add("%s: go=%s sol=%s", ctx, errClass(err), errClass(want))
			}
			continue
		}
		if err != nil {
			mm.add("%s: go=%s sol=ok", ctx, err)
			continue
		}
		if r.Op >= 1 && r.Op <= 4 && (!r0.Eq(&r.R0) || !r1.Eq(&r.R1)) {
			mm.add("%s: returns go=(%s,%s) sol=(%s,%s)", ctx, r0.Dec(), r1.Dec(), r.R0.Dec(), r.R1.Dec())
		}
		if pm := r.PostMarket.state(); v.Market != pm {
			mm.add("%s: post market go=%+v sol=%+v", ctx, v.Market, pm)
		}
		if pp := r.PostPosition.state(); v.Position != pp {
			mm.add("%s: post position go=%+v sol=%+v", ctx, v.Position, pp)
		}
		if !v.RateAtTarget.Eq(&r.PostRat.Abs) {
			mm.add("%s: post rateAtTarget go=%s sol=%s", ctx, v.RateAtTarget.Dec(), r.PostRat.Abs.Dec())
		}
	}
	t.Logf("morpho rows %d, mismatches %d", len(rows)-1, mm.n)
}

type finIrmRow struct {
	Header     bool        `json:"header"`
	Tag        string      `json:"tag"`
	Now        uint64      `json:"now,string"`
	Tsa        uint256.Int `json:"tsa"`
	Tba        uint256.Int `json:"tba"`
	LastUpdate uint256.Int `json:"lastUpdate"`
	Rat        signedDec   `json:"rat"`
	ViewOk     bool        `json:"viewOk"`
	View       string      `json:"view"`
	MutOk      bool        `json:"mutOk"`
	Mut        string      `json:"mut"`
	End        signedDec   `json:"end"`
}

// TestIrmEdges compares mmIrmBorrowRate with borrowRateView and the stored endRateAtTarget of borrowRate.
func TestIrmEdges(t *testing.T) {
	t.Parallel()
	var rows []finIrmRow
	loadGzFixture(t, "edges/mm_irm_edges.json.gz", &rows)
	require.Greater(t, len(rows), 600)
	mm := &finMismatches{t: t, area: "irm"}
	for i, r := range rows[1:] {
		m := mmMarket{TotalSupplyAssets: r.Tsa, TotalSupplyShares: r.Tsa, TotalBorrowAssets: r.Tba,
			TotalBorrowShares: r.Tba, LastUpdate: r.LastUpdate}
		avg, end, err := mmIrmBorrowRate(&m, &r.Rat.Abs, r.Now)
		ctx := fmt.Sprintf("row %d tag=%s tsa=%s tba=%s lastUpdate=%s rat=%s", i, r.Tag, r.Tsa.Dec(), r.Tba.Dec(),
			r.LastUpdate.Dec(), r.Rat.Abs.Dec())
		if !r.ViewOk {
			want := finRevert(t, r.View)
			if !finWantErr(err, want) {
				mm.add("%s: go=%s sol=%s", ctx, errClass(err), errClass(want))
			}
			continue
		}
		if err != nil {
			mm.add("%s: go=%s sol=ok", ctx, err)
			continue
		}
		if avg.Dec() != r.View || !r.MutOk || avg.Dec() != r.Mut || !end.Eq(&r.End.Abs) {
			mm.add("%s: go=(%s,%s) sol=(%s,%s,%s)", ctx, avg.Dec(), end.Dec(), r.View, r.Mut, r.End.Abs.Dec())
		}
	}
}

// ------------------------------------------------------------------ MorphoBlueAccount

type finRes struct {
	Ok     bool        `json:"ok"`
	V      uint256.Int `json:"v"`
	Revert string      `json:"revert"`
}

// check compares a (value, error) pair with a view result.
func (w *finRes) check(t *testing.T, mm *finMismatches, ctx string, got uint256.Int, err error) {
	t.Helper()
	if !w.Ok {
		want := finRevert(t, w.Revert)
		if !finWantErr(err, want) {
			mm.add("%s: go=%s(%s) sol=%s", ctx, errClass(err), got.Dec(), errClass(want))
		}
		return
	}
	if err != nil {
		mm.add("%s: go=%s sol=%s", ctx, err, w.V.Dec())
		return
	}
	if !got.Eq(&w.V) {
		mm.add("%s: go=%s sol=%s", ctx, got.Dec(), w.V.Dec())
	}
}

type finAccountRow struct {
	Header      bool        `json:"header"`
	Tag         string      `json:"tag"`
	Now         uint64      `json:"now,string"`
	Lltv        uint256.Int `json:"lltv"`
	Rat         signedDec   `json:"rat"`
	Price       uint256.Int `json:"price"`
	OracleDead  bool        `json:"oracleDead"`
	IrmDead     bool        `json:"irmDead"`
	Market      finMarket   `json:"market"`
	Position    finPosition `json:"position"`
	TryPosition struct {
		Ok           bool        `json:"ok"`
		Revert       string      `json:"revert"`
		Readable     bool        `json:"readable"`
		Collateral   uint256.Int `json:"collateral"`
		SupplyShares uint256.Int `json:"supplyShares"`
		Supplied     uint256.Int `json:"supplied"`
		Debt         uint256.Int `json:"debt"`
	} `json:"tryPosition"`
	DebtOf         finRes `json:"debtOf"`
	SuppliedOf     finRes `json:"suppliedOf"`
	FreeLiquidity  finRes `json:"freeLiquidity"`
	SharesToAssets []struct {
		Shares uint256.Int `json:"shares"`
		Res    finRes      `json:"res"`
	} `json:"sharesToAssets"`
	RateAfter []struct {
		DB  uint256.Int `json:"dB"`
		DS  uint256.Int `json:"dS"`
		Res struct {
			Ok     bool        `json:"ok"`
			Revert string      `json:"revert"`
			Rok    bool        `json:"rok"`
			Rate   uint256.Int `json:"rate"`
		} `json:"res"`
	} `json:"rateAfter"`
	OraclePrice struct {
		Ok bool        `json:"ok"`
		V  uint256.Int `json:"v"`
	} `json:"oraclePrice"`
	Op           uint64      `json:"op,string"`
	A            uint256.Int `json:"a"`
	S            uint256.Int `json:"s"`
	Ok           bool        `json:"ok"`
	Revert       string      `json:"revert"`
	R0           uint256.Int `json:"r0"`
	R1           uint256.Int `json:"r1"`
	PostMarket   finMarket   `json:"postMarket"`
	PostPosition finPosition `json:"postPosition"`
	PostRat      signedDec   `json:"postRat"`
}

// TestAccountEdges replays the deployed account's views and Router-driven mutators.
func TestAccountEdges(t *testing.T) {
	t.Parallel()
	var rows []finAccountRow
	loadGzFixture(t, "edges/mm_account_edges.json.gz", &rows)
	require.Greater(t, len(rows), 400)
	mm := &finMismatches{t: t, area: "account"}
	for i, r := range rows[1:] {
		base := mmVenueMarket{Market: r.Market.state(), Position: r.Position.state(), Lltv: r.Lltv, HasIrm: true,
			IrmReadable: !r.IrmDead, RateAtTarget: r.Rat.Abs, OracleOk: r.OraclePrice.Ok, OraclePrice: r.OraclePrice.V,
			OracleZero: !r.OracleDead && r.Price.IsZero()}
		ctx := fmt.Sprintf("row %d tag=%s now=%d market=%+v pos=%+v rat=%s irmDead=%v oracle=(%v,%s)", i, r.Tag, r.Now,
			r.Market, r.Position, r.Rat.Abs.Dec(), r.IrmDead, r.OraclePrice.Ok, r.OraclePrice.V.Dec())
		v := base
		tp, err := v.tryPosition(r.Now)
		if !r.TryPosition.Ok {
			if want := finRevert(t, r.TryPosition.Revert); !finWantErr(err, want) {
				mm.add("%s tryPosition: go=%s sol=%s", ctx, errClass(err), errClass(want))
			}
		} else if err != nil {
			mm.add("%s tryPosition: go=%s sol=ok", ctx, err)
		} else if tp.Readable != r.TryPosition.Readable || !tp.Collateral.Eq(&r.TryPosition.Collateral) ||
			!tp.SupplyShares.Eq(&r.TryPosition.SupplyShares) || !tp.Supplied.Eq(&r.TryPosition.Supplied) ||
			!tp.Debt.Eq(&r.TryPosition.Debt) {
			mm.add("%s tryPosition: go=%+v sol=%+v", ctx, tp, r.TryPosition)
		}
		got, err := v.debtOf(r.Now)
		r.DebtOf.check(t, mm, ctx+" debtOf", got, err)
		got, err = v.suppliedOf(r.Now)
		r.SuppliedOf.check(t, mm, ctx+" suppliedOf", got, err)
		r.FreeLiquidity.check(t, mm, ctx+" freeLiquidity", v.freeLiquidity(), nil)
		for _, s := range r.SharesToAssets {
			got, err = v.supplySharesToAssets(&s.Shares, r.Now)
			s.Res.check(t, mm, ctx+" supplySharesToAssets("+s.Shares.Dec()+")", got, err)
		}
		for _, d := range r.RateAfter {
			ok, rate, err := v.borrowRateAfter(&d.DB, &d.DS, r.Now)
			dctx := fmt.Sprintf("%s borrowRateAfter(%s,%s)", ctx, d.DB.Dec(), d.DS.Dec())
			if !d.Res.Ok {
				if want := finRevert(t, d.Res.Revert); !finWantErr(err, want) {
					mm.add("%s: go=%s sol=%s", dctx, errClass(err), errClass(want))
				}
			} else if err != nil || ok != d.Res.Rok || !rate.Eq(&d.Res.Rate) {
				mm.add("%s: go=(%v,%s,%v) sol=(%v,%s)", dctx, ok, rate.Dec(), err, d.Res.Rok, d.Res.Rate.Dec())
			}
		}
		if v != base {
			mm.add("%s: a view wrote state", ctx)
		}
		if r.Op == 0 {
			continue
		}
		var r0, r1 uint256.Int
		switch r.Op {
		case 1:
			r0, err = v.accountRepay(&r.A, r.Now)
		case 2:
			r0, r1, err = v.accountWithdraw(&r.A, &r.S, r.Now)
		case 3:
			err = v.accountBorrow(&r.A, r.Now)
		case 4:
			r0, err = v.accountSupply(&r.A, r.Now)
		case 5:
			err = v.accountSupplyCollateral(&r.A)
		case 6:
			err = v.accountWithdrawCollateral(&r.A, r.Now)
		}
		mctx := fmt.Sprintf("%s op=%d a=%s s=%s", ctx, r.Op, r.A.Dec(), r.S.Dec())
		if !r.Ok {
			want := finRevert(t, r.Revert)
			if !finWantErr(err, want) {
				mm.add("%s: go=%s sol=%s", mctx, errClass(err), errClass(want))
			}
			continue
		}
		if err != nil {
			mm.add("%s: go=%s sol=ok", mctx, err)
			continue
		}
		if !r0.Eq(&r.R0) || !r1.Eq(&r.R1) {
			mm.add("%s: returns go=(%s,%s) sol=(%s,%s)", mctx, r0.Dec(), r1.Dec(), r.R0.Dec(), r.R1.Dec())
		}
		if pm := r.PostMarket.state(); v.Market != pm {
			mm.add("%s: post market go=%+v sol=%+v", mctx, v.Market, pm)
		}
		if pp := r.PostPosition.state(); v.Position != pp {
			mm.add("%s: post position go=%+v sol=%+v", mctx, v.Position, pp)
		}
		if !v.RateAtTarget.Eq(&r.PostRat.Abs) {
			mm.add("%s: post rateAtTarget go=%s sol=%s", mctx, v.RateAtTarget.Dec(), r.PostRat.Abs.Dec())
		}
	}
	t.Logf("account rows %d, mismatches %d", len(rows)-1, mm.n)
}

// ------------------------------------------------------------------ MMRouter

type finRouterVenue struct {
	LoanIndex     uint8       `json:"loanIndex,string"`
	Lltv          uint256.Int `json:"lltv"`
	BorrowEnabled bool        `json:"borrowEnabled"`
	SupplyEnabled bool        `json:"supplyEnabled"`
	Retired       bool        `json:"retired"`
	DebtCap       uint256.Int `json:"debtCap"`
	SupplyCap     uint256.Int `json:"supplyCap"`
	MaxRate       uint256.Int `json:"maxRate"`
	ManagedColl   uint256.Int `json:"managedColl"`
	ManagedShares uint256.Int `json:"managedShares"`
	HasIrm        bool        `json:"hasIrm"`
	IrmDead       bool        `json:"irmDead"`
	Rat           signedDec   `json:"rat"`
	OracleOk      bool        `json:"oracleOk"`
	OraclePrice   uint256.Int `json:"oraclePrice"`
	AcctLltv      uint256.Int `json:"acctLltv"`
	Market        finMarket   `json:"market"`
	Position      finPosition `json:"position"`
}

type finRouterState struct {
	Paused        bool        `json:"paused"`
	Pin           uint256.Int `json:"pin"`
	Gap           uint256.Int `json:"gap"`
	Band          uint256.Int `json:"band"`
	MaxDrawn      uint8       `json:"maxDrawn,string"`
	BorrowOrder   []uint16    `json:"borrowOrder"`
	SupplyOrder   []uint16    `json:"supplyOrder"`
	WithdrawOrder []uint16    `json:"withdrawOrder"`
	RepayOrder    []uint16    `json:"repayOrder"`
	Loans         []struct {
		Decimals      uint8       `json:"decimals,string"`
		Scale         uint256.Int `json:"scale"`
		DebtCap       uint256.Int `json:"debtCap"`
		SupplyCap     uint256.Int `json:"supplyCap"`
		BorrowEnabled bool        `json:"borrowEnabled"`
		Retired       bool        `json:"retired"`
	} `json:"loans"`
	Venues []finRouterVenue `json:"venues"`
}

func (s *finRouterState) router() *mmRouter {
	r := &mmRouter{GlobalPaused: s.Paused, PinLtvWad: s.Pin, SafetyGapWad: s.Gap, OracleBandWad: s.Band,
		MaxDrawnAssets: s.MaxDrawn, BorrowOrder: s.BorrowOrder, SupplyOrder: s.SupplyOrder,
		WithdrawOrder: s.WithdrawOrder, RepayOrder: s.RepayOrder}
	for _, l := range s.Loans {
		r.Loans = append(r.Loans, mmLoan{Decimals: l.Decimals, LoanScale: l.Scale, DebtCap: l.DebtCap,
			SupplyCap: l.SupplyCap, BorrowEnabled: l.BorrowEnabled, Retired: l.Retired})
	}
	for _, v := range s.Venues {
		r.Venues = append(r.Venues, mmVenue{
			Morpho: mmVenueMarket{Market: v.Market.state(), Position: v.Position.state(), Lltv: v.AcctLltv,
				HasIrm: v.HasIrm, IrmReadable: !v.IrmDead, RateAtTarget: v.Rat.Abs, OracleOk: v.OracleOk,
				OraclePrice: v.OraclePrice},
			LoanIndex: v.LoanIndex, LltvWad: v.Lltv, BorrowEnabled: v.BorrowEnabled, SupplyEnabled: v.SupplyEnabled,
			Retired: v.Retired, DebtCap: v.DebtCap, SupplyCap: v.SupplyCap, MaxBorrowRateWad: v.MaxRate,
			ManagedCollateral: v.ManagedColl, ManagedSupplyShares: v.ManagedShares})
	}
	return r
}

type finCall struct {
	Ok  bool   `json:"ok"`
	Ret string `json:"ret"`
}

func (c *finCall) words(t *testing.T) []uint256.Int {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(c.Ret, "0x"))
	require.NoError(t, err)
	out := make([]uint256.Int, len(b)/32)
	for i := range out {
		out[i].SetBytes(b[32*i : 32*i+32])
	}
	return out
}

// finDynArray decodes the dynamic uint256[] whose head offset sits in word k.
func finDynArray(w []uint256.Int, k int) []uint256.Int {
	off := int(w[k].Uint64() / 32)
	n := int(w[off].Uint64())
	return w[off+1 : off+1+n]
}

// finRevertRouter extends finRevert with the live oracle's raw "dead" payload.
func finRevertRouter(t *testing.T, h string) error {
	if h == "0x64656164" {
		return errMMOracleReverted
	}
	return finRevert(t, h)
}

// checkWords compares a view call with the port's words (or its error).
func (c *finCall) checkWords(t *testing.T, mm *finMismatches, ctx string, got []uint256.Int, err error) {
	t.Helper()
	if !c.Ok {
		want := finRevertRouter(t, c.Ret)
		if !finWantErr(err, want) {
			mm.add("%s: go=%s sol=%s", ctx, errClass(err), errClass(want))
		}
		return
	}
	if err != nil {
		mm.add("%s: go=%s sol=ok %s", ctx, err, c.Ret)
		return
	}
	w := c.words(t)
	if len(w) != len(got) {
		mm.add("%s: go=%d words sol=%d words", ctx, len(got), len(w))
		return
	}
	for i := range w {
		if !w[i].Eq(&got[i]) {
			mm.add("%s: word %d go=%s sol=%s", ctx, i, got[i].Dec(), w[i].Dec())
			return
		}
	}
}

func finBool(b bool) uint256.Int {
	if b {
		return *uint256.NewInt(1)
	}
	return uint256.Int{}
}

type finRouterRow struct {
	Header   bool           `json:"header"`
	Scenario string         `json:"scenario"`
	Kind     uint8          `json:"kind,string"`
	Now      uint64         `json:"now,string"`
	State    finRouterState `json:"state"`
	Views    struct {
		Positions         finCall `json:"positions"`
		Position0         finCall `json:"position0"`
		Position1         finCall `json:"position1"`
		Position2         finCall `json:"position2"`
		Quarantine0       finCall `json:"quarantine0"`
		Quarantine1       finCall `json:"quarantine1"`
		Drawn             finCall `json:"drawn"`
		MinLltv           finCall `json:"minLltv"`
		Reclaimable       finCall `json:"reclaimable"`
		ReclaimableP1zero finCall `json:"reclaimableP1zero"`
		ReclaimableBadLen finCall `json:"reclaimableBadLen"`
		VenueViews        []struct {
			VenuePosition finCall `json:"venuePosition"`
			Health        finCall `json:"health"`
			HealthLow     finCall `json:"healthLow"`
			Readable      finCall `json:"readable"`
			FreeLiquidity finCall `json:"freeLiquidity"`
		} `json:"venueViews"`
	} `json:"views"`
	Ceilings []struct {
		Idx   uint8       `json:"idx,string"`
		Coll  uint256.Int `json:"coll"`
		Price uint256.Int `json:"price"`
		Res   finCall     `json:"res"`
	} `json:"ceilings"`
	Id     uint16         `json:"id,string"`
	Idx    uint8          `json:"idx,string"`
	A      uint256.Int    `json:"a"`
	CollIn uint256.Int    `json:"collIn"`
	Price  uint256.Int    `json:"price"`
	Price1 uint256.Int    `json:"price1"`
	Prop   bool           `json:"prop"`
	Pre    uint256.Int    `json:"pre"`
	PreOk  bool           `json:"preOk"`
	PreRet string         `json:"preRet"`
	Ok     bool           `json:"ok"`
	Ret    string         `json:"ret"`
	Post   finRouterState `json:"post"`
}

const (
	finP0 = 788432322552395
	finP1 = 262810774184131
)

func finRouterViews(t *testing.T, mm *finMismatches, r *finRouterRow, p *mmRouter) {
	now := r.Now
	ctx := "scenario " + r.Scenario
	v := &r.Views
	pos, err := p.positions(now)
	if v.Positions.Ok && err == nil {
		w := v.Positions.words(t)
		cmp := func(name string, sol []uint256.Int, got []uint256.Int) {
			for i := range sol {
				if i >= len(got) || !sol[i].Eq(&got[i]) {
					mm.add("%s positions.%s[%d]: go=%v sol=%s", ctx, name, i, got, sol[i].Dec())
					return
				}
			}
		}
		cmp("coll", finDynArray(w, 0), pos.Coll)
		cmp("sup", finDynArray(w, 1), pos.Sup)
		cmp("debt", finDynArray(w, 2), pos.Debt)
		if !w[3].Eq(&pos.TotalColl) {
			mm.add("%s positions.total: go=%s sol=%s", ctx, pos.TotalColl.Dec(), w[3].Dec())
		}
	} else {
		v.Positions.checkWords(t, mm, ctx+" positions", nil, err)
	}
	for i, c := range []finCall{v.Position0, v.Position1, v.Position2} {
		a, b, d, err := p.position(uint8(i), now)
		c.checkWords(t, mm, fmt.Sprintf("%s position(%d)", ctx, i), []uint256.Int{a, b, d}, err)
	}
	for i, c := range []finCall{v.Quarantine0, v.Quarantine1} {
		q, err := p.quarantine(uint8(i), now)
		c.checkWords(t, mm, fmt.Sprintf("%s quarantine(%d)", ctx, i), []uint256.Int{finBool(q.Any), q.FrozenDebt, q.FrozenColl}, err)
	}
	count, mask, err := p.drawn(now)
	v.Drawn.checkWords(t, mm, ctx+" drawn", []uint256.Int{*uint256.NewInt(uint64(count)), *uint256.NewInt(uint64(mask))}, err)
	ml, err := p.minLltv(now)
	v.MinLltv.checkWords(t, mm, ctx+" minLltv", []uint256.Int{ml}, err)
	rc, err := p.reclaimable([]uint256.Int{*uint256.NewInt(finP0), *uint256.NewInt(finP1)}, now)
	v.Reclaimable.checkWords(t, mm, ctx+" reclaimable", []uint256.Int{rc}, err)
	rc, err = p.reclaimable([]uint256.Int{*uint256.NewInt(finP0), {}}, now)
	v.ReclaimableP1zero.checkWords(t, mm, ctx+" reclaimable(P1=0)", []uint256.Int{rc}, err)
	rc, err = p.reclaimable([]uint256.Int{{}}, now)
	v.ReclaimableBadLen.checkWords(t, mm, ctx+" reclaimable(len 1)", []uint256.Int{rc}, err)
	for i, vv := range v.VenueViews {
		id := uint16(i)
		price := uint256.NewInt(finP0)
		if i == 3 {
			price = uint256.NewInt(finP1)
		}
		vp, err := p.venuePosition(id, now)
		vv.VenuePosition.checkWords(t, mm, fmt.Sprintf("%s venuePosition(%d)", ctx, i), vp[:], err)
		h, err := p.venueHealth(id, price, now)
		vv.Health.checkWords(t, mm, fmt.Sprintf("%s venueHealth(%d)", ctx, i), []uint256.Int{h}, err)
		half := new(uint256.Int).Rsh(price, 1)
		h, err = p.venueHealth(id, half, now)
		vv.HealthLow.checkWords(t, mm, fmt.Sprintf("%s venueHealth(%d, p/2)", ctx, i), []uint256.Int{h}, err)
		rd, err := p.read(&p.Venues[i], now)
		vv.Readable.checkWords(t, mm, fmt.Sprintf("%s venueReadable(%d)", ctx, i), []uint256.Int{finBool(rd.Readable)}, err)
		fl := p.Venues[i].Morpho.freeLiquidity()
		vv.FreeLiquidity.checkWords(t, mm, fmt.Sprintf("%s venueFreeLiquidity(%d)", ctx, i), []uint256.Int{fl}, nil)
	}
	for _, c := range r.Ceilings {
		got, err := p.fundingCeiling(c.Idx, &c.Coll, &c.Price, now)
		c.Res.checkWords(t, mm, fmt.Sprintf("%s fundingCeiling(%d,%s,%s)", ctx, c.Idx, c.Coll.Dec(), c.Price.Dec()),
			[]uint256.Int{got}, err)
	}
}

func finRouterOp(t *testing.T, mm *finMismatches, r *finRouterRow, base *mmRouter, now uint64) {
	p := base.clone()
	ctx := fmt.Sprintf("scenario %s kind=%d idx=%d id=%d a=%s collIn=%s price=%s price1=%s prop=%v pre=%s", r.Scenario,
		r.Kind, r.Idx, r.Id, r.A.Dec(), r.CollIn.Dec(), r.Price.Dec(), r.Price1.Dec(), r.Prop, r.Pre.Dec())
	var got []uint256.Int
	var err error
	switch r.Kind {
	case 1:
		w, b, po, e := p.fund(r.Idx, &r.A, &r.CollIn, &r.Price, now)
		got, err = []uint256.Int{w, b, po}, e
	case 2:
		x, e := p.repayCascade(r.Idx, &r.A, now)
		got, err = []uint256.Int{x}, e
	case 3:
		x, e := p.supplyCascade(r.Idx, &r.A, now)
		got, err = []uint256.Int{x}, e
	case 4, 5:
		x, e := p.reclaim(&r.A, []uint256.Int{r.Price, r.Price1}, r.Kind == 4, now)
		got, err = []uint256.Int{x}, e
	case 6:
		got, err = []uint256.Int{}, p.borrow(r.Id, &r.A, &r.Price, now)
	case 7:
		x, e := p.repay(r.Id, &r.A, now)
		got, err = []uint256.Int{x}, e
	case 8:
		x, e := p.supply(r.Id, &r.A, now)
		got, err = []uint256.Int{x}, e
	case 9:
		x, e := p.withdrawSuppliedEntry(r.Id, &r.A, now)
		got, err = []uint256.Int{x}, e
	case 10:
		if r.Prop {
			q := p.clone()
			_, perr := q.repay(r.Id, &r.Pre, now)
			if r.PreOk != (perr == nil) {
				mm.add("%s: pre-repay go=%s sol ok=%v %s", ctx, errClass(perr), r.PreOk, r.PreRet)
				return
			}
			if perr == nil {
				p = q
			}
		}
		got, err = []uint256.Int{}, p.withdrawCollateral(r.Id, &r.A, &r.Price, r.Prop, now)
	case 11:
		got, err = []uint256.Int{}, p.postCollateral(r.Id, &r.A)
	}
	c := finCall{Ok: r.Ok, Ret: r.Ret}
	c.checkWords(t, mm, ctx, got, err)
	if !r.Ok || err != nil {
		return
	}
	for i := range r.Post.Venues {
		want := r.Post.router().Venues[i]
		have := p.Venues[i]
		if have.Morpho.Market != want.Morpho.Market || have.Morpho.Position != want.Morpho.Position ||
			!have.Morpho.RateAtTarget.Eq(&want.Morpho.RateAtTarget) || !have.ManagedCollateral.Eq(&want.ManagedCollateral) ||
			!have.ManagedSupplyShares.Eq(&want.ManagedSupplyShares) {
			mm.add("%s: post venue %d go={m:%+v p:%+v rat:%s mc:%s ms:%s} sol={m:%+v p:%+v rat:%s mc:%s ms:%s}", ctx, i,
				have.Morpho.Market, have.Morpho.Position, have.Morpho.RateAtTarget.Dec(), have.ManagedCollateral.Dec(),
				have.ManagedSupplyShares.Dec(), want.Morpho.Market, want.Morpho.Position, want.Morpho.RateAtTarget.Dec(),
				want.ManagedCollateral.Dec(), want.ManagedSupplyShares.Dec())
		}
	}
}

// TestRouterEdges replays the deployed MMRouter's views, funding ceilings and entries over 26 scenarios.
func TestRouterEdges(t *testing.T) {
	t.Parallel()
	var rows []finRouterRow
	loadGzFixture(t, "edges/mm_router_edges.json.gz", &rows)
	require.Greater(t, len(rows), 1000)
	mm := &finMismatches{t: t, area: "router"}
	var base *mmRouter
	var now uint64
	ops := 0
	for i := range rows[1:] {
		r := &rows[i+1]
		if r.Kind == 0 {
			base, now = r.State.router(), r.Now
			finRouterViews(t, mm, r, base.clone())
			continue
		}
		ops++
		finRouterOp(t, mm, r, base, now)
	}
	t.Logf("router scenarios+ops %d (ops %d), mismatches %d", len(rows)-1, ops, mm.n)
}

// ------------------------------------------------------------------ FLAMMGateLib

type finGateRouter struct {
	sup, debt []uint256.Int
	posted    uint256.Int
	qAny      []bool
	qDebt     []uint256.Int
	qColl     []uint256.Int
}

func (f *finGateRouter) positions(uint64) (mmPositions, error) {
	return mmPositions{Coll: make([]uint256.Int, len(f.sup)), Sup: f.sup, Debt: f.debt, TotalColl: f.posted}, nil
}

func (f *finGateRouter) quarantine(idx uint8, _ uint64) (mmQuarantine, error) {
	return mmQuarantine{Any: f.qAny[idx], FrozenDebt: f.qDebt[idx], FrozenColl: f.qColl[idx]}, nil
}

// finWord is an int256 as its two's-complement word.
func finWord(x gateInt) uint256.Int {
	var z uint256.Int
	if x.Neg {
		z.Neg(&x.Abs)
		return z
	}
	return x.Abs
}

func finGint(x signedDec) gateInt { return gateInt(x) }

type finGateRow struct {
	Header bool `json:"header"`
	In     struct {
		Tag         string        `json:"tag"`
		N           int           `json:"n,string"`
		Scale       []uint256.Int `json:"scale"`
		Liquid      []uint256.Int `json:"liquid"`
		Supplied    []uint256.Int `json:"supplied"`
		Debt        []uint256.Int `json:"debt"`
		Price       []uint256.Int `json:"price"`
		Cross       []uint256.Int `json:"cross"`
		Usd         []uint256.Int `json:"usd"`
		OkCross     []bool        `json:"okCross"`
		OkUsd       []bool        `json:"okUsd"`
		QAny        []bool        `json:"qAny"`
		QDebt       []uint256.Int `json:"qDebt"`
		QColl       []uint256.Int `json:"qColl"`
		U0          []signedDec   `json:"u0"`
		Physical    uint256.Int   `json:"physical"`
		Posted      uint256.Int   `json:"posted"`
		Ltv         uint256.Int   `json:"ltv"`
		Phi         uint256.Int   `json:"phi"`
		Eps         uint256.Int   `json:"eps"`
		Gross0      uint256.Int   `json:"gross0"`
		Quarantined bool          `json:"quarantined"`
		UArg        signedDec     `json:"uArg"`
		HeadArg     uint256.Int   `json:"headArg"`
		LtvArg      uint256.Int   `json:"ltvArg"`
		PhiArg      uint256.Int   `json:"phiArg"`
		LltvArg     uint256.Int   `json:"lltvArg"`
		RouterLegs  int           `json:"routerLegs,string"`
	} `json:"in"`
	Legs []struct {
		NetL18         finCall `json:"netL18"`
		NetPW          finCall `json:"netPW"`
		RequiredPosted finCall `json:"requiredPosted"`
		HeadOf         finCall `json:"headOf"`
		RoomNative     finCall `json:"roomNative"`
	} `json:"legs"`
	ExposurePW        finCall `json:"exposurePW"`
	Gross             finCall `json:"gross"`
	BoundPW           finCall `json:"boundPW"`
	BoundOf           finCall `json:"boundOf"`
	Lift              finCall `json:"lift"`
	RoomWad           finCall `json:"roomWad"`
	StructuralDistWad finCall `json:"structuralDistWad"`
	RequiredPostedAll finCall `json:"requiredPostedAll"`
	NavAt             finCall `json:"navAt"`
	Context           finCall `json:"context"`
	AssertGate        finCall `json:"assertGate"`
	Anchor            finCall `json:"anchor"`
	Entry             finCall `json:"entry"`
	Exit              finCall `json:"exit"`
	TotalAssets       finCall `json:"totalAssets"`
}

// finGateWrapped reports a leg whose debt*scale or (supplied+liquid)*scale lands in [2^255, 2^256): Solidity's
// explicit uint256 -> int256 cast in netL18 wraps there. Those rows are compared like every other; the count is
// logged to show the wrap region stays covered.
func finGateWrapped(L *gateLeg) bool {
	var half uint256.Int
	half.Lsh(uOne, 255)
	d, err1 := gateMul(&L.Debt, &L.Scale)
	a, err2 := gateAdd(&L.Supplied, &L.Liquid)
	if err2 == nil {
		a, err2 = gateMul(&a, &L.Scale)
	}
	return (err1 == nil && !d.Lt(&half)) || (err2 == nil && !a.Lt(&half))
}

// TestGateEdges replays FLAMMGateLib's pure law and storage composites.
func TestGateEdges(t *testing.T) {
	t.Parallel()
	var rows []finGateRow
	loadGzFixture(t, "edges/gate_edges.json.gz", &rows)
	require.Greater(t, len(rows), 400)
	finGateReplay(t, "gate", rows)
}

// finGateReplay compares every harness call of every gate row (header element first) with the port.
func finGateReplay(t *testing.T, area string, rows []finGateRow) {
	mm := &finMismatches{t: t, area: area}
	wrapped := 0
	for ri := range rows[1:] {
		r := &rows[ri+1]
		in := &r.In
		b := gateBook{Physical: in.Physical, Posted: in.Posted}
		wrap := false
		for i := 0; i < in.N; i++ {
			b.Legs = append(b.Legs, gateLeg{Liquid: in.Liquid[i], Supplied: in.Supplied[i], Debt: in.Debt[i],
				Scale: in.Scale[i], PriceWad: in.Price[i], CrossWad: in.Cross[i]})
			wrap = wrap || finGateWrapped(&b.Legs[i])
		}
		pool := &gatePool{Physical: in.Physical, LtvWad: in.Ltv, PhiWad: in.Phi, RoomEpsilonWad: in.Eps}
		for i := 0; i < in.N; i++ {
			pool.Loans = append(pool.Loans, gateLoanCfg{Scale: in.Scale[i], Liquid: in.Liquid[i]})
			var pw, cw uint256.Int
			if in.OkCross[i] {
				pw = in.Price[i]
			}
			if i == 0 {
				cw = *uWad
			} else if in.OkUsd[0] && in.OkUsd[i] && !in.Usd[0].IsZero() {
				z, overflow := mulDiv(&in.Usd[i], uWad, &in.Usd[0])
				require.False(t, overflow)
				cw = *z
			}
			pool.PriceWad = append(pool.PriceWad, pw)
			pool.CrossWad = append(pool.CrossWad, cw)
		}
		// the composites price the book from the feed
		for i := 0; i < in.N; i++ {
			L := gateLeg{Liquid: in.Liquid[i], Supplied: in.Supplied[i], Debt: in.Debt[i], Scale: in.Scale[i]}
			wrap = wrap || finGateWrapped(&L)
		}
		if wrap {
			wrapped++
		}
		fr := &finGateRouter{sup: in.Supplied[:in.RouterLegs], debt: in.Debt[:in.RouterLegs], posted: in.Posted,
			qAny: in.QAny, qDebt: in.QDebt, qColl: in.QColl}
		ctx := fmt.Sprintf("row %d %s in=%+v", ri, in.Tag, *in)
		for i := 0; i < in.N; i++ {
			lc := fmt.Sprintf("%s leg %d", ctx, i)
			n, err := gateNetL18(&b.Legs[i])
			r.Legs[i].NetL18.checkWords(t, mm, lc+" netL18", []uint256.Int{finWord(n)}, err)
			n, err = gateNetPW(&b.Legs[i])
			r.Legs[i].NetPW.checkWords(t, mm, lc+" netPW", []uint256.Int{finWord(n)}, err)
			x, err := gateRequiredPosted(&in.Debt[i], &in.Scale[i], &in.LtvArg, &in.Price[i])
			r.Legs[i].RequiredPosted.checkWords(t, mm, lc+" requiredPosted", []uint256.Int{x}, err)
			x, err = gateHeadOf(&b, i, &in.HeadArg, &in.LtvArg)
			r.Legs[i].HeadOf.checkWords(t, mm, lc+" headOf", []uint256.Int{x}, err)
			x, err = gateRoomNative(pool, &b, i, &in.HeadArg)
			r.Legs[i].RoomNative.checkWords(t, mm, lc+" roomNative", []uint256.Int{x}, err)
		}
		x, err := gateExposurePW(&b)
		r.ExposurePW.checkWords(t, mm, ctx+" exposurePW", []uint256.Int{x}, err)
		x, err = gateGross(&b)
		r.Gross.checkWords(t, mm, ctx+" gross", []uint256.Int{x}, err)
		x, err = gateBoundPW(&in.Physical, &in.LtvArg)
		r.BoundPW.checkWords(t, mm, ctx+" boundPW", []uint256.Int{x}, err)
		x, err = gateBoundOf(&in.Physical, &in.Price[0], &in.LtvArg)
		r.BoundOf.checkWords(t, mm, ctx+" boundOf", []uint256.Int{x}, err)
		x, err = gateLift(&in.HeadArg, &in.LtvArg, &in.PhiArg)
		r.Lift.checkWords(t, mm, ctx+" lift", []uint256.Int{x}, err)
		x, err = gateRoomWad(finGint(in.UArg), &in.Physical, &in.Price[0], &in.LtvArg, &in.PhiArg)
		r.RoomWad.checkWords(t, mm, ctx+" roomWad", []uint256.Int{x}, err)
		x = gateStructuralDistWad(&in.LtvArg, &in.LltvArg)
		r.StructuralDistWad.checkWords(t, mm, ctx+" structuralDistWad", []uint256.Int{x}, nil)
		x, err = gateRequiredPostedAll(&b, &in.LtvArg)
		r.RequiredPostedAll.checkWords(t, mm, ctx+" requiredPostedAll", []uint256.Int{x}, err)
		x, err = gateNavAt(&b)
		r.NavAt.checkWords(t, mm, ctx+" navAt", []uint256.Int{x}, err)
		var ts uint256.Int
		ts.Mod(&in.HeadArg, new(uint256.Int).Lsh(uOne, 48))
		c, err := gateContext(&b, &in.Price[0], ts.Uint64(), &in.Gross0)
		r.Context.checkWords(t, mm, ctx+" context", []uint256.Int{c.PhysicalPoolAsset, c.PostedPoolAsset, c.LiquidLoanAsset,
			c.SuppliedLoanAsset, c.DebtLoanAsset, c.ShareSupply, c.PriceWad, *uint256.NewInt(c.PriceTs),
			*uint256.NewInt(uint64(c.LoanCount))}, err)

		now := uint64(0)
		r.AssertGate.checkWords(t, mm, ctx+" assertGate", []uint256.Int{}, gateAssertGate(pool, fr, now))
		u0, g0, q, err := gateAnchor(pool, fr, now)
		if r.Anchor.Ok && err == nil {
			w := r.Anchor.words(t)
			got := []uint256.Int{w[0], g0, finBool(q), *uint256.NewInt(uint64(len(u0)))}
			for _, u := range u0 {
				got = append(got, finWord(u))
			}
			r.Anchor.checkWords(t, mm, ctx+" anchor", got, nil)
		} else {
			r.Anchor.checkWords(t, mm, ctx+" anchor", nil, err)
		}
		inU0 := make([]gateInt, in.N)
		for i := range inU0 {
			inU0[i] = finGint(in.U0[i])
		}
		r.Entry.checkWords(t, mm, ctx+" assertEntryGate", []uint256.Int{},
			gateAssertEntryGate(pool, fr, now, inU0, in.Gross0, in.Quarantined))
		r.Exit.checkWords(t, mm, ctx+" assertExitNotWorsened", []uint256.Int{},
			gateAssertExitNotWorsened(pool, fr, now, inU0, in.Gross0))
		x, err = gateTotalAssets(pool, fr, now)
		r.TotalAssets.checkWords(t, mm, ctx+" totalAssets", []uint256.Int{x}, err)
	}
	t.Logf("%s rows %d (int256-wrap rows %d), mismatches %d", area, len(rows)-1, wrapped, mm.n)
}

// ------------------------------------------------------------------ FLAMMSwapLib settlement

type finSettlePool struct {
	Physical      uint256.Int `json:"physical"`
	Liquid        uint256.Int `json:"liquid"`
	ReserveTarget uint256.Int `json:"reserveTarget"`
	Features      uint256.Int `json:"features"`
	Ltv           uint256.Int `json:"ltv"`
	Phi           uint256.Int `json:"phi"`
	Eps           uint256.Int `json:"eps"`
	PriceWad      uint256.Int `json:"priceWad"`
}

type finSettleRow struct {
	Header     bool           `json:"header"`
	Tag        string         `json:"tag"`
	Buy        bool           `json:"buy"`
	AmountIn   uint256.Int    `json:"amountIn"`
	Now        uint64         `json:"now,string"`
	PrePool    finSettlePool  `json:"prePool"`
	PreRouter  finRouterState `json:"preRouter"`
	PreviewOk  bool           `json:"previewOk"`
	Used       uint256.Int    `json:"used"`
	Net        uint256.Int    `json:"net"`
	Ok         bool           `json:"ok"`
	Ret        string         `json:"ret"`
	SwapUsed   uint256.Int    `json:"swapUsed"`
	SwapNet    uint256.Int    `json:"swapNet"`
	PostPool   finSettlePool  `json:"postPool"`
	PostRouter finRouterState `json:"postRouter"`
}

// TestSettleEdges replays real swaps' settlement legs (the preview's used / net) against mmSettleSell / mmSettleBuy.
func TestSettleEdges(t *testing.T) {
	t.Parallel()
	var rows []finSettleRow
	loadGzFixture(t, "edges/mm_settle_edges.json.gz", &rows)
	require.Greater(t, len(rows), 150)
	mm := &finMismatches{t: t, area: "settle"}
	settled, reverted := 0, 0
	for i := range rows[1:] {
		r := &rows[i+1]
		if !r.PreviewOk {
			continue
		}
		pp := &r.PrePool
		pool := &gatePool{Physical: pp.Physical, LtvWad: pp.Ltv, PhiWad: pp.Phi, RoomEpsilonWad: pp.Eps, Features: pp.Features,
			Loans:    []gateLoanCfg{{Scale: r.PreRouter.Loans[0].Scale, Liquid: pp.Liquid, ReserveTarget: pp.ReserveTarget}},
			PriceWad: []uint256.Int{pp.PriceWad}, CrossWad: []uint256.Int{*uWad}}
		router := r.PreRouter.router()
		var err error
		if r.Buy {
			err = mmSettleBuy(pool, router, 0, &r.Used, &r.Net, r.Now)
		} else {
			err = mmSettleSell(pool, router, 0, &pool.PriceWad[0], &r.Used, &r.Net, r.Now)
		}
		ctx := fmt.Sprintf("row %d %s buy=%v in=%s used=%s net=%s pre=%+v", i, r.Tag, r.Buy, r.AmountIn.Dec(), r.Used.Dec(),
			r.Net.Dec(), *pp)
		c := finCall{Ok: r.Ok, Ret: r.Ret}
		c.checkWords(t, mm, ctx, []uint256.Int{}, err)
		if !r.Ok {
			reverted++
			continue
		}
		if err != nil {
			continue
		}
		settled++
		require.True(t, r.SwapUsed.Eq(&r.Used) && r.SwapNet.Eq(&r.Net), "preview/execute diverged: %s", ctx)
		if !pool.Physical.Eq(&r.PostPool.Physical) || !pool.Loans[0].Liquid.Eq(&r.PostPool.Liquid) {
			mm.add("%s: post pool go=(physical %s, liquid %s) sol=(physical %s, liquid %s)", ctx, pool.Physical.Dec(),
				pool.Loans[0].Liquid.Dec(), r.PostPool.Physical.Dec(), r.PostPool.Liquid.Dec())
		}
		want := r.PostRouter.router()
		for k := range want.Venues {
			have, w := &router.Venues[k], &want.Venues[k]
			if have.Morpho.Market != w.Morpho.Market || have.Morpho.Position != w.Morpho.Position ||
				!have.Morpho.RateAtTarget.Eq(&w.Morpho.RateAtTarget) || !have.ManagedCollateral.Eq(&w.ManagedCollateral) ||
				!have.ManagedSupplyShares.Eq(&w.ManagedSupplyShares) {
				mm.add("%s: post venue %d go={m:%+v p:%+v rat:%s mc:%s ms:%s} sol={m:%+v p:%+v rat:%s mc:%s ms:%s}", ctx, k,
					have.Morpho.Market, have.Morpho.Position, have.Morpho.RateAtTarget.Dec(), have.ManagedCollateral.Dec(),
					have.ManagedSupplyShares.Dec(), w.Morpho.Market, w.Morpho.Position, w.Morpho.RateAtTarget.Dec(),
					w.ManagedCollateral.Dec(), w.ManagedSupplyShares.Dec())
			}
		}
	}
	t.Logf("settle rows %d, settled %d, settlement reverts %d, mismatches %d", len(rows)-1, settled, reverted, mm.n)
}
