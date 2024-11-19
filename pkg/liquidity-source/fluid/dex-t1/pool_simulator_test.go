package dexT1

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
						Address:     "0x0B1a513ee24972DAEf112bC777a5610d4325C9e7",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("5264013433389911488"), bignumber.NewBig("2569095126840549696")},
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
				DexLimits:      limitsWide,
				SyncTimestamp:  time.Now().Unix() - 10,
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
						Address:     "0x0B1a513ee24972DAEf112bC777a5610d4325C9e7",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("5264013433389911488"), bignumber.NewBig("2569095126840549696")},
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
				DexLimits:      limitsWide,
				SyncTimestamp:  time.Now().Unix() - 10,
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
						Address:     "0x0B1a513ee24972DAEf112bC777a5610d4325C9e7",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("5264013433389911488"), bignumber.NewBig("2569095126840549696")},
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
				DexLimits:      limitsWide,
				SyncTimestamp:  time.Now().Unix() - 10,
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
		{
			name: "it should return error for swap amount exceeding reserve",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:     "0x0B1a513ee24972DAEf112bC777a5610d4325C9e7",
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
				DexLimits:      limitsWide,
				SyncTimestamp:  time.Now().Unix() - 10,
				Token0Decimals: 18,
				Token1Decimals: 18,
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("30000000000000000000"), // Exceeds reserve
					Token:  "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
				},
				TokenOut: "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
			},
			expectedAmountOut: nil,
			expectedError:     ErrInsufficientReserve,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountOut(tc.param)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				t.Logf("Expected Amount Out: %s", tc.expectedAmountOut.String())
				t.Logf("Result Amount: %s", result.TokenAmountOut.Amount.String())
				t.Logf("Fee Amount: %s", result.Fee.Amount.String())

				if tc.expectedAmountOut != nil {
					assert.Zero(t, tc.expectedAmountOut.Cmp(result.TokenAmountOut.Amount))
				}
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
						Address:     "0x0B1a513ee24972DAEf112bC777a5610d4325C9e7",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("5264013433389911488"), bignumber.NewBig("2569095126840549696")},
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
				DexLimits:      limitsWide,
				SyncTimestamp:  time.Now().Unix() - 10,
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
						Address:     "0x0B1a513ee24972DAEf112bC777a5610d4325C9e7",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("5264013433389911488"), bignumber.NewBig("2569095126840549696")},
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
				DexLimits:      limitsWide,
				SyncTimestamp:  time.Now().Unix() - 10,
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
						Address:     "0x0B1a513ee24972DAEf112bC777a5610d4325C9e7",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("5264013433389911488"), bignumber.NewBig("2569095126840549696")},
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
				DexLimits:      limitsWide,
				SyncTimestamp:  time.Now().Unix() - 10,
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
		{
			name: "it should return error for swap amount exceeding reserve",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:     "0x0B1a513ee24972DAEf112bC777a5610d4325C9e7",
						Exchange:    "fluid-dex-t1",
						Type:        "fluid-dex-t1",
						Tokens:      []string{"0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"},
						Reserves:    []*big.Int{bignumber.NewBig("8792764353113222"), bignumber.NewBig("10371036463574636")},
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
				DexLimits:      limitsWide,
				SyncTimestamp:  time.Now().Unix() - 10,
				Token0Decimals: 18,
				Token1Decimals: 18,
			},
			param: poolpkg.CalcAmountInParams{
				TokenAmountOut: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("8792764353113223"), // exceeds reserve0 (= reserve0 + 1)
					Token:  "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
				},
				TokenIn: "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
			},
			expectedError: ErrInsufficientReserve,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountIn(tc.param)

			if tc.expectedError != nil {
				assert.ErrorIs(t, tc.expectedError, err)
			} else {
				t.Logf("Expected Amount In: %s", tc.expectedAmountIn.String())
				t.Logf("Result Amount: %s", result.TokenAmountIn.Amount.String())
				t.Logf("Fee Amount: %s", result.Fee.Amount.String())

				if tc.expectedAmountIn != nil {
					assert.Zero(t, tc.expectedAmountIn.Cmp(result.TokenAmountIn.Amount))
				}
			}
		})
	}
}

