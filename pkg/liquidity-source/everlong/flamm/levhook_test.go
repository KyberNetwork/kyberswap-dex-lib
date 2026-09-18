package everlongflamm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// Hook-level fixtures recorded by testdata/gen/LevRecorder.sol: per row, the LeverContext core built for
// pool.previewLever(up, amount) (captured by etching a calldata-echo probe over the hook), the swap hook's
// bookFor legs and reservationPriceWad for that context, and the deployed/compiled hook's frame(ctx) and
// previewLever(ctx) with revert data. levFrameFor/levQuote must reproduce every frame word, every fill word
// and every revert.

type levFixtureUints []*uint256.Int

func (u *levFixtureUints) UnmarshalJSON(b []byte) error {
	var raw []json.Number
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*u = make(levFixtureUints, len(raw))
	for i, n := range raw {
		v, err := uint256.FromDecimal(n.String())
		if err != nil {
			return fmt.Errorf("word %d: %w", i, err)
		}
		(*u)[i] = v
	}
	return nil
}

type levHookFixture struct {
	Block     uint64          `json:"block"`
	Fields    []string        `json:"fields"`
	Rows      int             `json:"rows"`
	Scenarios []string        `json:"scenarios"`
	Stride    int             `json:"stride"`
	V         levFixtureUints `json:"v"`
}

type levHookRow map[string]*uint256.Int

func levLoadHookFixture(t *testing.T, path string) (levHookFixture, []levHookRow) {
	t.Helper()
	var fx levHookFixture
	require.NoError(t, json.Unmarshal(readFixture(t, path), &fx))
	require.Len(t, fx.Fields, fx.Stride)
	require.Len(t, fx.V, fx.Rows*fx.Stride)
	rows := make([]levHookRow, fx.Rows)
	for i := range rows {
		rows[i] = levHookRow{}
		for j, name := range fx.Fields {
			rows[i][name] = fx.V[i*fx.Stride+j]
		}
	}
	return fx, rows
}

func (r levHookRow) flag(name string) bool { return !r[name].IsZero() }

func (r levHookRow) context() (*leverContext, *levBook) {
	ctx := &leverContext{
		Pool: poolContext{
			PhysicalPoolAsset: *r["ctx.physicalPoolAsset"],
			PostedPoolAsset:   *r["ctx.postedPoolAsset"],
			LiquidLoanAsset:   *r["ctx.liquidLoanAsset"],
			SuppliedLoanAsset: *r["ctx.suppliedLoanAsset"],
			DebtLoanAsset:     *r["ctx.debtLoanAsset"],
			ShareSupply:       *r["ctx.shareSupply"],
			PriceWad:          *r["ctx.priceWad"],
			PriceTs:           r["ctx.priceTs"].Uint64(),
			LoanCount:         uint8(r["ctx.loanCount"].Uint64()),
		},
		Up:        r.flag("ctx.up"),
		SpreadPpm: *r["ctx.spreadPpm"],
		AmountIn:  *r["ctx.amountIn"],
		MaxOut:    *r["ctx.maxOut"],
	}
	book := &levBook{
		Rs:                  r["book.rs"],
		Is:                  r["book.is"],
		Rv:                  r["book.rv"],
		Iv:                  r["book.iv"],
		ReservationPriceWad: r["reservationPriceWad"],
	}
	return ctx, book
}

var levPanicSelector = [4]byte{0x4e, 0x48, 0x7b, 0x71}

// levExpectedRevert maps recorded revert data (length, selector, first argument word) to the Go error.
func levExpectedRevert(length, selector, arg *uint256.Int) error {
	if length.IsZero() {
		return errMulDivOverflow
	}
	var sel [4]byte
	s := selector.Uint64()
	sel[0], sel[1], sel[2], sel[3] = byte(s>>24), byte(s>>16), byte(s>>8), byte(s)
	if sel == levPanicSelector && arg.Uint64() == 0x11 {
		return errPanicArithmetic
	}
	if err, ok := revertSelectors[sel]; ok {
		return err
	}
	return fmt.Errorf("unmapped revert selector %x arg %s", sel, arg.Dec())
}

var levLoanScale = uint256.NewInt(1_000_000_000_000)

