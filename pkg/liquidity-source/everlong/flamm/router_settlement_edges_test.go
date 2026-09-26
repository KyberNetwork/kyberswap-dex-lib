package everlongflamm

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// Settlement edges for the financing module. The fixtures under testdata/edges/ come from
// testdata/gen/{RouterSettlementEdges,MorphoMarketEdges}.t.sol, run on a Base fork at block 51317000:
//   - router_settlement_edges_one_loan / _two_loans: the settlement legs (FLAMMSwapLib.settleSell, the buy branch
//     of execute, payLoan, takeLoan, releaseExcess), the gate composites and the Router entries, executed by
//     DELEGATECALL into the libraries the deployed pool implementation links, over the live pool record extended to
//     three USDC venues (and a fourth venue on a second loan asset in _two_loans), with amounts placed on the
//     thresholds each state implies rather than bounded by a preview;
//   - mm_market_edges: Morpho Blue transitions, the AdaptiveCurveIrm and the MorphoBlueAccount views and mutators on
//     realistic-magnitude states with 1-wei neighbours of every branch threshold.
// Every array opens with a header element.

func settleHex(t *testing.T, h string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(h, "0x"))
	require.NoError(t, err)
	return b
}

// settleRevert maps revert data onto the port's error vocabulary (nil for unmapped data, which fails the row).
func settleRevert(b []byte) error {
	if len(b) == 0 {
		return errMulDivOverflow
	}
	switch string(b) {
	case "irm down":
		return errMMIrmReverted
	case "oracle down":
		return errMMOracleReverted
	}
	if len(b) < 4 {
		return nil
	}
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
		return nil
	case "08c379a0":
		n := new(uint256.Int).SetBytes(b[36:68]).Uint64()
		switch string(b[68 : 68+n]) {
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
		case "oracle down":
			return errMMOracleReverted
		}
		return nil
	}
	return revertSelectors[sel]
}

type settleReport struct {
	t    *testing.T
	area string
	n    int
}

func (m *settleReport) add(format string, args ...any) {
	m.t.Helper()
	m.n++
	if m.n <= 60 {
		m.t.Errorf("[%s] "+format, append([]any{m.area}, args...)...)
	}
}

// settleArray decodes the dynamic uint256[] whose head offset sits in word `at` of `b`.
func settleArray(b []byte, at int) []uint256.Int {
	w := abiWords(b)
	off := int(w[at].Uint64()) / 32
	n := int(w[off].Uint64())
	return w[off+1 : off+1+n]
}

// settleUnwrap strips the `bytes` return of the harness's gate(bytes4).
func settleUnwrap(b []byte) []byte {
	w := abiWords(b)
	off := int(w[0].Uint64())
	n := int(w[off/32].Uint64())
	return b[off+32 : off+32+n]
}

// ------------------------------------------------------------------ state

type settleMarket struct {
	Tsa        uint256.Int `json:"tsa"`
	Tss        uint256.Int `json:"tss"`
	Tba        uint256.Int `json:"tba"`
	Tbs        uint256.Int `json:"tbs"`
	LastUpdate uint256.Int `json:"lastUpdate"`
	Fee        uint256.Int `json:"fee"`
}

func (m *settleMarket) state() mmMarket {
	return mmMarket{TotalSupplyAssets: m.Tsa, TotalSupplyShares: m.Tss, TotalBorrowAssets: m.Tba,
		TotalBorrowShares: m.Tbs, LastUpdate: m.LastUpdate, Fee: m.Fee}
}

type settlePosition struct {
	SupplyShares uint256.Int `json:"supplyShares"`
	BorrowShares uint256.Int `json:"borrowShares"`
	Collateral   uint256.Int `json:"collateral"`
}

func (p *settlePosition) state() mmPosition {
	return mmPosition{SupplyShares: p.SupplyShares, BorrowShares: p.BorrowShares, Collateral: p.Collateral}
}

