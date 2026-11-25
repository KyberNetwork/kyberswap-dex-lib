package aave

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

// legacyGolden is one pool's output from the unmodified big.Int simulator, captured before the uint256 port.
type legacyGolden struct {
	Frozen bool
	Pool   string
	Quotes []goldenQuote
	Under  []goldenQuote
	Dys    []goldenQuote
	VP     []string
	Lp     []string
	Upd    []string
}

type goldenQuote struct {
	In, Out, Amount, Dy, Fee, Err string
}

// quoteParity runs a native call and reports a panic as its error text.
func quoteParity[T any](f func() (T, error)) (res T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return f()
}

func loadAaveGoldens(t *testing.T, file string) []legacyGolden {
	b, err := os.ReadFile(file)
	require.NoError(t, err)
	var g []legacyGolden
	require.NoError(t, json.Unmarshal(b, &g))
	return g
}

func mustAaveU(t *testing.T, s string) *uint256.Int {
	t.Helper()
	x, err := uint256.FromDecimal(s)
	require.NoError(t, err)
	return x
}

// TestLegacyParity replays the frozen and unfrozen goldens through the native surface.
func TestLegacyParity(t *testing.T) {
	t.Parallel()
	for _, file := range []string{"testdata/legacy_goldens_frozen.json", "testdata/legacy_goldens_unfrozen.json"} {
		for pi, g := range loadAaveGoldens(t, file) {
			var e entity.Pool
			require.NoError(t, json.Unmarshal([]byte(g.Pool), &e))
			DepositFrozen = g.Frozen
			p, err := NewPoolSimulator(e)
			require.NoError(t, err, "%s pool %d", file, pi)

			for _, c := range g.Quotes {
				amt := mustAaveU(t, c.Amount)
				res, err := quoteParity(func() (*pool.CalcAmtOutResult, error) {
					return p.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: c.In, Amount: *amt}, TokenOut: c.Out})
				})
				checkAaveQuote(t, file, pi, "quote", c, err, res)
			}
			for _, c := range g.Under {
				amt := mustAaveU(t, c.Amount)
				res, err := quoteParity(func() (*pool.CalcAmtOutResult, error) {
					return p.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: c.In, Amount: *amt}, TokenOut: c.Out})
				})
				checkAaveQuote(t, file, pi, "underlying", c, err, res)
			}
		}
	}
}

func checkAaveQuote(t *testing.T, file string, pi int, kind string, c goldenQuote, err error, res *pool.CalcAmtOutResult) {
	t.Helper()
	if c.Err != "" {
		require.Error(t, err, "%s pool %d %s %s->%s %s: want %q", file, pi, kind, c.In, c.Out, c.Amount, c.Err)
		require.Equal(t, c.Err, err.Error(), "%s pool %d %s %s->%s %s", file, pi, kind, c.In, c.Out, c.Amount)
		return
	}
	require.NoError(t, err, "%s pool %d %s %s->%s %s", file, pi, kind, c.In, c.Out, c.Amount)
	require.Equal(t, c.Dy, res.TokenAmountOut.Amount.String(), "%s pool %d %s dy %s->%s %s", file, pi, kind, c.In, c.Out, c.Amount)
}

// TestLegacyParityState replays the GetDy, virtual price, liquidity math, and state sequence of each golden.
func TestLegacyParityState(t *testing.T) {
	t.Parallel()
	for _, file := range []string{"testdata/legacy_goldens_frozen.json", "testdata/legacy_goldens_unfrozen.json"} {
		for pi, g := range loadAaveGoldens(t, file) {
			var e entity.Pool
			require.NoError(t, json.Unmarshal([]byte(g.Pool), &e))
			DepositFrozen = g.Frozen
			p, err := NewPoolSimulator(e)
			require.NoError(t, err)
			n := len(e.Tokens)

			for _, c := range g.Dys {
				var i, j int
				_, _ = fmt.Sscanf(c.In, "%d", &i)
				_, _ = fmt.Sscanf(c.Out, "%d", &j)
				amt := mustAaveU(t, c.Amount)
				dy, fee, err := getDyParity(p, i, j, amt)
				if c.Err != "" {
					require.Error(t, err, "%s pool %d getDy %d->%d %s", file, pi, i, j, c.Amount)
					require.Equal(t, c.Err, err.Error(), "%s pool %d getDy %d->%d %s", file, pi, i, j, c.Amount)
					continue
				}
				require.NoError(t, err, "%s pool %d getDy %d->%d %s", file, pi, i, j, c.Amount)
				require.Equal(t, c.Dy, dy.String(), "%s pool %d getDy dy %d->%d %s", file, pi, i, j, c.Amount)
				require.Equal(t, c.Fee, fee.String(), "%s pool %d getDy fee %d->%d %s", file, pi, i, j, c.Amount)
			}

			vp, dCached, err := p.GetVirtualPrice()
			if len(g.VP) > 0 && g.VP[0] != "" && !isErrText(g.VP[0]) {
				require.NoError(t, err, "%s pool %d vp", file, pi)
				require.Equal(t, g.VP[0], vp.String(), "%s pool %d virtual price", file, pi)
				require.Equal(t, g.VP[1], dCached.String(), "%s pool %d D", file, pi)
			}

			var lp []string
			for _, a := range []string{"1000", "100000000000000000"} {
				w, _, werr := p.CalculateWithdrawOneCoin(mustAaveU(t, a), 0)
				if werr != nil {
					lp = append(lp, "err:"+werr.Error())
				} else {
					lp = append(lp, w.String())
				}
				ta, terr := p.CalculateTokenAmount([]*uint256.Int{mustAaveU(t, a), uint256.NewInt(0), uint256.NewInt(0)}, true)
				if terr != nil {
					lp = append(lp, "err:"+terr.Error())
				} else {
					lp = append(lp, ta.String())
				}
			}
			require.Equal(t, g.Lp, lp, "%s pool %d liquidity math", file, pi)

			var upd []string
			c := p.CloneState().(*PoolSimulator)
			for k := 0; k < 8; k++ {
				in := e.Tokens[k%n].Address
				o := e.Tokens[(k+1)%n].Address
				amt := uint256.NewInt(uint64(100000 * (k + 1)))
				res, err := quoteParity(func() (*pool.CalcAmtOutResult, error) {
					return p.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: in, Amount: *amt}, TokenOut: o})
				})
				if err != nil || res == nil {
					upd = append(upd, "err")
					continue
				}
				p.UpdateState(pool.UpdateStateParams{
					TokenAmountIn:  pool.TokenAmt{Token: in, Amount: *amt},
					TokenAmountOut: *res.TokenAmountOut,
				})
				for _, r := range p.Info.Reserves {
					upd = append(upd, r.String())
				}
			}
			for _, r := range c.Info.Reserves {
				upd = append(upd, "clone:"+r.String())
			}
			require.Equal(t, g.Upd, upd, "%s pool %d state sequence", file, pi)
		}
	}
}

func isErrText(s string) bool { return len(s) > 4 && s[:4] == "err:" }

func getDyParity(p *PoolSimulator, i, j int, amt *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	type pair struct{ dy, fee *uint256.Int }
	res, err := quoteParity(func() (pair, error) {
		dy, fee, err := p.GetDy(i, j, amt, nil)
		return pair{dy, fee}, err
	})
	return res.dy, res.fee, err
}
