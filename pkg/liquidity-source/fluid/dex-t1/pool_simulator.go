package dexT1

import (
	"errors"
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidAmountIn  = errors.New("invalid amountIn")
	ErrInvalidAmountOut = errors.New("invalid amount out")

	ErrInsufficientReserve    = errors.New("insufficient reserve: tokenOut amount exceeds reserve")
	ErrSwapAndArbitragePaused = errors.New("51043")

	ErrInsufficientWithdrawable = errors.New("insufficient reserve: tokenOut amount exceeds withdrawable limit")
	ErrInsufficientBorrowable   = errors.New("insufficient reserve: tokenOut amount exceeds borrowable limit")

	ErrInsufficientMaxPrice = errors.New("insufficient reserve: tokenOut amount exceeds max price limit")
)

type PoolSimulator struct {
	poolpkg.Pool
	StaticExtra

	CollateralReserves CollateralReserves
	DebtReserves       DebtReserves
	DexLimits          DexLimits

	Token0Decimals uint8
	Token1Decimals uint8

	SyncTimestamp            int64
	IsSwapAndArbitragePaused bool
}

var (
	// Uniswap takes total gas of 125k = 21k base gas & 104k swap (this is when user has token balance)
	// Fluid takes total gas of 175k = 21k base gas & 154k swap (this is when user has token balance),
	// with ETH swaps costing less (because no WETH conversion)
	defaultGas = Gas{Swap: 260000}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	fee := new(big.Int)
	fee.SetInt64(int64(entityPool.SwapFee * float64(FeePercentPrecision)))

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
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
		Token0Decimals:           entityPool.Tokens[0].Decimals,
		Token1Decimals:           entityPool.Tokens[1].Decimals,
		StaticExtra:              staticExtra,
		IsSwapAndArbitragePaused: extra.IsSwapAndArbitragePaused,
		SyncTimestamp:            entityPool.Timestamp,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.IsSwapAndArbitragePaused {
		return nil, ErrSwapAndArbitragePaused
	}

	if param.TokenAmountIn.Amount.Cmp(bignumber.ZeroBI) <= 0 {
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
	fee = new(big.Int).Div(fee, big.NewInt(Fee100PercentPrecision))

	amountInAfterFee := new(big.Int).Sub(param.TokenAmountIn.Amount, fee)

	collateralReserves := CollateralReserves{
		Token0RealReserves:      new(big.Int).Set(s.CollateralReserves.Token0RealReserves),
		Token1RealReserves:      new(big.Int).Set(s.CollateralReserves.Token1RealReserves),
		Token0ImaginaryReserves: new(big.Int).Set(s.CollateralReserves.Token0ImaginaryReserves),
		Token1ImaginaryReserves: new(big.Int).Set(s.CollateralReserves.Token1ImaginaryReserves),
	}

	debtReserves := DebtReserves{
		Token0RealReserves:      new(big.Int).Set(s.DebtReserves.Token0RealReserves),
		Token1RealReserves:      new(big.Int).Set(s.DebtReserves.Token1RealReserves),
		Token0ImaginaryReserves: new(big.Int).Set(s.DebtReserves.Token0ImaginaryReserves),
		Token1ImaginaryReserves: new(big.Int).Set(s.DebtReserves.Token1ImaginaryReserves),
	}

	dexLimits := DexLimits{
		BorrowableToken0: TokenLimit{
			Available:      new(big.Int).Set(s.DexLimits.BorrowableToken0.Available),
			ExpandsTo:      new(big.Int).Set(s.DexLimits.BorrowableToken0.ExpandsTo),
			ExpandDuration: new(big.Int).Set(s.DexLimits.BorrowableToken0.ExpandDuration),
		},
		BorrowableToken1: TokenLimit{
			Available:      new(big.Int).Set(s.DexLimits.BorrowableToken1.Available),
			ExpandsTo:      new(big.Int).Set(s.DexLimits.BorrowableToken1.ExpandsTo),
			ExpandDuration: new(big.Int).Set(s.DexLimits.BorrowableToken1.ExpandDuration),
		},
		WithdrawableToken0: TokenLimit{
			Available:      new(big.Int).Set(s.DexLimits.WithdrawableToken0.Available),
			ExpandsTo:      new(big.Int).Set(s.DexLimits.WithdrawableToken0.ExpandsTo),
			ExpandDuration: new(big.Int).Set(s.DexLimits.WithdrawableToken0.ExpandDuration),
		},
		WithdrawableToken1: TokenLimit{
			Available:      new(big.Int).Set(s.DexLimits.WithdrawableToken1.Available),
			ExpandsTo:      new(big.Int).Set(s.DexLimits.WithdrawableToken1.ExpandsTo),
			ExpandDuration: new(big.Int).Set(s.DexLimits.WithdrawableToken1.ExpandDuration),
		},
	}

	syncTimestamp := s.SyncTimestamp

	tokenAmountOut, err := swapIn(swap0To1, amountInAfterFee, collateralReserves, debtReserves,
		int64(tokenInDecimals), int64(tokenOutDecimals), dexLimits, syncTimestamp)
	if err != nil {
		return nil, err
	}

	if err := s.validateAmountOut(swap0To1, tokenAmountOut); err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: param.TokenOut, Amount: tokenAmountOut},
		Fee:            &poolpkg.TokenAmount{Token: param.TokenAmountIn.Token, Amount: fee},
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

func (s *PoolSimulator) CalcAmountIn(param poolpkg.CalcAmountInParams) (*poolpkg.CalcAmountInResult, error) {
	if s.IsSwapAndArbitragePaused {
		return nil, ErrSwapAndArbitragePaused
	}

	if param.TokenAmountOut.Amount.Cmp(bignumber.ZeroBI) <= 0 {
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

	collateralReserves := CollateralReserves{
		Token0RealReserves:      new(big.Int).Set(s.CollateralReserves.Token0RealReserves),
		Token1RealReserves:      new(big.Int).Set(s.CollateralReserves.Token1RealReserves),
		Token0ImaginaryReserves: new(big.Int).Set(s.CollateralReserves.Token0ImaginaryReserves),
		Token1ImaginaryReserves: new(big.Int).Set(s.CollateralReserves.Token1ImaginaryReserves),
	}

	debtReserves := DebtReserves{
		Token0RealReserves:      new(big.Int).Set(s.DebtReserves.Token0RealReserves),
		Token1RealReserves:      new(big.Int).Set(s.DebtReserves.Token1RealReserves),
		Token0ImaginaryReserves: new(big.Int).Set(s.DebtReserves.Token0ImaginaryReserves),
		Token1ImaginaryReserves: new(big.Int).Set(s.DebtReserves.Token1ImaginaryReserves),
	}

	dexLimits := DexLimits{
		BorrowableToken0: TokenLimit{
			Available:      new(big.Int).Set(s.DexLimits.BorrowableToken0.Available),
			ExpandsTo:      new(big.Int).Set(s.DexLimits.BorrowableToken0.ExpandsTo),
			ExpandDuration: new(big.Int).Set(s.DexLimits.BorrowableToken0.ExpandDuration),
		},
		BorrowableToken1: TokenLimit{
			Available:      new(big.Int).Set(s.DexLimits.BorrowableToken1.Available),
			ExpandsTo:      new(big.Int).Set(s.DexLimits.BorrowableToken1.ExpandsTo),
			ExpandDuration: new(big.Int).Set(s.DexLimits.BorrowableToken1.ExpandDuration),
		},
		WithdrawableToken0: TokenLimit{
			Available:      new(big.Int).Set(s.DexLimits.WithdrawableToken0.Available),
			ExpandsTo:      new(big.Int).Set(s.DexLimits.WithdrawableToken0.ExpandsTo),
			ExpandDuration: new(big.Int).Set(s.DexLimits.WithdrawableToken0.ExpandDuration),
		},
		WithdrawableToken1: TokenLimit{
			Available:      new(big.Int).Set(s.DexLimits.WithdrawableToken1.Available),
			ExpandsTo:      new(big.Int).Set(s.DexLimits.WithdrawableToken1.ExpandsTo),
			ExpandDuration: new(big.Int).Set(s.DexLimits.WithdrawableToken1.ExpandDuration),
		},
	}

	syncTimestamp := s.SyncTimestamp

	tokenAmountIn, err := swapOut(swap0To1, param.TokenAmountOut.Amount, collateralReserves, debtReserves,
		int64(tokenInDecimals), int64(tokenOutDecimals), dexLimits, syncTimestamp)
	if err != nil {
		return nil, err
	}

	// fee is applied on token in
	fee := new(big.Int).Mul(tokenAmountIn, s.Pool.Info.SwapFee)
	fee = new(big.Int).Div(fee, big.NewInt(Fee100PercentPrecision))

	amountInAfterFee := new(big.Int).Add(tokenAmountIn, fee)

	return &poolpkg.CalcAmountInResult{
		TokenAmountIn: &poolpkg.TokenAmount{Token: param.TokenIn, Amount: amountInAfterFee},
		Fee:           &poolpkg.TokenAmount{Token: param.TokenIn, Amount: fee},
		Gas:           defaultGas.Swap,
		SwapInfo: SwapInfo{
			HasNative:             s.HasNative,
			NewCollateralReserves: collateralReserves,
			NewDebtReserves:       debtReserves,
			NewDexLimits:          dexLimits,
		},
	}, nil
}

func (t *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = input.Amount
	var outputAmount = output.Amount

	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
		}
	}

	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		t.CollateralReserves = swapInfo.NewCollateralReserves
		t.DebtReserves = swapInfo.NewDebtReserves
		// Note: limits are updated, but are likely off for the input token until newly fetched.
		// Erring on the cautious side with too tight limits to avoid potential reverts.
		// See Comment Ref #4327563287
		t.DexLimits = swapInfo.NewDexLimits
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
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
	return new(big.Int).Div(numerator, denominator)
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
	return new(big.Int).Div(numerator, denominator)
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
 * @param {number} syncTime - timestamp in seconds when the limits were synced
 * @returns {number} amountOut - The calculated output amount.
 * @returns {error} - An error object if the operation fails.
 */
func swapInAdjusted(swap0To1 bool, amountToSwap *big.Int, colReserves CollateralReserves, debtReserves DebtReserves, outDecimals int64, currentLimits DexLimits, syncTime int64) (*big.Int, error) {
	var (
		colIReserveIn, colIReserveOut, debtIReserveIn, debtIReserveOut *big.Int
		colReserveOut, debtReserveOut                                  *big.Int
		borrowable, withdrawable                                       *big.Int
	)

	if swap0To1 {
		colReserveOut = colReserves.Token1RealReserves
		colIReserveIn = colReserves.Token0ImaginaryReserves
		colIReserveOut = colReserves.Token1ImaginaryReserves
		debtReserveOut = debtReserves.Token1RealReserves
		debtIReserveIn = debtReserves.Token0ImaginaryReserves
		debtIReserveOut = debtReserves.Token1ImaginaryReserves
		borrowable = getExpandedLimit(syncTime, currentLimits.BorrowableToken1)
		withdrawable = getExpandedLimit(syncTime, currentLimits.WithdrawableToken1)
	} else {
		colReserveOut = colReserves.Token0RealReserves
		colIReserveIn = colReserves.Token1ImaginaryReserves
		colIReserveOut = colReserves.Token0ImaginaryReserves
		debtReserveOut = debtReserves.Token0RealReserves
		debtIReserveIn = debtReserves.Token1ImaginaryReserves
		debtIReserveOut = debtReserves.Token0ImaginaryReserves
		borrowable = getExpandedLimit(syncTime, currentLimits.BorrowableToken0)
		withdrawable = getExpandedLimit(syncTime, currentLimits.WithdrawableToken0)
	}

	// bring borrowable and withdrawable from token decimals to 1e12 decimals, same as amounts
	var factor *big.Int
	if DexAmountsDecimals > outDecimals {
		factor = new(big.Int).Exp(bI10, big.NewInt(DexAmountsDecimals-outDecimals), nil)
		borrowable = new(big.Int).Mul(borrowable, factor)
		withdrawable = new(big.Int).Mul(withdrawable, factor)
	} else {
		factor = new(big.Int).Exp(bI10, big.NewInt(outDecimals-DexAmountsDecimals), nil)
		borrowable = new(big.Int).Div(borrowable, factor)
		withdrawable = new(big.Int).Div(withdrawable, factor)
	}

	// Check if all reserves of collateral pool are greater than 0
	colPoolEnabled := colReserves.Token0RealReserves.Cmp(bignumber.ZeroBI) > 0 &&
		colReserves.Token1RealReserves.Cmp(bignumber.ZeroBI) > 0 &&
		colReserves.Token0ImaginaryReserves.Cmp(bignumber.ZeroBI) > 0 &&
		colReserves.Token1ImaginaryReserves.Cmp(bignumber.ZeroBI) > 0

	// Check if all reserves of debt pool are greater than 0
	debtPoolEnabled := debtReserves.Token0RealReserves.Cmp(bignumber.ZeroBI) > 0 &&
		debtReserves.Token1RealReserves.Cmp(bignumber.ZeroBI) > 0 &&
		debtReserves.Token0ImaginaryReserves.Cmp(bignumber.ZeroBI) > 0 &&
		debtReserves.Token1ImaginaryReserves.Cmp(bignumber.ZeroBI) > 0

	var a *big.Int
	if colPoolEnabled && debtPoolEnabled {
		a = swapRoutingIn(amountToSwap, colIReserveOut, colIReserveIn, debtIReserveOut, debtIReserveIn)
	} else if debtPoolEnabled {
		a = big.NewInt(-1) // Route from debt pool
	} else if colPoolEnabled {
		a = new(big.Int).Add(amountToSwap, big.NewInt(1)) // Route from collateral pool
	} else {
		return nil, errors.New("no pools are enabled")
	}

	amountInCollateral := big.NewInt(0)
	amountOutCollateral := big.NewInt(0)
	amountInDebt := big.NewInt(0)
	amountOutDebt := big.NewInt(0)

	triggerUpdateDebtReserves := false
	triggerUpdateColReserves := false

	if a.Cmp(bignumber.ZeroBI) <= 0 {
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
		return nil, errors.New(ErrInsufficientReserve.Error())
	}

	if amountOutCollateral.Cmp(colReserveOut) > 0 {
		return nil, errors.New(ErrInsufficientReserve.Error())
	}

	if amountOutDebt.Cmp(borrowable) > 0 {
		return nil, errors.New(ErrInsufficientBorrowable.Error())
	}

	if amountOutCollateral.Cmp(withdrawable) > 0 {
		return nil, errors.New(ErrInsufficientWithdrawable.Error())
	}

	oldPrice := big.NewInt(0)
	newPrice := big.NewInt(0)
	priceDiff := big.NewInt(0)
	maxPriceDiff := big.NewInt(0)
	// from whatever pool higher amount of swap is routing we are taking that as final price, does not matter much because both pools final price should be same
	if amountInCollateral.Cmp(amountInDebt) > 0 {
		// new pool price from col pool
		if swap0To1 {
			oldPrice.Div(oldPrice.Mul(colIReserveOut, bI1e27), colIReserveIn)
			newPrice.Div(newPrice.Mul(new(big.Int).Sub(colIReserveOut, amountOutCollateral), bI1e27), new(big.Int).Add(colIReserveIn, amountInCollateral))
		} else {
			oldPrice.Div(oldPrice.Mul(colIReserveIn, bI1e27), colIReserveOut)
			newPrice.Div(newPrice.Mul(new(big.Int).Add(colIReserveIn, amountInCollateral), bI1e27), new(big.Int).Sub(colIReserveOut, amountOutCollateral))
		}
	} else {
		// new pool price from debt pool
		if swap0To1 {
			oldPrice.Div(oldPrice.Mul(debtIReserveOut, bI1e27), debtIReserveIn)
			newPrice.Div(newPrice.Mul(new(big.Int).Sub(debtIReserveOut, amountOutDebt), bI1e27), new(big.Int).Add(debtIReserveIn, amountInDebt))
		} else {
			oldPrice.Div(oldPrice.Mul(debtIReserveIn, bI1e27), debtIReserveOut)
			newPrice.Div(newPrice.Mul(new(big.Int).Add(debtIReserveIn, amountInDebt), bI1e27), new(big.Int).Sub(debtIReserveOut, amountOutDebt))
		}
	}
	priceDiff.Abs(priceDiff.Sub(oldPrice, newPrice))
	maxPriceDiff.Div(maxPriceDiff.Mul(oldPrice, big.NewInt(MaxPriceDiff)), bI100)
	if priceDiff.Cmp(maxPriceDiff) > 0 {
		// if price diff is > 5% then swap would revert.
		return nil, errors.New(ErrInsufficientMaxPrice.Error())
	}

	if triggerUpdateColReserves && triggerUpdateDebtReserves {
		updateBothReservesAndLimits(swap0To1, amountInCollateral, amountOutCollateral, amountInDebt, amountOutDebt, colReserves, debtReserves, currentLimits)
	} else if triggerUpdateColReserves {
		updateCollateralReservesAndLimits(swap0To1, amountToSwap, amountOutCollateral, colReserves, currentLimits)
	} else if triggerUpdateDebtReserves {
		updateDebtReservesAndLimits(swap0To1, amountToSwap, amountOutDebt, debtReserves, currentLimits)
	}

	return new(big.Int).Add(amountOutCollateral, amountOutDebt), nil
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
	syncTime int64,
) (*big.Int, error) {
	var amountInAdjusted *big.Int

	if inDecimals > DexAmountsDecimals {
		amountInAdjusted = new(big.Int).Div(amountIn, new(big.Int).Exp(bI10, big.NewInt(inDecimals-DexAmountsDecimals), nil))
	} else {
		amountInAdjusted = new(big.Int).Mul(amountIn, new(big.Int).Exp(bI10, big.NewInt(DexAmountsDecimals-inDecimals), nil))
	}

	amountOut, err := swapInAdjusted(swap0To1, amountInAdjusted, colReserves, debtReserves, outDecimals, currentLimits, syncTime)

	if err != nil {
		return nil, err
	}

	if outDecimals > DexAmountsDecimals {
		amountOut = new(big.Int).Mul(amountOut, new(big.Int).Exp(bI10, big.NewInt(outDecimals-DexAmountsDecimals), nil))
	} else {
		amountOut = new(big.Int).Div(amountOut, new(big.Int).Exp(bI10, big.NewInt(DexAmountsDecimals-outDecimals), nil))
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
	syncTime int64,
) (*big.Int, error) {
	var (
		colIReserveIn, colIReserveOut, debtIReserveIn, debtIReserveOut *big.Int
		colReserveOut, debtReserveOut                                  *big.Int
		borrowable, withdrawable                                       *big.Int
	)

	if swap0To1 {
		colReserveOut = colReserves.Token1RealReserves
		colIReserveIn = colReserves.Token0ImaginaryReserves
		colIReserveOut = colReserves.Token1ImaginaryReserves
		debtReserveOut = debtReserves.Token1RealReserves
		debtIReserveIn = debtReserves.Token0ImaginaryReserves
		debtIReserveOut = debtReserves.Token1ImaginaryReserves
		borrowable = getExpandedLimit(syncTime, currentLimits.BorrowableToken1)
		withdrawable = getExpandedLimit(syncTime, currentLimits.WithdrawableToken1)
	} else {
		colReserveOut = colReserves.Token0RealReserves
		colIReserveIn = colReserves.Token1ImaginaryReserves
		colIReserveOut = colReserves.Token0ImaginaryReserves
		debtReserveOut = debtReserves.Token0RealReserves
		debtIReserveIn = debtReserves.Token1ImaginaryReserves
		debtIReserveOut = debtReserves.Token0ImaginaryReserves
		borrowable = getExpandedLimit(syncTime, currentLimits.BorrowableToken0)
		withdrawable = getExpandedLimit(syncTime, currentLimits.WithdrawableToken0)
	}

	// bring borrowable and withdrawable from token decimals to 1e12 decimals, same as amounts
	var factor *big.Int
	if DexAmountsDecimals > outDecimals {
		factor = new(big.Int).Exp(bI10, big.NewInt(DexAmountsDecimals-outDecimals), nil)
		borrowable = new(big.Int).Mul(borrowable, factor)
		withdrawable = new(big.Int).Mul(withdrawable, factor)
	} else {
		factor = new(big.Int).Exp(bI10, big.NewInt(outDecimals-DexAmountsDecimals), nil)
		borrowable = new(big.Int).Div(borrowable, factor)
		withdrawable = new(big.Int).Div(withdrawable, factor)
	}

	// Check if all reserves of collateral pool are greater than 0
	colPoolEnabled := colReserves.Token0RealReserves.Cmp(bignumber.ZeroBI) > 0 &&
		colReserves.Token1RealReserves.Cmp(bignumber.ZeroBI) > 0 &&
		colReserves.Token0ImaginaryReserves.Cmp(bignumber.ZeroBI) > 0 &&
		colReserves.Token1ImaginaryReserves.Cmp(bignumber.ZeroBI) > 0

	// Check if all reserves of debt pool are greater than 0
	debtPoolEnabled := debtReserves.Token0RealReserves.Cmp(bignumber.ZeroBI) > 0 &&
		debtReserves.Token1RealReserves.Cmp(bignumber.ZeroBI) > 0 &&
		debtReserves.Token0ImaginaryReserves.Cmp(bignumber.ZeroBI) > 0 &&
		debtReserves.Token1ImaginaryReserves.Cmp(bignumber.ZeroBI) > 0

	var a *big.Int
	if colPoolEnabled && debtPoolEnabled {
		a = swapRoutingOut(amountOut, colIReserveIn, colIReserveOut, debtIReserveIn, debtIReserveOut)
	} else if debtPoolEnabled {
		a = big.NewInt(-1) // Route from debt pool
	} else if colPoolEnabled {
		a = new(big.Int).Add(amountOut, big.NewInt(1)) // Route from collateral pool
	} else {
		return nil, errors.New("no pools are enabled")
	}

	amountInCollateral := big.NewInt(0)
	amountOutCollateral := big.NewInt(0)
	amountInDebt := big.NewInt(0)
	amountOutDebt := big.NewInt(0)

	triggerUpdateDebtReserves := false
	triggerUpdateColReserves := false

	if a.Cmp(bignumber.ZeroBI) <= 0 {
		// Entire trade routes through debt pool
		amountOutDebt = amountOut
		amountInDebt = getAmountIn(amountOut, debtIReserveIn, debtIReserveOut)

		if amountOut.Cmp(debtReserveOut) > 0 {
			return nil, errors.New(ErrInsufficientReserve.Error())
		}

		triggerUpdateDebtReserves = true
	} else if a.Cmp(amountOut) >= 0 {
		// Entire trade routes through collateral pool
		amountOutCollateral = amountOut
		amountInCollateral = getAmountIn(amountOut, colIReserveIn, colIReserveOut)
		if amountOut.Cmp(colReserveOut) > 0 {
			return nil, errors.New(ErrInsufficientReserve.Error())
		}

		triggerUpdateColReserves = true
	} else {
		// Trade routes through both pools
		amountOutCollateral = a
		amountInCollateral = getAmountIn(a, colIReserveIn, colIReserveOut)
		amountOutDebt.Sub(amountOut, a)
		amountInDebt = getAmountIn(amountOutDebt, debtIReserveIn, debtIReserveOut)

		if amountOutDebt.Cmp(debtReserveOut) > 0 || amountOutCollateral.Cmp(colReserveOut) > 0 {
			return nil, errors.New(ErrInsufficientReserve.Error())
		}

		triggerUpdateDebtReserves = true
		triggerUpdateColReserves = true
	}

	if amountOutDebt.Cmp(borrowable) > 0 {
		return nil, errors.New(ErrInsufficientBorrowable.Error())
	}

	if amountOutCollateral.Cmp(withdrawable) > 0 {
		return nil, errors.New(ErrInsufficientWithdrawable.Error())
	}

	oldPrice := big.NewInt(0)
	newPrice := big.NewInt(0)
	priceDiff := big.NewInt(0)
	maxPriceDiff := big.NewInt(0)
	// from whatever pool higher amount of swap is routing we are taking that as final price, does not matter much because both pools final price should be same
	if amountOutCollateral.Cmp(amountOutDebt) > 0 {
		// new pool price from col pool
		if swap0To1 {
			oldPrice.Div(oldPrice.Mul(colIReserveOut, bI1e27), colIReserveIn)
			newPrice.Div(newPrice.Mul(new(big.Int).Sub(colIReserveOut, amountOutCollateral), bI1e27), new(big.Int).Add(colIReserveIn, amountInCollateral))
		} else {
			oldPrice.Div(oldPrice.Mul(colIReserveIn, bI1e27), colIReserveOut)
			newPrice.Div(newPrice.Mul(new(big.Int).Add(colIReserveIn, amountInCollateral), bI1e27), new(big.Int).Sub(colIReserveOut, amountOutCollateral))
		}
	} else {
		// new pool price from debt pool
		if swap0To1 {
			oldPrice.Div(oldPrice.Mul(debtIReserveOut, bI1e27), debtIReserveIn)
			newPrice.Div(newPrice.Mul(new(big.Int).Sub(debtIReserveOut, amountOutDebt), bI1e27), new(big.Int).Add(debtIReserveIn, amountInDebt))
		} else {
			oldPrice.Div(oldPrice.Mul(debtIReserveIn, bI1e27), debtIReserveOut)
			newPrice.Div(newPrice.Mul(new(big.Int).Add(debtIReserveIn, amountInDebt), bI1e27), new(big.Int).Sub(debtIReserveOut, amountOutDebt))
		}
	}
	priceDiff.Abs(priceDiff.Sub(oldPrice, newPrice))
	maxPriceDiff.Div(maxPriceDiff.Mul(oldPrice, big.NewInt(MaxPriceDiff)), bI100)
	if priceDiff.Cmp(maxPriceDiff) > 0 {
		// if price diff is > 5% then swap would revert.
		return nil, errors.New(ErrInsufficientMaxPrice.Error())
	}

	if triggerUpdateColReserves && triggerUpdateDebtReserves {
		updateBothReservesAndLimits(swap0To1, amountInCollateral, amountOutCollateral, amountInDebt, amountOutDebt, colReserves, debtReserves, currentLimits)
	} else if triggerUpdateColReserves {
		updateCollateralReservesAndLimits(swap0To1, amountInCollateral, amountOutCollateral, colReserves, currentLimits)
	} else if triggerUpdateDebtReserves {
		updateDebtReservesAndLimits(swap0To1, amountInDebt, amountOutDebt, debtReserves, currentLimits)
	}

	return new(big.Int).Add(amountInCollateral, amountInDebt), nil
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
	syncTime int64,
) (*big.Int, error) {
	var amountOutAdjusted *big.Int

	if outDecimals > DexAmountsDecimals {
		amountOutAdjusted = new(big.Int).Div(amountOut,
			new(big.Int).Exp(bI10, big.NewInt(outDecimals-DexAmountsDecimals), nil))
	} else {
		amountOutAdjusted = new(big.Int).Mul(amountOut,
			new(big.Int).Exp(bI10, big.NewInt(DexAmountsDecimals-outDecimals), nil))
	}

	amountIn, err := swapOutAdjusted(swap0To1, amountOutAdjusted, colReserves, debtReserves, outDecimals, currentLimits, syncTime)

	if err != nil {
		return nil, err
	}

	if inDecimals > DexAmountsDecimals {
		amountIn = new(big.Int).Mul(amountIn,
			new(big.Int).Exp(bI10, big.NewInt(inDecimals-DexAmountsDecimals), nil))
	} else {
		amountIn = new(big.Int).Div(amountIn,
			new(big.Int).Exp(bI10, big.NewInt(DexAmountsDecimals-inDecimals), nil))
	}

	return amountIn, nil
}

// Calculates the currently available swappable amount for a token limit considering expansion since last syncTime.
func getExpandedLimit(syncTime int64, limit TokenLimit) *big.Int {
	currentTime := time.Now().Unix() // get current time in seconds
	elapsedTime := currentTime - syncTime

	expandedAmount := big.NewInt(0).Set(limit.Available)

	if elapsedTime < 10 {
		// if almost no time has elapsed, return available amount
		return expandedAmount
	}

	if elapsedTime >= limit.ExpandDuration.Int64() {
		// if duration has passed, return max amount
		expandedAmount = limit.ExpandsTo
		return expandedAmount
	}

	expandedAmount = new(big.Int).Sub(limit.ExpandsTo, limit.Available)
	expandedAmount.Mul(expandedAmount, big.NewInt(elapsedTime))
	expandedAmount.Div(expandedAmount, limit.ExpandDuration)
	expandedAmount.Add(expandedAmount, limit.Available)

	return expandedAmount
}

func updateDebtReservesAndLimits(swap0To1 bool, amountIn, amountOut *big.Int, debtReserves DebtReserves, limits DexLimits) {
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

func updateCollateralReservesAndLimits(swap0To1 bool, amountIn, amountOut *big.Int, colReserves CollateralReserves, limits DexLimits) {
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

func updateBothReservesAndLimits(swap0To1 bool, amountInCollateral, amountOutCollateral, amountInDebt, amountOutDebt *big.Int,
	colReserves CollateralReserves, debtReserves DebtReserves, limits DexLimits) {
	if swap0To1 {
		colReserves.Token1RealReserves.Sub(colReserves.Token1RealReserves, amountOutCollateral)
		colReserves.Token1ImaginaryReserves.Sub(colReserves.Token1ImaginaryReserves, amountOutCollateral)
		colReserves.Token0RealReserves.Add(colReserves.Token0RealReserves, amountInCollateral)
		colReserves.Token0ImaginaryReserves.Add(colReserves.Token0ImaginaryReserves, amountInCollateral)

		debtReserves.Token1RealReserves.Sub(debtReserves.Token1RealReserves, amountOutDebt)
		debtReserves.Token1ImaginaryReserves.Sub(debtReserves.Token1ImaginaryReserves, amountOutDebt)
		debtReserves.Token0RealReserves.Add(debtReserves.Token0RealReserves, amountInDebt)
		debtReserves.Token0ImaginaryReserves.Add(debtReserves.Token0ImaginaryReserves, amountInDebt)

		limits.BorrowableToken1.Available.Sub(limits.BorrowableToken1.Available, amountOutDebt)
		limits.BorrowableToken1.ExpandsTo.Sub(limits.BorrowableToken1.ExpandsTo, amountOutDebt)
		limits.WithdrawableToken1.Available.Sub(limits.WithdrawableToken1.Available, amountOutCollateral)
		limits.WithdrawableToken1.ExpandsTo.Sub(limits.WithdrawableToken1.ExpandsTo, amountOutCollateral)
	} else {
		colReserves.Token1RealReserves.Add(colReserves.Token1RealReserves, amountInCollateral)
		colReserves.Token1ImaginaryReserves.Add(colReserves.Token1ImaginaryReserves, amountInCollateral)
		colReserves.Token0RealReserves.Sub(colReserves.Token0RealReserves, amountOutCollateral)
		colReserves.Token0ImaginaryReserves.Sub(colReserves.Token0ImaginaryReserves, amountOutCollateral)

		debtReserves.Token1RealReserves.Add(debtReserves.Token1RealReserves, amountInDebt)
		debtReserves.Token1ImaginaryReserves.Add(debtReserves.Token1ImaginaryReserves, amountInDebt)
		debtReserves.Token0RealReserves.Sub(debtReserves.Token0RealReserves, amountOutDebt)
		debtReserves.Token0ImaginaryReserves.Sub(debtReserves.Token0ImaginaryReserves, amountOutDebt)

		limits.BorrowableToken0.Available.Sub(limits.BorrowableToken0.Available, amountOutDebt)
		limits.BorrowableToken0.ExpandsTo.Sub(limits.BorrowableToken0.ExpandsTo, amountOutDebt)
		limits.WithdrawableToken0.Available.Sub(limits.WithdrawableToken0.Available, amountOutCollateral)
		limits.WithdrawableToken0.ExpandsTo.Sub(limits.WithdrawableToken0.ExpandsTo, amountOutCollateral)
	}
}
