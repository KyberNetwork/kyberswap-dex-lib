package math

import (
	"math/big"
)

func sortRatios(sqrtRatioA, sqrtRatioB *big.Int) (*big.Int, *big.Int) {
	if sqrtRatioA.Cmp(sqrtRatioB) == -1 {
		return sqrtRatioA, sqrtRatioB
	}
	return sqrtRatioB, sqrtRatioA
}

func Amount0Delta(sqrtRatioA, sqrtRatioB, liquidity *big.Int, roundUp bool) (*big.Int, error) {
	if liquidity.Sign() == 0 || sqrtRatioA.Cmp(sqrtRatioB) == 0 {
		return new(big.Int), nil
	}

	lower, upper := sortRatios(sqrtRatioA, sqrtRatioB)

	result0, err := MulDivOverflow(
		new(big.Int).Sub(upper, lower),
		new(big.Int).Lsh(liquidity, 128),
		upper,
		roundUp,
	)
	if err != nil {
		return nil, err
	}

	result, err := div(result0, lower, roundUp)
	if err != nil {
		return nil, err
	}

	if result.BitLen() > 128 {
		return nil, ErrAmount0DeltaOverflow
	}

	return result, nil
}

func Amount1Delta(sqrtRatioA, sqrtRatioB, liquidity *big.Int, roundUp bool) (*big.Int, error) {
	if liquidity.Sign() == 0 || sqrtRatioA.Cmp(sqrtRatioB) == 0 {
		return new(big.Int), nil
	}

	lower, upper := sortRatios(sqrtRatioA, sqrtRatioB)

	result, err := MulDivOverflow(
		liquidity,
		new(big.Int).Sub(upper, lower),
		TwoPow128,
		roundUp,
	)
	if err != nil {
		return nil, err
	}

	if result.BitLen() > 128 {
		return nil, ErrAmount1DeltaOverflow
	}

	return result, nil
}
