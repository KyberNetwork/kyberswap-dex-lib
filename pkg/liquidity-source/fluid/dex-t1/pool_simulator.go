package dexT1

import (
	"errors"
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	StaticExtra

	CollateralReserves CollateralReserves
	DebtReserves       DebtReserves
	DexLimits          DexLimits
	CenterPrice        *big.Int

	Token0Decimals uint8
	Token1Decimals uint8

	SyncTimestamp            int64
	IsSwapAndArbitragePaused bool
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	fee := big.NewInt(int64(entityPool.SwapFee * FeePercentPrecision))

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
			SwapFee:     fee,
		}},
		CollateralReserves:       extra.CollateralReserves,
		DebtReserves:             extra.DebtReserves,
		DexLimits:                extra.DexLimits,
		CenterPrice:              extra.CenterPrice,
		Token0Decimals:           entityPool.Tokens[0].Decimals,
		Token1Decimals:           entityPool.Tokens[1].Decimals,
		StaticExtra:              staticExtra,
		IsSwapAndArbitragePaused: extra.IsSwapAndArbitragePaused,
		SyncTimestamp:            entityPool.Timestamp,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.IsSwapAndArbitragePaused {
		return nil, ErrSwapAndArbitragePaused
	}

	if param.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	swap0To1 := param.TokenAmountIn.Token == s.Info.Tokens[0]

	var tokenInDecimals, tokenOutDecimals uint8
	if swap0To1 {
		tokenInDecimals = s.Token0Decimals
		tokenOutDecimals = s.Token1Decimals
	} else {
		tokenOutDecimals = s.Token0Decimals
		tokenInDecimals = s.Token1Decimals
	}

	// fee is applied on token in
	fee := new(big.Int).Mul(param.TokenAmountIn.Amount, s.Pool.Info.SwapFee)
	fee = fee.Div(fee, SIX_DECIMALS)

	amountInAfterFee := new(big.Int).Sub(param.TokenAmountIn.Amount, fee)

	collateralReserves := s.CollateralReserves.Clone()
	debtReserves := s.DebtReserves.Clone()
	dexLimits := s.DexLimits.Clone()
	syncTimestamp := s.SyncTimestamp
	centerPrice := s.CenterPrice

	tokenAmountOut, err := swapIn(swap0To1, amountInAfterFee, collateralReserves, debtReserves,
		int64(tokenInDecimals), int64(tokenOutDecimals), dexLimits, centerPrice, syncTimestamp)
	if err != nil {
		return nil, err
	}

	if err := s.validateAmountOut(swap0To1, tokenAmountOut); err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: tokenAmountOut},
		Fee:            &pool.TokenAmount{Token: param.TokenAmountIn.Token, Amount: fee},
		Gas:            defaultGas.Swap,
		SwapInfo: SwapInfo{
			HasNative:             s.HasNative,
			NewCollateralReserves: collateralReserves,
			NewDebtReserves:       debtReserves,
			NewDexLimits:          dexLimits,
		},
	}, nil
}

func (s *PoolSimulator) validateAmountOut(swap0To1 bool, tokenAmountOut *big.Int) error {
	if tokenAmountOut.Cmp(s.GetReserves()[lo.Ternary(swap0To1, 1, 0)]) > 0 {
		return ErrInsufficientReserve
	}

	return nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if s.IsSwapAndArbitragePaused {
		return nil, ErrSwapAndArbitragePaused
	}

	if param.TokenAmountOut.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}

	swap0To1 := param.TokenAmountOut.Token == s.Info.Tokens[1]

	if err := s.validateAmountOut(swap0To1, param.TokenAmountOut.Amount); err != nil {
		return nil, err
	}

	var tokenInDecimals, tokenOutDecimals uint8
	if swap0To1 {
		tokenInDecimals = s.Token0Decimals
		tokenOutDecimals = s.Token1Decimals
	} else {
		tokenOutDecimals = s.Token0Decimals
		tokenInDecimals = s.Token1Decimals
	}

	collateralReserves := s.CollateralReserves.Clone()
	debtReserves := s.DebtReserves.Clone()
	dexLimits := s.DexLimits.Clone()
	syncTimestamp := s.SyncTimestamp
	centerPrice := s.CenterPrice

	tokenAmountIn, err := swapOut(swap0To1, param.TokenAmountOut.Amount, collateralReserves, debtReserves,
		int64(tokenInDecimals), int64(tokenOutDecimals), dexLimits, centerPrice, syncTimestamp)
	if err != nil {
		return nil, err
	}

	// fee is applied on token in
	fee := new(big.Int).Mul(tokenAmountIn, s.Pool.Info.SwapFee)
	fee = fee.Div(fee, SIX_DECIMALS)

	amountInAfterFee := new(big.Int).Add(tokenAmountIn, fee)

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: param.TokenIn, Amount: amountInAfterFee},
		Fee:           &pool.TokenAmount{Token: param.TokenIn, Amount: fee},
		Gas:           defaultGas.Swap,
		SwapInfo: SwapInfo{
			HasNative:             s.HasNative,
			NewCollateralReserves: collateralReserves,
			NewDebtReserves:       debtReserves,
			NewDexLimits:          dexLimits,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	inputAmount, outputAmount := input.Amount, output.Amount

	for i := range s.Info.Tokens {
		if s.Info.Tokens[i] == input.Token {
			s.Info.Reserves[i] = new(big.Int).Add(s.Info.Reserves[i], inputAmount)
		}
		if s.Info.Tokens[i] == output.Token {
			s.Info.Reserves[i] = new(big.Int).Sub(s.Info.Reserves[i], outputAmount)
		}
	}

	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		s.CollateralReserves = swapInfo.NewCollateralReserves
		s.DebtReserves = swapInfo.NewDebtReserves
		// Note: limits are updated, but are likely off for the input token until newly fetched.
		// Erring on the cautious side with too tight limits to avoid potential reverts.
		// See Comment Ref #4327563287
		s.DexLimits = swapInfo.NewDexLimits
	}
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	return PoolMeta{
		BlockNumber:     s.Pool.Info.BlockNumber,
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
	}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	return lo.Ternary(valueobject.IsNative(tokenIn), "", s.GetAddress())
}

// ------------------------------------------------------------------------------------------------
// @dev the logic in the methods below mirrors the original Solidity code used in Dex, see
// https://github.com/Instadapp/fluid-contracts-public/tree/main/contracts/protocols/dex/poolT1
// ------------------------------------------------------------------------------------------------

