package everlongflamm

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// Stateful replay of the financing module (morpho.go, irm.go, account.go, router.go). The fixtures under
// testdata/edges/ are JSON lines written by testdata/gen/RouterSequences.t.sol, SwapSettlementSequences.t.sol and
// MorphoAccrualGrid.t.sol on a Base fork at block 51317000 (forge --isolate, one transaction per step):
//   - router_sequence_a / _b: pseudo-random sequences of Router transactions over four venues and two loan assets,
//     with time warps, third-party Morpho activity (including calls on the account's behalf and liquidations),
//     IRM outages, oracle moves and config changes between them; router_sequence_liquidation: a scripted
//     liquidation run;
//   - swap_settlement_sequence_a..d: pseudo-random sequences of real pool.swap calls through the deployed pool (one
//     venue, an inflated book, three venues); _e: a scripted run through releaseExcess's swallowed revert,
//     quarantine and the cross-transaction repay snapshot;
//   - mm_accrual_grid: AdaptiveCurveIrm, Blue accrual and the account views over written market states.
// The port REPLAYS each sequence: the Morpho markets, the account positions, rateAtTarget, the managed fields and
// the pool ledger are carried forward from the port's own transitions and compared with the chain before and after
// every step, so an error that compounds across steps surfaces where it first appears. Config, oracle and IRM
// readability come from the chain's pre-state (they are tracker inputs).

func finSeqLines(t *testing.T, name string) [][]byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/edges/" + name)
	require.NoError(t, err)
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	require.NoError(t, err)
	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	var out [][]byte
	for sc.Scan() {
		if len(bytes.TrimSpace(sc.Bytes())) > 0 {
			out = append(out, append([]byte(nil), sc.Bytes()...))
		}
	}
	require.NoError(t, sc.Err())
	return out
}

type finSeqRes struct {
	Ok   bool     `json:"ok"`
	Ret  string   `json:"ret"`
	Rets []string `json:"rets"`
}

func finSeqBytes(h string) []byte {
	b, err := hex.DecodeString(strings.TrimPrefix(h, "0x"))
	if err != nil {
		panic(err)
	}
	return b
}

func abiWords(b []byte) []uint256.Int {
	w := make([]uint256.Int, len(b)/32)
	for i := range w {
		w[i].SetBytes(b[32*i : 32*i+32])
	}
	return w
}