// levCheckHookRows replays every captured row and returns the number of (fills, reverts) compared.
func levCheckHookRows(t *testing.T, fx levHookFixture, rows []levHookRow) (int, int) {
	fills, reverts := 0, 0
	for i, r := range rows {
		if !r.flag("captured") {
			continue
		}
		msg := fmt.Sprintf("row %d scenario %q up=%v amount=%s override=%s", i, fx.Scenarios[r["scenario"].Uint64()],
			r.flag("up"), r["amount"].Dec(), r["hookAmountOverride"].Dec())
		ctx, book := r.context()

		f, err := levFrameFor(&ctx.Pool, book)
		if r.flag("frameOk") {
			require.NoError(t, err, msg)
			require.Equal(t, r["frame.cv"], f.Cv, "frame.cv "+msg)
			require.Equal(t, r["frame.d"], (*uint256.Int)(f.D), "frame.d "+msg)
			require.Equal(t, r["frame.v"], f.V, "frame.v "+msg)
			require.Equal(t, r["frame.s"], f.S, "frame.s "+msg)
			require.Equal(t, r["frame.xAnchor"], f.XAnchor, "frame.xAnchor "+msg)
		} else {
			require.Error(t, err, msg)
		}

		fill, err := levQuote(ctx, book)
		if r.flag("hookOk") {
			require.NoError(t, err, msg)
			require.Equal(t, r["hook.amountInUsed"], fill.AmountInUsed, "amountInUsed "+msg)
			require.Equal(t, r["hook.grossOut"], fill.GrossOut, "grossOut "+msg)
			require.Equal(t, r["hook.virtualLegL18"], fill.VirtualLegL18, "virtualLegL18 "+msg)
			require.Equal(t, r["hook.crAfterWad"], fill.CrAfterWad, "crAfterWad "+msg)
			fills++
		} else {
			want := levExpectedRevert(r["hook.revertLen"], r["hook.revertSelector"], r["hook.revertArg"])
			require.True(t, errors.Is(err, want), "revert %v, got %v: %s", want, err, msg)
			reverts++
		}

		// Where core accepted the fill, its payout is the hook's fill netted on the loan grid: this pins that
		// the captured context is the one core priced (core's own gates are not modelled here).
		if r.flag("poolOk") && r.flag("hookOk") && r["hookAmountOverride"].IsZero() {
			require.Equal(t, r["pool.spreadPpm"], &ctx.SpreadPpm, msg)
			require.Equal(t, r["pool.crAfterWad"], fill.CrAfterWad, msg)
			if ctx.Up {
				require.Equal(t, r["pool.amountInUsed"], fill.AmountInUsed, msg)
				net := new(uint256.Int).Sub(fill.GrossOut, fill.VirtualLegL18)
				require.Equal(t, r["pool.amountOut"], net.Div(net, levLoanScale), msg)
			} else {
				require.Equal(t, r["pool.amountOut"], fill.GrossOut, msg)
				pay := divCeil(new(uint256.Int).Sub(fill.AmountInUsed, fill.VirtualLegL18), levLoanScale)
				require.Equal(t, r["pool.amountInUsed"], minU(pay, r["amount"]), msg)
			}
		}
	}
	return fills, reverts
}

func TestLevHookLocalFixture(t *testing.T) {
	t.Parallel()
	fx, rows := levLoadHookFixture(t, "testdata/lev_hook_local_fixture.json.gz")
	fills, reverts := levCheckHookRows(t, fx, rows)
	t.Logf("%d rows: %d fills and %d reverts matched", len(rows), fills, reverts)
	require.Greater(t, fills, 50)
	require.Greater(t, reverts, 10)
}

