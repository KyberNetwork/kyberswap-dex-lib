package everlongflamm

import (
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// Edge replays for almcurve.go, fee.go and hook.go. The fixtures in testdata/edges/ come from generators written
// separately from the module fixtures' (testdata/gen/{AlmCurve,Fee,HookFill}Edges.t.sol): source-compiled
// harnesses for the internal curve and fee functions, the deployed Base AlmCurve library for
// reservesAt/swapExactInX96, and the live Base EverlongHook under overwritten storage. Each row is one call:
// {f, a: args, ok, r: return words, e: revert data}.

type swapEdgeRow struct {
	F  string   `json:"f"`
	A  []string `json:"a"`
	Ok bool     `json:"ok"`
	R  []string `json:"r"`
	E  string   `json:"e"`
}

func swapEdgeLoad(t *testing.T, name string) []swapEdgeRow {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "edges", name))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	zr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()
	var rows []swapEdgeRow
	require.NoError(t, json.NewDecoder(zr).Decode(&rows))
	return rows
}

func swapEdgeArgs(t *testing.T, r *swapEdgeRow) []uint256.Int {
	t.Helper()
	out := make([]uint256.Int, len(r.A))
	for i, s := range r.A {
		v, err := uint256.FromDecimal(s)
		require.NoError(t, err)
		out[i] = *v
	}
	return out
}

// swapEdgeWantErr decodes the recorded revert data, independently of the implementation's helpers, into the
// sentinel the port must return; why is non-empty when the data maps to no sentinel.
func swapEdgeWantErr(errHex string) (why string, want error) {
	data, err := hex.DecodeString(strings.TrimPrefix(errHex, "0x"))
	if err != nil {
		return "bad hex " + errHex, nil
	}
	if len(data) == 0 {
		return "", errMulDivOverflow
	}
	if len(data) == 36 && hex.EncodeToString(data[:4]) == "4e487b71" {
		switch data[35] {
		case 0x11:
			return "", errPanicArithmetic
		case 0x12:
			return "", errPanicDivZero
		}
		return "unmapped panic " + errHex, nil
	}
	if len(data) == 4 {
		switch hex.EncodeToString(data) {
		case "1615e638":
			return "", errFeeLnWadUndefined
		case "e3922b5c":
			return "", ErrCurveDomain
		case "fb620e8c":
			return "", ErrCurveAmplification
		case "b79f5ca9":
			return "", ErrCurveSpan
		}
	}
	return "unmapped revert " + errHex, nil
}

// swapEdgeChecker accumulates mismatches so a run reports all of them, not just the first.
type swapEdgeChecker struct {
	t      *testing.T
	fails  int
	checks int
}

func (c *swapEdgeChecker) fail(format string, args ...any) {
	c.fails++
	if c.fails <= 60 {
		c.t.Errorf(format, args...)
	}
}

// compare checks one row: an error exactly where the contract reverted (the right sentinel), else
// every return word equal.
func (c *swapEdgeChecker) compare(r *swapEdgeRow, got []uint256.Int, gotErr error) {
	c.checks++
	if !r.Ok {
		why, want := swapEdgeWantErr(r.E)
		if want == nil {
			c.fail("%s%v: %s", r.F, r.A, why)
			return
		}
		if !errors.Is(gotErr, want) {
			c.fail("%s%v: solidity reverted %v (%s), go returned err=%v out=%s", r.F, r.A, want, r.E, gotErr,
				swapEdgeFmt(got))
		}
		return
	}
	if gotErr != nil {
		c.fail("%s%v: solidity ok %v, go err %v", r.F, r.A, r.R, gotErr)
		return
	}
	if len(got) != len(r.R) {
		c.fail("%s%v: arity go %d solidity %d", r.F, r.A, len(got), len(r.R))
		return
	}
	for i := range got {
		if got[i].Dec() != r.R[i] {
			c.fail("%s%v: word %d go %s solidity %s (all go %s sol %v)", r.F, r.A, i, got[i].Dec(), r.R[i],
				swapEdgeFmt(got), r.R)
			return
		}
	}
}

func swapEdgeFmt(v []uint256.Int) string {
	s := make([]string, len(v))
	for i := range v {
		s[i] = v[i].Dec()
	}
	return "[" + strings.Join(s, " ") + "]"
}

func swapEdgeSup(a []uint256.Int) almSupport {
	return almSupport{AWad: a[0], XLo: a[1], XHi: a[2], YHi: a[3]}
}

func TestAlmCurveEdges(t *testing.T) {
	t.Parallel()
	swapEdgeCurveRows(t, "alm_curve_edges.json.gz")
}