var limitExpandTight, _ = new(big.Int).SetString("711907234052361388866", 10)
var limitsTight = DexLimits{
	WithdrawableToken0: TokenLimit{
		Available:      big.NewInt(456740438880263),
		ExpandsTo:      big.NewInt(0).Set(limitExpandTight),
		ExpandDuration: big.NewInt(600),
	},
	WithdrawableToken1: TokenLimit{
		Available:      big.NewInt(825179383432029),
		ExpandsTo:      big.NewInt(0).Set(limitExpandTight),
		ExpandDuration: big.NewInt(600),
	},
	BorrowableToken0: TokenLimit{
		Available:      big.NewInt(941825058374170),
		ExpandsTo:      big.NewInt(0).Set(limitExpandTight),
		ExpandDuration: big.NewInt(600),
	},
	BorrowableToken1: TokenLimit{
		Available:      big.NewInt(941825058374170),
		ExpandsTo:      big.NewInt(0).Set(limitExpandTight),
		ExpandDuration: big.NewInt(600),
	},
}

var limitWide, _ = new(big.Int).SetString("34242332879776515083099999", 10)
var limitsWide = DexLimits{
	WithdrawableToken0: TokenLimit{
		Available:      big.NewInt(0).Set(limitWide),
		ExpandsTo:      big.NewInt(0).Set(limitWide),
		ExpandDuration: bignumber.ZeroBI,
	},
	WithdrawableToken1: TokenLimit{
		Available:      big.NewInt(0).Set(limitWide),
		ExpandsTo:      big.NewInt(0).Set(limitWide),
		ExpandDuration: big.NewInt(22),
	},
	BorrowableToken0: TokenLimit{
		Available:      big.NewInt(0).Set(limitWide),
		ExpandsTo:      big.NewInt(0).Set(limitWide),
		ExpandDuration: bignumber.ZeroBI,
	},
	BorrowableToken1: TokenLimit{
		Available:      big.NewInt(0).Set(limitWide),
		ExpandsTo:      big.NewInt(0).Set(limitWide),
		ExpandDuration: big.NewInt(308),
	},
}

func NewColReservesOne() CollateralReserves {
	return CollateralReserves{
		Token0RealReserves:      big.NewInt(20000000006000000),
		Token1RealReserves:      big.NewInt(20000000000500000),
		Token0ImaginaryReserves: big.NewInt(389736659726997981),
		Token1ImaginaryReserves: big.NewInt(389736659619871949),
	}
}

func NewColReservesEmpty() CollateralReserves {
	return CollateralReserves{
		Token0RealReserves:      big.NewInt(0),
		Token1RealReserves:      big.NewInt(0),
		Token0ImaginaryReserves: big.NewInt(0),
		Token1ImaginaryReserves: big.NewInt(0),
	}
}

func NewDebtReservesEmpty() DebtReserves {
	return DebtReserves{
		Token0RealReserves:      big.NewInt(0),
		Token1RealReserves:      big.NewInt(0),
		Token0ImaginaryReserves: big.NewInt(0),
		Token1ImaginaryReserves: big.NewInt(0),
	}
}

func NewDebtReservesOne() DebtReserves {
	return DebtReserves{
		Token0RealReserves:      big.NewInt(9486832995556050),
		Token1RealReserves:      big.NewInt(9486832993079885),
		Token0ImaginaryReserves: big.NewInt(184868330099560759),
		Token1ImaginaryReserves: big.NewInt(184868330048879109),
	}
}

func assertSwapInResult(t *testing.T, swap0To1 bool, amountIn *big.Int, colReserves CollateralReserves, debtReserves DebtReserves, expectedAmountIn string, expectedAmountOut string, outDecimals int64, limits DexLimits, syncTime int64) {
	outAmt, _ := swapInAdjusted(swap0To1, amountIn, colReserves, debtReserves, outDecimals, limits, syncTime)

	require.Equal(t, expectedAmountIn, amountIn.String())
	require.Equal(t, expectedAmountOut, outAmt.String())
}

func assertSwapOutResult(t *testing.T, swap0To1 bool, amountOut *big.Int, colReserves CollateralReserves, debtReserves DebtReserves, expectedAmountIn string, expectedAmountOut string, outDecimals int64, limits DexLimits, syncTime int64) {
	inAmt, _ := swapOutAdjusted(swap0To1, amountOut, colReserves, debtReserves, outDecimals, limits, syncTime)

	require.Equal(t, expectedAmountIn, inAmt.String())
	require.Equal(t, expectedAmountOut, amountOut.String())
}

