package nomiswapstable

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestGetAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		entityPool        entity.Pool
		tokenAmountIn     pool.TokenAmount
		tokenOut          string
		expectedAmountOut *pool.TokenAmount
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
			tokenAmountIn: pool.TokenAmount{
				Token:  "0x55d398326f99059fF775485246999027B3197955",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			tokenOut: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
			expectedAmountOut: &pool.TokenAmount{
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
			tokenAmountIn: pool.TokenAmount{
				Token:  "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			tokenOut: "0x55d398326f99059fF775485246999027B3197955",
			expectedAmountOut: &pool.TokenAmount{
				Token:  "0x55d398326f99059fF775485246999027B3197955",
				Amount: bignumber.NewBig("999850607765728933"),
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			poolSim, err := NewPoolSimulator(tc.entityPool)
			assert.Nil(t, err)
			calcAmountOutResult, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
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

func TestCloneState(t *testing.T) {
	t.Parallel()
	p, err := NewPoolSimulator(entity.Pool{
		Address:  "0x1e40450f8e21bb68490d7d91ab422888fb3d60f1",
		Exchange: "nomiswap-stable",
		Type:     "nomiswap-stable",
		Reserves: entity.PoolReserves{"634281143717214551397393", "166371541522087916283731"},
		Tokens: []*entity.PoolToken{
			{Address: "0x55d398326f99059ff775485246999027b3197955", Decimals: 18},
			{Address: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", Decimals: 18},
		},
		Extra: `{"swapFee":6,"token0PrecisionMultiplier":"1","token1PrecisionMultiplier":"1","a":"200000"}`,
	})
	require.NoError(t, err)

	testutil.TestCloneState(t, p, pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x55d398326f99059ff775485246999027b3197955",
			Amount: big.NewInt(1e18),
		},
		TokenOut: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d",
	}, nil)
}
