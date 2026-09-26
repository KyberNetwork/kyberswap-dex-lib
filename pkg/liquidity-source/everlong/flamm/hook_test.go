package everlongflamm

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type hookFixtureState struct {
	K             string    `json:"k"`
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	AWad          string    `json:"aWad"`
	Sup           [4]string `json:"sup"`
	AnchorSqrtX96 string    `json:"anchorSqrtX96"`
	Rp            string    `json:"rp"`
	Kappa         string    `json:"kappa"`
	X             string    `json:"x"`
	Rs            string    `json:"rs"`
	Is            string    `json:"is"`
	Rv            string    `json:"rv"`
	Iv            string    `json:"iv"`
	RvWad         string    `json:"rvWad"`
	Fee           [8]string `json:"fee"`
	InvKappa      string    `json:"invKappa"`
	InvBand       string    `json:"invBand"`
	LoanScale     string    `json:"loanScale"`
}

type hookFixtureBook struct {
	Kappa string `json:"kappa"`
	Rs    string `json:"rs"`
	Is    string `json:"is"`
	Rv    string `json:"rv"`
	Iv    string `json:"iv"`
	X     string `json:"x"`
	Ok    bool   `json:"ok"`
	Err   string `json:"err"`
}

type hookFixtureFill struct {
	Used   string `json:"used"`
	Gross  string `json:"gross"`
	FeeOut string `json:"feeOut"`
	Spot   string `json:"spot"`
	Ok     bool   `json:"ok"`
	Err    string `json:"err"`
}

type hookFixtureRow struct {
	hookFixtureState
	S      string          `json:"s"`
	Phys   string          `json:"phys"`
	Posted string          `json:"posted"`
	In     bool            `json:"in"`
	Amt    string          `json:"amt"`
	Cap    string          `json:"cap"`
	FeeWad string          `json:"fee"`
	Book   hookFixtureBook `json:"book"`
	Pf     struct {
		V   string `json:"v"`
		Ok  bool   `json:"ok"`
		Err string `json:"err"`
	} `json:"pf"`
	Px   hookFixtureFill `json:"px"`
	Ex   hookFixtureFill `json:"ex"`
	Post hookFixtureBook `json:"post"`
}

