package dexT1

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func calculateReservesOutsideRange(geometricMeanPrice, priceAtRange, reserveX, reserveY *big.Int) (*big.Int, *big.Int) {
	// Calculate the three parts of the quadratic equation solution
	part1 := new(big.Int).Sub(priceAtRange, geometricMeanPrice)

	part2 := new(big.Int).Div(new(big.Int).Add(new(big.Int).Mul(geometricMeanPrice, reserveX), new(big.Int).Mul(reserveY, bI1e27)), new(big.Int).Mul(big.NewInt(2), part1))

	part3 := new(big.Int).Mul(reserveX, reserveY)

	var bI1e50, _ = new(big.Int).SetString("100000000000000000000000000000000000000000000000000", 10)
	// Handle potential overflow like in Solidity
	if part3.Cmp(bI1e50) < 0 {
		part3 = new(big.Int).Div(new(big.Int).Mul(part3, bI1e27), part1)
	} else {
		part3 = new(big.Int).Mul(new(big.Int).Div(part3, part1), bI1e27)
	}

	// Calculate xa (reserveXOutside)
	reserveXOutside := new(big.Int).Add(part2, new(big.Int).Sqrt(new(big.Int).Add(part3, new(big.Int).Mul(part2, part2))))

	// Calculate yb (reserveYOutside)
	reserveYOutside := new(big.Int).Div(new(big.Int).Mul(reserveXOutside, geometricMeanPrice), bI1e27)

	return reserveXOutside, reserveYOutside
}

func getApproxCenterPriceIn(amountToSwap *big.Int, swap0To1 bool, colReserves CollateralReserves, debtReserves DebtReserves) (*big.Int, error) {
	colPoolEnabled := colReserves.Token0RealReserves.Sign() > 0 &&
		colReserves.Token1RealReserves.Sign() > 0 &&
		colReserves.Token0ImaginaryReserves.Sign() > 0 &&
		colReserves.Token1ImaginaryReserves.Sign() > 0

	debtPoolEnabled := debtReserves.Token0RealReserves.Sign() > 0 &&
		debtReserves.Token1RealReserves.Sign() > 0 &&
		debtReserves.Token0ImaginaryReserves.Sign() > 0 &&
		debtReserves.Token1ImaginaryReserves.Sign() > 0

	var colIReserveIn, colIReserveOut, debtIReserveIn, debtIReserveOut *big.Int

	if swap0To1 {
		colIReserveIn = colReserves.Token0ImaginaryReserves
		colIReserveOut = colReserves.Token1ImaginaryReserves
		debtIReserveIn = debtReserves.Token0ImaginaryReserves
		debtIReserveOut = debtReserves.Token1ImaginaryReserves
	} else {
		colIReserveIn = colReserves.Token1ImaginaryReserves
		colIReserveOut = colReserves.Token0ImaginaryReserves
		debtIReserveIn = debtReserves.Token1ImaginaryReserves
		debtIReserveOut = debtReserves.Token0ImaginaryReserves
	}

	var a *big.Int
	if colPoolEnabled && debtPoolEnabled {
		a = swapRoutingIn(amountToSwap, colIReserveOut, colIReserveIn, debtIReserveOut, debtIReserveIn)
	} else if debtPoolEnabled {
		a = big.NewInt(-1) // Route from debt pool
	} else if colPoolEnabled {
		a = new(big.Int).Add(amountToSwap, big.NewInt(1)) // Route from collateral pool
	} else {
		return nil, errors.New("No pools are enabled")
	}

	amountInCollateral := new(big.Int)
	amountInDebt := new(big.Int)

	if a.Sign() <= 0 {
		amountInDebt = amountToSwap
	} else if a.Cmp(amountToSwap) >= 0 {
		amountInCollateral = amountToSwap
	} else {
		amountInCollateral = a
		amountInDebt = new(big.Int).Sub(amountToSwap, a)
	}

	var price *big.Int
	if amountInCollateral.Cmp(amountInDebt) > 0 {
		if swap0To1 {
			price = new(big.Int).Div(new(big.Int).Mul(colIReserveOut, bI1e27), colIReserveIn)
		} else {
			price = new(big.Int).Div(new(big.Int).Mul(colIReserveIn, bI1e27), colIReserveOut)
		}
	} else {
		if swap0To1 {
			price = new(big.Int).Div(new(big.Int).Mul(debtIReserveOut, bI1e27), debtIReserveIn)
		} else {
			price = new(big.Int).Div(new(big.Int).Mul(debtIReserveIn, bI1e27), debtIReserveOut)
		}
	}

	return price, nil
}