/**
 * Given an input amount of asset and pair reserves, returns the maximum output amount of the other asset.
 * @param {number} amountIn - The amount of input asset.
 * @param {number} iReserveIn - Imaginary token reserve with input amount.
 * @param {number} iReserveOut - Imaginary token reserve of output amount.
 * @returns {number} - The maximum output amount of the other asset.
 */
func getAmountOut(amountIn *big.Int, iReserveIn *big.Int, iReserveOut *big.Int) *big.Int {
	// Both numerator and denominator are scaled to 1e6 to factor in fee scaling.
	numerator := new(big.Int).Mul(amountIn, iReserveOut)
	denominator := new(big.Int).Add(iReserveIn, amountIn)

	// Using the swap formula: (AmountIn * iReserveY) / (iReserveX + AmountIn)
	return numerator.Div(numerator, denominator)
}

/**
 * Given an output amount of asset and pair reserves, returns the input amount of the other asset
 * @param {number} amountOut - Desired output amount of the asset.
 * @param {number} iReserveIn - Imaginary token reserve of input amount.
 * @param {number} iReserveOut - Imaginary token reserve of output amount.
 * @returns {number} - The input amount of the other asset.
 */
func getAmountIn(amountOut *big.Int, iReserveIn *big.Int, iReserveOut *big.Int) *big.Int {
	// Both numerator and denominator are scaled to 1e6 to factor in fee scaling.
	numerator := new(big.Int).Mul(amountOut, iReserveIn)
	denominator := new(big.Int).Sub(iReserveOut, amountOut)

	// Using the swap formula: (AmountOut * iReserveX) / (iReserveY - AmountOut)
	return numerator.Div(numerator, denominator)
}

/**
 * Calculates how much of a swap should go through the collateral pool.
 * @param {number} t - Total amount in.
 * @param {number} x - Imaginary reserves of token out of collateral.
 * @param {number} y - Imaginary reserves of token in of collateral.
 * @param {number} x2 - Imaginary reserves of token out of debt.
 * @param {number} y2 - Imaginary reserves of token in of debt.
 * @returns {number} a - How much swap should go through collateral pool. Remaining will go from debt.
 * @note If a < 0 then entire trade route through debt pool and debt pool arbitrage with col pool.
 * @note If a > t then entire trade route through col pool and col pool arbitrage with debt pool.
 * @note If a > 0 & a < t then swap will route through both pools.
 */
func swapRoutingIn(t *big.Int, x *big.Int, y *big.Int, x2 *big.Int, y2 *big.Int) *big.Int {
	var xyRoot, x2y2Root big.Int
	xyRoot.Mul(x, y).Mul(&xyRoot, bI1e18).Sqrt(&xyRoot)
	x2y2Root.Mul(x2, y2).Mul(&x2y2Root, bI1e18).Sqrt(&x2y2Root)

	var tmp big.Int
	numerator := new(big.Int)
	numerator.Mul(y2, &xyRoot).Add(numerator, tmp.Mul(t, &xyRoot)).Sub(numerator, tmp.Mul(y, &x2y2Root))
	denominator := tmp.Add(&xyRoot, &x2y2Root)
	return numerator.Div(numerator, denominator)
}

/**
 * Calculates how much of a swap should go through the collateral pool for output amount.
 * @param {number} t - Total amount out.
 * @param {number} x - Imaginary reserves of token in of collateral.
 * @param {number} y - Imaginary reserves of token out of collateral.
 * @param {number} x2 - Imaginary reserves of token in of debt.
 * @param {number} y2 - Imaginary reserves of token out of debt.
 * @returns {number} a - How much swap should go through collateral pool. Remaining will go from debt.
 * @note If a < 0 then entire trade route through debt pool and debt pool arbitrage with col pool.
 * @note If a > t then entire trade route through col pool and col pool arbitrage with debt pool.
 * @note If a > 0 & a < t then swap will route through both pools.
 */
func swapRoutingOut(t *big.Int, x *big.Int, y *big.Int, x2 *big.Int, y2 *big.Int) *big.Int {
	var xyRoot, x2y2Root big.Int
	xyRoot.Mul(x, y).Mul(&xyRoot, bI1e18).Sqrt(&xyRoot)
	x2y2Root.Mul(x2, y2).Mul(&x2y2Root, bI1e18).Sqrt(&x2y2Root)

	var tmp big.Int
	numerator := new(big.Int)
	numerator.Mul(t, &xyRoot).Add(numerator, tmp.Mul(y, &x2y2Root)).Sub(numerator, tmp.Mul(y2, &xyRoot))
	denominator := tmp.Add(&xyRoot, &x2y2Root)
	return numerator.Div(numerator, denominator)
}

/**
 * Checks if token0 reserves are sufficient compared to token1 reserves.
 * This helps prevent edge cases and ensures high precision in calculations.
 * @param {number} token0Reserves - The reserves of token0.
 * @param {number} token1Reserves - The reserves of token1.
 * @param {number} price - The current price used for calculation.
 * @returns {boolean} - Returns false if token0 reserves are too low, true otherwise.
 */
func verifyToken0Reserves(token0Reserves *big.Int, token1Reserves *big.Int, price *big.Int) bool {
	numerator := new(big.Int).Mul(token1Reserves, bI1e27)
	denominator := new(big.Int).Mul(price, MinSwapLiquidity)
	return token0Reserves.Cmp(numerator.Div(numerator, denominator)) >= 0
}

/**
 * Checks if token1 reserves are sufficient compared to token0 reserves.
 * This helps prevent edge cases and ensures high precision in calculations.
 * @param {number} token0Reserves - The reserves of token0.
 * @param {number} token1Reserves - The reserves of token1.
 * @param {number} price - The current price used for calculation.
 * @returns {boolean} - Returns false if token1 reserves are too low, true otherwise.
 */
func verifyToken1Reserves(token0Reserves *big.Int, token1Reserves *big.Int, price *big.Int) bool {
	numerator := new(big.Int).Mul(token0Reserves, price)
	denominator := new(big.Int).Mul(bI1e27, MinSwapLiquidity)
	return token1Reserves.Cmp(numerator.Div(numerator, denominator)) >= 0
}