// finSeqRevert maps chain revert data onto the port's error vocabulary; nil means unmapped (the row fails).
func finSeqRevert(b []byte) error {
	switch string(b) {
	case "":
		return errMulDivOverflow
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

type finSeqReport struct {
	t    *testing.T
	area string
	n    int
}

func (m *finSeqReport) add(format string, args ...any) {
	m.t.Helper()
	m.n++
	if m.n <= 80 {
		m.t.Errorf("[%s] "+format, append([]any{m.area}, args...)...)
	}
}

// finSeqSameResult compares one call: both succeed (then the return words must match) or both fail with the same class.
func finSeqSameResult(rep *finSeqReport, where string, chain finSeqRes, goErr error, goWords []uint256.Int) {
	if chain.Ok != (goErr == nil) {
		rep.add("%s: chain ok=%v ret=%s, go err=%v words=%v", where, chain.Ok, chain.Ret, goErr, goWords)
		return
	}
	if !chain.Ok {
		want := finSeqRevert(finSeqBytes(chain.Ret))
		if want == nil || !errors.Is(goErr, want) {
			rep.add("%s: chain revert %s (%v), go %v", where, chain.Ret, want, goErr)
		}
		return
	}
	got := abiWords(finSeqBytes(chain.Ret))
	if len(got) < len(goWords) {
		rep.add("%s: chain returned %d words, go %d", where, len(got), len(goWords))
		return
	}
	for i := range goWords {
		if !got[i].Eq(&goWords[i]) {
			rep.add("%s: word %d chain %s go %s", where, i, got[i].Dec(), goWords[i].Dec())
		}
	}
}

func finSeqDec[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(raw, &v))
	return v
}

// finSeqDiffRouter lists the fields where two Router records differ (the transient snapshot excluded).
func finSeqDiffRouter(a, b *mmRouter) []string {
	var d []string
	if a.GlobalPaused != b.GlobalPaused || !a.PinLtvWad.Eq(&b.PinLtvWad) || !a.SafetyGapWad.Eq(&b.SafetyGapWad) ||
		!a.OracleBandWad.Eq(&b.OracleBandWad) || a.MaxDrawnAssets != b.MaxDrawnAssets {
		d = append(d, "record scalars")
	}
	if !reflect.DeepEqual(a.Loans, b.Loans) {
		d = append(d, fmt.Sprintf("loans %+v vs %+v", a.Loans, b.Loans))
	}
	if !reflect.DeepEqual([][]uint16{a.BorrowOrder, a.SupplyOrder, a.WithdrawOrder, a.RepayOrder},
		[][]uint16{b.BorrowOrder, b.SupplyOrder, b.WithdrawOrder, b.RepayOrder}) {
		d = append(d, "orders")
	}
	if len(a.Venues) != len(b.Venues) {
		return append(d, "venue count")
	}
	for i := range a.Venues {
		x, y := a.Venues[i], b.Venues[i]
		if x.Morpho.Market != y.Morpho.Market {
			d = append(d, fmt.Sprintf("venue %d market %+v vs %+v", i, x.Morpho.Market, y.Morpho.Market))
		}
		if x.Morpho.Position != y.Morpho.Position {
			d = append(d, fmt.Sprintf("venue %d position %+v vs %+v", i, x.Morpho.Position, y.Morpho.Position))
		}
		if !x.Morpho.RateAtTarget.Eq(&y.Morpho.RateAtTarget) {
			d = append(d, fmt.Sprintf("venue %d rateAtTarget %s vs %s", i, x.Morpho.RateAtTarget.Dec(), y.Morpho.RateAtTarget.Dec()))
		}
		if !x.ManagedCollateral.Eq(&y.ManagedCollateral) || !x.ManagedSupplyShares.Eq(&y.ManagedSupplyShares) {
			d = append(d, fmt.Sprintf("venue %d managed (%s,%s) vs (%s,%s)", i, x.ManagedCollateral.Dec(),
				x.ManagedSupplyShares.Dec(), y.ManagedCollateral.Dec(), y.ManagedSupplyShares.Dec()))
		}
		x.Morpho.Market, y.Morpho.Market = mmMarket{}, mmMarket{}
		x.Morpho.Position, y.Morpho.Position = mmPosition{}, mmPosition{}
		x.Morpho.RateAtTarget, y.Morpho.RateAtTarget = uint256.Int{}, uint256.Int{}
		x.ManagedCollateral, y.ManagedCollateral = uint256.Int{}, uint256.Int{}
		x.ManagedSupplyShares, y.ManagedSupplyShares = uint256.Int{}, uint256.Int{}
		if x != y {
			d = append(d, fmt.Sprintf("venue %d config %+v vs %+v", i, x, y))
		}
	}
	return d
}

// finSeqCarry overwrites the fields the replay carries (market, position, rateAtTarget, managed) from src into dst.
func finSeqCarry(dst, src *mmRouter) {
	for i := range dst.Venues {
		if i >= len(src.Venues) {
			return
		}
		dst.Venues[i].Morpho.Market = src.Venues[i].Morpho.Market
		dst.Venues[i].Morpho.Position = src.Venues[i].Morpho.Position
		dst.Venues[i].Morpho.RateAtTarget = src.Venues[i].Morpho.RateAtTarget
		dst.Venues[i].ManagedCollateral = src.Venues[i].ManagedCollateral
		dst.Venues[i].ManagedSupplyShares = src.Venues[i].ManagedSupplyShares
	}
}

// ------------------------------------------------------------------ Router calldata

type finSeqCall struct {
	sel  string
	args []uint256.Int
	data []byte
}

func finSeqParseCall(h string) finSeqCall {
	b := finSeqBytes(h)
	return finSeqCall{sel: hex.EncodeToString(b[:4]), args: abiWords(b[4:]), data: b[4:]}
}

// finSeqDynUints decodes the uint256[] at head word `slot` of ABI data.
func finSeqDynUints(data []byte, slot int) []uint256.Int {
	w := abiWords(data)
	off := int(w[slot].Uint64()) / 32
	n := int(w[off].Uint64())
	return append([]uint256.Int(nil), w[off+1:off+1+n]...)
}

// finSeqExecRouter runs one Router entry (as the pool) on r, writing through; returns the ABI return words.
func finSeqExecRouter(r *mmRouter, c finSeqCall, now uint64) ([]uint256.Int, error) {
	a := c.args
	switch c.sel {
	case "80066e9c": // fund(idx, assets, to, collateralIn, priceWad)
		w, b, p, err := r.fund(uint8(a[0].Uint64()), &a[1], &a[3], &a[4], now)
		return []uint256.Int{w, b, p}, err
	case "5fee443f":
		x, err := r.repayCascade(uint8(a[0].Uint64()), &a[1], now)
		return []uint256.Int{x}, err
	case "c412f8c5":
		x, err := r.supplyCascade(uint8(a[0].Uint64()), &a[1], now)
		return []uint256.Int{x}, err
	case "7490faea", "4d8e8fb2":
		x, err := r.reclaim(&a[0], finSeqDynUints(c.data, 1), c.sel == "7490faea", now)
		return []uint256.Int{x}, err
	case "71840b97":
		return nil, r.borrow(uint16(a[0].Uint64()), &a[1], &a[3], now)
	case "6b738d98":
		x, err := r.repay(uint16(a[0].Uint64()), &a[1], now)
		return []uint256.Int{x}, err
	case "eb669e44":
		x, err := r.supply(uint16(a[0].Uint64()), &a[1], now)
		return []uint256.Int{x}, err
	case "940c2e93":
		x, err := r.withdrawSuppliedEntry(uint16(a[0].Uint64()), &a[1], now)
		return []uint256.Int{x}, err
	case "335efc32":
		return nil, r.postCollateral(uint16(a[0].Uint64()), &a[1])
	case "9f82c026":
		return nil, r.withdrawCollateral(uint16(a[0].Uint64()), &a[1], &a[2], !a[3].IsZero(), now)
	}
	panic("unknown selector " + c.sel)
}

// finSeqPayload checks the uint256 payload of InsufficientLiquidity / InsufficientCollateral where the port can
// name it.
func finSeqPayload(rep *finSeqReport, where string, chain finSeqRes, want *uint256.Int) {
	b := finSeqBytes(chain.Ret)
	if chain.Ok || want == nil || len(b) != 36 {
		return
	}
	got := new(uint256.Int).SetBytes(b[4:36])
	if !got.Eq(want) {
		rep.add("%s: revert payload chain %s port %s", where, got.Dec(), want.Dec())
	}
}

// ------------------------------------------------------------------ the Router sequence

type finSeqEnv struct {
	Kind  string      `json:"kind"`
	Venue int         `json:"venue"`
	Op    int         `json:"op"`
	A     uint256.Int `json:"a"`
	S     uint256.Int `json:"s"`
	Wpre  mmPosition  `json:"wpre"`
	Wpost mmPosition  `json:"wpost"`
	Mpre  mmMarket    `json:"mpre"`
	Mpost mmMarket    `json:"mpost"`
	Rpre  uint256.Int `json:"rpre"`
	Rpost uint256.Int `json:"rpost"`
	Res   finSeqRes   `json:"res"`
	Who   string      `json:"who"`
	Data  string      `json:"data"`
}

type finSeqRouterViews struct {
	Prices  []uint256.Int `json:"prices"`
	CollIns []uint256.Int `json:"collIns"`
	Router  struct {
		Positions     finSeqRes   `json:"positions"`
		Drawn         finSeqRes   `json:"drawn"`
		MinLltv       finSeqRes   `json:"minLltv"`
		Reclaimable   finSeqRes   `json:"reclaimable"`
		Quarantine    []finSeqRes `json:"quarantine"`
		Ceiling       []finSeqRes `json:"ceiling"`
		VenuePosition []finSeqRes `json:"venuePosition"`
		Health        []finSeqRes `json:"health"`
	} `json:"router"`
	Acct *struct {
		Venue int `json:"venue"`
		V     struct {
			TryPosition   finSeqRes     `json:"tryPosition"`
			DebtOf        finSeqRes     `json:"debtOf"`
			SuppliedOf    finSeqRes     `json:"suppliedOf"`
			FreeLiquidity finSeqRes     `json:"freeLiquidity"`
			DB            []uint256.Int `json:"dB"`
			DS            []uint256.Int `json:"dS"`
			Rates         []finSeqRes   `json:"rates"`
		} `json:"v"`
	} `json:"acct"`
}

type finSeqRouterRow struct {
	I   int             `json:"i"`
	T   uint256.Int     `json:"t"`
	Env []finSeqEnv     `json:"env"`
	Pre json.RawMessage `json:"pre"`
	Op  struct {
		Kind string          `json:"kind"`
		Data json.RawMessage `json:"data"`
	} `json:"op"`
	Res   finSeqRes         `json:"res"`
	Post  json.RawMessage   `json:"post"`
	Views finSeqRouterViews `json:"views"`
}

// finSeqCarryThirdParty checks the carried market (and, for a call on the account's behalf, its position) against
// the chain before a third-party Morpho call, then takes the chain's post-call figures so a modelling error there
// is reported once by finSeqThirdParty rather than cascading.
func finSeqCarryThirdParty(rep *finSeqReport, where string, e *finSeqEnv, cv *mmVenueMarket) {
	if cv.Market != e.Mpre || !cv.RateAtTarget.Eq(&e.Rpre) {
		rep.add("%s carried market drifted before third-party op: go %+v rat %s, chain %+v rat %s", where,
			cv.Market, cv.RateAtTarget.Dec(), e.Mpre, e.Rpre.Dec())
	}
	cv.Market, cv.RateAtTarget = e.Mpost, e.Rpost
	if e.Who == "account" {
		if cv.Position != e.Wpre {
			rep.add("%s carried position drifted before a call on the account's behalf: go %+v, chain %+v", where, cv.Position, e.Wpre)
		}
		cv.Position = e.Wpost
	}
}

// finSeqThirdParty models a third party's Morpho call on venue v's market with the port's Blue transitions and
// compares the market, the caller's position and rateAtTarget with the chain.
func finSeqThirdParty(rep *finSeqReport, where string, e *finSeqEnv, flags mmVenueMarket, now uint64) {
	m := flags
	m.Market, m.Position, m.RateAtTarget = e.Mpre, e.Wpre, e.Rpre
	var err error
	switch e.Op { // 0-4 the third party's own position; 5-7 on the financing account's behalf
	case 0, 5:
		_, err = m.morphoSupply(&e.A, now)
	case 1:
		_, _, err = m.morphoWithdraw(&e.A, &e.S, now)
	case 2:
		_, err = m.morphoBorrow(&e.A, now)
	case 3, 7:
		_, _, err = m.morphoRepay(&e.A, &e.S, now)
	case 4, 6:
		err = m.morphoSupplyCollateral(&e.A)
	default:
		panic("unknown third-party op")
	}
	if e.Res.Ok != (err == nil) {
		rep.add("%s third-party op %d a=%s s=%s: chain ok=%v ret=%s, go %v", where, e.Op, e.A.Dec(), e.S.Dec(), e.Res.Ok, e.Res.Ret, err)
		return
	}
	if !e.Res.Ok {
		if want := finSeqRevert(finSeqBytes(e.Res.Ret)); want == nil || !errors.Is(err, want) {
			rep.add("%s third-party op %d: chain %s, go %v", where, e.Op, e.Res.Ret, err)
		}
		return
	}
	if m.Market != e.Mpost || m.Position != e.Wpost || !m.RateAtTarget.Eq(&e.Rpost) {
		rep.add("%s third-party op %d a=%s s=%s: go market %+v pos %+v rat %s, chain %+v %+v %s", where, e.Op, e.A.Dec(),
			e.S.Dec(), m.Market, m.Position, m.RateAtTarget.Dec(), e.Mpost, e.Wpost, e.Rpost.Dec())
	}
}

func TestRouterSequences(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"router_sequence_a.jsonl.gz", "router_sequence_b.jsonl.gz", "router_sequence_liquidation.jsonl.gz"} {
		t.Run(name, func(t *testing.T) { finSeqRouterSequence(t, name) })
	}
}

