package nomiswapstable

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGetAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		entityPool        entity.Pool
		tokenAmountIn     poolPkg.TokenAmount
		tokenOut          string
		expectedAmountOut *poolPkg.TokenAmount
		expectedErr       error
	}{
		{
			name: "test token0 as tokenIn",
			entityPool: entity.Pool{
				Address:  "0x1e40450F8E21BB68490D7D91Ab422888Fb3D60f1",
				Exchange: "nomiswap",
				Type:     "nomiswap-stable",
				Reserves: []string{
					"53332989360391363843011",
					"74994257625190868514451",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x55d398326f99059fF775485246999027B3197955",
						Swappable: true,
					},
					{
						Address:   "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
						Swappable: true,
					},
				},
				Extra: "{\"swapFee\":6,\"token0PrecisionMultiplier\":1,\"token1PrecisionMultiplier\":1,\"a\":200000}",
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x55d398326f99059fF775485246999027B3197955",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			tokenOut: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
				Amount: bignumber.NewBig("1000029391004839352"),
			},
			expectedErr: nil,
		}, {
			name: "test token1 as tokenIn",
			entityPool: entity.Pool{
				Address:  "0x1e40450F8E21BB68490D7D91Ab422888Fb3D60f1",
				Exchange: "nomiswap",
				Type:     "nomiswap-stable",
				Reserves: []string{
					"53332989360391363843011",
					"74994257625190868514451",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x55d398326f99059fF775485246999027B3197955",
						Swappable: true,
					},
					{
						Address:   "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
						Swappable: true,
					},
				},
				Extra: "{\"swapFee\":6,\"token0PrecisionMultiplier\":1,\"token1PrecisionMultiplier\":1,\"a\":200000}",
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			tokenOut: "0x55d398326f99059fF775485246999027B3197955",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0x55d398326f99059fF775485246999027B3197955",
				Amount: bignumber.NewBig("999850607765728933"),
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := NewPoolSimulator(tc.entityPool)
			assert.Nil(t, err)
			calcAmountOutResult, err := testutil.MustConcurrentSafe[*poolPkg.CalcAmountOutResult](t, func() (any, error) {
				return pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
					Limit:         nil,
				})
			})

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedAmountOut, calcAmountOutResult.TokenAmountOut)
		})
	}
}