/**
 * Calculates the output amount for a given input amount in a swap operation.
 * @param {boolean} swap0To1 - Direction of the swap. True if swapping token0 for token1, false otherwise.
 * @param {number} amountToSwap - The amount of input token to be swapped scaled to 1e12.
 * @param {Object} colReserves - The reserves of the collateral pool scaled to 1e12.
 * @param {number} colReserves.token0RealReserves - Real reserves of token0 in the collateral pool.
 * @param {number} colReserves.token1RealReserves - Real reserves of token1 in the collateral pool.
 * @param {number} colReserves.token0ImaginaryReserves - Imaginary reserves of token0 in the collateral pool.
 * @param {number} colReserves.token1ImaginaryReserves - Imaginary reserves of token1 in the collateral pool.
 * @param {Object} debtReserves - The reserves of the debt pool scaled to 1e12.
 * @param {number} debtReserves.token0RealReserves - Real reserves of token0 in the debt pool.
 * @param {number} debtReserves.token1RealReserves - Real reserves of token1 in the debt pool.
 * @param {number} debtReserves.token0ImaginaryReserves - Imaginary reserves of token0 in the debt pool.
 * @param {number} debtReserves.token1ImaginaryReserves - Imaginary reserves of token1 in the debt pool.
 * @param {number} outDecimals - The number of decimals for the output token.
 * @param {Object} currentLimits - current borrowable & withdrawable of the pool. in token decimals.
 * @param {Object} currentLimits.borrowableToken0 - token0 borrow limit
 * @param {number} currentLimits.borrowableToken0.available - token0 instant borrowable available
 * @param {number} currentLimits.borrowableToken0.expandsTo - token0 maximum amount the available borrow amount expands to
 * @param {number} currentLimits.borrowableToken0.expandDuration - duration for token0 available to grow to expandsTo
 * @param {Object} currentLimits.borrowableToken1 - token1 borrow limit
 * @param {number} currentLimits.borrowableToken1.available - token1 instant borrowable available
 * @param {number} currentLimits.borrowableToken1.expandsTo - token1 maximum amount the available borrow amount expands to
 * @param {number} currentLimits.borrowableToken1.expandDuration - duration for token1 available to grow to expandsTo
 * @param {Object} currentLimits.withdrawableToken0 - token0 withdraw limit
 * @param {number} currentLimits.withdrawableToken0.available - token0 instant withdrawable available
 * @param {number} currentLimits.withdrawableToken0.expandsTo - token0 maximum amount the available withdraw amount expands to
 * @param {number} currentLimits.withdrawableToken0.expandDuration - duration for token0 available to grow to expandsTo
 * @param {Object} currentLimits.withdrawableToken1 - token1 withdraw limit
 * @param {number} currentLimits.withdrawableToken1.available - token1 instant withdrawable available
 * @param {number} currentLimits.withdrawableToken1.expandsTo - token1 maximum amount the available withdraw amount expands to
 * @param {number} currentLimits.withdrawableToken1.expandDuration - duration for token1 available to grow to expandsTo
 * @param {number} centerPrice - current center price used to verify reserves ratio
 * @param {number} syncTime - timestamp in seconds when the limits were synced
 * @returns {number} amountOut - The calculated output amount.
 * @returns {error} - An error object if the operation fails.
 */
