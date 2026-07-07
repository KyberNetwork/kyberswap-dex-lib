package pufeth

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
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
			// tx: 0x43dc3b135e5ac0d5f5b8a35b90e0f08c7ad01703caaffdec4b3778f1d3e72a2e
			name: "[stETH] it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{PUFETH, STETH, WSTETH},
					},
				},
				totalSupply: number.NewUint256("379989503452489947895013"),
				totalAssets: number.NewUint256("382649667359278267721330"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("1300000000000000000"),
					Token:  STETH,
				},
				TokenOut: PUFETH,
			},
			expectedAmountOut: bignumber.NewBig("1290962456330641932"),
		},
		// tx: 0x6f70b2fe87399781e5f4a323bf310346dbe9eb0bd70e398cf91ff1bb1845f93c
		{
			name: "[wstETH] it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{PUFETH, STETH, WSTETH},
					},
				},
				totalSupply:      number.NewUint256("379677392580527064900714"),
				totalAssets:      number.NewUint256("382335371516233372457736"),
				totalPooledEther: number.NewUint256("9408886941382666867434878"),
				totalShares:      number.NewUint256("8085737150987915500442326"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("87589218056035970428"),
					Token:  WSTETH,
				},
				TokenOut: PUFETH,
			},
			expectedAmountOut: bignumber.NewBig("101213755613114753371"),
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
