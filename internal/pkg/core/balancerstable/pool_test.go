package balancerstable

import (
	"testing"

	poolpkg "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
				Address: "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
				Weight:  250000000000000000,
			},
			&entity.PoolToken{
				Address: "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063",
				Weight:  250000000000000000,
			},
			&entity.PoolToken{
				Address: "0xa3fa99a148fa48d14ed51d610c367c61876997f1",
				Weight:  250000000000000000,
			},
			&entity.PoolToken{
				Address: "0xc2132d05d31c914a87c6611c10748aeb04b58e8f",
				Weight:  250000000000000000,
			},
		},
		Extra:       "{\"amplificationParameter\":{\"value\":60000,\"isUpdating\":false,\"precision\":1000}}",
		StaticExtra: "{\"vaultAddress\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\",\"poolId\":\"0x06df3b2bbb68adc8b0e302443692037ed9f91b42000000000000000000000012\",\"tokenDecimals\":[6,18,18,6]}",
	}
	var pool, _ = NewPool(pair)
	var tokenAmountIn = poolpkg.TokenAmount{
		Token:     "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
		Amount:    utils.NewBig10("100000000000"),
		AmountUsd: 100000,
	}
	var tokenOut = "0xc2132d05d31c914a87c6611c10748aeb04b58e8f"
	result, _ := pool.CalcAmountOut(tokenAmountIn, tokenOut)
	assert.NotNil(t, result.TokenAmountOut)
	assert.NotNil(t, result.Fee)
	assert.NotNil(t, result.Gas)
	assert.Equal(t, "99832311090", result.TokenAmountOut.Amount.String())
	logrus.Info(result.TokenAmountOut.Amount.String())
	logrus.Info(result.Fee.Amount.String())
	logrus.Info(result.Gas)
}
