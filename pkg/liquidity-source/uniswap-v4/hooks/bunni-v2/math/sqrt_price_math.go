package math

import (
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

func GetTickAtSqrtPrice(sqrtPriceX96 *uint256.Int) (int, error) {
	return v3Utils.GetTickAtSqrtRatioV2(sqrtPriceX96)
}

func GetSqrtPriceAtTick(tick int) (*uint256.Int, error) {
	var price uint256.Int
	err := v3Utils.GetSqrtRatioAtTickV2(tick, &price)
	return &price, err
}

func GetNextSqrtPriceFromInput(sqrtPX96, liquidity, amountIn *uint256.Int, zeroForOne bool) (*uint256.Int, error) {
	var price uint256.Int
	v3Utils.GetNextSqrtPriceFromInput(sqrtPX96, liquidity, amountIn, zeroForOne, &price)
	return &price, nil
}

func GetNextSqrtPriceFromOutput(sqrtPX96, liquidity, amountOut *uint256.Int, zeroForOne bool) (*uint256.Int, error) {
	var price uint256.Int
	v3Utils.GetNextSqrtPriceFromOutput(sqrtPX96, liquidity, amountOut, zeroForOne, &price)
	return &price, nil
}

func GetAmount0Delta(sqrtPriceAX96, sqrtPriceBX96, liquidity *uint256.Int, roundUp bool) (*uint256.Int, error) {
	var result uint256.Int
	err := v3Utils.GetAmount0DeltaV2(sqrtPriceAX96, sqrtPriceBX96, liquidity, roundUp, &result)
	return &result, err
}

func GetAmount1Delta(sqrtPriceAX96, sqrtPriceBX96, liquidity *uint256.Int, roundUp bool) (*uint256.Int, error) {
	var result uint256.Int
	err := v3Utils.GetAmount1DeltaV2(sqrtPriceAX96, sqrtPriceBX96, liquidity, roundUp, &result)
	return &result, err
}

// GetNextSqrtPriceFromAmount0RoundingUp gets the next sqrt price given a delta of currency0
// Always rounds up, because in the exact output case (increasing price) we need to move the price at least
// far enough to get the desired output amount, and in the exact input case (decreasing price) we need to move the
// price less in order to not send too much output.
// The most precise formula for this is liquidity * sqrtPX96 / (liquidity +- amount * sqrtPX96),
// if this is impossible because of overflow, we calculate liquidity / (liquidity / sqrtPX96 +- amount).
func GetNextSqrtPriceFromAmount0RoundingUp(sqrtPX96, liquidity, amount *uint256.Int, add bool) (*uint256.Int, error) {
	// we short circuit amount == 0 because the result is otherwise not guaranteed to equal the input price
	if amount.IsZero() {
		return sqrtPX96.Clone(), nil
	}

	// numerator1 = liquidity << FixedPoint96.RESOLUTION (Q96)
	var numerator1 uint256.Int
	numerator1.Lsh(liquidity, 96)

	if add {
		// product = amount * sqrtPX96
		var product uint256.Int
		product.Mul(amount, sqrtPX96)

		// Check if product / amount == sqrtPX96 (overflow check)
		var check uint256.Int
		check.Div(&product, amount)
		if check.Eq(sqrtPX96) {
			// denominator = numerator1 + product
			var denominator uint256.Int
			denominator.Add(&numerator1, &product)

			// Check if denominator >= numerator1 (overflow check)
			if denominator.Cmp(&numerator1) >= 0 {
				// always fits in 160 bits
				// return FullMath.mulDivRoundingUp(numerator1, sqrtPX96, denominator)
				return FullMulDivUp(&numerator1, sqrtPX96, &denominator)
			}
		}

		// denominator is checked for overflow
		// return UnsafeMath.divRoundingUp(numerator1, (numerator1 / sqrtPX96) + amount)
		var temp uint256.Int
		temp.Div(&numerator1, sqrtPX96)
		temp.Add(&temp, amount)

		res := check // reuse
		v3Utils.DivRoundingUp(&res, &numerator1, &temp)

		return &res, nil
	} else {
		// product = amount * sqrtPX96
		var product uint256.Int
		product.Mul(amount, sqrtPX96)

		// Check if product / amount != sqrtPX96 or numerator1 <= product (overflow/underflow check)
		var check uint256.Int
		check.Div(&product, amount)
		if !check.Eq(sqrtPX96) || numerator1.Cmp(&product) <= 0 {
			return nil, ErrOverflow
		}

		// denominator = numerator1 - product
		denominator := check // reuse
		denominator.Sub(&numerator1, &product)

		// return FullMath.mulDivRoundingUp(numerator1, sqrtPX96, denominator)
		return FullMulDivUp(&numerator1, sqrtPX96, &denominator)
	}
}

// GetNextSqrtPriceFromAmount1RoundingDown gets the next sqrt price given a delta of currency1
// Always rounds down, because in the exact output case (decreasing price) we need to move the price at least
// far enough to get the desired output amount, and in the exact input case (increasing price) we need to move the
// price less in order to not send too much output.
// The most precise formula for this is liquidity * sqrtPX96 / (liquidity +- amount * sqrtPX96),
// if this is impossible because of overflow, we calculate liquidity / (liquidity / sqrtPX96 +- amount).
func GetNextSqrtPriceFromAmount1RoundingDown(sqrtPX96, liquidity, amount *uint256.Int, add bool) (*uint256.Int, error) {
	// we short circuit amount == 0 because the result is otherwise not guaranteed to equal the input price
	if amount.IsZero() {
		return sqrtPX96.Clone(), nil
	}

	// numerator1 = liquidity << FixedPoint96.RESOLUTION (Q96)
	var numerator1 uint256.Int
	numerator1.Lsh(liquidity, 96)

	if add {
		// product = amount * sqrtPX96
		var product uint256.Int
		product.Mul(amount, sqrtPX96)

		// Check if product / amount == sqrtPX96 (overflow check)
		var check uint256.Int
		check.Div(&product, amount)
		if check.Eq(sqrtPX96) {
			// denominator = numerator1 + product
			var denominator uint256.Int
			denominator.Add(&numerator1, &product)

			// Check if denominator >= numerator1 (overflow check)
			if denominator.Cmp(&numerator1) >= 0 {
				// always fits in 160 bits
				// return FullMath.mulDiv(numerator1, sqrtPX96, denominator) (rounding down)
				return FullMulDiv(&numerator1, sqrtPX96, &denominator)
			}
		}

		// denominator is checked for overflow
		// return UnsafeMath.div(numerator1, (numerator1 / sqrtPX96) + amount) (rounding down)
		temp := check // reuse
		temp.Div(&numerator1, sqrtPX96)
		temp.Add(&temp, amount)

		var result uint256.Int
		result.Div(&numerator1, &temp)
		return &result, nil
	} else {
		// product = amount * sqrtPX96
		var product uint256.Int
		product.Mul(amount, sqrtPX96)

		// Check if product / amount != sqrtPX96 or numerator1 <= product (overflow/underflow check)
		var check uint256.Int
		check.Div(&product, amount)
		if !check.Eq(sqrtPX96) || numerator1.Cmp(&product) <= 0 {
			return nil, ErrOverflow
		}

		// denominator = numerator1 - product
		denominator := check // reuse
		denominator.Sub(&numerator1, &product)

		// return FullMath.mulDiv(numerator1, sqrtPX96, denominator) (rounding down)
		return FullMulDiv(&numerator1, sqrtPX96, &denominator)
	}
}
