package plain

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// legacyGolden is one pool's output from the unmodified big.Int simulator (captured before the uint256 port).
type legacyGolden struct {
	Pool string
	Amts []goldenAmt
	Ins  []goldenAmt
	Upd  []string
}

type goldenAmt struct {
	In, Out, Amount, Dy, Fee, Err string
}

// quoteParity runs a native call and recovers a checked-math overflow panic as its error text, as the legacy golden does.
func quoteParity[T any](f func() (T, error)) (res T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return f()
}

func loadGoldens(t *testing.T) []legacyGolden {
	b, err := os.ReadFile("testdata/legacy_goldens.json")
	require.NoError(t, err)
	var g []legacyGolden
	require.NoError(t, json.Unmarshal(b, &g))
	return g
}

func mustDec(t *testing.T, s string) uint256.Int {
	t.Helper()
	var x uint256.Int
	require.NoError(t, x.SetFromDecimal(s))
	return x
}

// TestLegacyParity replays every legacy golden through the native uint256 surface: quotes, exact-out, and state updates.
func TestLegacyParity(t *testing.T) {
	t.Parallel()
	goldens := loadGoldens(t)
	require.NotEmpty(t, goldens)
	for pi, g := range goldens {
		var e entity.Pool
		require.NoError(t, json.Unmarshal([]byte(g.Pool), &e))
		p, err := NewPoolSimulator(e)
		require.NoError(t, err, "pool %d", pi)

		for _, c := range g.Amts {
			amt := mustDec(t, c.Amount)
			res, err := quoteParity(func() (*pool.CalcAmtOutResult, error) {
				return p.CalcAmtOut(pool.CalcAmtOutParams{
					TokenAmountIn: pool.TokenAmt{Token: c.In, Amount: amt},
					TokenOut:      c.Out,
				})
			})
			checkGoldenResult(t, pi, "out", c, err, res)
		}
		for _, c := range g.Ins {
			amt := mustDec(t, c.Amount)
			res, err := quoteParity(func() (*pool.CalcAmtInResult, error) {
				return p.CalcAmtIn(pool.CalcAmtInParams{
					TokenAmountOut: pool.TokenAmt{Token: c.Out, Amount: amt},
					TokenIn:        c.In,
				})
			})
			checkGoldenIn(t, pi, c, err, res)
		}
	}
}

func checkGoldenResult(t *testing.T, pi int, kind string, c goldenAmt, err error, res *pool.CalcAmtOutResult) {
	t.Helper()
	if c.Err != "" {
		require.Error(t, err, "pool %d %s %s->%s %s: want error %q", pi, kind, c.In, c.Out, c.Amount, c.Err)
		require.Equal(t, wantErrText(c.Err, err), strings.TrimSpace(err.Error()), "pool %d %s %s->%s %s", pi, kind, c.In, c.Out, c.Amount)
		return
	}
	require.NoError(t, err, "pool %d %s %s->%s %s", pi, kind, c.In, c.Out, c.Amount)
	require.Equal(t, c.Dy, res.TokenAmountOut.Amount.String(), "pool %d out dy %s->%s %s", pi, c.In, c.Out, c.Amount)
}

func checkGoldenIn(t *testing.T, pi int, c goldenAmt, err error, res *pool.CalcAmtInResult) {
	t.Helper()
	if c.Err != "" {
		require.Error(t, err, "pool %d in %s<-%s %s: want error %q", pi, c.In, c.Out, c.Amount, c.Err)
		require.Equal(t, wantErrText(c.Err, err), strings.TrimSpace(err.Error()), "pool %d in %s<-%s %s", pi, c.In, c.Out, c.Amount)
		return
	}
	require.NoError(t, err, "pool %d in %s<-%s %s", pi, c.In, c.Out, c.Amount)
	require.Equal(t, c.Dy, res.TokenAmountIn.Amount.String(), "pool %d in dy %s<-%s %s", pi, c.In, c.Out, c.Amount)
}

// wantErrText maps a legacy overflow panic to ErrOverflow, the typed error now returned by getD.
// Every other golden error text must match the native error exactly.
func wantErrText(golden string, err error) string {
	if golden == "panic: uint256 overflow" && errors.Is(err, ErrOverflow) {
		return strings.TrimSpace(err.Error())
	}
	return golden
}

// TestLegacyParityUpdateState replays the golden state sequence: quote, then UpdateState, then read reserves.
// A failed quote records "err" and skips the update, as the legacy capture did.
func TestLegacyParityUpdateState(t *testing.T) {
	t.Parallel()
	for pi, g := range loadGoldens(t) {
		var e entity.Pool
		require.NoError(t, json.Unmarshal([]byte(g.Pool), &e))
		p, err := NewPoolSimulator(e)
		require.NoError(t, err)
		n := len(e.Tokens)
		var got []string
		for k := 0; k < 3; k++ {
			in := e.Tokens[k%n].Address
			out := e.Tokens[(k+1)%n].Address
			amt := *uint256.NewInt(uint64(1000000 * (k + 1)))
			res, err := quoteParity(func() (*pool.CalcAmtOutResult, error) {
				return p.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: in, Amount: amt}, TokenOut: out})
			})
			if err != nil || res == nil {
				got = append(got, "err")
				continue
			}
			p.UpdateState(pool.UpdateStateParams{
				TokenAmountIn:  pool.TokenAmt{Token: in, Amount: amt},
				TokenAmountOut: *res.TokenAmountOut,
			})
			for _, r := range p.Info.Reserves {
				got = append(got, r.String())
			}
		}
		require.Equal(t, g.Upd, got, "pool %d state sequence", pi)
	}
}