func getApproxCenterPriceOut(amountOut *big.Int, swap0To1 bool, colReserves CollateralReserves, debtReserves DebtReserves) (*big.Int, error) {
	colPoolEnabled := colReserves.Token0RealReserves.Sign() > 0 &&
		colReserves.Token1RealReserves.Sign() > 0 &&
		colReserves.Token0ImaginaryReserves.Sign() > 0 &&
		colReserves.Token1ImaginaryReserves.Sign() > 0

	debtPoolEnabled := debtReserves.Token0RealReserves.Sign() > 0 &&
		debtReserves.Token1RealReserves.Sign() > 0 &&
		debtReserves.Token0ImaginaryReserves.Sign() > 0 &&
		debtReserves.Token1ImaginaryReserves.Sign() > 0

	var colIReserveIn, colIReserveOut, debtIReserveIn, debtIReserveOut *big.Int

	if swap0To1 {
		colIReserveIn = colReserves.Token0ImaginaryReserves
		colIReserveOut = colReserves.Token1ImaginaryReserves
		debtIReserveIn = debtReserves.Token0ImaginaryReserves
		debtIReserveOut = debtReserves.Token1ImaginaryReserves
	} else {
		colIReserveIn = colReserves.Token1ImaginaryReserves
		colIReserveOut = colReserves.Token0ImaginaryReserves
		debtIReserveIn = debtReserves.Token1ImaginaryReserves
		debtIReserveOut = debtReserves.Token0ImaginaryReserves
	}

	var a *big.Int
	if colPoolEnabled && debtPoolEnabled {
		a = swapRoutingOut(amountOut, colIReserveIn, colIReserveOut, debtIReserveIn, debtIReserveOut)
	} else if debtPoolEnabled {
		a = big.NewInt(-1) // Route from debt pool
	} else if colPoolEnabled {
		a = new(big.Int).Add(amountOut, big.NewInt(1)) // Route from collateral pool
	} else {
		return nil, errors.New("No pools are enabled")
	}

	amountInCollateral := new(big.Int)
	amountInDebt := new(big.Int)

	if a.Sign() <= 0 {
		amountInDebt = getAmountIn(amountOut, debtIReserveIn, debtIReserveOut)
	} else if a.Cmp(amountOut) >= 0 {
		amountInCollateral = getAmountIn(amountOut, colIReserveIn, colIReserveOut)
	} else {
		amountInCollateral = getAmountIn(a, colIReserveIn, colIReserveOut)
		amountInDebt = getAmountIn(new(big.Int).Sub(amountOut, a), debtIReserveIn, debtIReserveOut)
	}

	var price *big.Int
	if amountInCollateral.Cmp(amountInDebt) > 0 {
		if swap0To1 {
			price = new(big.Int).Div(new(big.Int).Mul(colIReserveOut, bI1e27), colIReserveIn)
		} else {
			price = new(big.Int).Div(new(big.Int).Mul(colIReserveIn, bI1e27), colIReserveOut)
		}
	} else {
		if swap0To1 {
			price = new(big.Int).Div(new(big.Int).Mul(debtIReserveOut, bI1e27), debtIReserveIn)
		} else {
			price = new(big.Int).Div(new(big.Int).Mul(debtIReserveIn, bI1e27), debtIReserveOut)
		}
	}

	return price, nil
}

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
				DexLimits:      limitsWide(),
				CenterPrice:    big.NewInt(1),
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
				DexLimits:      limitsWide(),
				CenterPrice:    big.NewInt(1),
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
				DexLimits:      limitsWide(),
				CenterPrice:    new(big.Int).Mul(big.NewInt(1.2e18), big.NewInt(1e9)),
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
				DexLimits:      limitsWide(),
				CenterPrice:    big.NewInt(1),
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
				if result == nil {
					t.Log("Result is nil")
				} else {
					t.Logf("Result Amount: %s", result.TokenAmountOut.Amount.String())
					t.Logf("Fee Amount: %s", result.Fee.Amount.String())

				}

				if tc.expectedAmountOut != nil {
					assert.Zero(t, tc.expectedAmountOut.Cmp(result.TokenAmountOut.Amount))
				}

			}
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
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
				DexLimits:      limitsWide(),
				CenterPrice:    big.NewInt(1),
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
				DexLimits:      limitsWide(),
				CenterPrice:    big.NewInt(1),
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
				DexLimits:      limitsWide(),
				CenterPrice:    new(big.Int).Mul(big.NewInt(1.2e18), big.NewInt(1e9)),
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
				DexLimits:      limitsWide(),
				CenterPrice:    big.NewInt(1),
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