func swapInAdjusted(swap0To1 bool, amountToSwap *big.Int, colReserves CollateralReserves, debtReserves DebtReserves,
	outDecimals int64, currentLimits DexLimits, centerPrice *big.Int, syncTime int64) (*big.Int, error) {
	var (
		colIReserveIn, colIReserveOut, debtIReserveIn, debtIReserveOut *big.Int
		colReserveIn, colReserveOut, debtReserveIn, debtReserveOut     *big.Int
		borrowable, withdrawable                                       *big.Int
	)

	if swap0To1 {
		colReserveIn = colReserves.Token0RealReserves
		colReserveOut = colReserves.Token1RealReserves
		colIReserveIn = colReserves.Token0ImaginaryReserves
		colIReserveOut = colReserves.Token1ImaginaryReserves
		debtReserveIn = debtReserves.Token0RealReserves
		debtReserveOut = debtReserves.Token1RealReserves
		debtIReserveIn = debtReserves.Token0ImaginaryReserves
		debtIReserveOut = debtReserves.Token1ImaginaryReserves
		borrowable = getExpandedLimit(syncTime, currentLimits.BorrowableToken1)
		withdrawable = getExpandedLimit(syncTime, currentLimits.WithdrawableToken1)
	} else {
		colReserveIn = colReserves.Token1RealReserves
		colReserveOut = colReserves.Token0RealReserves
		colIReserveIn = colReserves.Token1ImaginaryReserves
		colIReserveOut = colReserves.Token0ImaginaryReserves
		debtReserveIn = debtReserves.Token1RealReserves
		debtReserveOut = debtReserves.Token0RealReserves
		debtIReserveIn = debtReserves.Token1ImaginaryReserves
		debtIReserveOut = debtReserves.Token0ImaginaryReserves
		borrowable = getExpandedLimit(syncTime, currentLimits.BorrowableToken0)
		withdrawable = getExpandedLimit(syncTime, currentLimits.WithdrawableToken0)
	}

	// bring borrowable and withdrawable from token decimals to 1e12 decimals, same as amounts
	var factor *big.Int
	if DexAmountsDecimals > outDecimals {
		factor = bignumber.TenPowInt(DexAmountsDecimals - outDecimals)
		borrowable = new(big.Int).Mul(borrowable, factor)
		withdrawable = new(big.Int).Mul(withdrawable, factor)
	} else {
		factor = bignumber.TenPowInt(outDecimals - DexAmountsDecimals)
		borrowable = new(big.Int).Div(borrowable, factor)
		withdrawable = new(big.Int).Div(withdrawable, factor)
	}

	// Check if all reserves of collateral pool are greater than 0
	colPoolEnabled := colReserves.Token0RealReserves.Sign() > 0 &&
		colReserves.Token1RealReserves.Sign() > 0 &&
		colReserves.Token0ImaginaryReserves.Sign() > 0 &&
		colReserves.Token1ImaginaryReserves.Sign() > 0

	// Check if all reserves of debt pool are greater than 0
	debtPoolEnabled := debtReserves.Token0RealReserves.Sign() > 0 &&
		debtReserves.Token1RealReserves.Sign() > 0 &&
		debtReserves.Token0ImaginaryReserves.Sign() > 0 &&
		debtReserves.Token1ImaginaryReserves.Sign() > 0

	var a *big.Int
	if colPoolEnabled && debtPoolEnabled {
		a = swapRoutingIn(amountToSwap, colIReserveOut, colIReserveIn, debtIReserveOut, debtIReserveIn)
	} else if debtPoolEnabled {
		a = big.NewInt(-1) // Route from debt pool
	} else if colPoolEnabled {
		a = new(big.Int).Add(amountToSwap, bignumber.One) // Route from collateral pool
	} else {
		return nil, errors.New("no pools are enabled")
	}

	amountInCollateral := new(big.Int)
	amountOutCollateral := new(big.Int)
	amountInDebt := new(big.Int)
	amountOutDebt := new(big.Int)

	triggerUpdateDebtReserves := false
	triggerUpdateColReserves := false

	if a.Sign() <= 0 {
		// Entire trade routes through debt pool
		amountInDebt = amountToSwap
		amountOutDebt = getAmountOut(amountToSwap, debtIReserveIn, debtIReserveOut)

		triggerUpdateDebtReserves = true
	} else if a.Cmp(amountToSwap) >= 0 {
		// Entire trade routes through collateral pool
		amountInCollateral = amountToSwap
		amountOutCollateral = getAmountOut(amountToSwap, colIReserveIn, colIReserveOut)

		triggerUpdateColReserves = true
	} else {
		// Trade routes through both pools
		amountInDebt.Sub(amountToSwap, a)

		amountInCollateral = a
		amountOutCollateral = getAmountOut(a, colIReserveIn, colIReserveOut)
		amountInDebt.Sub(amountToSwap, a)
		amountOutDebt = getAmountOut(amountInDebt, debtIReserveIn, debtIReserveOut)

		triggerUpdateDebtReserves = true
		triggerUpdateColReserves = true
	}

	if amountOutDebt.Cmp(debtReserveOut) > 0 {
		return nil, ErrInsufficientReserve
	}

	if amountOutCollateral.Cmp(colReserveOut) > 0 {
		return nil, ErrInsufficientReserve
	}

	if amountOutDebt.Cmp(borrowable) > 0 {
		return nil, ErrInsufficientBorrowable
	}

	if amountOutCollateral.Cmp(withdrawable) > 0 {
		return nil, ErrInsufficientWithdrawable
	}

	if amountInCollateral.Sign() > 0 {
		reservesRatioValid := false
		if swap0To1 {
			reservesRatioValid = verifyToken1Reserves(new(big.Int).Add(colReserveIn, amountInCollateral),
				new(big.Int).Sub(colReserveOut, amountOutCollateral), centerPrice)
		} else {
			reservesRatioValid = verifyToken0Reserves(new(big.Int).Sub(colReserveOut, amountOutCollateral),
				new(big.Int).Add(colReserveIn, amountInCollateral), centerPrice)
		}
		if !reservesRatioValid {
			return nil, ErrVerifyReservesRatiosInvalid
		}
	}
	if amountInDebt.Sign() > 0 {
		reservesRatioValid := false
		if swap0To1 {
			reservesRatioValid = verifyToken1Reserves(new(big.Int).Add(debtReserveIn, amountInDebt),
				new(big.Int).Sub(debtReserveOut, amountOutDebt), centerPrice)
		} else {
			reservesRatioValid = verifyToken0Reserves(new(big.Int).Sub(debtReserveOut, amountOutDebt),
				new(big.Int).Add(debtReserveIn, amountInDebt), centerPrice)
		}
		if !reservesRatioValid {
			return nil, ErrVerifyReservesRatiosInvalid
		}
	}

	oldPrice, newPrice := new(big.Int), new(big.Int)
	priceDiff, maxPriceDiff := new(big.Int), new(big.Int)
	// from whatever pool higher amount of swap is routing we are taking that as final price, does not matter much because both pools final price should be same
	if amountInCollateral.Cmp(amountInDebt) > 0 {
		// new pool price from col pool
		if swap0To1 {
			oldPrice.Div(oldPrice.Mul(colIReserveOut, bI1e27), colIReserveIn)
			newPrice.Div(newPrice.Mul(newPrice.Sub(colIReserveOut, amountOutCollateral), bI1e27),
				new(big.Int).Add(colIReserveIn, amountInCollateral))
		} else {
			oldPrice.Div(oldPrice.Mul(colIReserveIn, bI1e27), colIReserveOut)
			newPrice.Div(newPrice.Mul(newPrice.Add(colIReserveIn, amountInCollateral), bI1e27),
				new(big.Int).Sub(colIReserveOut, amountOutCollateral))
		}
	} else {
		// new pool price from debt pool
		if swap0To1 {
			oldPrice.Div(oldPrice.Mul(debtIReserveOut, bI1e27), debtIReserveIn)
			newPrice.Div(newPrice.Mul(newPrice.Sub(debtIReserveOut, amountOutDebt), bI1e27),
				new(big.Int).Add(debtIReserveIn, amountInDebt))
		} else {
			oldPrice.Div(oldPrice.Mul(debtIReserveIn, bI1e27), debtIReserveOut)
			newPrice.Div(newPrice.Mul(newPrice.Add(debtIReserveIn, amountInDebt), bI1e27),
				new(big.Int).Sub(debtIReserveOut, amountOutDebt))
		}
	}
	priceDiff.Abs(priceDiff.Sub(oldPrice, newPrice))
	maxPriceDiff.Div(maxPriceDiff.Mul(oldPrice, MaxPriceDiff), TWO_DECIMALS)
	if priceDiff.Cmp(maxPriceDiff) > 0 {
		// if price diff is > 5% then swap would revert.
		return nil, ErrInsufficientMaxPrice
	}

	if triggerUpdateColReserves {
		updateCollateralReservesAndLimits(swap0To1, amountToSwap, amountOutCollateral, colReserves, currentLimits)
	}

	if triggerUpdateDebtReserves {
		updateDebtReservesAndLimits(swap0To1, amountToSwap, amountOutDebt, debtReserves, currentLimits)
	}

	return amountOutCollateral.Add(amountOutCollateral, amountOutDebt), nil
}