type settleVenue struct {
	LoanIndex     uint8          `json:"loanIndex,string"`
	Lltv          uint256.Int    `json:"lltv"`
	BorrowEnabled bool           `json:"borrowEnabled"`
	SupplyEnabled bool           `json:"supplyEnabled"`
	Retired       bool           `json:"retired"`
	DebtCap       uint256.Int    `json:"debtCap"`
	SupplyCap     uint256.Int    `json:"supplyCap"`
	MaxRate       uint256.Int    `json:"maxRate"`
	ManagedColl   uint256.Int    `json:"managedColl"`
	ManagedShares uint256.Int    `json:"managedShares"`
	HasIrm        bool           `json:"hasIrm"`
	IrmDead       bool           `json:"irmDead"`
	Rat           signedDec      `json:"rat"`
	OracleOk      bool           `json:"oracleOk"`
	OraclePrice   uint256.Int    `json:"oraclePrice"`
	OracleZero    bool           `json:"oracleZero"`
	Market        settleMarket   `json:"market"`
	Position      settlePosition `json:"position"`
}

type settleLoan struct {
	Decimals      uint8       `json:"decimals,string"`
	Scale         uint256.Int `json:"scale"`
	DebtCap       uint256.Int `json:"debtCap"`
	SupplyCap     uint256.Int `json:"supplyCap"`
	BorrowEnabled bool        `json:"borrowEnabled"`
	Retired       bool        `json:"retired"`
}

type settleRouter struct {
	Paused   bool          `json:"paused"`
	Pin      uint256.Int   `json:"pin"`
	Gap      uint256.Int   `json:"gap"`
	Band     uint256.Int   `json:"band"`
	MaxDrawn uint8         `json:"maxDrawn,string"`
	Bo       []uint16      `json:"bo"`
	So       []uint16      `json:"so"`
	Wo       []uint16      `json:"wo"`
	Ro       []uint16      `json:"ro"`
	Loans    []settleLoan  `json:"loans"`
	Venues   []settleVenue `json:"venues"`
}

type settlePool struct {
	Physical uint256.Int   `json:"physical"`
	Features uint256.Int   `json:"features"`
	Ltv      uint256.Int   `json:"ltv"`
	Phi      uint256.Int   `json:"phi"`
	Eps      uint256.Int   `json:"eps"`
	Liquid   []uint256.Int `json:"liquid"`
	Reserve  []uint256.Int `json:"reserve"`
	Scale    []uint256.Int `json:"scale"`
	PriceWad []uint256.Int `json:"priceWad"`
	CrossWad []uint256.Int `json:"crossWad"`
}

type settleState struct {
	Now    uint64       `json:"now,string"`
	Pool   settlePool   `json:"pool"`
	Router settleRouter `json:"router"`
}

func (s *settleState) build() (*gatePool, *mmRouter) {
	p := &gatePool{Physical: s.Pool.Physical, LtvWad: s.Pool.Ltv, PhiWad: s.Pool.Phi, RoomEpsilonWad: s.Pool.Eps,
		Features: s.Pool.Features, PriceWad: s.Pool.PriceWad, CrossWad: s.Pool.CrossWad}
	for i := range s.Pool.Liquid {
		p.Loans = append(p.Loans, gateLoanCfg{Scale: s.Pool.Scale[i], Liquid: s.Pool.Liquid[i], ReserveTarget: s.Pool.Reserve[i]})
	}
	x := &s.Router
	r := &mmRouter{GlobalPaused: x.Paused, PinLtvWad: x.Pin, SafetyGapWad: x.Gap, OracleBandWad: x.Band,
		MaxDrawnAssets: x.MaxDrawn, BorrowOrder: x.Bo, SupplyOrder: x.So, WithdrawOrder: x.Wo, RepayOrder: x.Ro}
	for _, l := range x.Loans {
		r.Loans = append(r.Loans, mmLoan{Decimals: l.Decimals, LoanScale: l.Scale, DebtCap: l.DebtCap,
			SupplyCap: l.SupplyCap, BorrowEnabled: l.BorrowEnabled, Retired: l.Retired})
	}
	for _, v := range x.Venues {
		r.Venues = append(r.Venues, mmVenue{
			Morpho: mmVenueMarket{Market: v.Market.state(), Position: v.Position.state(), Lltv: v.Lltv,
				HasIrm: v.HasIrm, IrmReadable: !v.IrmDead, RateAtTarget: v.Rat.Abs, OracleOk: v.OracleOk,
				OraclePrice: v.OraclePrice, OracleZero: v.OracleZero},
			LoanIndex: v.LoanIndex, LltvWad: v.Lltv, BorrowEnabled: v.BorrowEnabled, SupplyEnabled: v.SupplyEnabled,
			Retired: v.Retired, DebtCap: v.DebtCap, SupplyCap: v.SupplyCap, MaxBorrowRateWad: v.MaxRate,
			ManagedCollateral: v.ManagedColl, ManagedSupplyShares: v.ManagedShares})
	}
	return p, r
}

