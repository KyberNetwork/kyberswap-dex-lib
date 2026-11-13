package tricryptong

import (
	"fmt"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// CalcAmtOut must match CalcAmountOut on every field the router reads.
func TestCalcAmtOutMatchesCalcAmountOut(t *testing.T) {
	t.Parallel()

	var poolEntity entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolF5JSON), &poolEntity))
	p, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	tokens := []string{
		"0xdac17f958d2ee523a2206206994597c13d831ec7", // USDT
		"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", // WBTC
		"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", // WETH
	}
	amounts := []string{
		"0", "1", "2", "1000", "123456789",
		"1000000000", "10000000000", "100000000000",
		"1000000000000000000", "1000000000000000000000000", "1000000000000000000000000000000",
	}

	for _, in := range tokens {
		for _, out := range tokens {
			if in == out {
				continue
			}
			for _, amt := range amounts {
				t.Run(fmt.Sprintf("%s->%s/%s", in[:8], out[:8], amt), func(t *testing.T) {
					var amtIn uint256.Int
					require.NoError(t, amtIn.SetFromDecimal(amt))
					bigRes, bigErr := p.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: in, Amount: amtIn.ToBig()},
						TokenOut:      out,
					})
					u256Res, u256Err := p.CalcAmtOut(pool.CalcAmtOutParams{
						TokenAmountIn: pool.TokenAmt{Token: in, Amount: amtIn},
						TokenOut:      out,
					})

					require.Equal(t, bigErr, u256Err)
					if bigErr != nil {
						return
					}
					assert.Equal(t, bigRes.TokenAmountOut.Token, u256Res.TokenAmountOut.Token)
					assert.Equal(t, bigRes.TokenAmountOut.Amount.String(), u256Res.TokenAmountOut.Amount.Dec())
					assert.Equal(t, bigRes.Fee.Token, u256Res.Fee.Token)
					assert.Equal(t, bigRes.Fee.Amount.String(), u256Res.Fee.Amount.Dec())
					assert.Equal(t, bigRes.Gas, u256Res.Gas)
					assert.Equal(t, bigRes.SwapInfo, u256Res.SwapInfo)
					assert.Nil(t, u256Res.RemainingTokenAmountIn)
				})
			}
		}
	}
}
