package beets_ss

import (
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
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
						Reserves: []*big.Int{bignumber.NewBig(defaultReserve), bignumber.NewBig(defaultReserve)},
					},
				},
				totalSupply: bignumber.NewUint256("55239936004195121896978015"),
				totalAssets: bignumber.NewUint256("55319744731539794782367353"),
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: big.NewInt(1e18),
				Token:  strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDSonic]),
			},
			tokenOut:          Beets_Staked_Sonic_Address,
			expectedAmountOut: bignumber.NewBig("998557319312806390"),
			expectedError:     nil,
		},
		{
			name: "Swap from wS to stS with zero assets",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  Beets_Staked_Sonic_Address,
						Tokens:   []string{Beets_Staked_Sonic_Address, strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDSonic])},
						Reserves: []*big.Int{bignumber.NewBig(defaultReserve), bignumber.NewBig(defaultReserve)},
					},
				},
				totalSupply: bignumber.NewUint256("0"),
				totalAssets: bignumber.NewUint256("0"),
			},
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: bignumber.NewBig(defaultReserve),
				Token:  strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDSonic]),
			},
			tokenOut:          Beets_Staked_Sonic_Address,
			expectedAmountOut: bignumber.NewBig(defaultReserve),
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