func limitsTight() DexLimits {
	return DexLimits{
		WithdrawableToken0: TokenLimit{
			Available:      big.NewInt(456740438880263),
			ExpandsTo:      new(big.Int).Set(limitExpandTight),
			ExpandDuration: big.NewInt(600),
		},
		WithdrawableToken1: TokenLimit{
			Available:      big.NewInt(825179383432029),
			ExpandsTo:      new(big.Int).Set(limitExpandTight),
			ExpandDuration: big.NewInt(600),
		},
		BorrowableToken0: TokenLimit{
			Available:      big.NewInt(941825058374170),
			ExpandsTo:      new(big.Int).Set(limitExpandTight),
			ExpandDuration: big.NewInt(600),
		},
		BorrowableToken1: TokenLimit{
			Available:      big.NewInt(941825058374170),
			ExpandsTo:      new(big.Int).Set(limitExpandTight),
			ExpandDuration: big.NewInt(600),
		},
	}
}

var limitWide, _ = new(big.Int).SetString("34242332879776515083099999", 10)

func limitsWide() DexLimits {
	return DexLimits{
		WithdrawableToken0: TokenLimit{
			Available:      new(big.Int).Set(limitWide),
			ExpandsTo:      new(big.Int).Set(limitWide),
			ExpandDuration: bignumber.ZeroBI,
		},
		WithdrawableToken1: TokenLimit{
			Available:      new(big.Int).Set(limitWide),
			ExpandsTo:      new(big.Int).Set(limitWide),
			ExpandDuration: big.NewInt(22),
		},
		BorrowableToken0: TokenLimit{
			Available:      new(big.Int).Set(limitWide),
			ExpandsTo:      new(big.Int).Set(limitWide),
			ExpandDuration: bignumber.ZeroBI,
		},
		BorrowableToken1: TokenLimit{
			Available:      new(big.Int).Set(limitWide),
			ExpandsTo:      new(big.Int).Set(limitWide),
			ExpandDuration: big.NewInt(308),
		},
	}
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
		Token0RealReserves:      new(big.Int),
		Token1RealReserves:      new(big.Int),
		Token0ImaginaryReserves: new(big.Int),
		Token1ImaginaryReserves: new(big.Int),
	}
}

