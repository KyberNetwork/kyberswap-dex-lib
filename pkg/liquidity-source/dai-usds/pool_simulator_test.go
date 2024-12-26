package dai_usds

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
)

var testPool = entity.Pool{
	Address:  "0x3225737a9bbb6473cb4a45b7244aca2befdb276a",
	Exchange: "dai-usds",
	Type:     "dai-usds",
	Reserves: []string{
		"10000000000000000000",
		"10000000000000000000",
	},
	Tokens: []*entity.PoolToken{
		{
			Address:   "0x6b175474e89094c44da98b954eedeac495271d0f",
			Swappable: true,
		},
		{
			Address:   "0xdc035d45d973e3ec169d2276ddab16f1e407384f",
			Swappable: true,
		},
	},
}

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
			name:       "test token0 as tokenIn",
			entityPool: testPool,
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x6b175474e89094c44da98b954eedeac495271d0f",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			tokenOut: "0xdc035d45d973e3ec169d2276ddab16f1e407384f",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0xdc035d45d973e3ec169d2276ddab16f1e407384f",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			expectedErr: nil,
		}, {
			name:       "test token1 as tokenIn",
			entityPool: testPool,
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0xdc035d45d973e3ec169d2276ddab16f1e407384f",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			tokenOut: "0x6b175474e89094c44da98b954eedeac495271d0f",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0x6b175474e89094c44da98b954eedeac495271d0f",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := NewPoolSimulator(tc.entityPool)
			assert.Nil(t, err)
			calcAmountOutResult, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
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