// TestLevVenueGolden ports test/flamm/lev/VenueGolden.t.sol's bit-exact sequence: rows 0-3 of the local fixture are
// the pre-step states of leverUp 0.05 BTC, leverDown out1/2, leverUp 0.05 BTC, leverDown (out1-out1/2+out3)/4
// and row 4 the post-sequence probe. The Go frame must land on every pinned CV/D and the Go fill, netted the
// way core settles it, on every pinned payout.
func TestLevVenueGolden(t *testing.T) {
	t.Parallel()
	_, rows := levLoadHookFixture(t, "testdata/lev_hook_local_fixture.json.gz")
	u := uint256.MustFromDecimal
	cv := []*uint256.Int{u("1999999999999999997898758"), u("2009999999999999997888251"), u("2007538233999999997890838"),
		u("2017538233999999997880332"), u("2015692939999999997882270")}
	d := []*uint256.Int{u("999999999999999997898758"), u("1009937593495999997888251"), u("1007468796959999997890838"),
		u("1017388072002999997880332"), u("1015541054419999997882270")}
	outs := []uint64{4_937_593_496, 1_230_883, 4_919_275_043, 922_647}

	var out1, out3 uint64
	for k := 0; k < 5; k++ {
		r := rows[k]
		require.Zero(t, r["scenario"].Uint64())
		ctx, book := r.context()
		f, err := levFrameFor(&ctx.Pool, book)
		require.NoError(t, err)
		require.Equal(t, cv[k], f.Cv, "cv%d", k)
		require.Equal(t, d[k], (*uint256.Int)(f.D), "d%d", k)
		if k == 4 {
			break
		}
		switch k {
		case 0, 2:
			require.Equal(t, uint64(5e6), r["amount"].Uint64())
		case 1:
			require.Equal(t, out1/2, r["amount"].Uint64())
		case 3:
			require.Equal(t, (out1-out1/2+out3)/4, r["amount"].Uint64())
		}
		fill, err := levQuote(ctx, book)
		require.NoError(t, err)
		var out uint64
		if ctx.Up {
			net := new(uint256.Int).Sub(fill.GrossOut, fill.VirtualLegL18)
			out = net.Div(net, levLoanScale).Uint64()
		} else {
			out = fill.GrossOut.Uint64()
		}
		require.Equal(t, outs[k], out, "out%d", k+1)
		switch k {
		case 0:
			out1 = out
		case 2:
			out3 = out
		}
	}
}

// TestLevFixtureDigests pins every generated lev fixture to the digest recorded in testdata/README.md.
func TestLevFixtureDigests(t *testing.T) {
	t.Parallel()
	for path, want := range map[string]string{
		"testdata/lev_curve_fork_fixture.json.gz": "798735119ee4e322ec929a75aa48d8855e630f622fc20aa4d3a27a54c30d4e9f",
		"testdata/lev_hook_fork_fixture.json.gz":  "f5d027dc34dbc37289edbf91312d5adf67217bef1a94e02883bee49319580641",
		"testdata/lev_hook_local_fixture.json.gz": "6e8b8d178b2f07c24aa0b4b94021b48a44f50455829786f11f07b92dbeabd59f",
		"testdata/lev_hook_band_fixture.json.gz":  "52e04fdf28c6224faa48e1a4cf581be3d8d070b5a0a53b65c47cf4651d2cb90d",
	} {
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "0x"+want, levSha256Hex(raw), path)
	}
}

func TestLevHookForkFixture(t *testing.T) {
	t.Parallel()
	fx, rows := levLoadHookFixture(t, "testdata/lev_hook_fork_fixture.json.gz")
	require.EqualValues(t, 51_317_000, fx.Block)
	fills, reverts := levCheckHookRows(t, fx, rows)
	t.Logf("%d rows: %d fills and %d reverts matched", len(rows), fills, reverts)
	require.Greater(t, fills, 150)
	require.Greater(t, reverts, 100)
}

// TestLevAssertAnchorAndBandFixture replays _assertAnchorAndBand, exposed by a harness over the compiled hook
// (testdata/gen/LevGoldenFixture.t.sol), across dust and in-domain states with the pre-anchor straddling the
// post-state's own anchor.
func TestLevAssertAnchorAndBandFixture(t *testing.T) {
	t.Parallel()
	var fx struct {
		Stride int             `json:"stride"`
		V      levFixtureUints `json:"v"`
	}
	require.NoError(t, json.Unmarshal(readFixture(t, "testdata/lev_hook_band_fixture.json.gz"), &fx))
	require.Equal(t, 6, fx.Stride)
	drops := 0
	for i := 0; i < len(fx.V)/fx.Stride; i++ {
		r := fx.V[i*fx.Stride : (i+1)*fx.Stride]
		msg := fmt.Sprintf("row %d %v", i, r)
		err := levAssertAnchorAndBand(r[0], r[1], r[2])
		if !r[3].IsZero() {
			require.NoError(t, err, msg)
			continue
		}
		want := levExpectedRevert(r[4], r[5], uZero)
		require.True(t, errors.Is(err, want), "revert %v, got %v: %s", want, err, msg)
		if errors.Is(err, ErrFillValueDrop) {
			drops++
		}
	}
	require.Greater(t, drops, 100)
}
