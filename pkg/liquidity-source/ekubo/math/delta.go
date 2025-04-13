package math

import (
	"fmt"
	"math/big"
)

func sortRatios(sqrtRatioA, sqrtRatioB *big.Int) (*big.Int, *big.Int) {
	if sqrtRatioA.Cmp(sqrtRatioB) == -1 {
		return sqrtRatioA, sqrtRatioB
	}
	return sqrtRatioB, sqrtRatioA
}

func amount0Delta(sqrtRatioA, sqrtRatioB, liquidity *big.Int, roundUp bool) (*big.Int, error) {
	lower, upper := sortRatios(sqrtRatioA, sqrtRatioB)

	if liquidity.Sign() == 0 || lower.Cmp(upper) == 0 {
		return new(big.Int), nil
	}

	result0, err := muldiv(
		new(big.Int).Sub(upper, lower),
		new(big.Int).Lsh(liquidity, 128),
		upper,
		roundUp,
	)
	if err != nil {
		return nil, fmt.Errorf("muldiv error: %w", err)
	}

	result, remainder := result0.DivMod(
		result0,
		lower,
		new(big.Int),
	)

	if roundUp && remainder.Sign() != 0 {
		result.Add(result, One)
	}

	if result.Cmp(TwoPow128) != -1 {
		return nil, ErrOverflow
	}

	return result, nil
}

func amount1Delta(sqrtRatioA, sqrtRatioB, liquidity *big.Int, roundUp bool) (*big.Int, error) {
	lower, upper := sortRatios(sqrtRatioA, sqrtRatioB)

	if liquidity.Sign() == 0 || lower.Cmp(upper) == 0 {
		return new(big.Int), nil
	}

	result, err := muldiv(
		liquidity,
		new(big.Int).Sub(upper, lower),
		TwoPow128,
		roundUp,
	)
	if err != nil {
		return nil, fmt.Errorf("muldiv error: %w", err)
	}

	if result.Cmp(TwoPow128) != -1 {
		return nil, ErrOverflow
	}

	return result, nil
}