// settleDiff compares the written parts of a post-state.
func settleDiff(p *gatePool, r *mmRouter, want *settleState) string {
	var d []string
	if !p.Physical.Eq(&want.Pool.Physical) {
		d = append(d, fmt.Sprintf("physical go=%s sol=%s", p.Physical.Dec(), want.Pool.Physical.Dec()))
	}
	for i := range p.Loans {
		if !p.Loans[i].Liquid.Eq(&want.Pool.Liquid[i]) {
			d = append(d, fmt.Sprintf("liquid[%d] go=%s sol=%s", i, p.Loans[i].Liquid.Dec(), want.Pool.Liquid[i].Dec()))
		}
	}
	for i := range r.Venues {
		v, w := &r.Venues[i], &want.Router.Venues[i]
		if wm := w.Market.state(); v.Morpho.Market != wm {
			d = append(d, fmt.Sprintf("v%d market go=%+v sol=%+v", i, v.Morpho.Market, wm))
		}
		if wp := w.Position.state(); v.Morpho.Position != wp {
			d = append(d, fmt.Sprintf("v%d position go=%+v sol=%+v", i, v.Morpho.Position, wp))
		}
		if !v.Morpho.RateAtTarget.Eq(&w.Rat.Abs) {
			d = append(d, fmt.Sprintf("v%d rat go=%s sol=%s", i, v.Morpho.RateAtTarget.Dec(), w.Rat.Abs.Dec()))
		}
		if !v.ManagedCollateral.Eq(&w.ManagedColl) || !v.ManagedSupplyShares.Eq(&w.ManagedShares) {
			d = append(d, fmt.Sprintf("v%d managed go=(%s,%s) sol=(%s,%s)", i, v.ManagedCollateral.Dec(),
				v.ManagedSupplyShares.Dec(), w.ManagedColl.Dec(), w.ManagedShares.Dec()))
		}
	}
	return strings.Join(d, "; ")
}

// ------------------------------------------------------------------ settlement fixtures

type settleRow struct {
	Header   bool            `json:"header"`
	Scenario string          `json:"scenario"`
	S        int             `json:"s"`
	State    *settleState    `json:"state"`
	Op       string          `json:"op"`
	Args     json.RawMessage `json:"args"`
	Ok       bool            `json:"ok"`
	Ret      string          `json:"ret"`
	Post     *settleState    `json:"post"`
}

type settleGateArgs struct {
	U0 []signedDec `json:"u0"`
	G0 uint256.Int `json:"g0"`
	Q0 bool        `json:"q0"`
}

func (a *settleGateArgs) u0() []gateInt {
	u := make([]gateInt, len(a.U0))
	for i, x := range a.U0 {
		u[i] = gateInt(x)
	}
	return u
}

