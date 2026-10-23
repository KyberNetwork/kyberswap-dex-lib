package v1

import (
	"context"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Routers keep one simulator per pool and give each request a CloneState copy to swap on. A swap
// applied to the clone must not change what the original quotes, or every later quote on that
// pool prices liquidity that does not exist.
func TestUpdateBalanceOnCloneLeavesOriginalQuotesUntouched(t *testing.T) {
	t.Parallel()
	swaps := 0
	for _, data := range poolList {
		var ep entity.Pool
		require.NoError(t, json.Unmarshal([]byte(data), &ep))
		original, err := NewPoolSimulator(ep)
		require.NoError(t, err)

		type quoteResult struct {
			res *poolpkg.CalcAmountOutResult
			err error
		}
		tokenIns := [2]poolpkg.TokenAmount{}
		for i := range 2 {
			tokenIns[i] = poolpkg.TokenAmount{Token: ep.Tokens[i].Address,
				Amount: new(big.Int).Div(original.GetReserves()[i], big.NewInt(10))}
		}
		quote := func(sim poolpkg.IPoolSimulator, i int) quoteResult {
			res, err := poolpkg.CalcAmountOut(context.Background(), sim, tokenIns[i], ep.Tokens[1-i].Address, nil)
			return quoteResult{res, err}
		}
		before := [2]quoteResult{quote(original, 0), quote(original, 1)}

		for i := range 2 {
			if before[i].err != nil {
				continue
			}
			clone := original.CloneState()
			q := quote(clone, i)
			require.Equal(t, before[i], q, "pool %s: a fresh clone quotes differently from the original", ep.Address)
			clone.UpdateBalance(poolpkg.UpdateBalanceParams{TokenAmountIn: tokenIns[i],
				TokenAmountOut: *q.res.TokenAmountOut, SwapInfo: q.res.SwapInfo})
			swaps++
			for j := range 2 {
				require.Equal(t, before[j], quote(original, j),
					"pool %s: swap %d on the clone changed the original's quote %d", ep.Address, i, j)
			}
		}
	}
	require.Positive(t, swaps, "no fixture pool could swap, so nothing was tested")
}