func finSeqRouterSequence(t *testing.T, name string) {
	rows := finSeqLines(t, name)
	rep := &finSeqReport{t: t, area: name}
	var cur *mmRouter
	var nOps, nOk, nViews int
	for _, raw := range rows {
		var row finSeqRouterRow
		require.NoError(t, json.Unmarshal(raw, &row))
		now := row.T.Uint64()
		pre := finSeqDec[mmRouter](t, row.Pre)
		post := finSeqDec[mmRouter](t, row.Post)
		where := fmt.Sprintf("step %d", row.I)

		// third-party Morpho activity, replayed on the carried market
		for i := range row.Env {
			e := &row.Env[i]
			switch e.Kind {
			case "tp":
				if cur != nil {
					finSeqCarryThirdParty(rep, where, e, &cur.Venues[e.Venue].Morpho)
				}
				finSeqThirdParty(rep, where, e, pre.Venues[e.Venue].Morpho, now)
			case "liq":
				// Blue liquidation is not ported: check the carried figures, then take the chain's
				if cur != nil {
					e.Who = "account"
					finSeqCarryThirdParty(rep, where, e, &cur.Venues[e.Venue].Morpho)
				}
			case "poolcall":
				if cur != nil {
					c := finSeqParseCall(e.Data)
					trial := cur.clone()
					words, err := finSeqExecRouter(trial, c, now)
					finSeqSameResult(rep, where+" poolcall "+c.sel, e.Res, err, words)
					if err == nil {
						cur = trial
					}
				}
			}
		}

		// the carried state against the chain's pre-state
		g := pre.clone()
		if cur != nil {
			finSeqCarry(g, cur)
			if d := finSeqDiffRouter(g, &pre); len(d) > 0 {
				rep.add("%s carried state drifted from chain pre-state: %v", where, d)
				g = pre.clone()
			}
		}

		// the transaction
		nOps++
		trial := g.clone()
		switch row.Op.Kind {
		case "call":
			var h string
			require.NoError(t, json.Unmarshal(row.Op.Data, &h))
			c := finSeqParseCall(h)
			words, err := finSeqExecRouter(trial, c, now)
			finSeqSameResult(rep, where+" "+c.sel, row.Res, err, words)
			switch {
			case c.sel == "80066e9c" && errors.Is(err, ErrInsufficientLiquidity):
				plan, perr := g.clone().buildPlan(uint8(c.args[0].Uint64()), &c.args[1], &c.args[3], &c.args[4], now)
				if perr == nil && !plan.Remaining.IsZero() {
					finSeqPayload(rep, where+" fund", row.Res, &plan.Remaining)
				}
			case c.sel == "7490faea" && errors.Is(err, ErrInsufficientCollateral) && len(words) == 1:
				var short uint256.Int
				short.Sub(&c.args[0], &words[0])
				finSeqPayload(rep, where+" reclaim", row.Res, &short)
			case (c.sel == "940c2e93" || c.sel == "9f82c026") && (errors.Is(err, ErrInsufficientLiquidity) || errors.Is(err, ErrInsufficientCollateral)):
				finSeqPayload(rep, where+" "+c.sel, row.Res, &c.args[1])
			}
			if err == nil {
				g = trial
				nOk++
			}
		case "bundle":
			var hs []string
			require.NoError(t, json.Unmarshal(row.Op.Data, &hs))
			c0, c1 := finSeqParseCall(hs[0]), finSeqParseCall(hs[1])
			w0, err := finSeqExecRouter(trial, c0, now)
			if err == nil {
				_, err = finSeqExecRouter(trial, c1, now)
			}
			if row.Res.Ok != (err == nil) {
				rep.add("%s bundle: chain ok=%v ret=%s, go %v", where, row.Res.Ok, row.Res.Ret, err)
			} else if !row.Res.Ok {
				if want := finSeqRevert(finSeqBytes(row.Res.Ret)); want == nil || !errors.Is(err, want) {
					rep.add("%s bundle: chain %s, go %v", where, row.Res.Ret, err)
				}
			} else {
				finSeqSameResult(rep, where+" bundle repay", finSeqRes{Ok: true, Ret: row.Res.Rets[0]}, nil, w0)
				g = trial
				nOk++
			}
		}
		g.endTransaction()
		if d := finSeqDiffRouter(g, &post); len(d) > 0 {
			rep.add("%s post-state: %v", where, d)
			g = post.clone()
		}
		cur = g

		// views on the chain's post-state
		nViews += finSeqRouterViewsCheck(rep, where, &post, &row.Views, now)
	}
	t.Logf("%s: %d steps, %d transactions committed, %d view rows, %d mismatches", name, nOps, nOk, nViews, rep.n)
	require.Zero(t, rep.n)
}

