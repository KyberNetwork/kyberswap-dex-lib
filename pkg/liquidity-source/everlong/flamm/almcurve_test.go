package everlongflamm

import (
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// almLoadFixture decodes a JSON fixture from testdata, transparently gunzipping *.gz.
func almLoadFixture(t *testing.T, name string, v any) {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", name))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	var dec *json.Decoder
	if strings.HasSuffix(name, ".gz") {
		zr, err := gzip.NewReader(f)
		require.NoError(t, err)
		defer func() { _ = zr.Close() }()
		dec = json.NewDecoder(zr)
	} else {
		dec = json.NewDecoder(f)
	}
	require.NoError(t, dec.Decode(v))
}

// almU parses a decimal fixture field.
func almU(t *testing.T, s string) *uint256.Int {
	t.Helper()
	v, err := uint256.FromDecimal(s)
	require.NoError(t, err, s)
	return v
}

// almRevertErr maps recorded revert data onto the port's error: a custom-error selector onto its
// sentinel, Panic(0x11/0x12) and the empty mulDiv revert onto the module's arithmetic errors.
func almRevertErr(t *testing.T, errHex string) error {
	t.Helper()
	data, err := hex.DecodeString(strings.TrimPrefix(errHex, "0x"))
	require.NoError(t, err)
	switch {
	case len(data) == 0:
		return errMulDivOverflow
	case len(data) == 36 && hex.EncodeToString(data[:4]) == "4e487b71":
		switch data[35] {
		case 0x11:
			return errPanicArithmetic
		case 0x12:
			return errPanicDivZero
		}
	case len(data) == 4:
		if hex.EncodeToString(data) == "1615e638" {
			return errFeeLnWadUndefined
		}
		if e, ok := revertSelectors[[4]byte(data)]; ok {
			return e
		}
	}
	t.Fatalf("unmapped revert data %s", errHex)
	return nil
}

// almExpectErr asserts the port refused exactly where the contract reverted.
func almExpectErr(t *testing.T, ok bool, errHex string, got error, ctx any) bool {
	t.Helper()
	if ok {
		require.NoError(t, got, "%+v", ctx)
		return true
	}
	want := almRevertErr(t, errHex)
	require.Truef(t, errors.Is(got, want), "want %v got %v at %+v", want, got, ctx)
	return false
}

type almGridRow struct {
	K        string    `json:"k"`
	A        string    `json:"a"`
	Up       string    `json:"up"`
	Dn       string    `json:"dn"`
	X        string    `json:"x"`
	Y        string    `json:"y"`
	Held     string    `json:"held"`
	P        string    `json:"p"`
	Sup      [4]string `json:"sup"`
	Anchor   string    `json:"anchor"`
	Kappa    string    `json:"kappa"`
	StableIn bool      `json:"stableIn"`
	Amt      string    `json:"amt"`
	Out      string    `json:"out"`
	XAfter   string    `json:"xAfter"`
	Unspent  string    `json:"unspent"`
	Stable   string    `json:"stable"`
	Volatile string    `json:"volatile"`
	Ok       bool      `json:"ok"`
	Err      string    `json:"err"`
}

func (r *almGridRow) support(t *testing.T) almSupport {
	return almSupport{AWad: *almU(t, r.Sup[0]), XLo: *almU(t, r.Sup[1]), XHi: *almU(t, r.Sup[2]), YHi: *almU(t, r.Sup[3])}
}

// TestAlmCurveGrid replays every row the deployed Base AlmCurve library (and the live hook's spot, for
// priceAtX) produced: supportFor, yAtX, priceAtX, reservesAt and swapExactInX96 over amplifications,
// spans, coordinates across and beyond the domain, anchors, scales and log-spaced amounts, reverts
// included.
func TestAlmCurveGrid(t *testing.T) {
	t.Parallel()
	var rows []almGridRow
	almLoadFixture(t, "alm_curve_grid.json.gz", &rows)
	require.Greater(t, len(rows), 10000)
	counts := map[string]int{}
	for i := range rows {
		r := &rows[i]
		counts[r.K]++
		switch r.K {
		case "sup":
			sup, err := almSupportFor(almU(t, r.A), almU(t, r.Up), almU(t, r.Dn))
			if almExpectErr(t, r.Ok, r.Err, err, r) {
				require.Equal(t, r.support(t), sup, "%+v", r)
			}
		case "y":
			y, err := almYAtX(almU(t, r.X), almU(t, r.A))
			if almExpectErr(t, r.Ok, r.Err, err, r) {
				require.Equal(t, r.Y, y.Dec(), "%+v", r)
			}
			full := almSupport{AWad: *almU(t, r.A), XLo: *almMinXWad, XHi: *almMaxXWad}
			st, vol, err := almReservesAt(&full, uQ96, uWad, almU(t, r.X))
			if almExpectErr(t, r.Ok, r.Err, err, r) {
				require.Equal(t, r.Y, st.Dec(), "%+v", r)
				require.Equal(t, r.Held, vol.Dec(), "%+v", r)
			}
		case "p":
			p, err := almPriceAtX(almU(t, r.X), almU(t, r.A))
			if almExpectErr(t, r.Ok, r.Err, err, r) {
				require.Equal(t, r.P, p.Dec(), "%+v", r)
			}
		case "res":
			sup := r.support(t)
			st, vol, err := almReservesAt(&sup, almU(t, r.Anchor), almU(t, r.Kappa), almU(t, r.X))
			if almExpectErr(t, r.Ok, r.Err, err, r) {
				require.Equal(t, r.Stable, st.Dec(), "%+v", r)
				require.Equal(t, r.Volatile, vol.Dec(), "%+v", r)
			}
		case "swap":
			sup := r.support(t)
			x, amt := almU(t, r.X), almU(t, r.Amt)
			out, xAfter, unspent, err := almSwapExactInX96(&sup, almU(t, r.Anchor), almU(t, r.Kappa), x, r.StableIn, amt)
			if almExpectErr(t, r.Ok, r.Err, err, r) {
				require.Equal(t, r.Out, out.Dec(), "%+v", r)
				require.Equal(t, r.XAfter, xAfter.Dec(), "%+v", r)
				require.Equal(t, r.Unspent, unspent.Dec(), "%+v", r)
			}
		default:
			t.Fatalf("unknown row kind %q", r.K)
		}
	}
	t.Logf("rows by kind: %v", counts)
}

// TestAlmSwapZeroInputIsNoOp pins the guards the bisection relies on: zero input never moves x, and a
// stable-in fill never lands right of the starting coordinate.
func TestAlmSwapZeroInputIsNoOp(t *testing.T) {
	t.Parallel()
	sup, err := almSupportFor(almU(t, "34000000000000000000"), uint256.NewInt(6e18), uint256.NewInt(6e18))
	require.NoError(t, err)
	x := uint256.NewInt(532533306204662064)
	for _, stableIn := range []bool{true, false} {
		out, xAfter, unspent, err := almSwapExactIn(&sup, x, stableIn, uZero)
		require.NoError(t, err)
		require.True(t, out.IsZero() && unspent.IsZero())
		require.Equal(t, *x, xAfter)
	}
	_, xAfter, _, err := almSwapExactIn(&sup, x, false, uOne)
	require.NoError(t, err)
	require.False(t, xAfter.Gt(x))
}