func NewDebtReservesEmpty() DebtReserves {
	return DebtReserves{
		Token0RealReserves:      new(big.Int),
		Token1RealReserves:      new(big.Int),
		Token0ImaginaryReserves: new(big.Int),
		Token1ImaginaryReserves: new(big.Int),
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

func assertSwapInResult(t *testing.T, swap0To1 bool, amountIn *big.Int, colReserves CollateralReserves, debtReserves DebtReserves, expectedAmountIn string, expectedAmountOut string, inDecimals, outDecimals int64, limits DexLimits, syncTime int64) {
	price, _ := getApproxCenterPriceIn(amountIn, swap0To1, colReserves, debtReserves)
	outAmt, _ := swapInAdjusted(swap0To1, amountIn, colReserves, debtReserves, inDecimals, outDecimals, limits, price, syncTime)

	require.Equal(t, expectedAmountIn, amountIn.String())
	require.Equal(t, expectedAmountOut, outAmt.String())
}

func assertSwapOutResult(t *testing.T, swap0To1 bool, amountOut *big.Int, colReserves CollateralReserves, debtReserves DebtReserves, expectedAmountIn string, expectedAmountOut string, inDecimals, outDecimals int64, limits DexLimits, syncTime int64) {
	price, _ := getApproxCenterPriceOut(amountOut, swap0To1, colReserves, debtReserves)
	inAmt, _ := swapOutAdjusted(swap0To1, amountOut, colReserves, debtReserves, inDecimals, outDecimals, limits, price, syncTime)

	require.Equal(t, expectedAmountIn, inAmt.String())
	require.Equal(t, expectedAmountOut, amountOut.String())
}

func TestPoolSimulator_SwapIn(t *testing.T) {
	t.Parallel()
	t.Run("TestPoolSimulator_SwapIn", func(t *testing.T) {
		assertSwapInResult(t, true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), "1000000000000000", "998262697204710000000", 12, 18, limitsWide(), time.Now().Unix()-10)
		assertSwapInResult(t, true, big.NewInt(1e15), NewColReservesEmpty(), NewDebtReservesOne(), "1000000000000000", "994619847016724000000", 12, 18, limitsWide(), time.Now().Unix()-10)
		assertSwapInResult(t, true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesEmpty(), "1000000000000000", "997440731289905000000", 12, 18, limitsWide(), time.Now().Unix()-10)
		assertSwapInResult(t, false, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), "1000000000000000", "998262697752553000000", 12, 18, limitsWide(), time.Now().Unix()-10)
		assertSwapInResult(t, false, big.NewInt(1e15), NewColReservesEmpty(), NewDebtReservesOne(), "1000000000000000", "994619847560607000000", 12, 18, limitsWide(), time.Now().Unix()-10)
		assertSwapInResult(t, false, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesEmpty(), "1000000000000000", "997440731837532000000", 12, 18, limitsWide(), time.Now().Unix()-10)
	})
}

func TestPoolSimulator_SwapInLimits(t *testing.T) {
	t.Parallel()
	t.Run("TestPoolSimulator_SwapInLimits", func(t *testing.T) {
		// when limits hit
		price, _ := getApproxCenterPriceIn(big.NewInt(1e15), true, NewColReservesOne(), NewDebtReservesOne())
		outAmt, err := swapInAdjusted(true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), 18, 18, limitsTight(), price, time.Now().Unix()-10)
		require.Nil(t, outAmt)
		require.EqualError(t, err, ErrInsufficientBorrowable.Error())

		// when expanded
		price, _ = getApproxCenterPriceIn(big.NewInt(1e15), true, NewColReservesOne(), NewDebtReservesOne())
		outAmt, _ = swapInAdjusted(true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), 18, 18, limitsTight(), price, time.Now().Unix()-6000)
		require.Equal(t, "998262697204710", outAmt.String())

		// when price diff hit
		price, _ = getApproxCenterPriceIn(big.NewInt(3e16), true, NewColReservesOne(), NewDebtReservesOne())
		outAmt, err = swapInAdjusted(true, big.NewInt(3e16), NewColReservesOne(), NewDebtReservesOne(), 18, 18, limitsWide(), price, time.Now().Unix()-10)
		require.Nil(t, outAmt)
		require.EqualError(t, err, ErrInsufficientMaxPrice.Error())

		// when reserves limt is hit
		price, _ = getApproxCenterPriceIn(big.NewInt(5e16), true, NewColReservesOne(), NewDebtReservesOne())
		outAmt, err = swapInAdjusted(true, big.NewInt(5e16), NewColReservesOne(), NewDebtReservesOne(), 18, 18, limitsWide(), price, time.Now().Unix()-10)
		require.Nil(t, outAmt)
		require.EqualError(t, err, ErrInsufficientReserve.Error())
	})
}

func TestPoolSimulator_swapInAdjusted(t *testing.T) {
	t.Parallel()
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
		price, _ := getApproxCenterPriceIn(amountIn, true, colReserves, debtReserves)
		outAmt, _ := swapInAdjusted(true, amountIn, colReserves, debtReserves, 18, 18, limitsWide(), price, time.Now().Unix()-10)

		require.Equal(t, expectedAmountOut, new(big.Int).Mul(outAmt, big.NewInt(1e6)).String())
	})
}

