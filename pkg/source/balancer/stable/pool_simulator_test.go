package balancerstable

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwap(t *testing.T) {
	var pair = entity.Pool{
		Address:    "0x06df3b2bbb68adc8b0e302443692037ed9f91b42",
		ReserveUsd: 0,
		SwapFee:    0.0004,
		Exchange:   "balancer",
		Type:       "balancer-stable",
		Timestamp:  13529165,
		Reserves: []string{"4362365955985",
			"4342743177527924936049411",
			"6921895060068041759669604",
			"4198113236810"},
		Tokens: entity.PoolTokens{
			&entity.PoolToken{
				Address: "A",
				Weight:  250000000000000000,
			},
			&entity.PoolToken{
				Address: "B",
				Weight:  250000000000000000,
			},
			&entity.PoolToken{
				Address: "C",
				Weight:  250000000000000000,
			},
			&entity.PoolToken{
				Address: "D",
				Weight:  250000000000000000,
			},
		},
		Extra:       "{\"amplificationParameter\":{\"value\":60000,\"isUpdating\":false,\"precision\":1000}}",
		StaticExtra: "{\"vaultAddress\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\",\"poolId\":\"0x06df3b2bbb68adc8b0e302443692037ed9f91b42000000000000000000000012\",\"tokenDecimals\":[6,18,18,6]}",
	}
	var p, err = NewPoolSimulator(pair)
	require.Nil(t, err)
	assert.Equal(t, []string{"B", "C", "D"}, p.CanSwapTo("A"))
	assert.Equal(t, 0, len(p.CanSwapTo("Ax")))

	var tokenAmountIn = pool.TokenAmount{
		Token:     "A",
		Amount:    bignumber.NewBig10("100000000000"),
		AmountUsd: 100000,
	}
	var tokenOut = "D"
	result, _ := p.CalcAmountOut(tokenAmountIn, tokenOut)
	assert.NotNil(t, result.TokenAmountOut)
	assert.NotNil(t, result.Fee)
	assert.NotNil(t, result.Gas)
	assert.Equal(t, "99832311090", result.TokenAmountOut.Amount.String())
}