func finSeqRouterViewsCheck(rep *finSeqReport, where string, st *mmRouter, v *finSeqRouterViews, now uint64) int {
	n := 0
	r := st.clone()
	pos, err := r.positions(now)
	var pw []uint256.Int
	if err == nil {
		pw = append(pw, pos.TotalColl)
	}
	finSeqSameResult(rep, where+" positions.total", finSeqResWord(v.Router.Positions, 3), err, pw)
	if err == nil && v.Router.Positions.Ok {
		data := finSeqBytes(v.Router.Positions.Ret)
		for k, want := range [][]uint256.Int{pos.Coll, pos.Sup, pos.Debt} {
			got := finSeqDynUints(data, k)
			if !reflect.DeepEqual(got, want) {
				rep.add("%s positions[%d] chain %v go %v", where, k, got, want)
			}
		}
	}
	n++
	c, m, err := r.drawn(now)
	finSeqSameResult(rep, where+" drawn", v.Router.Drawn, err, []uint256.Int{*uint256.NewInt(uint64(c)), *uint256.NewInt(uint64(m))})
	lltv, err := r.minLltv(now)
	finSeqSameResult(rep, where+" minLltv", v.Router.MinLltv, err, []uint256.Int{lltv})
	rc, err := r.reclaimable(v.Prices, now)
	finSeqSameResult(rep, where+" reclaimable", v.Router.Reclaimable, err, []uint256.Int{rc})
	n += 3
	for i := range v.Router.Quarantine {
		q, err := r.quarantine(uint8(i), now)
		any := uint256.Int{}
		if q.Any {
			any.SetOne()
		}
		finSeqSameResult(rep, fmt.Sprintf("%s quarantine(%d)", where, i), v.Router.Quarantine[i], err, []uint256.Int{any, q.FrozenDebt, q.FrozenColl})
		n++
	}
	k := 0
	for i := range st.Loans {
		for j := range v.CollIns {
			x, err := r.fundingCeiling(uint8(i), &v.CollIns[j], &v.Prices[i], now)
			finSeqSameResult(rep, fmt.Sprintf("%s fundingCeiling(%d,%s,%s)", where, i, v.CollIns[j].Dec(), v.Prices[i].Dec()), v.Router.Ceiling[k], err, []uint256.Int{x})
			k++
			n++
		}
	}
	for id := range st.Venues {
		p, err := r.venuePosition(uint16(id), now)
		finSeqSameResult(rep, fmt.Sprintf("%s venuePosition(%d)", where, id), v.Router.VenuePosition[id], err, p[:])
		h, err := r.venueHealth(uint16(id), &v.Prices[st.Venues[id].LoanIndex], now)
		finSeqSameResult(rep, fmt.Sprintf("%s venueHealth(%d)", where, id), v.Router.Health[id], err, []uint256.Int{h})
		n += 2
	}
	if a := v.Acct; a != nil {
		vm := &r.Venues[a.Venue].Morpho
		tp, err := vm.tryPosition(now)
		rd := uint256.Int{}
		if tp.Readable {
			rd.SetOne()
		}
		finSeqSameResult(rep, fmt.Sprintf("%s tryPosition(%d)", where, a.Venue), a.V.TryPosition, err, []uint256.Int{rd, tp.Collateral, tp.SupplyShares, tp.Supplied, tp.Debt})
		d, err := vm.debtOf(now)
		finSeqSameResult(rep, fmt.Sprintf("%s debtOf(%d)", where, a.Venue), a.V.DebtOf, err, []uint256.Int{d})
		s, err := vm.suppliedOf(now)
		finSeqSameResult(rep, fmt.Sprintf("%s suppliedOf(%d)", where, a.Venue), a.V.SuppliedOf, err, []uint256.Int{s})
		fl := vm.freeLiquidity()
		finSeqSameResult(rep, fmt.Sprintf("%s freeLiquidity(%d)", where, a.Venue), a.V.FreeLiquidity, nil, []uint256.Int{fl})
		n += 4
		for i := range a.V.Rates {
			ok, rate, err := vm.borrowRateAfter(&a.V.DB[i], &a.V.DS[i], now)
			okw := uint256.Int{}
			if ok {
				okw.SetOne()
			}
			finSeqSameResult(rep, fmt.Sprintf("%s borrowRateAfter(%d,%s,%s)", where, a.Venue, a.V.DB[i].Dec(), a.V.DS[i].Dec()), a.V.Rates[i], err, []uint256.Int{okw, rate})
			n++
		}
	}
	return n
}

