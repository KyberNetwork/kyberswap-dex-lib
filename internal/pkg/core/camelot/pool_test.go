package camelot

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	poolPkg "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
)

func TestPool_getAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		pool              Pool
		amountIn          *big.Int
		tokenIn           string
		expectedAmountOut *big.Int
	}{
		{
			name: "it should return correct amount when swap from 0 to 1 stableSwap",
			pool: Pool{
				Pool: poolPkg.Pool{
					Info: poolPkg.PoolInfo{
						Tokens: []string{
							"0x5979d7b546e38e414f7e9822514be443a4800529",
							"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
						},
						Reserves: []*big.Int{
							utils.NewBig("1481252219344464578434"),
							utils.NewBig("3236537897421945761324"),
						},
					},
				},
				Token0FeePercent:     big.NewInt(40),
				Token1FeePercent:     big.NewInt(40),
				PrecisionMultiplier0: big.NewInt(1000000000000000000),
				PrecisionMultiplier1: big.NewInt(1000000000000000000),
				StableSwap:           true,
				FeeDenominator:       big.NewInt(100000),
			},
			amountIn:          utils.NewBig("1000000000000000000000"),
			tokenIn:           "0x5979d7b546e38e414f7e9822514be443a4800529",
			expectedAmountOut: utils.NewBig("1022352385458443941729"),
		},
		{
			name: "it should return correct amount when swap from 1 to 0 stableSwap",
			pool: Pool{
				Pool: poolPkg.Pool{
					Info: poolPkg.PoolInfo{
						Tokens: []string{
							"0x5979d7b546e38e414f7e9822514be443a4800529",
							"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
						},
						Reserves: []*big.Int{
							utils.NewBig("1481252219344464578434"),
							utils.NewBig("3236537897421945761324"),
						},
					},
				},
				Token0FeePercent:     big.NewInt(40),
				Token1FeePercent:     big.NewInt(40),
				PrecisionMultiplier0: big.NewInt(1000000000000000000),
				PrecisionMultiplier1: big.NewInt(1000000000000000000),
				StableSwap:           true,
				FeeDenominator:       big.NewInt(100000),
			},
			amountIn:          utils.NewBig("1000000000000000000000"),
			tokenIn:           "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			expectedAmountOut: utils.NewBig("708007269566256823589"),
		},
		{
			name: "it should return correct amount when swap from 0 to 1",
			pool: Pool{
				Pool: poolPkg.Pool{
					Info: poolPkg.PoolInfo{
						Tokens: []string{
							"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
							"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
						},
						Reserves: []*big.Int{
							utils.NewBig("4446200649353806287147"),
							utils.NewBig("7387929715114"),
						},
					},
				},
				Token0FeePercent:     big.NewInt(300),
				Token1FeePercent:     big.NewInt(300),
				PrecisionMultiplier0: big.NewInt(1000000000000000000),
				PrecisionMultiplier1: big.NewInt(1000000),
				StableSwap:           false,
				FeeDenominator:       big.NewInt(100000),
			},
			amountIn:          utils.NewBig("1000000000000000000000"),
			tokenIn:           "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			expectedAmountOut: utils.NewBig("1353204924908"),
		},
		{
			name: "it should return correct amount when swap from 1 to 0",
			pool: Pool{
				Pool: poolPkg.Pool{
					Info: poolPkg.PoolInfo{
						Tokens: []string{
							"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
							"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
						},
						Reserves: []*big.Int{
							utils.NewBig("4446169910492564197660"),
							utils.NewBig("7387985285550"),
						},
					},
				},
				Token0FeePercent:     big.NewInt(300),
				Token1FeePercent:     big.NewInt(300),
				PrecisionMultiplier0: big.NewInt(1000000000000000000),
				PrecisionMultiplier1: big.NewInt(1000000),
				StableSwap:           false,
				FeeDenominator:       big.NewInt(100000),
			},
			amountIn:          utils.NewBig("1000000000000000000000"),
			tokenIn:           "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			expectedAmountOut: utils.NewBig("4446169877545485328691"),
		},
		{
			name: "it should return correct amount when swap from 1 to 0 with owner fee",
			pool: Pool{
				Pool: poolPkg.Pool{
					Info: poolPkg.PoolInfo{
						Tokens: []string{
							"0x5979d7b546e38e414f7e9822514be443a4800529",
							"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
						},
						Reserves: []*big.Int{
							utils.NewBig("1118199781455197144456"),
							utils.NewBig("2468857930499458348529"),
						},
					},
				},
				Token0FeePercent:     big.NewInt(40),
				Token1FeePercent:     big.NewInt(40),
				PrecisionMultiplier0: big.NewInt(1000000000000000000),
				PrecisionMultiplier1: big.NewInt(1000000000000000000),
				StableSwap:           true,
				Factory: &Factory{
					FeeTo:         "0x6a63830e24f9a2f9c295fb2150107d0390ed1448",
					OwnerFeeShare: big.NewInt(40000),
				},
				FeeDenominator: big.NewInt(100000),
			},
			amountIn:          utils.NewBig("1000000000000000000"),
			tokenIn:           "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			expectedAmountOut: utils.NewBig("898082364338357291"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			amountOut := tc.pool.getAmountOut(tc.amountIn, tc.tokenIn)

			assert.Equal(t, tc.expectedAmountOut, amountOut)
		})
	}
}
