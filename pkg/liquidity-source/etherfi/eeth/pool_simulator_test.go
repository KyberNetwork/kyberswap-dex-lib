package eeth

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		poolSimulator     *PoolSimulator
		param             poolpkg.CalcAmountOutParams
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			// tx: 0xc54ef903cfd952d9e0e5ae1e3061f8456ca5588ef35dd8a88ab790f9a87fc5c0
			name: "it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x35fa164735182de50811e8e2e824cfb9b6118ac2"},
					},
				},
				totalPooledEther: bignumber.NewBig("478349632983976798301885"),
				totalShares:      bignumber.NewBig("463434527744908632824686"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("10000000000000000"),
					Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				},
				TokenOut: "0x35fa164735182de50811e8e2e824cfb9b6118ac2",
			},
			expectedAmountOut: bignumber.NewBig("9999999999999999"),
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

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()
	t.Run("it should update balance correctly", func(t *testing.T) {
		poolSimulator := &PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x35fa164735182de50811e8e2e824cfb9b6118ac2"},
				},
			},
			totalPooledEther: bignumber.NewBig("478349632983976798301885"),
			totalShares:      bignumber.NewBig("463434527744908632824686"),
		}

		params := poolpkg.UpdateBalanceParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Amount: bignumber.NewBig("10000000000000000"),
			},
			TokenAmountOut: poolpkg.TokenAmount{
				Amount: bignumber.NewBig("9688196578180132"),
			},
		}

		poolSimulator.UpdateBalance(params)

		assert.Zero(t, poolSimulator.totalPooledEther.Cmp(bignumber.NewBig("478349642983976798301885")))
		assert.Zero(t, poolSimulator.totalShares.Cmp(bignumber.NewBig("463434537433105211004818")))
	})
}