// finSeqResWord narrows a successful ABI return to the single head word at `slot`.
func finSeqResWord(r finSeqRes, slot int) finSeqRes {
	if !r.Ok {
		return r
	}
	w := abiWords(finSeqBytes(r.Ret))
	b := w[slot].Bytes32()
	return finSeqRes{Ok: true, Ret: "0x" + hex.EncodeToString(b[:])}
}

// ------------------------------------------------------------------ the swap sequence

type finSeqSwapState struct {
	Pool   gatePool `json:"pool"`
	Router mmRouter `json:"router"`
}

type finSeqSwapRow struct {
	I   int             `json:"i"`
	T   uint256.Int     `json:"t"`
	Env []finSeqEnv     `json:"env"`
	Pre finSeqSwapState `json:"pre"`
	Op  struct {
		Sell     bool        `json:"sell"`
		AmountIn uint256.Int `json:"amountIn"`
	} `json:"op"`
	Preview finSeqRes       `json:"preview"`
	Res     finSeqRes       `json:"res"`
	Post    finSeqSwapState `json:"post"`
	Views   struct {
		finSeqRouterViews
		TotalAssets       finSeqRes `json:"totalAssets"`
		LoanPosition      finSeqRes `json:"loanPosition"`
		PoolAssetPosition finSeqRes `json:"poolAssetPosition"`
	} `json:"views"`
}