func swapEdgeCurveRows(t *testing.T, name string) {
	rows := swapEdgeLoad(t, name)
	c := &swapEdgeChecker{t: t}
	seen := map[string]int{}
	for i := range rows {
		r := &rows[i]
		a := swapEdgeArgs(t, r)
		seen[r.F]++
		switch r.F {
		case "cWad":
			// cWad = fourK - WAD as int256, compared in two's complement.
			_, fourK, err := almNegC(&a[0])
			var c256 uint256.Int
			c256.Sub(&fourK, uWad)
			c.compare(r, []uint256.Int{c256}, err)
		case "yAtX":
			y, err := almYAtX(&a[0], &a[1])
			c.compare(r, []uint256.Int{y}, err)
		case "priceAtX":
			p, err := almPriceAtX(&a[0], &a[1])
			c.compare(r, []uint256.Int{p}, err)
		case "xAtPrice":
			x, err := almXAtPrice(&a[0], &a[1])
			c.compare(r, []uint256.Int{x}, err)
		case "supportFor":
			s, err := almSupportFor(&a[0], &a[1], &a[2])
			c.compare(r, []uint256.Int{s.AWad, s.XLo, s.XHi, s.YHi}, err)
		case "heldAt":
			sup := swapEdgeSup(a)
			v, s, err := almHeldAt(&sup, &a[4])
			c.compare(r, []uint256.Int{v, s}, err)
		case "swapExactIn":
			sup := swapEdgeSup(a)
			out, xa, un, err := almSwapExactIn(&sup, &a[4], !a[5].IsZero(), &a[6])
			c.compare(r, []uint256.Int{out, xa, un}, err)
		case "reservesAt":
			sup := swapEdgeSup(a)
			st, vol, err := almReservesAt(&sup, &a[4], &a[5], &a[6])
			c.compare(r, []uint256.Int{st, vol}, err)
		case "swapExactInX96":
			sup := swapEdgeSup(a)
			out, xa, un, err := almSwapExactInX96(&sup, &a[4], &a[5], &a[6], !a[7].IsZero(), &a[8])
			c.compare(r, []uint256.Int{out, xa, un}, err)
		default:
			t.Fatalf("unknown fn %s", r.F)
		}
	}
	t.Logf("%s rows %d checks %d mismatches %d by fn %v", name, len(rows), c.checks, c.fails, seen)
	require.Zero(t, c.fails)
}

func swapEdgeFee(a []uint256.Int) feeParams {
	return feeParams{MidFeeWad: a[0], OutFeeWad: a[1], GammaWad: a[2], SigmaRefWad: a[3], VolBetaWad: a[4],
		VolMinWad: a[5], VolMaxWad: a[6], DirSkewWad: a[7]}
}

func TestFeeEdges(t *testing.T) {
	t.Parallel()
	rows := swapEdgeLoad(t, "fee_edges.json.gz")
	c := &swapEdgeChecker{t: t}
	seen := map[string]int{}
	for i := range rows {
		r := &rows[i]
		a := swapEdgeArgs(t, r)
		seen[r.F]++
		switch r.F {
		case "lnWad":
			v, err := feeLnWad(&a[0])
			c.compare(r, []uint256.Int{v}, err) // int256 in two's complement, as the harness returned it
		case "logRatioAbsWad":
			v, err := feeLogRatioAbsWad(&a[0], &a[1])
			c.compare(r, []uint256.Int{v}, err)
		case "reductionG":
			st := feeState{ReserveStable: a[0], ReserveVolatile: a[1], AnchorWad: a[2]}
			g, w, err := feeReductionG(&st, &a[3])
			c.compare(r, []uint256.Int{g, w}, err)
		case "volMultiplier":
			p := swapEdgeFee(a)
			v, err := feeVolMultiplier(&p, &a[8], &a[9])
			c.compare(r, []uint256.Int{v}, err)
		case "fillFee":
			p := swapEdgeFee(a)
			st := feeState{ReserveStable: a[8], ReserveVolatile: a[9], AnchorWad: a[10], SpotWad: a[11], RvWad: a[12]}
			f, err := feeFillFee(&p, &st, !a[13].IsZero(), &a[14], &a[15])
			c.compare(r, []uint256.Int{f}, err)
		default:
			t.Fatalf("unknown fn %s", r.F)
		}
	}
	t.Logf("fee rows %d checks %d mismatches %d by fn %v", len(rows), c.checks, c.fails, seen)
	require.Zero(t, c.fails)
}