/**
 * Calculates the output amount for a given input amount in a swap operation.
 * @param {boolean} swap0To1 - Direction of the swap. True if swapping token0 for token1, false otherwise.
 * @param {number} amountToSwap - The amount of input token to be swapped.
 * @param {Object} colReserves - The reserves of the collateral pool.
 * @param {number} colReserves.token0RealReserves - Real reserves of token0 in the collateral pool.
 * @param {number} colReserves.token1RealReserves - Real reserves of token1 in the collateral pool.
 * @param {number} colReserves.token0ImaginaryReserves - Imaginary reserves of token0 in the collateral pool.
 * @param {number} colReserves.token1ImaginaryReserves - Imaginary reserves of token1 in the collateral pool.
 * @param {Object} debtReserves - The reserves of the debt pool.
 * @param {number} debtReserves.token0RealReserves - Real reserves of token0 in the debt pool.
 * @param {number} debtReserves.token1RealReserves - Real reserves of token1 in the debt pool.
 * @param {number} debtReserves.token0ImaginaryReserves - Imaginary reserves of token0 in the debt pool.
 * @param {number} debtReserves.token1ImaginaryReserves - Imaginary reserves of token1 in the debt pool.
 * @param {number} inDecimals - The number of decimals for the input token.
 * @param {number} outDecimals - The number of decimals for the output token.
 * @param {Object} currentLimits - current borrowable & withdrawable of the pool. in token decimals.
 * @param {Object} currentLimits.borrowableToken0 - token0 borrow limit
 * @param {number} currentLimits.borrowableToken0.available - token0 instant borrowable available
 * @param {number} currentLimits.borrowableToken0.expandsTo - token0 maximum amount the available borrow amount expands to
 * @param {number} currentLimits.borrowableToken0.expandDuration - duration for token0 available to grow to expandsTo
 * @param {Object} currentLimits.borrowableToken1 - token1 borrow limit
 * @param {number} currentLimits.borrowableToken1.available - token1 instant borrowable available
 * @param {number} currentLimits.borrowableToken1.expandsTo - token1 maximum amount the available borrow amount expands to
 * @param {number} currentLimits.borrowableToken1.expandDuration - duration for token1 available to grow to expandsTo
 * @param {Object} currentLimits.withdrawableToken0 - token0 withdraw limit
 * @param {number} currentLimits.withdrawableToken0.available - token0 instant withdrawable available
 * @param {number} currentLimits.withdrawableToken0.expandsTo - token0 maximum amount the available withdraw amount expands to
 * @param {number} currentLimits.withdrawableToken0.expandDuration - duration for token0 available to grow to expandsTo
 * @param {Object} currentLimits.withdrawableToken1 - token1 withdraw limit
 * @param {number} currentLimits.withdrawableToken1.available - token1 instant withdrawable available
 * @param {number} currentLimits.withdrawableToken1.expandsTo - token1 maximum amount the available withdraw amount expands to
 * @param {number} currentLimits.withdrawableToken1.expandDuration - duration for token1 available to grow to expandsTo
 * @param {number} centerPrice - current center price used to verify reserves ratio
 * @param {number} syncTime - timestamp in seconds when the limits were synced
 * @returns {number} amountOut - The calculated output amount.
 * @returns {error} - An error object if the operation fails.
 */
func swapIn(
	swap0To1 bool,
	amountIn *big.Int,
	colReserves CollateralReserves,
	debtReserves DebtReserves,
	inDecimals int64,
	outDecimals int64,
	currentLimits DexLimits,
	centerPrice *big.Int,
	syncTime int64,
) (*big.Int, error) {
	var amountInAdjusted *big.Int

	if inDecimals > DexAmountsDecimals {
		amountInAdjusted = new(big.Int).Div(amountIn, bignumber.TenPowInt(inDecimals-DexAmountsDecimals))
	} else {
		amountInAdjusted = new(big.Int).Mul(amountIn, bignumber.TenPowInt(DexAmountsDecimals-inDecimals))
	}

	if amountInAdjusted.Cmp(SIX_DECIMALS) < 0 || amountIn.Cmp(TWO_DECIMALS) < 0 {
		return nil, ErrInvalidAmountIn
	}

	amountOut, err := swapInAdjusted(swap0To1, amountInAdjusted, colReserves, debtReserves, outDecimals, currentLimits,
		centerPrice, syncTime)

	if err != nil {
		return nil, err
	}

	if outDecimals > DexAmountsDecimals {
		amountOut = new(big.Int).Mul(amountOut, bignumber.TenPowInt(outDecimals-DexAmountsDecimals))
	} else {
		amountOut = new(big.Int).Div(amountOut, bignumber.TenPowInt(DexAmountsDecimals-outDecimals))
	}

	return amountOut, nil
}

/**
 * Calculates the input amount for a given output amount in a swap operation.
 * @param {boolean} swap0to1 - Direction of the swap. True if swapping token0 for token1, false otherwise.
 * @param {number} amountOut - The amount of output token to be swapped scaled to 1e12.
 * @param {Object} colReserves - The reserves of the collateral pool scaled to 1e12.
 * @param {number} colReserves.token0RealReserves - Real reserves of token0 in the collateral pool.
 * @param {number} colReserves.token1RealReserves - Real reserves of token1 in the collateral pool.
 * @param {number} colReserves.token0ImaginaryReserves - Imaginary reserves of token0 in the collateral pool.
 * @param {number} colReserves.token1ImaginaryReserves - Imaginary reserves of token1 in the collateral pool.
 * @param {Object} debtReserves - The reserves of the debt pool scaled to 1e12.
 * @param {number} debtReserves.token0RealReserves - Real reserves of token0 in the debt pool.
 * @param {number} debtReserves.token1RealReserves - Real reserves of token1 in the debt pool.
 * @param {number} debtReserves.token0ImaginaryReserves - Imaginary reserves of token0 in the debt pool.
 * @param {number} debtReserves.token1ImaginaryReserves - Imaginary reserves of token1 in the debt pool.
 * @param {number} outDecimals - The number of decimals for the output token.
 * @param {Object} currentLimits - current borrowable & withdrawable of the pool. in token decimals.
 * @param {Object} currentLimits.borrowableToken0 - token0 borrow limit
 * @param {number} currentLimits.borrowableToken0.available - token0 instant borrowable available
 * @param {number} currentLimits.borrowableToken0.expandsTo - token0 maximum amount the available borrow amount expands to
 * @param {number} currentLimits.borrowableToken0.expandDuration - duration for token0 available to grow to expandsTo
 * @param {Object} currentLimits.borrowableToken1 - token1 borrow limit
 * @param {number} currentLimits.borrowableToken1.available - token1 instant borrowable available
 * @param {number} currentLimits.borrowableToken1.expandsTo - token1 maximum amount the available borrow amount expands to
 * @param {number} currentLimits.borrowableToken1.expandDuration - duration for token1 available to grow to expandsTo
 * @param {Object} currentLimits.withdrawableToken0 - token0 withdraw limit
 * @param {number} currentLimits.withdrawableToken0.available - token0 instant withdrawable available
 * @param {number} currentLimits.withdrawableToken0.expandsTo - token0 maximum amount the available withdraw amount expands to
 * @param {number} currentLimits.withdrawableToken0.expandDuration - duration for token0 available to grow to expandsTo
 * @param {Object} currentLimits.withdrawableToken1 - token1 withdraw limit
 * @param {number} currentLimits.withdrawableToken1.available - token1 instant withdrawable available
 * @param {number} currentLimits.withdrawableToken1.expandsTo - token1 maximum amount the available withdraw amount expands to
 * @param {number} currentLimits.withdrawableToken1.expandDuration - duration for token1 available to grow to expandsTo
 * @param {number} centerPrice - current center price used to verify reserves ratio
 * @param {number} syncTime - timestamp in seconds when the limits were synced
 * @returns {number} amountIn - The calculated input amount required for the swap.
 * @returns {error} - An error object if the operation fails.
 */