func TestPoolSimulator_SwapOut(t *testing.T) {
	t.Parallel()
	t.Run("TestPoolSimulator_SwapOut", func(t *testing.T) {
		assertSwapOutResult(t, true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), "1001743360284199", "1000000000000000", 18, 18, limitsWide(), time.Now().Unix()-10)
		assertSwapOutResult(t, true, big.NewInt(1e15), NewColReservesEmpty(), NewDebtReservesOne(), "1005438674786548", "1000000000000000", 18, 18, limitsWide(), time.Now().Unix()-10)
		assertSwapOutResult(t, true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesEmpty(), "1002572435818386", "1000000000000000", 18, 18, limitsWide(), time.Now().Unix()-10)
		assertSwapOutResult(t, false, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), "1001743359733488", "1000000000000000", 18, 18, limitsWide(), time.Now().Unix()-10)
		assertSwapOutResult(t, false, big.NewInt(1e15), NewColReservesEmpty(), NewDebtReservesOne(), "1005438674233767", "1000000000000000", 18, 18, limitsWide(), time.Now().Unix()-10)
		assertSwapOutResult(t, false, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesEmpty(), "1002572435266527", "1000000000000000", 18, 18, limitsWide(), time.Now().Unix()-10)
	})
}

func TestPoolSimulator_SwapOutLimits(t *testing.T) {
	t.Parallel()
	t.Run("TestPoolSimulator_SwapInLimits", func(t *testing.T) {
		// when limits hit
		price, _ := getApproxCenterPriceOut(big.NewInt(1e15), true, NewColReservesOne(), NewDebtReservesOne())
		outAmt, err := swapOutAdjusted(true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), 18, 18, limitsTight(), price, time.Now().Unix()-10)
		require.Nil(t, outAmt)
		require.EqualError(t, err, ErrInsufficientBorrowable.Error())

		// when expanded
		price, _ = getApproxCenterPriceOut(big.NewInt(1e15), true, NewColReservesOne(), NewDebtReservesOne())
		outAmt, _ = swapOutAdjusted(true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), 18, 18, limitsTight(), price, time.Now().Unix()-6000)
		require.Equal(t, "1001743360284199", outAmt.String())

		// when price diff hit
		price, _ = getApproxCenterPriceOut(big.NewInt(2e16), true, NewColReservesOne(), NewDebtReservesOne())
		outAmt, err = swapOutAdjusted(true, big.NewInt(2e16), NewColReservesOne(), NewDebtReservesOne(), 18, 18, limitsWide(), price, time.Now().Unix()-10)
		require.Nil(t, outAmt)
		require.EqualError(t, err, ErrInsufficientMaxPrice.Error())

		// when reserves limt is hit
		price, _ = getApproxCenterPriceOut(big.NewInt(3e16), true, NewColReservesOne(), NewDebtReservesOne())
		outAmt, err = swapOutAdjusted(true, big.NewInt(3e16), NewColReservesOne(), NewDebtReservesOne(), 18, 18, limitsWide(), price, time.Now().Unix()-10)
		require.Nil(t, outAmt)
		require.EqualError(t, err, ErrInsufficientReserve.Error())
	})
}

func TestPoolSimulator_SwapInOut(t *testing.T) {
	t.Parallel()
	t.Run("TestPoolSimulator_SwapInOut", func(t *testing.T) {
		assertSwapInResult(t, true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), "1000000000000000", "998262697204710", 18, 18, limitsWide(), time.Now().Unix()-10)

		assertSwapOutResult(t, true, big.NewInt(998262697204710), NewColReservesOne(), NewDebtReservesOne(), "999999999999998", "998262697204710", 18, 18, limitsWide(), time.Now().Unix()-10)

		assertSwapInResult(t, false, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesOne(), "1000000000000000", "998262697752553", 18, 18, limitsWide(), time.Now().Unix()-10)

		assertSwapOutResult(t, false, big.NewInt(998262697752553), NewColReservesOne(), NewDebtReservesOne(), "999999999999998", "998262697752553", 18, 18, limitsWide(), time.Now().Unix()-10)
	})
}

func TestPoolSimulator_SwapInOutDebtEmpty(t *testing.T) {
	t.Parallel()
	t.Run("TestPoolSimulator_SwapInOutDebtEmpty", func(t *testing.T) {
		assertSwapInResult(t, true, big.NewInt(1e15), NewColReservesEmpty(), NewDebtReservesOne(), "1000000000000000", "994619847016724", 18, 18, limitsWide(), time.Now().Unix()-10)

		assertSwapOutResult(t, true, big.NewInt(994619847016724), NewColReservesEmpty(), NewDebtReservesOne(), "999999999999999", "994619847016724", 18, 18, limitsWide(), time.Now().Unix()-10)

		assertSwapInResult(t, false, big.NewInt(1e15), NewColReservesEmpty(), NewDebtReservesOne(), "1000000000000000", "994619847560607", 18, 18, limitsWide(), time.Now().Unix()-10)

		assertSwapOutResult(t, false, big.NewInt(994619847560607), NewColReservesEmpty(), NewDebtReservesOne(), "999999999999999", "994619847560607", 18, 18, limitsWide(), time.Now().Unix()-10)
	})

}