func TestPoolSimulator_SwapIn(t *testing.T) {
	t.Run("TestPoolSimulator_SwapIn", func(t *testing.T) {
		assertSwapInResult(t, true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), "1000000000000000", "998262697204710", 18, limitsWide, time.Now().Unix()-10)
		assertSwapInResult(t, true, big.NewInt(1e15), NewColReservesEmpty(), NewDebtReservesOne(), "1000000000000000", "994619847016724", 18, limitsWide, time.Now().Unix()-10)
		assertSwapInResult(t, true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesEmpty(), "1000000000000000", "997440731289905", 18, limitsWide, time.Now().Unix()-10)
		assertSwapInResult(t, false, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), "1000000000000000", "998262697752553", 18, limitsWide, time.Now().Unix()-10)
		assertSwapInResult(t, false, big.NewInt(1e15), NewColReservesEmpty(), NewDebtReservesOne(), "1000000000000000", "994619847560607", 18, limitsWide, time.Now().Unix()-10)
		assertSwapInResult(t, false, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesEmpty(), "1000000000000000", "997440731837532", 18, limitsWide, time.Now().Unix()-10)
	})
}

func TestPoolSimulator_SwapInLimits(t *testing.T) {
	t.Run("TestPoolSimulator_SwapInLimits", func(t *testing.T) {
		// when limits hit
		outAmt, err := swapInAdjusted(true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), 18, limitsTight, time.Now().Unix()-10)
		require.Nil(t, outAmt)
		require.EqualError(t, err, ErrInsufficientBorrowable.Error())

		// when expanded
		outAmt, _ = swapInAdjusted(true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), 18, limitsTight, time.Now().Unix()-6000)
		require.Equal(t, "998262697204710", outAmt.String())

		// when price diff hit
		outAmt, err = swapInAdjusted(true, big.NewInt(3e16), NewColReservesOne(), NewDebtReservesOne(), 18, limitsWide, time.Now().Unix()-10)
		require.Nil(t, outAmt)
		require.EqualError(t, err, ErrInsufficientMaxPrice.Error())

		// when reserves limt is hit
		outAmt, err = swapInAdjusted(true, big.NewInt(5e16), NewColReservesOne(), NewDebtReservesOne(), 18, limitsWide, time.Now().Unix()-10)
		require.Nil(t, outAmt)
		require.EqualError(t, err, ErrInsufficientReserve.Error())
	})
}

func TestPoolSimulator_swapInAdjusted(t *testing.T) {
	t.Run("TestPoolSimulator_SwapInCompareEstimateIn", func(t *testing.T) {
		expectedAmountOut := "1180035404724000000"

		colReserves := CollateralReserves{
			Token0RealReserves:      big.NewInt(2169934539358),
			Token1RealReserves:      big.NewInt(19563846299171),
			Token0ImaginaryReserves: big.NewInt(62490032619260838),
			Token1ImaginaryReserves: big.NewInt(73741038977020279),
		}
		debtReserves := DebtReserves{
			Token0Debt:              big.NewInt(16590678644536),
			Token1Debt:              big.NewInt(2559733858855),
			Token0RealReserves:      big.NewInt(2169108220421),
			Token1RealReserves:      big.NewInt(19572550738602),
			Token0ImaginaryReserves: big.NewInt(62511862774117387),
			Token1ImaginaryReserves: big.NewInt(73766803277429176),
		}

		amountIn := big.NewInt(1e12)
		outAmt, _ := swapInAdjusted(true, amountIn, colReserves, debtReserves, 18, limitsWide, time.Now().Unix()-10)

		require.Equal(t, expectedAmountOut, big.NewInt(0).Mul(outAmt, big.NewInt(1e6)).String())
	})
}

func TestPoolSimulator_SwapOut(t *testing.T) {
	t.Run("TestPoolSimulator_SwapOut", func(t *testing.T) {
		assertSwapOutResult(t, true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), "1001743360284199", "1000000000000000", 18, limitsWide, time.Now().Unix()-10)
		assertSwapOutResult(t, true, big.NewInt(1e15), NewColReservesEmpty(), NewDebtReservesOne(), "1005438674786548", "1000000000000000", 18, limitsWide, time.Now().Unix()-10)
		assertSwapOutResult(t, true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesEmpty(), "1002572435818386", "1000000000000000", 18, limitsWide, time.Now().Unix()-10)
		assertSwapOutResult(t, false, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), "1001743359733488", "1000000000000000", 18, limitsWide, time.Now().Unix()-10)
		assertSwapOutResult(t, false, big.NewInt(1e15), NewColReservesEmpty(), NewDebtReservesOne(), "1005438674233767", "1000000000000000", 18, limitsWide, time.Now().Unix()-10)
		assertSwapOutResult(t, false, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesEmpty(), "1002572435266527", "1000000000000000", 18, limitsWide, time.Now().Unix()-10)
	})
}

