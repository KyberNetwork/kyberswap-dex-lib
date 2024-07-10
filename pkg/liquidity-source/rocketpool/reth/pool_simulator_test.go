package reth

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		poolSimulator     *PoolSimulator
		param             poolpkg.CalcAmountOutParams
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			// tx: 0xc10375968ca89dc807568cb40f57eef2792cae5a6438f083834b4609691ceef2
			name: "it should return correct amount (deposit)",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0xae78736cd615f374d3085123a210448e74fc6393"},
					},
				},
				depositEnabled:         true,
				minimumDeposit:         bignumber.NewBig("10000000000000000"),
				balance:                bignumber.NewBig("17963940799090443727000"),
				maximumDepositPoolSize: bignumber.NewBig("18000000000000000000000"),
				depositFee:             bignumber.NewBig("500000000000000"),
				totalRETHSupply:        bignumber.NewBig("563912813663573766722840"),
				totalETHBalance:        bignumber.NewBig("619583685490020782650352"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("15000000000000000000"),
					Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				},
				TokenOut: "0xae78736cd615f374d3085123a210448e74fc6393",
			},
			expectedAmountOut: bignumber.NewBig("13645392957957896455"),
		},
		{
			// tx: 0x8eb5a21d1d0bddd30a74628bb246db25c7f28826130c408c45a94a64ee8b5701
			name: "it should return correct amount (burn)",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0xae78736cd615f374d3085123a210448e74fc6393"},
					},
				},
				totalETHBalance: bignumber.NewBig("612577958207564412422016"),
				totalRETHSupply: bignumber.NewBig("557175055422211468874658"),
				excessBalance:   bignumber.NewBig("20053486147767716199171"),
				rETHBalance:     bignumber.NewBig("4217179378819361040197"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("8000000000000000000"),
					Token:  "0xae78736cd615f374d3085123a210448e74fc6393",
				},
				TokenOut: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			},
			expectedAmountOut: bignumber.NewBig("8795482888132817700"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountOut(tc.param)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			}

			if tc.expectedAmountOut != nil {
				assert.Zero(t, tc.expectedAmountOut.Cmp(result.TokenAmountOut.Amount))
			}
		})
	}
}