func TestPoolSimulator_SwapInOutColEmpty(t *testing.T) {
	t.Parallel()
	t.Run("TestPoolSimulator_SwapInOutColEmpty", func(t *testing.T) {
		assertSwapInResult(t, true, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesEmpty(), "1000000000000000", "997440731289905", 18, 18, limitsWide(), time.Now().Unix()-10)

		assertSwapOutResult(t, true, big.NewInt(997440731289905), NewColReservesOne(), NewDebtReservesEmpty(), "999999999999999", "997440731289905", 18, 18, limitsWide(), time.Now().Unix()-10)

		assertSwapInResult(t, false, big.NewInt(1e15), NewColReservesOne(), NewDebtReservesEmpty(), "1000000000000000", "997440731837532", 18, 18, limitsWide(), time.Now().Unix()-10)

		assertSwapOutResult(t, false, big.NewInt(997440731837532), NewColReservesOne(), NewDebtReservesEmpty(), "999999999999999", "997440731837532", 18, 18, limitsWide(), time.Now().Unix()-10)
	})
}

func NewVerifyRatioColReserves() CollateralReserves {
	return CollateralReserves{
		Token0RealReserves:      big.NewInt(2_000_000 * 1e6 * 1e6), // e.g. 2M USDC
		Token1RealReserves:      big.NewInt(15_000 * 1e6 * 1e6),    // e.g. 15 USDT
		Token0ImaginaryReserves: new(big.Int),
		Token1ImaginaryReserves: new(big.Int),
	}
}
func NewVerifyRatioDebtReserves() DebtReserves {
	return DebtReserves{
		Token0RealReserves:      big.NewInt(2_000_000 * 1e6 * 1e6), // e.g. 2M USDC
		Token1RealReserves:      big.NewInt(15_000 * 1e6 * 1e6),    // e.g. 15 USDT
		Token0ImaginaryReserves: new(big.Int),
		Token1ImaginaryReserves: new(big.Int),
	}
}
func TestSwapInVerifyReservesInRange(t *testing.T) {
	t.Parallel()
	t.Run("TestSwapInVerifyReservesInRange", func(t *testing.T) {
		decimals := int64(6)

		colReserves := NewVerifyRatioColReserves()
		debtReserves := NewVerifyRatioDebtReserves()

		// Ignore the boolean return value
		price, _ := new(big.Int).SetString("1000001000000000000000000000", 10)

		reserveXOutside, reserveYOutside := calculateReservesOutsideRange(
			bI1e27,
			price,
			colReserves.Token0RealReserves,
			colReserves.Token1RealReserves,
		)
		colReserves.Token0ImaginaryReserves = new(big.Int).Add(reserveXOutside, colReserves.Token0RealReserves)
		colReserves.Token1ImaginaryReserves = new(big.Int).Add(reserveYOutside, colReserves.Token1RealReserves)
		reserveXOutside, reserveYOutside = calculateReservesOutsideRange(
			bI1e27,
			price,
			debtReserves.Token0RealReserves,
			debtReserves.Token1RealReserves,
		)
		debtReserves.Token0ImaginaryReserves = new(big.Int).Add(reserveXOutside, debtReserves.Token0RealReserves)
		debtReserves.Token1ImaginaryReserves = new(big.Int).Add(reserveYOutside, debtReserves.Token1RealReserves)

		// expected required ratio:
		// token1Reserves must be > (token0Reserves * price) / (1e27 * MIN_SWAP_LIQUIDITY)
		// so 2M / 2e4, which is 100

		// Test for swap amount 14_905, revert should hit
		swapAmount := big.NewInt(14_905 * 1e6 * 1e6)
		price, _ = getApproxCenterPriceIn(swapAmount, true, colReserves, NewDebtReservesEmpty())
		result, _ := swapInAdjusted(true, swapAmount, colReserves, NewDebtReservesEmpty(), decimals, decimals, limitsWide(), price, time.Now().Unix()-10)
		require.Nil(t, result, "FAIL: reserves ratio verification revert NOT hit for col reserves when swap amount %d", 14_905)
		price, _ = getApproxCenterPriceIn(swapAmount, true, NewColReservesEmpty(), debtReserves)
		result, _ = swapInAdjusted(true, swapAmount, NewColReservesEmpty(), debtReserves, decimals, decimals, limitsWide(), price, time.Now().Unix()-10)
		require.Nil(t, result, "FAIL: reserves ratio verification revert NOT hit for debt reserves when swap amount %d", 14_905)

		// refresh reserves
		colReserves = NewVerifyRatioColReserves()
		debtReserves = NewVerifyRatioDebtReserves()
		reserveXOutside, reserveYOutside = calculateReservesOutsideRange(
			bI1e27,
			price,
			colReserves.Token0RealReserves,
			colReserves.Token1RealReserves,
		)
		colReserves.Token0ImaginaryReserves = new(big.Int).Add(reserveXOutside, colReserves.Token0RealReserves)
		colReserves.Token1ImaginaryReserves = new(big.Int).Add(reserveYOutside, colReserves.Token1RealReserves)
		reserveXOutside, reserveYOutside = calculateReservesOutsideRange(
			bI1e27,
			price,
			debtReserves.Token0RealReserves,
			debtReserves.Token1RealReserves,
		)
		debtReserves.Token0ImaginaryReserves = new(big.Int).Add(reserveXOutside, debtReserves.Token0RealReserves)
		debtReserves.Token1ImaginaryReserves = new(big.Int).Add(reserveYOutside, debtReserves.Token1RealReserves)

		// Test for swap amount 14_895, revert should NOT hit
		swapAmount = big.NewInt(14_895 * 1e6 * 1e6)
		err := error(nil)
		price, _ = getApproxCenterPriceIn(swapAmount, true, colReserves, NewDebtReservesEmpty())
		result, err = swapInAdjusted(true, swapAmount, colReserves, NewDebtReservesEmpty(), decimals, decimals, limitsWide(), price, time.Now().Unix()-10)
		require.NoError(t, err, "Error during swapInAdjusted for col reserves")
		require.NotNil(t, result, "FAIL: reserves ratio verification revert hit for col reserves when swap amount %d", 14_895)
		price, _ = getApproxCenterPriceIn(swapAmount, true, NewColReservesEmpty(), debtReserves)
		result, _ = swapInAdjusted(true, swapAmount, NewColReservesEmpty(), debtReserves, decimals, decimals, limitsWide(), price, time.Now().Unix()-10)
		require.NotNil(t, result, "FAIL: reserves ratio verification revert hit for debt reserves when swap amount %d", 14_895)
	})
}