func TestPoolSimulator_SwapOutLimits(t *testing.T) {
	t.Run("TestPoolSimulator_SwapInLimits", func(t *testing.T) {
		// when limits hit
		outAmt, err := swapOutAdjusted(true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), 18, limitsTight, time.Now().Unix()-10)
		require.Nil(t, outAmt)
		require.EqualError(t, err, ErrInsufficientBorrowable.Error())

		// when expanded
		outAmt, _ = swapOutAdjusted(true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), 18, limitsTight, time.Now().Unix()-6000)
		require.Equal(t, "1001743360284199", outAmt.String())

		// when price diff hit
		outAmt, err = swapOutAdjusted(true, big.NewInt(2e16), NewColReservesOne(), NewDebtReservesOne(), 18, limitsWide, time.Now().Unix()-10)
		require.Nil(t, outAmt)
		require.EqualError(t, err, ErrInsufficientMaxPrice.Error())

		// when reserves limt is hit
		outAmt, err = swapOutAdjusted(true, big.NewInt(3e16), NewColReservesOne(), NewDebtReservesOne(), 18, limitsWide, time.Now().Unix()-10)
		require.Nil(t, outAmt)
		require.EqualError(t, err, ErrInsufficientReserve.Error())
	})
}

func TestPoolSimulator_SwapInOut(t *testing.T) {
	t.Run("TestPoolSimulator_SwapInOut", func(t *testing.T) {
		assertSwapInResult(t, true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), "1000000000000000", "998262697204710", 18, limitsWide, time.Now().Unix()-10)

		assertSwapOutResult(t, true, big.NewInt(998262697204710), NewColReservesOne(), NewDebtReservesOne(), "999999999999998", "998262697204710", 18, limitsWide, time.Now().Unix()-10)

		assertSwapInResult(t, false, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), "1000000000000000", "998262697752553", 18, limitsWide, time.Now().Unix()-10)

		assertSwapOutResult(t, false, big.NewInt(998262697752553), NewColReservesOne(), NewDebtReservesOne(), "999999999999998", "998262697752553", 18, limitsWide, time.Now().Unix()-10)
	})
}

func TestPoolSimulator_SwapInOutDebtEmpty(t *testing.T) {
	t.Run("TestPoolSimulator_SwapInOutDebtEmpty", func(t *testing.T) {
		assertSwapInResult(t, true, big.NewInt(1e15), NewColReservesEmpty(), NewDebtReservesOne(), "1000000000000000", "994619847016724", 18, limitsWide, time.Now().Unix()-10)

		assertSwapOutResult(t, true, big.NewInt(994619847016724), NewColReservesEmpty(), NewDebtReservesOne(), "999999999999999", "994619847016724", 18, limitsWide, time.Now().Unix()-10)

		assertSwapInResult(t, false, big.NewInt(1e15), NewColReservesEmpty(), NewDebtReservesOne(), "1000000000000000", "994619847560607", 18, limitsWide, time.Now().Unix()-10)

		assertSwapOutResult(t, false, big.NewInt(994619847560607), NewColReservesEmpty(), NewDebtReservesOne(), "999999999999999", "994619847560607", 18, limitsWide, time.Now().Unix()-10)
	})

}

func TestPoolSimulator_SwapInOutColEmpty(t *testing.T) {
	t.Run("TestPoolSimulator_SwapInOutColEmpty", func(t *testing.T) {
		assertSwapInResult(t, true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesEmpty(), "1000000000000000", "997440731289905", 18, limitsWide, time.Now().Unix()-10)

		assertSwapOutResult(t, true, big.NewInt(997440731289905), NewColReservesOne(), NewDebtReservesEmpty(), "999999999999999", "997440731289905", 18, limitsWide, time.Now().Unix()-10)

		assertSwapInResult(t, false, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesEmpty(), "1000000000000000", "997440731837532", 18, limitsWide, time.Now().Unix()-10)

		assertSwapOutResult(t, false, big.NewInt(997440731837532), NewColReservesOne(), NewDebtReservesEmpty(), "999999999999999", "997440731837532", 18, limitsWide, time.Now().Unix()-10)
	})
}
