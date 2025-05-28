package weeth

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
			// tx: 0xf559588e97560b7ef8644b5c3c2fe4c95d9654659b6cc502fb43353dfaf4b168
			name: "it should return correct amount (wrap)",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{"0x35fa164735182de50811e8e2e824cfb9b6118ac2", "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee"},
					},
				},
				totalPooledEther: bignumber.NewBig("479746451523543911039175"),
				totalShares:      bignumber.NewBig("464768412137509601320862"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("31259633999414378581"),
					Token:  "0x35fa164735182de50811e8e2e824cfb9b6118ac2",
				},
				TokenOut: "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee",
			},
			expectedAmountOut: bignumber.NewBig("30283685083587393838"),
		},
		{
			// tx: 0x86b280508b052ea51a421a10265f24135918179dbd685e3bda2a98d849ed4b48
			name: "it should return correct amount (unwrap)",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{"0x35fa164735182de50811e8e2e824cfb9b6118ac2", "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee"},
					},
				},
				totalPooledEther: bignumber.NewBig("482437159360194010684174"),
				totalShares:      bignumber.NewBig("467375114083494601305331"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("29089204898328349375"),
					Token:  "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee",
				},
				TokenOut: "0x35fa164735182de50811e8e2e824cfb9b6118ac2",
			},
			expectedAmountOut: bignumber.NewBig("30026659435463879113"),
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
