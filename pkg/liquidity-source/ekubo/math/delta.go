package math

import (
	"math/big"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func sortRatios(sqrtRatioA, sqrtRatioB *big.Int) (*big.Int, *big.Int) {
	if sqrtRatioA.Cmp(sqrtRatioB) == -1 {
		return sqrtRatioA, sqrtRatioB
	}
	return sqrtRatioB, sqrtRatioA
}

func amount0Delta(sqrtRatioA, sqrtRatioB, liquidity *big.Int, roundUp bool) (*big.Int, error) {
	if liquidity.Sign() == 0 || sqrtRatioA.Cmp(sqrtRatioB) == 0 {
		return new(big.Int), nil
	}

	lower, upper := sortRatios(sqrtRatioA, sqrtRatioB)

	result0, err := mulDivOverflow(
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

	if result.Cmp(bignum.MAX_UINT_128) > 0 {
		return nil, ErrAmount0DeltaOverflow
	}

	return result, nil
}

func amount1Delta(sqrtRatioA, sqrtRatioB, liquidity *big.Int, roundUp bool) (*big.Int, error) {
	if liquidity.Sign() == 0 || sqrtRatioA.Cmp(sqrtRatioB) == 0 {
		return new(big.Int), nil
	}

	lower, upper := sortRatios(sqrtRatioA, sqrtRatioB)

	result, err := mulDivOverflow(
		liquidity,
		new(big.Int).Sub(upper, lower),
		TwoPow128,
		roundUp,
	)
	if err != nil {
		return nil, err
	}

	if result.Cmp(bignum.MAX_UINT_128) > 0 {
		return nil, ErrAmount1DeltaOverflow
	}

	return result, nil
}