func swapOutAdjusted(
	swap0To1 bool,
	amountOut *big.Int,
	colReserves CollateralReserves,
	debtReserves DebtReserves,
	outDecimals int64,
	currentLimits DexLimits,
	centerPrice *big.Int,
	syncTime int64,
) (*big.Int, error) {
	var (
		colIReserveIn, colIReserveOut, debtIReserveIn, debtIReserveOut *big.Int
		colReserveIn, colReserveOut, debtReserveIn, debtReserveOut     *big.Int
		borrowable, withdrawable                                       *big.Int
	)

	if swap0To1 {
		colReserveIn = colReserves.Token0RealReserves
		colReserveOut = colReserves.Token1RealReserves
		colIReserveIn = colReserves.Token0ImaginaryReserves
		colIReserveOut = colReserves.Token1ImaginaryReserves
		debtReserveIn = debtReserves.Token0RealReserves
		debtReserveOut = debtReserves.Token1RealReserves
		debtIReserveIn = debtReserves.Token0ImaginaryReserves
		debtIReserveOut = debtReserves.Token1ImaginaryReserves
		borrowable = getExpandedLimit(syncTime, currentLimits.BorrowableToken1)
		withdrawable = getExpandedLimit(syncTime, currentLimits.WithdrawableToken1)
	} else {
		colReserveIn = colReserves.Token1RealReserves
		colReserveOut = colReserves.Token0RealReserves
		colIReserveIn = colReserves.Token1ImaginaryReserves
		colIReserveOut = colReserves.Token0ImaginaryReserves
		debtReserveIn = debtReserves.Token1RealReserves
		debtReserveOut = debtReserves.Token0RealReserves
		debtIReserveIn = debtReserves.Token1ImaginaryReserves
		debtIReserveOut = debtReserves.Token0ImaginaryReserves
		borrowable = getExpandedLimit(syncTime, currentLimits.BorrowableToken0)
		withdrawable = getExpandedLimit(syncTime, currentLimits.WithdrawableToken0)
	}

	// bring borrowable and withdrawable from token decimals to 1e12 decimals, same as amounts
	var factor *big.Int
	if DexAmountsDecimals > outDecimals {
		factor = bignumber.TenPowInt(DexAmountsDecimals - outDecimals)
		borrowable = new(big.Int).Mul(borrowable, factor)
		withdrawable = new(big.Int).Mul(withdrawable, factor)
	} else {
		factor = bignumber.TenPowInt(outDecimals - DexAmountsDecimals)
		borrowable = new(big.Int).Div(borrowable, factor)
		withdrawable = new(big.Int).Div(withdrawable, factor)
	}

	// Check if all reserves of collateral pool are greater than 0
	colPoolEnabled := colReserves.Token0RealReserves.Sign() > 0 &&
		colReserves.Token1RealReserves.Sign() > 0 &&
		colReserves.Token0ImaginaryReserves.Sign() > 0 &&
		colReserves.Token1ImaginaryReserves.Sign() > 0

	// Check if all reserves of debt pool are greater than 0
	debtPoolEnabled := debtReserves.Token0RealReserves.Sign() > 0 &&
		debtReserves.Token1RealReserves.Sign() > 0 &&
		debtReserves.Token0ImaginaryReserves.Sign() > 0 &&
		debtReserves.Token1ImaginaryReserves.Sign() > 0

	var a *big.Int
	if colPoolEnabled && debtPoolEnabled {
		a = swapRoutingOut(amountOut, colIReserveIn, colIReserveOut, debtIReserveIn, debtIReserveOut)
	} else if debtPoolEnabled {
		a = big.NewInt(-1) // Route from debt pool
	} else if colPoolEnabled {
		a = new(big.Int).Add(amountOut, bignumber.One) // Route from collateral pool
	} else {
		return nil, errors.New("no pools are enabled")
	}

	amountInCollateral, amountOutCollateral := new(big.Int), new(big.Int)
	amountInDebt, amountOutDebt := new(big.Int), new(big.Int)

	triggerUpdateDebtReserves := false
	triggerUpdateColReserves := false

	if a.Sign() <= 0 {
		// Entire trade routes through debt pool
		amountOutDebt = amountOut
		amountInDebt = getAmountIn(amountOut, debtIReserveIn, debtIReserveOut)

		if amountOut.Cmp(debtReserveOut) > 0 {
			return nil, ErrInsufficientReserve
		}

		triggerUpdateDebtReserves = true
	} else if a.Cmp(amountOut) >= 0 {
		// Entire trade routes through collateral pool
		amountOutCollateral = amountOut
		amountInCollateral = getAmountIn(amountOut, colIReserveIn, colIReserveOut)
		if amountOut.Cmp(colReserveOut) > 0 {
			return nil, ErrInsufficientReserve
		}

		triggerUpdateColReserves = true
	} else {
		// Trade routes through both pools
		amountOutCollateral = a
		amountInCollateral = getAmountIn(a, colIReserveIn, colIReserveOut)
		amountOutDebt.Sub(amountOut, a)
		amountInDebt = getAmountIn(amountOutDebt, debtIReserveIn, debtIReserveOut)

		if amountOutDebt.Cmp(debtReserveOut) > 0 || amountOutCollateral.Cmp(colReserveOut) > 0 {
			return nil, ErrInsufficientReserve
		}

		triggerUpdateDebtReserves = true
		triggerUpdateColReserves = true
	}

	if amountOutDebt.Cmp(borrowable) > 0 {
		return nil, ErrInsufficientBorrowable
	}

	if amountOutCollateral.Cmp(withdrawable) > 0 {
		return nil, ErrInsufficientWithdrawable
	}

	if amountInCollateral.Sign() > 0 {
		reservesRatioValid := false
		if swap0To1 {
			reservesRatioValid = verifyToken1Reserves(new(big.Int).Add(colReserveIn, amountInCollateral),
				new(big.Int).Sub(colReserveOut, amountOutCollateral), centerPrice)
		} else {
			reservesRatioValid = verifyToken0Reserves(new(big.Int).Sub(colReserveOut, amountOutCollateral),
				new(big.Int).Add(colReserveIn, amountInCollateral), centerPrice)
		}
		if !reservesRatioValid {
			return nil, ErrVerifyReservesRatiosInvalid
		}
	}
	if amountInDebt.Sign() > 0 {
		reservesRatioValid := false
		if swap0To1 {
			reservesRatioValid = verifyToken1Reserves(new(big.Int).Add(debtReserveIn, amountInDebt),
				new(big.Int).Sub(debtReserveOut, amountOutDebt), centerPrice)
		} else {
			reservesRatioValid = verifyToken0Reserves(new(big.Int).Sub(debtReserveOut, amountOutDebt),
				new(big.Int).Add(debtReserveIn, amountInDebt), centerPrice)
		}
		if !reservesRatioValid {
			return nil, ErrVerifyReservesRatiosInvalid
		}
	}

	oldPrice, newPrice := new(big.Int), new(big.Int)
	priceDiff, maxPriceDiff := new(big.Int), new(big.Int)
	// from whatever pool higher amount of swap is routing we are taking that as final price, does not matter much because both pools final price should be same
	if amountOutCollateral.Cmp(amountOutDebt) > 0 {
		// new pool price from col pool
		if swap0To1 {
			oldPrice.Div(oldPrice.Mul(colIReserveOut, bI1e27), colIReserveIn)
			newPrice.Div(newPrice.Mul(newPrice.Sub(colIReserveOut, amountOutCollateral), bI1e27),
				new(big.Int).Add(colIReserveIn, amountInCollateral))
		} else {
			oldPrice.Div(oldPrice.Mul(colIReserveIn, bI1e27), colIReserveOut)
			newPrice.Div(newPrice.Mul(newPrice.Add(colIReserveIn, amountInCollateral), bI1e27),
				new(big.Int).Sub(colIReserveOut, amountOutCollateral))
		}
	} else {
		// new pool price from debt pool
		if swap0To1 {
			oldPrice.Div(oldPrice.Mul(debtIReserveOut, bI1e27), debtIReserveIn)
			newPrice.Div(newPrice.Mul(newPrice.Sub(debtIReserveOut, amountOutDebt), bI1e27),
				new(big.Int).Add(debtIReserveIn, amountInDebt))
		} else {
			oldPrice.Div(oldPrice.Mul(debtIReserveIn, bI1e27), debtIReserveOut)
			newPrice.Div(newPrice.Mul(newPrice.Add(debtIReserveIn, amountInDebt), bI1e27),
				new(big.Int).Sub(debtIReserveOut, amountOutDebt))
		}
	}
	priceDiff.Abs(priceDiff.Sub(oldPrice, newPrice))
	maxPriceDiff.Div(maxPriceDiff.Mul(oldPrice, MaxPriceDiff), TWO_DECIMALS)
	if priceDiff.Cmp(maxPriceDiff) > 0 {
		// if price diff is > 5% then swap would revert.
		return nil, ErrInsufficientMaxPrice
	}

	if triggerUpdateColReserves {
		updateCollateralReservesAndLimits(swap0To1, amountInCollateral, amountOutCollateral, colReserves, currentLimits)
	}

	if triggerUpdateDebtReserves {
		updateDebtReservesAndLimits(swap0To1, amountInDebt, amountOutDebt, debtReserves, currentLimits)
	}

	return amountInCollateral.Add(amountInCollateral, amountInDebt), nil
}

