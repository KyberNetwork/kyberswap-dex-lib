package poolsidev1

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		tokenAmountIn     poolpkg.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name: "[SwapTest1] Swap USDM for ARBINAUTS: check amountOut",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0xaDAa99e01aCCD6C3944a6cec33Da6fb09e2C6b16",
						Tokens:   []string{"0x59D9356E565Ab3A36dD77763Fc0d87fEaf85508C", "0x836975C507bfF631FCD7FBa875e9127C8A50dBa6"},
						Reserves: []*big.Int{utils.NewBig("105317811426017560516256"), utils.NewBig("18710208568631997600301116596")},
					},
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("2400000000000"),
				Token:  "0x59D9356E565Ab3A36dD77763Fc0d87fEaf85508C",
			},
			tokenOut:          "0x836975C507bfF631FCD7FBa875e9127C8A50dBa6",
			expectedAmountOut: utils.NewBig("425092265551443270"),
			expectedError:     nil,
		},
		{
			name: "[SwapTest2] Swap ARBINAUTS for USDM: check amountOut",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0xaDAa99e01aCCD6C3944a6cec33Da6fb09e2C6b16",
						Tokens:   []string{"0x59D9356E565Ab3A36dD77763Fc0d87fEaf85508C", "0x836975C507bfF631FCD7FBa875e9127C8A50dBa6"},
						Reserves: []*big.Int{utils.NewBig("105317811426017560516256"), utils.NewBig("18710208568631997600301116596")},
					},
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("746000000000000"),
				Token:  "0x836975C507bfF631FCD7FBa875e9127C8A50dBa6",
			},
			tokenOut:          "0x59D9356E565Ab3A36dD77763Fc0d87fEaf85508C",
			expectedAmountOut: utils.NewBig("4186558678"),
			expectedError:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
				return tc.poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
					Limit:         nil,
				})
			})

			if tc.expectedError != nil {
				assert.ErrorIs(t, tc.expectedError, err)
			} else {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		params           poolpkg.UpdateBalanceParams
		expectedReserves []*big.Int
	}{
		{
			name: "[UpdateBalance1] Swap USDM for ARBINAUTS: check Reserves",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0xaDAa99e01aCCD6C3944a6cec33Da6fb09e2C6b16",
						Tokens:   []string{"0x59D9356E565Ab3A36dD77763Fc0d87fEaf85508C", "0x836975C507bfF631FCD7FBa875e9127C8A50dBa6"},
						Reserves: []*big.Int{utils.NewBig("105317811426017560516256"), utils.NewBig("18710208568631997600301116596")},
					},
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x59D9356E565Ab3A36dD77763Fc0d87fEaf85508C", Amount: utils.NewBig("2400000000000")},
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x836975C507bfF631FCD7FBa875e9127C8A50dBa6", Amount: utils.NewBig("425092265551443270")},
				Fee:            poolpkg.TokenAmount{Token: "0x59D9356E565Ab3A36dD77763Fc0d87fEaf85508C", Amount: utils.NewBig("7200000000")},
			},
			expectedReserves: []*big.Int{utils.NewBig("105317811428417560516256"), utils.NewBig("18710208568206905334749673326")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.poolSimulator.UpdateBalance(tc.params)

			assert.Equal(t, 0, tc.poolSimulator.Info.Reserves[0].Cmp(tc.expectedReserves[0]))
			assert.Equal(t, 0, tc.poolSimulator.Info.Reserves[1].Cmp(tc.expectedReserves[1]))
		})
	}
}
