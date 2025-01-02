package beets_ss

import (
	"math/big"
	"strings"
	"testing"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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
			name: "Swap from wS to stS successfully",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  Beets_Staked_Sonic_Address,
						Tokens:   []string{Beets_Staked_Sonic_Address, strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDSonic])},
						Reserves: []*big.Int{utils.NewBig(defaultReserve), utils.NewBig(defaultReserve)},
					},
				},
				totalSupply: utils.NewUint256("55239936004195121896978015"),
				totalAssets: utils.NewUint256("55319744731539794782367353"),
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: big.NewInt(1e18),
				Token:  strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDSonic]),
			},
			tokenOut:          Beets_Staked_Sonic_Address,
			expectedAmountOut: utils.NewBig("998557319312806390"),
			expectedError:     nil,
		},
		{
			name: "Swap from wS to stS with zero assets",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  Beets_Staked_Sonic_Address,
						Tokens:   []string{Beets_Staked_Sonic_Address, strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDSonic])},
						Reserves: []*big.Int{utils.NewBig(defaultReserve), utils.NewBig(defaultReserve)},
					},
				},
				totalSupply: utils.NewUint256("0"),
				totalAssets: utils.NewUint256("0"),
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig(defaultReserve),
				Token:  strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDSonic]),
			},
			tokenOut:          Beets_Staked_Sonic_Address,
			expectedAmountOut: utils.NewBig(defaultReserve),
			expectedError:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.poolSimulator.CanSwapFrom(tc.tokenAmountIn.Token), []string{Beets_Staked_Sonic_Address})
			assert.Equal(t, tc.poolSimulator.CanSwapFrom(tc.tokenOut), []string{})

			result, err := tc.poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tc.tokenAmountIn,
				TokenOut:      tc.tokenOut,
				Limit:         nil,
			})

			if tc.expectedError != nil {
				assert.ErrorIs(t, tc.expectedError, err)
			} else {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}
