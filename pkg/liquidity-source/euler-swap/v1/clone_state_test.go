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
// applied to the clone must leave the original exactly as built, or every later quote on that
// pool sees liquidity that does not exist.
func TestUpdateBalanceOnCloneLeavesOriginalUntouched(t *testing.T) {
	t.Parallel()
	swaps := 0
	for _, data := range poolList {
		var ep entity.Pool
		require.NoError(t, json.Unmarshal([]byte(data), &ep))
		original, err := NewPoolSimulator(ep)
		require.NoError(t, err)
		asBuilt, err := NewPoolSimulator(ep)
		require.NoError(t, err)

		for i := range 2 {
			clone := original.CloneState()
			tokenIn := poolpkg.TokenAmount{Token: ep.Tokens[i].Address,
				Amount: new(big.Int).Div(original.GetReserves()[i], big.NewInt(10))}
			res, err := poolpkg.CalcAmountOut(context.Background(), clone, tokenIn, ep.Tokens[1-i].Address, nil)
			if err != nil {
				continue
			}
			clone.UpdateBalance(poolpkg.UpdateBalanceParams{TokenAmountIn: tokenIn,
				TokenAmountOut: *res.TokenAmountOut, SwapInfo: res.SwapInfo})
			swaps++
		}
		require.Equal(t, asBuilt, original, "pool %s changed by a swap on its clone", ep.Address)
	}
	require.Positive(t, swaps, "no fixture pool could swap, so nothing was tested")
}