// The "fee" key is the fee row (an array) on state rows and feeWad (a string) on fill rows.
func (r *hookFixtureRow) UnmarshalJSON(data []byte) error {
	type plain hookFixtureRow
	var probe struct {
		K string `json:"k"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	if probe.K == "state" {
		return json.Unmarshal(data, &r.hookFixtureState)
	}
	var aux struct {
		plain
		FeeWad string `json:"fee"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = hookFixtureRow(aux.plain)
	r.K, r.FeeWad = probe.K, aux.FeeWad
	if r.K == "spot" {
		return json.Unmarshal(data, &r.Pf)
	}
	return nil
}

func (f *hookFixtureState) state(t *testing.T) hookState {
	return hookState{
		AWad:                *almU(t, f.AWad),
		Support:             almSupport{AWad: *almU(t, f.Sup[0]), XLo: *almU(t, f.Sup[1]), XHi: *almU(t, f.Sup[2]), YHi: *almU(t, f.Sup[3])},
		AnchorSqrtX96:       *almU(t, f.AnchorSqrtX96),
		ReservationPriceWad: *almU(t, f.Rp),
		Kappa:               *almU(t, f.Kappa),
		XWad:                *almU(t, f.X),
		ReserveStable:       *almU(t, f.Rs),
		IdleStable:          *almU(t, f.Is),
		ReserveVolatile:     *almU(t, f.Rv),
		IdleVolatile:        *almU(t, f.Iv),
		RvWad:               *almU(t, f.RvWad),
		Fee:                 feeParamsOf(t, f.Fee),
		InvSkewKappaWad:     *almU(t, f.InvKappa),
		InvSkewBandWad:      *almU(t, f.InvBand),
		LoanScale:           *almU(t, f.LoanScale),
	}
}

func hookRequireBook(t *testing.T, want *hookFixtureBook, got *hookBook, ctx any) {
	t.Helper()
	require.Equal(t, []string{want.Kappa, want.Rs, want.Is, want.Rv, want.Iv, want.X},
		[]string{got.Kappa.Dec(), got.Rs.Dec(), got.Is.Dec(), got.Rv.Dec(), got.Iv.Dec(), got.X.Dec()}, "%+v", ctx)
}

func hookRequireFill(t *testing.T, want *hookFixtureFill, got *hookFillResult, ctx any) {
	t.Helper()
	require.Equal(t, []string{want.Used, want.Gross, want.FeeOut, want.Spot},
		[]string{got.AmountInUsed.Dec(), got.GrossOut.Dec(), got.FeeOut.Dec(), got.SpotAfterWad.Dec()}, "%+v", ctx)
}

// hookReplay checks one recorded context against bookFor, previewFeeWad, previewExactIn and the
// post-state executeExactIn commits, reverts included, and that the receiver is never mutated.
func hookReplay(t *testing.T, st *hookState, r *hookFixtureRow) (filled bool) {
	t.Helper()
	pre := *st
	ctx := swapContext{
		Pool:         poolContext{PhysicalPoolAsset: *almU(t, r.Phys), PostedPoolAsset: *almU(t, r.Posted)},
		PoolAssetIn:  r.In,
		AmountIn:     *almU(t, r.Amt),
		MaxAmountOut: *almU(t, r.Cap),
	}
	feeWad := almU(t, r.FeeWad)

	b, err := st.bookFor(&ctx.Pool)
	if almExpectErr(t, r.Book.Ok, r.Book.Err, err, r) {
		hookRequireBook(t, &r.Book, &b, r)
	}
	f, err := st.previewFeeWad(&ctx)
	if almExpectErr(t, r.Pf.Ok, r.Pf.Err, err, r) {
		require.Equal(t, r.Pf.V, f.Dec(), "%+v", r)
	}
	fr, err := st.previewExactIn(&ctx, feeWad)
	if almExpectErr(t, r.Px.Ok, r.Px.Err, err, r) {
		hookRequireFill(t, &r.Px, &fr, r)
	}
	fr, post, err := st.executeExactIn(&ctx, feeWad)
	if almExpectErr(t, r.Ex.Ok, r.Ex.Err, err, r) {
		hookRequireFill(t, &r.Ex, &fr, r)
		got := hookBook{Kappa: post.Kappa, Rs: post.ReserveStable, Is: post.IdleStable, Rv: post.ReserveVolatile,
			Iv: post.IdleVolatile, X: post.XWad}
		hookRequireBook(t, &r.Post, &got, r)
		// execute writes only the book
		post.Kappa, post.ReserveStable, post.IdleStable = pre.Kappa, pre.ReserveStable, pre.IdleStable
		post.ReserveVolatile, post.IdleVolatile, post.XWad = pre.ReserveVolatile, pre.IdleVolatile, pre.XWad
		require.Equal(t, pre, post)
		filled = !fr.AmountInUsed.IsZero()
	}
	require.Equal(t, pre, *st, "receiver mutated")
	return filled
}

// hookRowKey is a fill context's sampling stream (state, direction) and outcome class (sampleRows): whether bookFor
// and previewExactIn answer, with which revert, and whether it filled.
func hookRowKey(r *hookFixtureRow) (stream, class string) {
	if r.K == "state" || r.K == "spot" {
		return "", ""
	}
	return fmt.Sprintf("%s|%v", r.S, r.In), fmt.Sprintf("%v%s|%v%s|%v", r.Book.Ok, r.Book.Err, r.Px.Ok, r.Px.Err,
		r.Ex.Used == "0")
}

// TestHookFillGrid replays the live Base EverlongHook (storage overwritten per state: curve rows,
// anchors, books on and off the curve, empty and retracted books, fee rows) over gross != accounted,
// both directions, dust-to-overflow amounts, binding and non-binding output caps and fee rates (sampled under -race,
// fixture_sample_test.go).
func TestHookFillGrid(t *testing.T) {
	t.Parallel()
	var rows []hookFixtureRow
	almLoadFixture(t, "hook_fill_grid.json.gz", &rows)
	keep := sampleRows(len(rows), rowSampling{stride: 16, rare: 16},
		func(i int) (string, string) { return hookRowKey(&rows[i]) })
	states := map[string]hookState{}
	var fills, filled, reverted, spots int
	for i := range rows {
		if !keep[i] {
			continue
		}
		r := &rows[i]
		if r.K == "state" {
			states[r.ID] = r.state(t)
			continue
		}
		st, ok := states[r.S]
		require.True(t, ok)
		if r.K == "spot" {
			sp, err := st.spot()
			if almExpectErr(t, r.Pf.Ok, r.Pf.Err, err, r) {
				require.Equal(t, r.Pf.V, sp.Dec(), "%+v", r)
			}
			spots++
			continue
		}
		if hookReplay(t, &st, r) {
			filled++
		}
		if !r.Px.Ok {
			reverted++
		}
		fills++
	}
	require.Equal(t, len(states), spots)
	if !fixturesSampled() {
		require.Greater(t, fills, 9000)
		require.Greater(t, filled, 3000)
		require.Greater(t, reverted, 300)
	}
	t.Logf("contexts %d (%d of %d rows kept), filled %d, reverted %d", fills, kept(keep), len(rows), filled, reverted)
}

// TestHookLiveSwap reproduces the only real Swap on the Base pool: the 15000-sat sell of tx
// 0x46c3cd72a5860b2fe546e5a2130e066314e3777027151661e1e4f19a935901fa against the hook at block 51302915
// (the constructor seed book rescaled to 219423 sats), fee 17499999999999999, netting 11301759 USDC.
func TestHookLiveSwap(t *testing.T) {
	t.Parallel()
	var rows []hookFixtureRow
	almLoadFixture(t, "hook_live_swap.json", &rows)
	require.Len(t, rows, 2)
	st := rows[0].state(t)
	require.True(t, hookReplay(t, &st, &rows[1]))

	ctx := swapContext{
		Pool:         poolContext{PhysicalPoolAsset: *uint256.NewInt(219423)},
		PoolAssetIn:  true,
		AmountIn:     *uint256.NewInt(15000),
		MaxAmountOut: *almU(t, "101043362000000000000"),
	}
	fee, err := st.previewFeeWad(&ctx)
	require.NoError(t, err)
	require.Equal(t, "17499999999999999", fee.Dec())
	fr, post, err := st.executeExactIn(&ctx, &fee)
	require.NoError(t, err)
	var net uint256.Int
	net.Sub(&fr.GrossOut, &fr.FeeOut)
	net.Div(&net, &st.LoanScale)
	require.Equal(t, "11301759", net.Dec())
	require.Equal(t, "15000", fr.AmountInUsed.Dec())
	// the committed book is the one the chain stored (read back at block 51310000)
	require.Equal(t, "12779948558379", post.Kappa.Dec())
	require.Equal(t, "157080117684030704791", post.ReserveStable.Dec())
	require.Equal(t, "201304154388180895", post.IdleStable.Dec())
	require.Equal(t, "234423", post.ReserveVolatile.Dec())
	require.Equal(t, "532533306204662064", post.XWad.Dec())
}
