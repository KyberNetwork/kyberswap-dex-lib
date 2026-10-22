package msgpack

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPeachHooksRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("../liquidity-source/uniswap/v4/hooks/peach/testdata/live_pools.json")
	require.NoError(t, err)
	var fixture struct {
		Cases []struct {
			Name    string
			ChainID valueobject.ChainID
			Pool    entity.Pool
		}
	}
	require.NoError(t, json.Unmarshal(raw, &fixture))
	require.Len(t, fixture.Cases, 5)
	for _, c := range fixture.Cases {
		t.Run(c.Name, func(t *testing.T) {
			var extra v4.Extra
			require.NoError(t, json.Unmarshal([]byte(c.Pool.Extra), &extra))
			r0, r1 := v4.EstimateReservesFromTicks(extra.SqrtPriceX96, extra.Ticks)
			c.Pool.Reserves = []string{r0.String(), r1.String()}
			sim, err := v4.NewPoolSimulator(c.Pool, c.ChainID)
			require.NoError(t, err)
			// Token1 -> token0 has executable liquidity on all five fixtures, including BNB.
			amount := new(big.Int).Exp(big.NewInt(10), big.NewInt(max(int64(c.Pool.Tokens[1].Decimals)-6, 6)), nil)
			params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: c.Pool.Tokens[1].Address, Amount: amount}, TokenOut: c.Pool.Tokens[0].Address}
			for _, state := range []string{"fresh", "accepted"} {
				encoded, err := EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{sim.GetAddress(): sim})
				require.NoError(t, err)
				decoded, err := DecodePoolSimulatorsMap(encoded)
				require.NoError(t, err)
				restored := decoded[sim.GetAddress()].(*v4.PoolSimulator)
				want, err := sim.CalcAmountOut(params)
				require.NoError(t, err, state)
				got, err := restored.CalcAmountOut(params)
				require.NoError(t, err, state)
				require.Equal(t, want.TokenAmountOut, got.TokenAmountOut)
				require.Equal(t, want.Gas, got.Gas)
				require.Equal(t, sim.GetExchange(), restored.GetExchange())
				if state == "fresh" {
					sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *want.TokenAmountOut, Fee: *want.Fee, SwapInfo: want.SwapInfo})
					params = pool.CalcAmountOutParams{TokenAmountIn: *want.TokenAmountOut, TokenOut: params.TokenAmountIn.Token}
				}
			}
		})
	}
}