func finSeqDiffPool(a, b *gatePool) []string {
	var d []string
	if !a.Physical.Eq(&b.Physical) {
		d = append(d, fmt.Sprintf("physical %s vs %s", a.Physical.Dec(), b.Physical.Dec()))
	}
	for i := range a.Loans {
		if !a.Loans[i].Liquid.Eq(&b.Loans[i].Liquid) {
			d = append(d, fmt.Sprintf("liquid[%d] %s vs %s", i, a.Loans[i].Liquid.Dec(), b.Loans[i].Liquid.Dec()))
		}
	}
	return d
}

func TestSwapSettlementSequences(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"swap_settlement_sequence_a.jsonl.gz", "swap_settlement_sequence_b.jsonl.gz", "swap_settlement_sequence_c.jsonl.gz", "swap_settlement_sequence_d.jsonl.gz", "swap_settlement_sequence_e.jsonl.gz"} {
		t.Run(name, func(t *testing.T) {
			rows := finSeqLines(t, name)
			rep := &finSeqReport{t: t, area: name}
			var curPool *gatePool
			var curRouter *mmRouter
			var nSwaps, nOk, nFail, nViews int
			for _, raw := range rows {
				var row finSeqSwapRow
				require.NoError(t, json.Unmarshal(raw, &row))
				now := row.T.Uint64()
				where := fmt.Sprintf("step %d", row.I)
				resyncPhysical := false
				for i := range row.Env {
					e := &row.Env[i]
					switch e.Kind {
					case "tp":
						if curRouter != nil {
							finSeqCarryThirdParty(rep, where, e, &curRouter.Venues[e.Venue].Morpho)
						}
						finSeqThirdParty(rep, where, e, row.Pre.Router.Venues[e.Venue].Morpho, now)
					case "poolcall":
						// a Router entry called as the pool between swaps (custody outside the tracked ledger)
						if curRouter != nil {
							c := finSeqParseCall(e.Data)
							trial := curRouter.clone()
							words, err := finSeqExecRouter(trial, c, now)
							finSeqSameResult(rep, where+" poolcall "+c.sel, e.Res, err, words)
							if err == nil {
								trial.endTransaction()
								curRouter = trial
							}
						}
					case "phys":
						resyncPhysical = true
					}
				}

				pool := row.Pre.Pool.clone()
				router := row.Pre.Router.clone()
				if curRouter != nil {
					finSeqCarry(router, curRouter)
					if d := finSeqDiffRouter(router, &row.Pre.Router); len(d) > 0 {
						rep.add("%s carried router drifted: %v", where, d)
						router = row.Pre.Router.clone()
					}
					pool.Physical = curPool.Physical
					if resyncPhysical {
						pool.Physical = row.Pre.Pool.Physical
					}
					for i := range pool.Loans {
						pool.Loans[i].Liquid = curPool.Loans[i].Liquid
					}
					if d := finSeqDiffPool(pool, &row.Pre.Pool); len(d) > 0 {
						rep.add("%s carried pool drifted: %v", where, d)
						pool = row.Pre.Pool.clone()
					}
				}

				if row.Preview.Ok {
					nSwaps++
					src := row.Preview
					if row.Res.Ok {
						src = row.Res
					}
					w := abiWords(finSeqBytes(src.Ret))
					used, net := w[0], w[1]
					var err error
					if row.Op.Sell {
						err = mmSettleSell(pool, router, 0, &pool.PriceWad[0], &used, &net, now)
					} else {
						err = mmSettleBuy(pool, router, 0, &used, &net, now)
					}
					if row.Res.Ok != (err == nil) {
						rep.add("%s %s used=%s net=%s: chain ok=%v ret=%s, go %v", where, map[bool]string{true: "sell", false: "buy"}[row.Op.Sell],
							used.Dec(), net.Dec(), row.Res.Ok, row.Res.Ret, err)
					} else if !row.Res.Ok {
						nFail++
						if want := finSeqRevert(finSeqBytes(row.Res.Ret)); want == nil || !errors.Is(err, want) {
							rep.add("%s settlement revert: chain %s (%v), go %v", where, row.Res.Ret, want, err)
						}
					} else {
						nOk++
					}
				} else if row.Res.Ok {
					rep.add("%s preview reverted %s but the swap succeeded", where, row.Preview.Ret)
				}
				if d := finSeqDiffPool(pool, &row.Post.Pool); len(d) > 0 {
					rep.add("%s post pool: %v", where, d)
					pool = row.Post.Pool.clone()
				}
				if d := finSeqDiffRouter(router, &row.Post.Router); len(d) > 0 {
					rep.add("%s post router: %v", where, d)
					router = row.Post.Router.clone()
				}
				curPool, curRouter = pool, router

				// views on the chain post-state
				pp, pr := row.Post.Pool.clone(), row.Post.Router.clone()
				nViews += finSeqRouterViewsCheck(rep, where, pr, &row.Views.finSeqRouterViews, now)
				ta, err := gateTotalAssets(pp, pr, now)
				finSeqSameResult(rep, where+" totalAssets", row.Views.TotalAssets, err, []uint256.Int{ta})
				_, sup, debt, err := pr.position(0, now)
				finSeqSameResult(rep, where+" loanPosition", row.Views.LoanPosition, err, []uint256.Int{pp.Loans[0].Liquid, sup, debt})
				pos, err := pr.positions(now)
				var gross uint256.Int
				gross.Add(&pp.Physical, &pos.TotalColl)
				finSeqSameResult(rep, where+" poolAssetPosition", row.Views.PoolAssetPosition, err, []uint256.Int{pp.Physical, gross})
				nViews += 3
			}
			t.Logf("%s: %d steps, %d settled swaps, %d settlement reverts, %d view rows, %d mismatches", name, len(rows), nOk, nFail, nViews, rep.n)
			require.Zero(t, rep.n)
		})
	}
}

