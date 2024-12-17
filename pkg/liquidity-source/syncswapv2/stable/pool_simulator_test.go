package syncswapv2stable

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
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
			name: "test token1 as tokenIn",
			entityPool: entity.Pool{
				Address:  "0xd5a1a9680f083237c10c6357e72b37cafe1fb5de",
				Exchange: "syncswapv2-stable",
				Type:     "syncswapv2-stable",
				Reserves: []string{
					"1771167531",
					"8079308863505801735196",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x3355df6d4c9c3035724fd0e3914de96a5a83aaf4",
						Swappable: true,
					},
					{
						Address:   "0x5fc44e95eaa48f9eb84be17bd3ac66b6a82af709",
						Swappable: true,
					},
				},
				Extra: "{\"swapFee0To1\":100,\"swapFee1To0\":100,\"token0PrecisionMultiplier\":1000000000000,\"token1PrecisionMultiplier\":1,\"A\":80}",
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x5fc44e95eaa48f9eb84be17bd3ac66b6a82af709",
				Amount: bignumber.NewBig("100000000000000000000"),
			},
			tokenOut: "0x3355df6d4c9c3035724fd0e3914de96a5a83aaf4",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0x3355df6d4c9c3035724fd0e3914de96a5a83aaf4",
				Amount: bignumber.NewBig("95366156"),
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
