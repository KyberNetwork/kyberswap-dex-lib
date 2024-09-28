package dexT1

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
						Address:     "0x6d83f60eEac0e50A1250760151E81Db2a278e03a",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("18760613183894"), bignumber.NewBig("22123580158026")},
						BlockNumber: 20836530,
						SwapFee:     bignumber.NewBig("100"),
					},
				},
				CollateralReserves: CollateralReserves{
					Token0RealReserves:      bignumber.NewBig("2169934539358"),
					Token1RealReserves:      bignumber.NewBig("19563846299171"),
					Token0ImaginaryReserves: bignumber.NewBig("62490032619260838"),
					Token1ImaginaryReserves: bignumber.NewBig("73741038977020279"),
				},
				DebtReserves: DebtReserves{
					Token0Debt:              bignumber.NewBig("16590678644536"),
					Token1Debt:              bignumber.NewBig("2559733858855"),
					Token0RealReserves:      bignumber.NewBig("2169108220421"),
					Token1RealReserves:      bignumber.NewBig("19572550738602"),
					Token0ImaginaryReserves: bignumber.NewBig("62511862774117387"),
					Token1ImaginaryReserves: bignumber.NewBig("73766803277429176"),
				},
				Token0Decimals: 18,
				Token1Decimals: 18,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("1000000000000000000"), // 1 wstETH
					Token:  "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
				},
				TokenOut: "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
			},
			// expected amount without fee see math_test.go 1180035404724000000
			// for resolver estimateSwapIn result at very similar reserves values (hardcoded reserves above taken some blocks before).
			// resolver says estimateSwapIn result should be 1179917367073000000
			// we get here incl. fee 0.01% -> 1179917402128000000.
			expectedAmountOut: bignumber.NewBig("1179917402128000000"),
		},
		{
			name: "it should return correct amount for 0.5 wstETH",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:     "0x6d83f60eEac0e50A1250760151E81Db2a278e03a",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("18760613183894"), bignumber.NewBig("22123580158026")},
						BlockNumber: 20836530,
						SwapFee:     bignumber.NewBig("100"),
					},
				},
				CollateralReserves: CollateralReserves{
					Token0RealReserves:      bignumber.NewBig("2169934539358"),
					Token1RealReserves:      bignumber.NewBig("19563846299171"),
					Token0ImaginaryReserves: bignumber.NewBig("62490032619260838"),
					Token1ImaginaryReserves: bignumber.NewBig("73741038977020279"),
				},
				DebtReserves: DebtReserves{
					Token0Debt:              bignumber.NewBig("16590678644536"),
					Token1Debt:              bignumber.NewBig("2559733858855"),
					Token0RealReserves:      bignumber.NewBig("2169108220421"),
					Token1RealReserves:      bignumber.NewBig("19572550738602"),
					Token0ImaginaryReserves: bignumber.NewBig("62511862774117387"),
					Token1ImaginaryReserves: bignumber.NewBig("73766803277429176"),
				},
				Token0Decimals: 18,
				Token1Decimals: 18,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("500000000000000000"), // 0.5 wstETH
					Token:  "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
				},
				TokenOut: "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
			},
			// approximately expected:
			// 1179917402128000000 / 2 =
			//  589958701064000000
			expectedAmountOut: bignumber.NewBig("589961060629000000"),
		},
		{
			name: "it should return correct amount for 0.8 ETH",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:     "0x6d83f60eEac0e50A1250760151E81Db2a278e03a",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("18760613183894"), bignumber.NewBig("22123580158026")},
						BlockNumber: 20836530,
						SwapFee:     bignumber.NewBig("100"),
					},
				},
				CollateralReserves: CollateralReserves{
					Token0RealReserves:      bignumber.NewBig("2169934539358"),
					Token1RealReserves:      bignumber.NewBig("19563846299171"),
					Token0ImaginaryReserves: bignumber.NewBig("62490032619260838"),
					Token1ImaginaryReserves: bignumber.NewBig("73741038977020279"),
				},
				DebtReserves: DebtReserves{
					Token0Debt:              bignumber.NewBig("16590678644536"),
					Token1Debt:              bignumber.NewBig("2559733858855"),
					Token0RealReserves:      bignumber.NewBig("2169108220421"),
					Token1RealReserves:      bignumber.NewBig("19572550738602"),
					Token0ImaginaryReserves: bignumber.NewBig("62511862774117387"),
					Token1ImaginaryReserves: bignumber.NewBig("73766803277429176"),
				},
				Token0Decimals: 18,
				Token1Decimals: 18,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("800000000000000000"), // 0.8 ETH
					Token:  "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
				},
				TokenOut: "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
			},
			// for approximate check expected value check:
			// if 1e18 wstETH from test above results in 1179917402128000000 ETH
			// then for 1 ETH we should get 0.847516951776864996 WSTETH
			// and following for 0.8 ETH 0.678013561421491997.
			expectedAmountOut: bignumber.NewBig("677868867152000000"),
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
			t.Logf("Fee Amount: %s", result.Fee.Amount.String())

			if tc.expectedAmountOut != nil {
				assert.Zero(t, tc.expectedAmountOut.Cmp(result.TokenAmountOut.Amount))
			}
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	testCases := []struct {
		name             string
		poolSimulator    *PoolSimulator
		param            poolpkg.CalcAmountInParams
		expectedAmountIn *big.Int
		expectedError    error
	}{
		{
			name: "it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:     "0x6d83f60eEac0e50A1250760151E81Db2a278e03a",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("18760613183894"), bignumber.NewBig("22123580158026")},
						BlockNumber: 20836530,
						SwapFee:     bignumber.NewBig("100"),
					},
				},
				CollateralReserves: CollateralReserves{
					Token0RealReserves:      bignumber.NewBig("2169934539358"),
					Token1RealReserves:      bignumber.NewBig("19563846299171"),
					Token0ImaginaryReserves: bignumber.NewBig("62490032619260838"),
					Token1ImaginaryReserves: bignumber.NewBig("73741038977020279"),
				},
				DebtReserves: DebtReserves{
					Token0Debt:              bignumber.NewBig("16590678644536"),
					Token1Debt:              bignumber.NewBig("2559733858855"),
					Token0RealReserves:      bignumber.NewBig("2169108220421"),
					Token1RealReserves:      bignumber.NewBig("19572550738602"),
					Token0ImaginaryReserves: bignumber.NewBig("62511862774117387"),
					Token1ImaginaryReserves: bignumber.NewBig("73766803277429176"),
				},
				Token0Decimals: 18,
				Token1Decimals: 18,
			},
			param: poolpkg.CalcAmountInParams{
				TokenAmountOut: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("1179917402128000000"),
					Token:  "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
				},
				TokenIn: "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
			},
			// expected very close to 1000000000000000000
			expectedAmountIn: bignumber.NewBig("999999989997999800"),
		},
		{
			name: "it should return correct amount for 0.5 wstETH",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:     "0x6d83f60eEac0e50A1250760151E81Db2a278e03a",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("18760613183894"), bignumber.NewBig("22123580158026")},
						BlockNumber: 20836530,
						SwapFee:     bignumber.NewBig("100"),
					},
				},
				CollateralReserves: CollateralReserves{
					Token0RealReserves:      bignumber.NewBig("2169934539358"),
					Token1RealReserves:      bignumber.NewBig("19563846299171"),
					Token0ImaginaryReserves: bignumber.NewBig("62490032619260838"),
					Token1ImaginaryReserves: bignumber.NewBig("73741038977020279"),
				},
				DebtReserves: DebtReserves{
					Token0Debt:              bignumber.NewBig("16590678644536"),
					Token1Debt:              bignumber.NewBig("2559733858855"),
					Token0RealReserves:      bignumber.NewBig("2169108220421"),
					Token1RealReserves:      bignumber.NewBig("19572550738602"),
					Token0ImaginaryReserves: bignumber.NewBig("62511862774117387"),
					Token1ImaginaryReserves: bignumber.NewBig("73766803277429176"),
				},
				Token0Decimals: 18,
				Token1Decimals: 18,
			},
			param: poolpkg.CalcAmountInParams{
				TokenAmountOut: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("589961060629000000"),
					Token:  "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
				},
				TokenIn: "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
			},
			// expected very close to 500000000000000000
			expectedAmountIn: bignumber.NewBig("499999994997999800"),
		},
		{
			name: "it should return correct amount for 0.8 ETH",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:     "0x6d83f60eEac0e50A1250760151E81Db2a278e03a",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("18760613183894"), bignumber.NewBig("22123580158026")},
						BlockNumber: 20836530,
						SwapFee:     bignumber.NewBig("100"),
					},
				},
				CollateralReserves: CollateralReserves{
					Token0RealReserves:      bignumber.NewBig("2169934539358"),
					Token1RealReserves:      bignumber.NewBig("19563846299171"),
					Token0ImaginaryReserves: bignumber.NewBig("62490032619260838"),
					Token1ImaginaryReserves: bignumber.NewBig("73741038977020279"),
				},
				DebtReserves: DebtReserves{
					Token0Debt:              bignumber.NewBig("16590678644536"),
					Token1Debt:              bignumber.NewBig("2559733858855"),
					Token0RealReserves:      bignumber.NewBig("2169108220421"),
					Token1RealReserves:      bignumber.NewBig("19572550738602"),
					Token0ImaginaryReserves: bignumber.NewBig("62511862774117387"),
					Token1ImaginaryReserves: bignumber.NewBig("73766803277429176"),
				},
				Token0Decimals: 18,
				Token1Decimals: 18,
			},
			param: poolpkg.CalcAmountInParams{
				TokenAmountOut: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("677868867152000000"),
					Token:  "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
				},
				TokenIn: "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
			},
			// expected very close to 800000000000000000
			expectedAmountIn: bignumber.NewBig("799999991997999800"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountIn(tc.param)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			}

			t.Logf("Expected Amount In: %s", tc.expectedAmountIn.String())
			t.Logf("Result Amount: %s", result.TokenAmountIn.Amount.String())
			t.Logf("Fee Amount: %s", result.Fee.Amount.String())

			if tc.expectedAmountIn != nil {
				assert.Zero(t, tc.expectedAmountIn.Cmp(result.TokenAmountIn.Amount))
			}
		})
	}
}
