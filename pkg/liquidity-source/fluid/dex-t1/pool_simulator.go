package dexT1

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/samber/lo"
)

var (
	ErrInvalidAmountIn  = errors.New("invalid amountIn")
	ErrInvalidAmountOut = errors.New("invalid amount out")
)

type PoolSimulator struct {
	poolpkg.Pool

	DexReservesResolver string
	CollateralReserves  CollateralReserves
	DebtReserves        DebtReserves

	Token0Decimals uint8
	Token1Decimals uint8
}

var (
	defaultGas = Gas{Swap: 150000}
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
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
			SwapFee:     fee,
		}},
		CollateralReserves:  extra.CollateralReserves,
		DebtReserves:        extra.DebtReserves,
		Token0Decimals:      entityPool.Tokens[0].Decimals,
		Token1Decimals:      entityPool.Tokens[1].Decimals,
		DexReservesResolver: staticExtra.DexReservesResolver,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
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

	_, tokenAmountOut, err := swapIn(swap0To1, amountInAfterFee, s.CollateralReserves, s.DebtReserves, int64(tokenInDecimals), int64(tokenOutDecimals))
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: param.TokenOut, Amount: tokenAmountOut},
		Fee:            &poolpkg.TokenAmount{Token: param.TokenAmountIn.Token, Amount: fee},
		Gas:            defaultGas.Swap,
		SwapInfo: StaticExtra{
			DexReservesResolver: s.DexReservesResolver,
		},
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param poolpkg.CalcAmountInParams) (*poolpkg.CalcAmountInResult, error) {
	if param.TokenAmountOut.Amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInvalidAmountOut
	}

	swap0To1 := param.TokenAmountOut.Token == s.Info.Tokens[1]

	var tokenInDecimals, tokenOutDecimals uint8
	if swap0To1 {
		tokenInDecimals = s.Token0Decimals
		tokenOutDecimals = s.Token1Decimals
	} else {
		tokenOutDecimals = s.Token0Decimals
		tokenInDecimals = s.Token1Decimals
	}

	tokenAmountIn, _, err := swapOut(swap0To1, param.TokenAmountOut.Amount, s.CollateralReserves, s.DebtReserves, int64(tokenInDecimals), int64(tokenOutDecimals))
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
		SwapInfo: StaticExtra{
			DexReservesResolver: s.DexReservesResolver,
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
	bI1e18 := new(big.Int)
	bI1e18.SetString(String1e18, 10) // 1e18

	xyProduct := new(big.Int).Mul(x, y)
	xyProduct = new(big.Int).Mul(xyProduct, bI1e18)
	xyRoot := new(big.Int).Sqrt(xyProduct)

	x2y2Product := new(big.Int).Mul(x2, y2)
	x2y2Product = new(big.Int).Mul(x2y2Product, bI1e18)
	x2y2Root := new(big.Int).Sqrt(x2y2Product)

	y2xyRoot := new(big.Int).Mul(y2, xyRoot)
	txyRoot := new(big.Int).Mul(t, xyRoot)
	yx2y2Root := new(big.Int).Mul(y, x2y2Root)
	sum := new(big.Int).Add(y2xyRoot, txyRoot)
	sum = new(big.Int).Sub(sum, yx2y2Root)
	denominator := new(big.Int).Add(xyRoot, x2y2Root)
	a := new(big.Int).Div(sum, denominator)
	return a
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
	bI1e18 := new(big.Int)
	bI1e18.SetString(String1e18, 10) // 1e18

	xyProduct := new(big.Int).Mul(x, y)
	xyProduct = new(big.Int).Mul(xyProduct, bI1e18)
	xyRoot := new(big.Int).Sqrt(xyProduct)

	x2y2Product := new(big.Int).Mul(x2, y2)
	x2y2Product = new(big.Int).Mul(x2y2Product, bI1e18)
	x2y2Root := new(big.Int).Sqrt(x2y2Product)

	txyRoot := new(big.Int).Mul(t, xyRoot)
	yx2y2Root := new(big.Int).Mul(y, x2y2Root)
	y2xyRoot := new(big.Int).Mul(y2, xyRoot)
	sum := new(big.Int).Add(txyRoot, yx2y2Root)
	sum = new(big.Int).Sub(sum, y2xyRoot)
	denominator := new(big.Int).Add(xyRoot, x2y2Root)
	a := new(big.Int).Div(sum, denominator)
	return a
}

/**
 * Calculates the output amount for a given input amount in a swap operation.
 * @param {boolean} swap0To1 - Direction of the swap. True if swapping token0 for token1, false otherwise.
 * @param {number} amountToSwap - The amount of input token to be swapped scaled to 1e12.
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
 * @returns {Object} An object containing the input amount and the calculated output amount.
 * @returns {number} amountIn - The input amount.
 * @returns {number} amountOut - The calculated output amount.
 * @returns {error} - An error object if the operation fails.
 */
