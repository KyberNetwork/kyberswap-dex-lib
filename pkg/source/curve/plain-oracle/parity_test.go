package plainoracle

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
	Pool   string
	Quotes []goldenQuote
	Dys    []goldenQuote
	VP     []string
	Upd    []string
	Lp     []string
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

func loadLegacyGoldens(t *testing.T) []legacyGolden {
	b, err := os.ReadFile("testdata/legacy_goldens.json")
	require.NoError(t, err)
	var g []legacyGolden
	require.NoError(t, json.Unmarshal(b, &g))
	return g
}

func mustU(t *testing.T, s string) *uint256.Int {
	t.Helper()
	x, err := uint256.FromDecimal(s)
	require.NoError(t, err)
	return x
}

// TestLegacyParity replays every legacy golden through the native uint256 surface.
func TestLegacyParity(t *testing.T) {
	t.Parallel()
	goldens := loadLegacyGoldens(t)
	require.NotEmpty(t, goldens)
	for pi, g := range goldens {
		var e entity.Pool
		require.NoError(t, json.Unmarshal([]byte(g.Pool), &e))
		p, err := NewPoolSimulator(e)
		require.NoError(t, err, "pool %d", pi)

		for _, c := range g.Quotes {
			amt := mustU(t, c.Amount)
			res, err := quoteParity(func() (*pool.CalcAmtOutResult, error) {
				return p.CalcAmtOut(pool.CalcAmtOutParams{
					TokenAmountIn: pool.TokenAmt{Token: c.In, Amount: *amt},
					TokenOut:      c.Out,
				})
			})
			if c.Err != "" {
				require.Error(t, err, "pool %d quote %s->%s %s", pi, c.In, c.Out, c.Amount)
				require.Equal(t, c.Err, err.Error(), "pool %d quote %s->%s %s", pi, c.In, c.Out, c.Amount)
				continue
			}
			require.NoError(t, err, "pool %d quote %s->%s %s", pi, c.In, c.Out, c.Amount)
			require.Equal(t, c.Dy, res.TokenAmountOut.Amount.String(), "pool %d dy %s->%s %s", pi, c.In, c.Out, c.Amount)
		}
		for _, c := range g.Dys {
			i, j := mustIdx(t, c.In), mustIdx(t, c.Out)
			amt := mustU(t, c.Amount)
			dy, fee, err := quoteParityGetDy(p, i, j, amt, nil)
			if c.Err != "" {
				require.Error(t, err, "pool %d getDy %d->%d %s", pi, i, j, c.Amount)
				require.Equal(t, c.Err, err.Error(), "pool %d getDy %d->%d %s", pi, i, j, c.Amount)
				continue
			}
			require.NoError(t, err, "pool %d getDy %d->%d %s", pi, i, j, c.Amount)
			require.Equal(t, c.Dy, dy.String(), "pool %d getDy dy", pi)
			require.Equal(t, c.Fee, fee.String(), "pool %d getDy fee", pi)
		}
	}
}

func mustIdx(t *testing.T, s string) int {
	t.Helper()
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	require.NoError(t, err)
	return i
}

func quoteParityGetDy(p *PoolSimulator, i, j int, amt, dCached *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	type pair struct{ dy, fee *uint256.Int }
	res, err := quoteParity(func() (pair, error) {
		dy, fee, err := p.GetDy(i, j, amt, dCached)
		return pair{dy, fee}, err
	})
	return res.dy, res.fee, err
}

// TestLegacyParityState replays the legacy virtual price, liquidity math, and state update sequence.
func TestLegacyParityState(t *testing.T) {
	t.Parallel()
	for pi, g := range loadLegacyGoldens(t) {
		var e entity.Pool
		require.NoError(t, json.Unmarshal([]byte(g.Pool), &e))
		p, err := NewPoolSimulator(e)
		require.NoError(t, err)

		vp, dCached, err := p.GetVirtualPrice()
		require.NoError(t, err, "pool %d", pi)
		require.Equal(t, g.VP[0], vp.String(), "pool %d virtual price", pi)
		require.Equal(t, g.VP[1], dCached.String(), "pool %d D", pi)
		dy2, _, err := p.GetDy(0, 1, uint256.NewInt(100000), dCached)
		require.NoError(t, err)
		require.Equal(t, g.VP[2], dy2.String(), "pool %d getDy with cached D", pi)

		var lp []string
		for _, amt := range []string{"1000", "100000000000000000"} {
			w, _, werr := p.CalculateWithdrawOneCoin(mustU(t, amt), 0)
			if werr != nil {
				lp = append(lp, "err:"+werr.Error())
			} else {
				lp = append(lp, w.String())
			}
			ta, terr := p.CalculateTokenAmount([]*uint256.Int{mustU(t, amt), uint256.NewInt(0)}, true)
			if terr != nil {
				lp = append(lp, "err:"+terr.Error())
			} else {
				lp = append(lp, ta.String())
			}
		}
		require.Equal(t, g.Lp, lp, "pool %d liquidity math", pi)

		var upd []string
		n := len(e.Tokens)
		c := p.CloneState().(*PoolSimulator)
		for k := 0; k < 8; k++ {
			in := e.Tokens[k%n].Address
			out := e.Tokens[(k+1)%n].Address
			amt := uint256.NewInt(uint64(100000 * (k + 1)))
			res, err := quoteParity(func() (*pool.CalcAmtOutResult, error) {
				return p.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: in, Amount: *amt}, TokenOut: out})
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
		require.Equal(t, g.Upd, upd, "pool %d state sequence", pi)
	}
}