func settleReplay(t *testing.T, file string) {
	var rows []settleRow
	loadGzFixture(t, file, &rows)
	require.Greater(t, len(rows), 300)
	mm := &settleReport{t: t, area: file}
	states := map[int]*settleState{}
	tags := map[int]string{}
	var nOps int
	for i := 1; i < len(rows); i++ {
		row := &rows[i]
		if row.State != nil {
			states[row.S], tags[row.S] = row.State, row.Scenario
			continue
		}
		st := states[row.S]
		require.NotNil(t, st, "row %d", i)
		nOps++
		pool, r := st.build()
		now := st.Now
		var args []uint256.Int
		var gargs settleGateArgs
		if len(row.Args) > 0 && row.Args[0] == '[' {
			var ss []string
			require.NoError(t, json.Unmarshal(row.Args, &ss))
			args = make([]uint256.Int, len(ss))
			for k, s := range ss {
				require.NoError(t, args[k].SetFromDecimal(s))
			}
		} else {
			require.NoError(t, json.Unmarshal(row.Args, &gargs))
		}
		ret := settleHex(t, row.Ret)
		ctx := fmt.Sprintf("row %d scenario=%s op=%s args=%s", i, tags[row.S], row.Op, string(row.Args))
		var err error
		var outs []uint256.Int
		mutates := false
		a := func(k int) *uint256.Int { return &args[k] }
		idx := func(k int) uint8 { return uint8(args[k].Uint64()) }
		switch row.Op {
		case "positions":
			var ps mmPositions
			ps, err = r.positions(now)
			if err == nil {
				outs = append(append(append(append(outs, ps.Coll...), ps.Sup...), ps.Debt...), ps.TotalColl)
			}
		case "drawn":
			var c, m uint8
			c, m, err = r.drawn(now)
			outs = []uint256.Int{*uint256.NewInt(uint64(c)), *uint256.NewInt(uint64(m))}
		case "minLltv":
			var l uint256.Int
			l, err = r.minLltv(now)
			outs = []uint256.Int{l}
		case "reclaimable":
			var x uint256.Int
			x, err = r.reclaimable(pool.PriceWad, now)
			outs = []uint256.Int{x}
		case "quarantine":
			var q mmQuarantine
			q, err = r.quarantine(idx(0), now)
			b := uint256.Int{}
			if q.Any {
				b.SetOne()
			}
			outs = []uint256.Int{b, q.FrozenDebt, q.FrozenColl}
		case "fundingCeiling":
			var x uint256.Int
			x, err = r.fundingCeiling(idx(0), a(1), a(2), now)
			outs = []uint256.Int{x}
		case "totalAssets":
			var x uint256.Int
			x, err = gateTotalAssets(pool, r, now)
			outs = []uint256.Int{x}
		case "assertGate":
			err = gateAssertGate(pool, r, now)
		case "anchor":
			var u0 []gateInt
			var g0 uint256.Int
			var q0 bool
			u0, g0, q0, err = gateAnchor(pool, r, now)
			if err == nil && row.Ok {
				inner := settleUnwrap(ret)
				w := abiWords(inner)
				arr := settleArray(inner, 0)
				if len(arr) != len(u0) {
					mm.add("%s: anchor length go=%d sol=%d", ctx, len(u0), len(arr))
					continue
				}
				for k := range arr {
					var want gateInt
					if arr[k].Sign() < 0 {
						want.Neg = true
						want.Abs.Neg(&arr[k])
					} else {
						want.Abs = arr[k]
					}
					if want != u0[k] {
						mm.add("%s: anchor u0[%d] go=%+v sol=%+v", ctx, k, u0[k], want)
					}
				}
				if !g0.Eq(&w[1]) || q0 != !w[2].IsZero() {
					mm.add("%s: anchor go=(%s,%v) sol=(%s,%v)", ctx, g0.Dec(), q0, w[1].Dec(), !w[2].IsZero())
				}
			}
		case "entryGate":
			err = gateAssertEntryGate(pool, r, now, gargs.u0(), gargs.G0, gargs.Q0)
		case "exitGate":
			err = gateAssertExitNotWorsened(pool, r, now, gargs.u0(), gargs.G0)
		case "sell":
			mutates = true
			err = mmSettleSell(pool, r, idx(0), a(1), a(2), a(3), now)
		case "buy":
			mutates = true
			err = mmSettleBuy(pool, r, idx(0), a(1), a(2), now)
		case "release":
			mutates = true
			err = mmReleaseExcess(pool, r, now)
		case "take":
			mutates = true
			err = mmTakeLoan(pool, r, idx(0), a(1), now)
		case "pay":
			mutates = true
			err = mmPayLoan(pool, r, idx(0), a(1), a(2), now)
		case "fund":
			mutates = true
			var w, b, p uint256.Int
			w, b, p, err = r.fund(idx(0), a(1), a(2), a(3), now)
			outs = []uint256.Int{w, b, p}
		case "repayCascade":
			mutates = true
			var x uint256.Int
			x, err = r.repayCascade(idx(0), a(1), now)
			outs = []uint256.Int{x}
		case "supplyCascade":
			mutates = true
			var x uint256.Int
			x, err = r.supplyCascade(idx(0), a(1), now)
			outs = []uint256.Int{x}
		case "reclaim", "reclaimBestEffort":
			mutates = true
			var x uint256.Int
			x, err = r.reclaim(a(0), pool.PriceWad, row.Op == "reclaim", now)
			outs = []uint256.Int{x}
		case "borrow":
			mutates = true
			err = r.borrow(uint16(args[0].Uint64()), a(1), a(2), now)
		case "withdrawSupplied":
			mutates = true
			var x uint256.Int
			x, err = r.withdrawSuppliedEntry(uint16(args[0].Uint64()), a(1), now)
			outs = []uint256.Int{x}
		case "postCollateral":
			mutates = true
			err = r.postCollateral(uint16(args[0].Uint64()), a(1))
		case "withdrawCollateral":
			mutates = true
			err = r.withdrawCollateral(uint16(args[0].Uint64()), a(1), a(2), !args[3].IsZero(), now)
		case "supply":
			mutates = true
			var x uint256.Int
			x, err = r.supply(uint16(args[0].Uint64()), a(1), now)
			outs = []uint256.Int{x}
		case "repayWithdraw":
			// MMRouter.repay then a proportional withdrawCollateral inside one transaction
			mutates = true
			id := uint16(args[0].Uint64())
			if _, err = r.repay(id, a(1), now); err == nil {
				err = r.withdrawCollateral(id, a(2), uZero, true, now)
			}
		default:
			t.Fatalf("unknown op %s", row.Op)
		}
		if !row.Ok {
			want := settleRevert(ret)
			if want == nil || !errors.Is(err, want) {
				mm.add("%s: go=%s sol=revert %s (%s)", ctx, errClass(err), row.Ret, errClass(want))
			}
			continue
		}
		if err != nil {
			mm.add("%s: go=%s sol=ok", ctx, err)
			continue
		}
		if len(outs) > 0 && row.Op != "anchor" {
			data := ret
			if row.Op == "totalAssets" {
				data = settleUnwrap(ret)
			}
			var want []uint256.Int
			if row.Op == "positions" {
				want = append(append(append(settleArray(data, 0), settleArray(data, 1)...), settleArray(data, 2)...), abiWords(data)[3])
			} else {
				want = abiWords(data)
			}
			if len(want) != len(outs) {
				mm.add("%s: outs go=%v sol=%v", ctx, outs, want)
			} else {
				for k := range outs {
					if !outs[k].Eq(&want[k]) {
						mm.add("%s: out[%d] go=%s sol=%s", ctx, k, outs[k].Dec(), want[k].Dec())
					}
				}
			}
		}
		if mutates {
			require.NotNil(t, row.Post, ctx)
			if d := settleDiff(pool, r, row.Post); d != "" {
				mm.add("%s: post-state %s", ctx, d)
			}
		}
	}
	t.Logf("%s: %d ops over %d scenarios, mismatches %d", file, nOps, len(states), mm.n)
}

