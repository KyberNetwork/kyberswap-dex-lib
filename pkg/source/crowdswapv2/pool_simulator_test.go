package crowdswapv2

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestGetAmountOut(t *testing.T) {
	poolAddress := "0x1788f8dec1c2054d653f8330eedcdf3dfbeb42ac"
	token0Address := "0x2aa69e007c32cf6637511353b89dce0b473851a9"
	token1Address := "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91"

	testCases := []struct {
		name              string
		entityPool        entity.Pool
		tokenAmountIn     poolPkg.TokenAmount
		tokenOut          string
		swapFee           *big.Int
		expectedAmountOut *poolPkg.TokenAmount
		expectedErr       error
	}{
		{
			name: "test token0 as tokenIn",
			entityPool: entity.Pool{
				Address:  poolAddress,
				SwapFee:  0.3, //%
				Exchange: "crowdswapv2",
				Type:     "v2",
				Reserves: []string{
					"38819698878426432914729",
					"46113879614283",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   token0Address,
						Swappable: true,
					},
					{
						Address:   token1Address,
						Swappable: true,
					},
				},
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  token0Address,
				Amount: utils.NewBig("100000000000000000000"),
			},
			tokenOut: token1Address,
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  token1Address,
				Amount: utils.NewBig("118130133815"),
			},
			expectedErr: nil,
		},
		{
			name: "test token1 as tokenIn",
			entityPool: entity.Pool{
				Address:  poolAddress,
				SwapFee:  0.3, //%
				Exchange: "crowdswapv2",
				Type:     "v2",
				Reserves: []string{
					"38819698878426432914729",
					"46113879614283",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   token0Address,
						Swappable: true,
					},
					{
						Address:   token1Address,
						Swappable: true,
					},
				},
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  token1Address,
				Amount: utils.NewBig("10000000000000"),
			},
			tokenOut: token0Address,
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  token0Address,
				Amount: utils.NewBig("6900956219144033296901"),
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := NewPoolSimulator(tc.entityPool)
			assert.Nil(t, err)
			calcAmountOutResult, err := pool.CalcAmountOut(tc.tokenAmountIn, tc.tokenOut)

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedAmountOut, calcAmountOutResult.TokenAmountOut)
		})
	}
}
