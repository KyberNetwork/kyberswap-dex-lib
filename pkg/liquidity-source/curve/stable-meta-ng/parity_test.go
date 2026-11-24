package stablemetang

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// goldenRow is one legacy result. Err is the legacy error text; "" means a value.
type goldenRow struct {
	In, Out, Amount, Dy, Fee, Err string
}

// goldenPool is one pool's legacy output: quotes, exact-out, and a state walk.
type goldenPool struct {
	Pool   string
	Quotes []goldenRow
	Ins    []goldenRow
	Upd    []string
}

// changedRows lists golden rows that may change under the negative-result rule (negative D1 and negative reserve).
// Every other row must match legacy exactly. Empty: no golden row changed.
var changedRows = map[string]string{}

func loadMetaNgGolden(t *testing.T) []goldenPool {
	t.Helper()
	b, err := os.ReadFile("testdata/legacy_goldens.json")
	require.NoError(t, err)
	var g []goldenPool
	require.NoError(t, json.Unmarshal(b, &g))
	return g
}

func parityErr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// safeQuote runs a native call and turns a panic into "panic: <value>", the form the golden records.
func safeQuote(s *PoolSimulator, p pool.CalcAmtOutParams) (res *pool.CalcAmtOutResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return s.CalcAmtOut(p)
}

func safeExactOut(s *PoolSimulator, p pool.CalcAmtInParams) (res *pool.CalcAmtInResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return s.CalcAmtIn(p)
}

func keyOf(pool int, kind string, idx int) string {
	return fmt.Sprintf("%d/%s/%d", pool, kind, idx)
}

// TestLegacyParity replays the golden quote and exact-out rows through the native uint256 surface.
func TestLegacyParity(t *testing.T) {
	t.Parallel()
	for pi, gp := range loadMetaNgGolden(t) {
		s := newParitySim(t, gp)
		for idx, c := range gp.Quotes {
			key := keyOf(pi, "Quotes", idx)
			amt := mustAmt(t, c.Amount)
			res, err := safeQuote(s, pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: c.In, Amount: amt}, TokenOut: c.Out})
			checkRow(t, key, c, parityErr(err), res, nil)
		}
		for idx, c := range gp.Ins {
			key := keyOf(pi, "Ins", idx)
			amt := mustAmt(t, c.Amount)
			res, err := safeExactOut(s, pool.CalcAmtInParams{TokenAmountOut: pool.TokenAmt{Token: c.Out, Amount: amt}, TokenIn: c.In})
			checkRow(t, key, c, parityErr(err), nil, res)
		}
	}
}

// TestLegacyParityState replays each golden state walk: quote, then UpdateState, reading reserves after each step.
// A quote the native path rejects leaves the state unchanged and records "err:<text>", as the golden does.
func TestLegacyParityState(t *testing.T) {
	t.Parallel()
	for pi, gp := range loadMetaNgGolden(t) {
		s := newParitySim(t, gp)
		toks := s.GetTokens()
		var upd []string
		for k := range 8 {
			amt := mustAmt(t, fmt.Sprint(100000*(k+1)))
			res, err := s.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: toks[0], Amount: amt}, TokenOut: toks[1]})
			if err != nil {
				upd = append(upd, "err:"+err.Error())
				continue
			}
			s.UpdateState(pool.UpdateStateParams{TokenAmountIn: pool.TokenAmt{Token: toks[0], Amount: amt},
				TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo})
			for _, r := range s.GetReserves() {
				upd = append(upd, r.Dec())
			}
		}
		require.Equal(t, gp.Upd, upd, "pool %d: state walk", pi)
	}
}

func mustAmt(t *testing.T, dec string) uint256.Int {
	t.Helper()
	var amt uint256.Int
	require.NoError(t, amt.SetFromDecimal(dec))
	return amt
}

// newParitySim builds the meta pool and its base pool from the fixtures embedded in the golden pool entry.
func newParitySim(t *testing.T, gp goldenPool) *PoolSimulator {
	t.Helper()
	var e entity.Pool
	require.NoError(t, json.Unmarshal([]byte(gp.Pool), &e))
	s, err := NewPoolSimulator(e, parityBasePools(t))
	require.NoError(t, err)
	return s
}

// checkRow compares one native result with its golden row. Rows in changedRows must produce the
// negative-result error; every other row must match legacy exactly.
func checkRow(t *testing.T, key string, c goldenRow, gotErr string, out *pool.CalcAmtOutResult, in *pool.CalcAmtInResult) {
	t.Helper()
	if want, ok := changedRows[key]; ok {
		require.Equal(t, want, gotErr, key)
		return
	}
	if c.Err != "" {
		require.Equal(t, c.Err, gotErr, key)
		return
	}
	require.Empty(t, gotErr, key)
	if out != nil {
		require.Equal(t, c.Dy, out.TokenAmountOut.Amount.Dec(), key)
		require.Equal(t, c.Fee, out.Fee.Amount.Dec(), key)
	} else {
		require.Equal(t, c.Dy, in.TokenAmountIn.Amount.Dec(), key)
		require.Equal(t, c.Fee, in.Fee.Amount.Dec(), key)
	}
}
