package everlongflamm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// quotesDigest is the sha256 of every quote of one fixed sweep over the replay fixtures' recorded refreshes (Base's,
// the lever's and BSC's: each pool's first, warm, volatile and capped states), and how many quotes it took. On a
// fresh simulator of each state, every ladder is quoted at each of its points, half way to each from the one below,
// at half its first and one past its top; then, after a fill at each ladder's first point, every ladder again at
// each of its points, the pool moved. A quote's line is its fixture, state, pool, direction and input with the
// whole result: the output, the fee, the input left, the gas and the swap info, or the error. `limit` gives each
// simulator the swap limit its quotes are asked with, nil for none; no fill is applied with it.
func quotesDigest(t *testing.T, limit func() pool.SwapLimit) (string, int) {
	t.Helper()
	h, n := sha256.New(), 0
	line := func(where string, s *PoolSimulator, p entity.Pool, l Ladder, amount *uint256.Int, lim pool.SwapLimit) {
		res, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{
			Token: p.Tokens[l.In].Address, Amount: amount.ToBig()}, TokenOut: p.Tokens[l.Out].Address, Limit: lim})
		out := fmt.Sprint(err)
		if err == nil {
			out = fmt.Sprintf("%s %s %s %s %s %s %d %+v", res.TokenAmountOut.Token, res.TokenAmountOut.Amount,
				res.Fee.Token, res.Fee.Amount, res.RemainingTokenAmountIn.Token, res.RemainingTokenAmountIn.Amount,
				res.Gas, res.SwapInfo)
		}
		_, _ = fmt.Fprintf(h, "%s|%s|%d>%d|%s|%s\n", where, p.Address, l.In, l.Out, amount.Dec(), out)
		n++
	}
	for _, path := range []string{"testdata/base-fork.json.gz", "testdata/base-fork-lever.json.gz",
		"testdata/bsc-fork.json.gz"} {
		fx := loadFixture(t, path)
		for state, pools := range [][]entity.Pool{fx.Pools, fx.Warm, fx.Volatile, fx.Capped} {
			for _, p := range pools {
				p = quoting(t, p)
				where := fmt.Sprintf("%s|%d", path, state)
				fresh := func() *PoolSimulator {
					s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
					require.NoError(t, err)
					return s
				}
				ladders := extraOf(t, p).Ladders
				s, lim := fresh(), limit()
				for _, l := range ladders {
					var below, a uint256.Int
					for k, pt := range l.Points {
						if k == 0 {
							line(where, s, p, l, a.Rsh(pt[0], 1), lim)
						} else {
							line(where, s, p, l, a.Rsh(a.Add(&below, pt[0]), 1), lim)
						}
						line(where, s, p, l, pt[0], lim)
						below.Set(pt[0])
					}
					line(where, s, p, l, a.AddUint64(&below, 1), lim)
				}
				for _, first := range ladders {
					s, lim := fresh(), limit()
					in := pool.TokenAmount{Token: p.Tokens[first.In].Address, Amount: first.Points[0][0].ToBig()}
					res, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: in,
						TokenOut: p.Tokens[first.Out].Address, Limit: lim})
					if err != nil {
						continue
					}
					s.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in, TokenAmountOut: *res.TokenAmountOut,
						Fee: *res.Fee, SwapInfo: res.SwapInfo})
					for _, l := range ladders {
						for _, pt := range l.Points {
							line(fmt.Sprintf("%s|after %d>%d", where, first.In, first.Out), s, p, l, pt[0], lim)
						}
					}
				}
			}
		}
	}
	require.Positive(t, n)
	return hex.EncodeToString(h.Sum(nil)), n
}

// The digest of the sweep with no swap limit, logged: the figure TestWithNoSwapLimitEveryQuoteIsUnchanged pins.
func TestQuotesDigestWithNoSwapLimit(t *testing.T) {
	d, n := quotesDigest(t, func() pool.SwapLimit { return nil })
	t.Logf("%d quotes, sha256 %s", n, d)
}
