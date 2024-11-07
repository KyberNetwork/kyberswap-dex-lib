package rseth

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
			// tx: 0x1509967c8aa688eb70d1c8d73a8cb71221e4f6b0a0648c7f2a7fd22062548236
			name: "it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{
							"0xa1290d69c65a6fe4df752f95823fae25cb99e5a7", // rsETH
							"0xa35b1b31ce002fbf2058d22f30f95d405200a15b", // ETHx
						},
					},
				},

				minAmountToDeposit:  bignumber.NewBig("100000000000000"),
				totalDepositByAsset: map[string]*big.Int{"0xa35b1b31ce002fbf2058d22f30f95d405200a15b": bignumber.NewBig("802460400000000000000")},
				depositLimitByAsset: map[string]*big.Int{"0xa35b1b31ce002fbf2058d22f30f95d405200a15b": bignumber.NewBig("4197539600000000000000")},
				priceByAsset:        map[string]*big.Int{"0xa35b1b31ce002fbf2058d22f30f95d405200a15b": bignumber.NewBig("1015786347348446492")},
				rsETHPrice:          bignumber.NewBig("1000000000000000000"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("2756400000000000000"),
					Token:  "0xa35b1b31ce002fbf2058d22f30f95d405200a15b",
				},
				TokenOut: "0xa1290d69c65a6fe4df752f95823fae25cb99e5a7",
			},
			expectedAmountOut: bignumber.NewBig("2799913487831257910"),
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