// swapEdgeHookState builds the hook storage and context from a row: a[0..24] state, a[25..30] context.
func swapEdgeHookState(a []uint256.Int) (hookState, swapContext, uint256.Int) {
	st := hookState{
		AWad:                a[0],
		Support:             almSupport{AWad: a[1], XLo: a[2], XHi: a[3], YHi: a[4]},
		AnchorSqrtX96:       a[5],
		ReservationPriceWad: a[6],
		Kappa:               a[7],
		XWad:                a[8],
		ReserveStable:       a[9],
		IdleStable:          a[10],
		ReserveVolatile:     a[11],
		IdleVolatile:        a[12],
		RvWad:               a[13],
		Fee:                 swapEdgeFee(a[14:22]),
		InvSkewKappaWad:     a[22],
		InvSkewBandWad:      a[23],
		LoanScale:           a[24],
	}
	ctx := swapContext{
		Pool:         poolContext{PhysicalPoolAsset: a[25], PostedPoolAsset: a[26]},
		PoolAssetIn:  !a[27].IsZero(),
		AmountIn:     a[28],
		MaxAmountOut: a[29],
	}
	return st, ctx, a[30]
}

func swapEdgeHookRows(t *testing.T, name string) {
	rows := swapEdgeLoad(t, name)
	c := &swapEdgeChecker{t: t}
	seen := map[string]int{}
	filled := 0
	keep := sampleRows(len(rows), rowSampling{stride: 16, rare: 32}, func(i int) (string, string) {
		r := &rows[i]
		return r.F, fmt.Sprintf("%v|%s", r.Ok, r.E)
	})
	for i := range rows {
		if !keep[i] {
			continue
		}
		r := &rows[i]
		a := swapEdgeArgs(t, r)
		require.Len(t, a, 31)
		st, ctx, feeWad := swapEdgeHookState(a)
		before := st
		seen[r.F]++
		switch r.F {
		case "hook.spot":
			v, err := st.spot()
			c.compare(r, []uint256.Int{v}, err)
		case "hook.bookFor":
			b, err := st.bookFor(&ctx.Pool)
			c.compare(r, []uint256.Int{b.Kappa, b.Rs, b.Is, b.Rv, b.Iv, b.X}, err)
		case "hook.previewFeeWad":
			v, err := st.previewFeeWad(&ctx)
			c.compare(r, []uint256.Int{v}, err)
		case "hook.previewExactIn":
			fr, err := st.previewExactIn(&ctx, &feeWad)
			c.compare(r, []uint256.Int{fr.AmountInUsed, fr.GrossOut, fr.FeeOut, fr.SpotAfterWad}, err)
		case "hook.executeExactIn":
			fr, post, err := st.executeExactIn(&ctx, &feeWad)
			// storage read back in slot order: kappa(16), x(17), rs(18), is(19), rv(20), iv(21)
			c.compare(r, []uint256.Int{fr.AmountInUsed, fr.GrossOut, fr.FeeOut, fr.SpotAfterWad, post.Kappa, post.XWad,
				post.ReserveStable, post.IdleStable, post.ReserveVolatile, post.IdleVolatile}, err)
			if err == nil && !fr.AmountInUsed.IsZero() {
				filled++
			}
			// the rest of the storage is untouched, and the receiver is never mutated
			if err == nil {
				post.Kappa, post.XWad, post.ReserveStable, post.IdleStable = st.Kappa, st.XWad, st.ReserveStable, st.IdleStable
				post.ReserveVolatile, post.IdleVolatile = st.ReserveVolatile, st.IdleVolatile
				if post != before {
					c.fail("%s%v: executeExactIn changed non-book fields", r.F, r.A)
				}
			}
		default:
			t.Fatalf("unknown fn %s", r.F)
		}
		if st != before {
			c.fail("%s%v: receiver mutated", r.F, r.A)
		}
	}
	t.Logf("%s rows %d (%d kept) checks %d filled %d mismatches %d by fn %v", name, len(rows), kept(keep), c.checks, filled,
		c.fails, seen)
	require.Zero(t, c.fails)
}

func TestHookEdges(t *testing.T) {
	t.Parallel()
	swapEdgeHookRows(t, "hook_fill_edges.json.gz")
}

// LOAN_SCALE is immutable on the deployed hook (1e12); these rows patch its PUSH32 sites to 1 and 1e10.
func TestHookEdgesLoanScale(t *testing.T) {
	t.Parallel()
	swapEdgeHookRows(t, "hook_fill_loan_scale_edges.json.gz")
}

// The real 15000-sat sell (tx 0x46c3cd72..., block 51302916): the context captured by a calldata tap on a
// re-driven swap whose committed book equals the transaction's.
func TestHookEdgesLiveTx(t *testing.T) {
	t.Parallel()
	swapEdgeHookRows(t, "hook_live_tx_edges.json.gz")
}