func swapInAdjusted(swap0To1 bool, amountToSwap *big.Int, colReserves CollateralReserves, debtReserves DebtReserves) (*big.Int, *big.Int, error) {
	var (
		colIReserveIn, colIReserveOut, debtIReserveIn, debtIReserveOut *big.Int
	)

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
		return nil, nil, errors.New("no pools are enabled")
	}

	var amountOutCollateral, amountOutDebt *big.Int = bignumber.ZeroBI, bignumber.ZeroBI
	if a.Cmp(bignumber.ZeroBI) <= 0 {
		// Entire trade routes through debt pool
		amountOutDebt = getAmountOut(amountToSwap, debtIReserveIn, debtIReserveOut)
	} else if a.Cmp(amountToSwap) >= 0 {
		// Entire trade routes through collateral pool
		amountOutCollateral = getAmountOut(amountToSwap, colIReserveIn, colIReserveOut)
	} else {
		// Trade routes through both pools
		amountOutCollateral = getAmountOut(a, colIReserveIn, colIReserveOut)
		amountOutDebt = getAmountOut(new(big.Int).Sub(amountToSwap, a), debtIReserveIn, debtIReserveOut)
	}

	return amountToSwap, new(big.Int).Add(amountOutCollateral, amountOutDebt), nil
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
 * @returns {number} amountIn - The input amount.
 * @returns {number} amountOut - The calculated output amount scaled to token decimals
 * @returns {error} - An error object if the operation fails.
 */
func swapIn(
	swap0To1 bool,
	amountIn *big.Int,
	colReserves CollateralReserves,
	debtReserves DebtReserves,
	inDecimals int64,
	outDecimals int64,
) (*big.Int, *big.Int, error) {
	var amountInAdjusted *big.Int

	if inDecimals > DexAmountsDecimals {
		amountInAdjusted = new(big.Int).Div(amountIn, new(big.Int).Exp(big.NewInt(10), big.NewInt(inDecimals-DexAmountsDecimals), nil))
	} else {
		amountInAdjusted = new(big.Int).Mul(amountIn, new(big.Int).Exp(big.NewInt(10), big.NewInt(DexAmountsDecimals-inDecimals), nil))
	}

	_, amountOut, err := swapInAdjusted(swap0To1, amountInAdjusted, colReserves, debtReserves)

	if err != nil {
		return nil, nil, err
	}

	if outDecimals > DexAmountsDecimals {
		amountOut = new(big.Int).Mul(amountOut, new(big.Int).Exp(big.NewInt(10), big.NewInt(outDecimals-DexAmountsDecimals), nil))
	} else {
		amountOut = new(big.Int).Div(amountOut, new(big.Int).Exp(big.NewInt(10), big.NewInt(DexAmountsDecimals-outDecimals), nil))
	}

	return amountIn, amountOut, nil
}

/**
 * Calculates the input amount for a given output amount in a swap operation.
 * @param {boolean} swap0to1 - Direction of the swap. True if swapping token0 for token1, false otherwise.
 * @param {number} amountOut - The amount of output token to be swapped scaled to 1e12.
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
 * @returns {number} amountIn - The calculated input amount required for the swap.
 * @returns {number} amountOut - The specified output amount of the swap.
 * @returns {error} - An error object if the operation fails.
 */
func swapOutAdjusted(swap0to1 bool, amountOut *big.Int, colReserves CollateralReserves, debtReserves DebtReserves) (*big.Int, *big.Int, error) {
	var (
		colIReserveIn, colIReserveOut, debtIReserveIn, debtIReserveOut *big.Int
	)

	if swap0to1 {
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
		return nil, nil, errors.New("no pools are enabled")
	}

	var amountInCollateral, amountInDebt *big.Int = bignumber.ZeroBI, bignumber.ZeroBI
	if a.Cmp(bignumber.ZeroBI) <= 0 {
		// Entire trade routes through debt pool
		amountInDebt = getAmountIn(amountOut, debtIReserveIn, debtIReserveOut)
	} else if a.Cmp(amountOut) >= 0 {
		// Entire trade routes through collateral pool
		amountInCollateral = getAmountIn(amountOut, colIReserveIn, colIReserveOut)
	} else {
		// Trade routes through both pools
		amountInCollateral = getAmountIn(a, colIReserveIn, colIReserveOut)
		amountInDebt = getAmountIn(new(big.Int).Sub(amountOut, a), debtIReserveIn, debtIReserveOut)
	}

	return new(big.Int).Add(amountInCollateral, amountInDebt), amountOut, nil
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
 * @returns {number} amountIn - The calculated input amount required for the swap scaled to token decimals.
 * @returns {number} amountOut - The specified output amount of the swap.
 * @returns {error} - An error object if the operation fails.
 */
func swapOut(
	swap0To1 bool,
	amountOut *big.Int,
	colReserves CollateralReserves,
	debtReserves DebtReserves,
	inDecimals int64,
	outDecimals int64,
) (*big.Int, *big.Int, error) {
	var amountOutAdjusted *big.Int

	if outDecimals > DexAmountsDecimals {
		amountOutAdjusted = new(big.Int).Div(amountOut, new(big.Int).Exp(big.NewInt(10), big.NewInt(outDecimals-DexAmountsDecimals), nil))
	} else {
		amountOutAdjusted = new(big.Int).Mul(amountOut, new(big.Int).Exp(big.NewInt(10), big.NewInt(DexAmountsDecimals-outDecimals), nil))
	}

	amountIn, _, err := swapOutAdjusted(swap0To1, amountOutAdjusted, colReserves, debtReserves)

	if err != nil {
		return nil, nil, err
	}

	if inDecimals > DexAmountsDecimals {
		amountIn = new(big.Int).Mul(amountIn, new(big.Int).Exp(big.NewInt(10), big.NewInt(inDecimals-DexAmountsDecimals), nil))
	} else {
		amountIn = new(big.Int).Div(amountIn, new(big.Int).Exp(big.NewInt(10), big.NewInt(DexAmountsDecimals-inDecimals), nil))
	}

	return amountIn, amountOut, nil
}