/**
 * Calculates the input amount for a given output amount in a swap operation.
 * @param {boolean} swap0to1 - Direction of the swap. True if swapping token0 for token1, false otherwise.
 * @param {number} amountOut - The amount of output token to be swapped.
 * @param {Object} colReserves - The reserves of the collateral pool.
 * @param {number} colReserves.token0RealReserves - Real reserves of token0 in the collateral pool.
 * @param {number} colReserves.token1RealReserves - Real reserves of token1 in the collateral pool.
 * @param {number} colReserves.token0ImaginaryReserves - Imaginary reserves of token0 in the collateral pool.
 * @param {number} colReserves.token1ImaginaryReserves - Imaginary reserves of token1 in the collateral pool.
 * @param {Object} debtReserves - The reserves of the debt pool.
 * @param {number} debtReserves.token0RealReserves - Real reserves of token0 in the debt pool.
 * @param {number} debtReserves.token1RealReserves - Real reserves of token1 in the debt pool.
 * @param {number} debtReserves.token0ImaginaryReserves - Imaginary reserves of token0 in the debt pool.
 * @param {number} debtReserves.token1ImaginaryReserves - Imaginary reserves of token1 in the debt pool.
 * @param {number} inDecimals - The number of decimals for the input token.
 * @param {number} outDecimals - The number of decimals for the output token.
 * @param {Object} currentLimits - current borrowable & withdrawable of the pool. in token decimals.
 * @param {Object} currentLimits.borrowableToken0 - token0 borrow limit
 * @param {number} currentLimits.borrowableToken0.available - token0 instant borrowable available
 * @param {number} currentLimits.borrowableToken0.expandsTo - token0 maximum amount the available borrow amount expands to
 * @param {number} currentLimits.borrowableToken0.expandDuration - duration for token0 available to grow to expandsTo
 * @param {Object} currentLimits.borrowableToken1 - token1 borrow limit
 * @param {number} currentLimits.borrowableToken1.available - token1 instant borrowable available
 * @param {number} currentLimits.borrowableToken1.expandsTo - token1 maximum amount the available borrow amount expands to
 * @param {number} currentLimits.borrowableToken1.expandDuration - duration for token1 available to grow to expandsTo
 * @param {Object} currentLimits.withdrawableToken0 - token0 withdraw limit
 * @param {number} currentLimits.withdrawableToken0.available - token0 instant withdrawable available
 * @param {number} currentLimits.withdrawableToken0.expandsTo - token0 maximum amount the available withdraw amount expands to
 * @param {number} currentLimits.withdrawableToken0.expandDuration - duration for token0 available to grow to expandsTo
 * @param {Object} currentLimits.withdrawableToken1 - token1 withdraw limit
 * @param {number} currentLimits.withdrawableToken1.available - token1 instant withdrawable available
 * @param {number} currentLimits.withdrawableToken1.expandsTo - token1 maximum amount the available withdraw amount expands to
 * @param {number} currentLimits.withdrawableToken1.expandDuration - duration for token1 available to grow to expandsTo
 * @param {number} centerPrice - current center price used to verify reserves ratio
 * @param {number} syncTime - timestamp in seconds when the limits were synced
 * @returns {number} amountIn - The calculated input amount required for the swap.
 * @returns {error} - An error object if the operation fails.
 */