func NewVerifyRatioColReservesSwapOut() CollateralReserves {
	return CollateralReserves{
		Token0RealReserves:      big.NewInt(15_000 * 1e6 * 1e6),    // e.g. 15 USDT
		Token1RealReserves:      big.NewInt(2_000_000 * 1e6 * 1e6), // e.g. 2M USDC
		Token0ImaginaryReserves: new(big.Int),
		Token1ImaginaryReserves: new(big.Int),
	}
}
func NewVerifyRatioDebtReservesSwapOut() DebtReserves {
	return DebtReserves{
		Token0RealReserves:      big.NewInt(15_000 * 1e6 * 1e6),    // e.g. 15 USDT
		Token1RealReserves:      big.NewInt(2_000_000 * 1e6 * 1e6), // e.g. 2M USDC
		Token0ImaginaryReserves: new(big.Int),
		Token1ImaginaryReserves: new(big.Int),
	}
}

func TestSwapOutVerifyReservesInRange(t *testing.T) {
	t.Parallel()
	t.Run("TestSwapOutVerifyReservesInRange", func(t *testing.T) {
		decimals := int64(6)

		colReserves := NewVerifyRatioColReservesSwapOut()
		debtReserves := NewVerifyRatioDebtReservesSwapOut()

		// Ignore the boolean return value
		price, _ := new(big.Int).SetString("1000001000000000000000000000", 10)

		reserveXOutside, reserveYOutside := calculateReservesOutsideRange(
			bI1e27,
			price,
			colReserves.Token0RealReserves,
			colReserves.Token1RealReserves,
		)
		colReserves.Token0ImaginaryReserves = new(big.Int).Add(reserveXOutside, colReserves.Token0RealReserves)
		colReserves.Token1ImaginaryReserves = new(big.Int).Add(reserveYOutside, colReserves.Token1RealReserves)
		reserveXOutside, reserveYOutside = calculateReservesOutsideRange(
			bI1e27,
			price,
			debtReserves.Token0RealReserves,
			debtReserves.Token1RealReserves,
		)
		debtReserves.Token0ImaginaryReserves = new(big.Int).Add(reserveXOutside, debtReserves.Token0RealReserves)
		debtReserves.Token1ImaginaryReserves = new(big.Int).Add(reserveYOutside, debtReserves.Token1RealReserves)

		// expected required ratio:
		// token0Reserves >= (token1Reserves * 1e27) / (price * MIN_SWAP_LIQUIDITY)
		// so 2M / 0.85e4, which is 235.29 -> swap amount @~14_764

		// Test for swap amount 14_766, revert should hit
		swapAmount := big.NewInt(14_766 * 1e6 * 1e6)
		price, _ = getApproxCenterPriceOut(swapAmount, false, colReserves, NewDebtReservesEmpty())
		result, _ := swapOutAdjusted(false, swapAmount, colReserves, NewDebtReservesEmpty(), decimals, decimals, limitsWide(), price, time.Now().Unix()-10)
		require.Nil(t, result, "FAIL: reserves ratio verification revert NOT hit for col reserves when swap amount %d", 14_766)
		price, _ = getApproxCenterPriceOut(swapAmount, false, NewColReservesEmpty(), debtReserves)
		result, _ = swapOutAdjusted(false, swapAmount, NewColReservesEmpty(), debtReserves, decimals, decimals, limitsWide(), price, time.Now().Unix()-10)
		require.Nil(t, result, "FAIL: reserves ratio verification revert NOT hit for debt reserves when swap amount %d", 14_766)

		// refresh reserves
		colReserves = NewVerifyRatioColReservesSwapOut()
		debtReserves = NewVerifyRatioDebtReservesSwapOut()
		reserveXOutside, reserveYOutside = calculateReservesOutsideRange(
			bI1e27,
			price,
			colReserves.Token0RealReserves,
			colReserves.Token1RealReserves,
		)
		colReserves.Token0ImaginaryReserves = new(big.Int).Add(reserveXOutside, colReserves.Token0RealReserves)
		colReserves.Token1ImaginaryReserves = new(big.Int).Add(reserveYOutside, colReserves.Token1RealReserves)
		reserveXOutside, reserveYOutside = calculateReservesOutsideRange(
			bI1e27,
			price,
			debtReserves.Token0RealReserves,
			debtReserves.Token1RealReserves,
		)
		debtReserves.Token0ImaginaryReserves = new(big.Int).Add(reserveXOutside, debtReserves.Token0RealReserves)
		debtReserves.Token1ImaginaryReserves = new(big.Int).Add(reserveYOutside, debtReserves.Token1RealReserves)

		// Test for swap amount 14_762, revert should NOT hit
		swapAmount = big.NewInt(14_762 * 1e6 * 1e6)
		err := error(nil)
		price, _ = getApproxCenterPriceOut(swapAmount, false, colReserves, NewDebtReservesEmpty())
		result, err = swapOutAdjusted(false, swapAmount, colReserves, NewDebtReservesEmpty(), decimals, decimals, limitsWide(), price, time.Now().Unix()-10)
		require.NoError(t, err, "Error during swapOutAdjusted for col reserves")
		require.NotNil(t, result, "FAIL: reserves ratio verification revert hit for col reserves when swap amount %d", 14_762)
		price, _ = getApproxCenterPriceOut(swapAmount, false, NewColReservesEmpty(), debtReserves)
		result, _ = swapOutAdjusted(false, swapAmount, NewColReservesEmpty(), debtReserves, decimals, decimals, limitsWide(), price, time.Now().Unix()-10)
		require.NotNil(t, result, "FAIL: reserves ratio verification revert hit for debt reserves when swap amount %d", 14_762)
	})
}
