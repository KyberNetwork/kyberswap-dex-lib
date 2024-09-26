package vaultT1

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
			name: "it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:     "0x40D9b8417E6E1DcD358f04E3328bCEd061018A82",
						Exchange:    "fluid-vault-t1",
						Type:        "fluid-vault-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee"},
						Reserves:    []*big.Int{bignumber.NewBig("86232802856618560"), bignumber.NewBig("97976286699627227")},
						BlockNumber: 20812089,
						SwapFee:     big.NewInt(0), // no swap fee on liquidations
					},
				},
				Ratio: bignumber.NewBig("1136183487651849280183370224"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("1000000000000000000"), // 1 wstETH
					Token:  "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
				},
				TokenOut: "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee",
			},
			expectedAmountOut: bignumber.NewBig("1136183487651849280"),
		},
		{
			name: "it should return correct amount for 0.5 wstETH",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:     "0x40D9b8417E6E1DcD358f04E3328bCEd061018A82",
						Exchange:    "fluid-vault-t1",
						Type:        "fluid-vault-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee"},
						Reserves:    []*big.Int{bignumber.NewBig("86232802856618560"), bignumber.NewBig("97976286699627227")},
						BlockNumber: 20812089,
						SwapFee:     big.NewInt(0), // no swap fee on liquidations
					},
				},
				Ratio: bignumber.NewBig("1136183487651849280183370224"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("500000000000000000"), // 0.5 wstETH
					Token:  "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
				},
				TokenOut: "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee",
			},
			expectedAmountOut: bignumber.NewBig("568091743825924640"), // 1136183487651849280 / 2
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountOut(tc.param)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			}

			t.Logf("Expected Amount Out: %s", tc.expectedAmountOut.String())
			t.Logf("Result Amount: %s", result.TokenAmountOut.Amount.String())

			if tc.expectedAmountOut != nil {
				assert.Zero(t, tc.expectedAmountOut.Cmp(result.TokenAmountOut.Amount))
			}
		})
	}
}