func swapOut(
	swap0To1 bool,
	amountOut *big.Int,
	colReserves CollateralReserves,
	debtReserves DebtReserves,
	inDecimals int64,
	outDecimals int64,
	currentLimits DexLimits,
	centerPrice *big.Int,
	syncTime int64,
) (*big.Int, error) {
	var amountOutAdjusted *big.Int

	if outDecimals > DexAmountsDecimals {
		amountOutAdjusted = new(big.Int).Div(amountOut,
			bignumber.TenPowInt(outDecimals-DexAmountsDecimals))
	} else {
		amountOutAdjusted = new(big.Int).Mul(amountOut,
			bignumber.TenPowInt(DexAmountsDecimals-outDecimals))
	}

	amountIn, err := swapOutAdjusted(swap0To1, amountOutAdjusted, colReserves, debtReserves, outDecimals, currentLimits,
		centerPrice, syncTime)

	if err != nil {
		return nil, err
	}

	if inDecimals > DexAmountsDecimals {
		amountIn = new(big.Int).Mul(amountIn, bignumber.TenPowInt(inDecimals-DexAmountsDecimals))
	} else {
		amountIn = new(big.Int).Div(amountIn, bignumber.TenPowInt(DexAmountsDecimals-inDecimals))
	}

	return amountIn, nil
}

// Calculates the currently available swappable amount for a token limit considering expansion since last syncTime.
func getExpandedLimit(syncTime int64, limit TokenLimit) *big.Int {
	currentTime := time.Now().Unix() // get current time in seconds
	elapsedTime := currentTime - syncTime

	expandedAmount := limit.Available

	if elapsedTime < 10 {
		// if almost no time has elapsed, return available amount
		return expandedAmount
	} else if elapsedTime >= limit.ExpandDuration.Int64() {
		// if duration has passed, return max amount
		return limit.ExpandsTo
	}

	expandedAmount = new(big.Int).Sub(limit.ExpandsTo, limit.Available)
	expandedAmount.Mul(expandedAmount, big.NewInt(elapsedTime))
	expandedAmount.Div(expandedAmount, limit.ExpandDuration)
	expandedAmount.Add(expandedAmount, limit.Available)
	return expandedAmount
}

func updateCollateralReservesAndLimits(swap0To1 bool, amountIn, amountOut *big.Int, colReserves CollateralReserves,
	limits DexLimits) {
	if swap0To1 {
		colReserves.Token0RealReserves.Add(colReserves.Token0RealReserves, amountIn)
		colReserves.Token0ImaginaryReserves.Add(colReserves.Token0ImaginaryReserves, amountIn)
		colReserves.Token1RealReserves.Sub(colReserves.Token1RealReserves, amountOut)
		colReserves.Token1ImaginaryReserves.Sub(colReserves.Token1ImaginaryReserves, amountOut)

		limits.WithdrawableToken1.Available.Sub(limits.WithdrawableToken1.Available, amountOut)
		limits.WithdrawableToken1.ExpandsTo.Sub(limits.WithdrawableToken1.ExpandsTo, amountOut)
	} else {
		colReserves.Token0RealReserves.Sub(colReserves.Token0RealReserves, amountOut)
		colReserves.Token0ImaginaryReserves.Sub(colReserves.Token0ImaginaryReserves, amountOut)
		colReserves.Token1RealReserves.Add(colReserves.Token1RealReserves, amountIn)
		colReserves.Token1ImaginaryReserves.Add(colReserves.Token1ImaginaryReserves, amountIn)

		limits.WithdrawableToken0.Available.Sub(limits.WithdrawableToken0.Available, amountOut)
		limits.WithdrawableToken0.ExpandsTo.Sub(limits.WithdrawableToken0.ExpandsTo, amountOut)
	}
}

func updateDebtReservesAndLimits(swap0To1 bool, amountIn, amountOut *big.Int, debtReserves DebtReserves,
	limits DexLimits) {
	if swap0To1 {
		debtReserves.Token0RealReserves.Add(debtReserves.Token0RealReserves, amountIn)
		debtReserves.Token0ImaginaryReserves.Add(debtReserves.Token0ImaginaryReserves, amountIn)
		debtReserves.Token1RealReserves.Sub(debtReserves.Token1RealReserves, amountOut)
		debtReserves.Token1ImaginaryReserves.Sub(debtReserves.Token1ImaginaryReserves, amountOut)

		// Comment Ref #4327563287
		// if expandTo for borrowable and withdrawable match, that means they are a hard limit like liquidity layer balance
		// or utilization limit. In that case, the available swap amount should increase by `amountIn` but it's not guaranteed
		// because the actual borrow limit / withdrawal limit could be the limiting factor now, which could be even
		// only +1 bigger. So not updating in amount to avoid any revert. The same applies on all other similar cases in the code
		// below. Note a swap would anyway trigger an event, so the proper limits will be fetched shortly after the swap.
		limits.BorrowableToken1.Available.Sub(limits.BorrowableToken1.Available, amountOut)
		limits.BorrowableToken1.ExpandsTo.Sub(limits.BorrowableToken1.ExpandsTo, amountOut)
	} else {
		debtReserves.Token0RealReserves.Sub(debtReserves.Token0RealReserves, amountOut)
		debtReserves.Token0ImaginaryReserves.Sub(debtReserves.Token0ImaginaryReserves, amountOut)
		debtReserves.Token1RealReserves.Add(debtReserves.Token1RealReserves, amountIn)
		debtReserves.Token1ImaginaryReserves.Add(debtReserves.Token1ImaginaryReserves, amountIn)

		limits.BorrowableToken0.Available.Sub(limits.BorrowableToken0.Available, amountOut)
		limits.BorrowableToken0.ExpandsTo.Sub(limits.BorrowableToken0.ExpandsTo, amountOut)
	}
}