// ------------------------------------------------------------------ the IRM / Blue / account grid

type finSeqGridRow struct {
	I     int           `json:"i"`
	T     uint256.Int   `json:"t"`
	Down  bool          `json:"down"`
	State mmVenueMarket `json:"state"`
	View  finSeqRes     `json:"view"`
	Acct  struct {
		TryPosition finSeqRes   `json:"tryPosition"`
		DebtOf      finSeqRes   `json:"debtOf"`
		SuppliedOf  finSeqRes   `json:"suppliedOf"`
		Shares      uint256.Int `json:"shares"`
		S2a         finSeqRes   `json:"s2a"`
		DB          uint256.Int `json:"dB"`
		DS          uint256.Int `json:"dS"`
		Rate        finSeqRes   `json:"rate"`
	} `json:"acct"`
	Rate     finSeqRes     `json:"rate"`
	RatAfter uint256.Int   `json:"ratAfter"`
	Accrue   finSeqRes     `json:"accrue"`
	After    mmVenueMarket `json:"after"`
}

func TestMorphoAccrualGrid(t *testing.T) {
	t.Parallel()
	rep := &finSeqReport{t: t, area: "accrual-grid"}
	rows := finSeqLines(t, "mm_accrual_grid.jsonl.gz")
	n := 0
	for _, raw := range rows {
		var row finSeqGridRow
		require.NoError(t, json.Unmarshal(raw, &row))
		now := row.T.Uint64()
		where := fmt.Sprintf("grid %d", row.I)
		st := row.State
		if st.IrmReadable == row.Down {
			rep.add("%s: fixture readability flag %v with outage %v", where, st.IrmReadable, row.Down)
		}

		// AdaptiveCurveIrm: the view, and borrowRate's stored endRateAtTarget
		avg, end, err := mmIrmBorrowRate(&st.Market, &st.RateAtTarget, now)
		if row.Down {
			err = errMMIrmReverted
		}
		finSeqSameResult(rep, where+" borrowRateView", row.View, err, []uint256.Int{avg})
		finSeqSameResult(rep, where+" borrowRate", row.Rate, err, []uint256.Int{avg})
		if row.Rate.Ok && !end.Eq(&row.RatAfter) {
			rep.add("%s endRateAtTarget chain %s go %s", where, row.RatAfter.Dec(), end.Dec())
		}

		// account views
		vm := st
		tp, err := vm.tryPosition(now)
		rd := uint256.Int{}
		if tp.Readable {
			rd.SetOne()
		}
		finSeqSameResult(rep, where+" tryPosition", row.Acct.TryPosition, err, []uint256.Int{rd, tp.Collateral, tp.SupplyShares, tp.Supplied, tp.Debt})
		d, err := vm.debtOf(now)
		finSeqSameResult(rep, where+" debtOf", row.Acct.DebtOf, err, []uint256.Int{d})
		s, err := vm.suppliedOf(now)
		finSeqSameResult(rep, where+" suppliedOf", row.Acct.SuppliedOf, err, []uint256.Int{s})
		a, err := vm.supplySharesToAssets(&row.Acct.Shares, now)
		finSeqSameResult(rep, where+" supplySharesToAssets", row.Acct.S2a, err, []uint256.Int{a})
		ok, rate, err := vm.borrowRateAfter(&row.Acct.DB, &row.Acct.DS, now)
		okw := uint256.Int{}
		if ok {
			okw.SetOne()
		}
		finSeqSameResult(rep, fmt.Sprintf("%s borrowRateAfter(%s,%s)", where, row.Acct.DB.Dec(), row.Acct.DS.Dec()), row.Acct.Rate, err, []uint256.Int{okw, rate})

		// Morpho accrueInterest
		acc := st
		err = acc.morphoAccrue(now)
		finSeqSameResult(rep, where+" accrueInterest", row.Accrue, err, nil)
		if row.Accrue.Ok && err == nil && (acc.Market != row.After.Market || !acc.RateAtTarget.Eq(&row.After.RateAtTarget)) {
			rep.add("%s accrue: go %+v rat %s, chain %+v rat %s", where, acc.Market, acc.RateAtTarget.Dec(), row.After.Market, row.After.RateAtTarget.Dec())
		}
		n++
	}
	t.Logf("grid: %d rows, %d mismatches", n, rep.n)
	require.Zero(t, rep.n)
}