// TestRouterSettlementEdges replays the settlement / gate / Router op grid of both runs.
func TestRouterSettlementEdges(t *testing.T) {
	t.Parallel()
	settleReplay(t, "edges/router_settlement_edges_one_loan.json.gz")
	settleReplay(t, "edges/router_settlement_edges_two_loans.json.gz")
}

// ------------------------------------------------------------------ Blue / IRM / account fixtures

type blueState struct {
	Tsa        uint256.Int `json:"tsa"`
	Tss        uint256.Int `json:"tss"`
	Tba        uint256.Int `json:"tba"`
	Tbs        uint256.Int `json:"tbs"`
	Elapsed    uint64      `json:"elapsed,string"`
	Fee        uint256.Int `json:"fee"`
	Sup        uint256.Int `json:"sup"`
	Bor        uint256.Int `json:"bor"`
	Coll       uint256.Int `json:"coll"`
	Rat        signedDec   `json:"rat"`
	OracleP    uint256.Int `json:"oracleP"`
	OracleMode uint8       `json:"oracleMode,string"`
	IrmDead    bool        `json:"irmDead"`
	NoIrm      bool        `json:"noIrm"`
}

type blueRow struct {
	Header       bool           `json:"header"`
	Tag          string         `json:"tag"`
	Now          uint64         `json:"now,string"`
	Lltv         uint256.Int    `json:"lltv"`
	St           blueState      `json:"st"`
	Op           uint8          `json:"op,string"`
	A            uint256.Int    `json:"a"`
	B            uint256.Int    `json:"b"`
	Ok           bool           `json:"ok"`
	Ret          string         `json:"ret"`
	PostMarket   settleMarket   `json:"postMarket"`
	PostPosition settlePosition `json:"postPosition"`
	PostRat      signedDec      `json:"postRat"`
}

// TestMorphoMarketEdges replays the Blue / IRM / account rows.
func TestMorphoMarketEdges(t *testing.T) {
	t.Parallel()
	var rows []blueRow
	loadGzFixture(t, "edges/mm_market_edges.json.gz", &rows)
	require.Greater(t, len(rows), 1000)
	mm := &settleReport{t: t, area: "blue"}
	boolWord := func(b bool) uint256.Int {
		var z uint256.Int
		if b {
			z.SetOne()
		}
		return z
	}
	for i := 1; i < len(rows); i++ {
		r := &rows[i]
		s := &r.St
		var lu uint256.Int
		lu.SetUint64(r.Now - s.Elapsed)
		v := mmVenueMarket{
			Market: mmMarket{TotalSupplyAssets: s.Tsa, TotalSupplyShares: s.Tss, TotalBorrowAssets: s.Tba,
				TotalBorrowShares: s.Tbs, LastUpdate: lu, Fee: s.Fee},
			Position: mmPosition{SupplyShares: s.Sup, BorrowShares: s.Bor, Collateral: s.Coll},
			Lltv:     r.Lltv, HasIrm: !s.NoIrm, IrmReadable: !s.IrmDead, RateAtTarget: s.Rat.Abs,
			OracleOk:   s.OracleMode == 0 && !s.OracleP.IsZero(),
			OracleZero: s.OracleMode == 2 || (s.OracleMode == 0 && s.OracleP.IsZero()),
		}
		if v.OracleOk {
			v.OraclePrice = s.OracleP
		}
		now := r.Now
		var outs []uint256.Int
		var err error
		switch r.Op {
		case 0:
			err = v.morphoAccrue(now)
		case 1:
			var sh uint256.Int
			sh, err = v.morphoSupply(&r.A, now)
			outs = []uint256.Int{r.A, sh}
		case 2:
			var x, y uint256.Int
			x, y, err = v.morphoWithdraw(&r.A, &r.B, now)
			outs = []uint256.Int{x, y}
		case 3:
			var sh uint256.Int
			sh, err = v.morphoBorrow(&r.A, now)
			outs = []uint256.Int{r.A, sh}
		case 4:
			var x, y uint256.Int
			x, y, err = v.morphoRepay(&r.A, &r.B, now)
			outs = []uint256.Int{x, y}
		case 5:
			err = v.morphoSupplyCollateral(&r.A)
		case 6:
			err = v.morphoWithdrawCollateral(&r.A, now)
		case 10:
			var p mmVenueRead
			p, err = v.tryPosition(now)
			outs = []uint256.Int{boolWord(p.Readable), p.Collateral, p.SupplyShares, p.Supplied, p.Debt}
		case 11:
			var x uint256.Int
			x, err = v.debtOf(now)
			outs = []uint256.Int{x}
		case 12:
			var x uint256.Int
			x, err = v.suppliedOf(now)
			outs = []uint256.Int{x}
		case 13:
			var x uint256.Int
			x, err = v.supplySharesToAssets(&r.A, now)
			outs = []uint256.Int{x}
		case 14:
			outs = []uint256.Int{v.freeLiquidity()}
		case 15:
			var ok bool
			var x uint256.Int
			ok, x, err = v.borrowRateAfter(&r.A, &r.B, now)
			outs = []uint256.Int{boolWord(ok), x}
		case 16:
			outs = []uint256.Int{boolWord(v.OracleOk), v.OraclePrice}
		case 20:
			err = v.accountSupplyCollateral(&r.A)
		case 21:
			err = v.accountWithdrawCollateral(&r.A, now)
		case 22:
			err = v.accountBorrow(&r.A, now)
		case 23:
			var x uint256.Int
			x, err = v.accountRepay(&r.A, now)
			outs = []uint256.Int{x}
		case 24:
			var x uint256.Int
			x, err = v.accountSupply(&r.A, now)
			outs = []uint256.Int{x}
		case 25:
			var x, y uint256.Int
			x, y, err = v.accountWithdraw(&r.A, &r.B, now)
			outs = []uint256.Int{x, y}
		case 30, 31:
			if !v.IrmReadable {
				err = errMMIrmReverted
				break
			}
			if s.NoIrm {
				// the IRM read directly for a market Blue never let it touch: rateAtTarget is unset
				v.RateAtTarget.Clear()
			}
			var avg, endRat uint256.Int
			avg, endRat, err = mmIrmBorrowRate(&v.Market, &v.RateAtTarget, now)
			outs = []uint256.Int{avg}
			if r.Op == 31 && err == nil {
				v.RateAtTarget = endRat
			}
		default:
			t.Fatalf("unknown op %d", r.Op)
		}
		ctx := fmt.Sprintf("row %d tag=%s op=%d a=%s b=%s st=%+v", i, r.Tag, r.Op, r.A.Dec(), r.B.Dec(), *s)
		ret := settleHex(t, r.Ret)
		if !r.Ok {
			want := settleRevert(ret)
			if want == nil || !errors.Is(err, want) {
				mm.add("%s: go=%s sol=revert %s (%s)", ctx, errClass(err), r.Ret, errClass(want))
			}
			continue
		}
		if err != nil {
			mm.add("%s: go=%s sol=ok", ctx, err)
			continue
		}
		if len(outs) > 0 {
			w := abiWords(ret)
			if len(w) < len(outs) {
				mm.add("%s: outs go=%v sol=%s", ctx, outs, r.Ret)
				continue
			}
			for k := range outs {
				if !outs[k].Eq(&w[k]) {
					mm.add("%s: out[%d] go=%s sol=%s", ctx, k, outs[k].Dec(), w[k].Dec())
				}
			}
		}
		if pm := r.PostMarket.state(); v.Market != pm {
			mm.add("%s: post market go=%+v sol=%+v", ctx, v.Market, pm)
		}
		if pp := r.PostPosition.state(); v.Position != pp {
			mm.add("%s: post position go=%+v sol=%+v", ctx, v.Position, pp)
		}
		if !s.NoIrm && !v.RateAtTarget.Eq(&r.PostRat.Abs) {
			mm.add("%s: post rat go=%s sol=%s", ctx, v.RateAtTarget.Dec(), r.PostRat.Abs.Dec())
		}
	}
	t.Logf("blue rows %d, mismatches %d", len(rows)-1, mm.n)
}
