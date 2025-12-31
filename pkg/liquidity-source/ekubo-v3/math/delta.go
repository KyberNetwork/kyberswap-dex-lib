package math

import (
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func sortRatios(sqrtRatioA, sqrtRatioB *uint256.Int) (*uint256.Int, *uint256.Int) {
	if sqrtRatioA.Lt(sqrtRatioB) {
		return sqrtRatioA, sqrtRatioB
	}
	return sqrtRatioB, sqrtRatioA
}

func Amount0Delta(sqrtRatioA, sqrtRatioB, liquidity *uint256.Int, roundUp bool) (*uint256.Int, error) {
	if liquidity.IsZero() || sqrtRatioA.Eq(sqrtRatioB) {
		return big256.U0, nil
	}

	lower, upper := sortRatios(sqrtRatioA, sqrtRatioB)

	var tmp, tmp2 uint256.Int
	result0 := lo.Ternary(roundUp, big256.MulDivUp, big256.MulDivDown)(
		&tmp,
		tmp.Sub(upper, lower),
		tmp2.Lsh(liquidity, 128),
		upper,
	)

	result, err := div(result0, lower, roundUp)
	if err != nil {
		return nil, err
	} else if result.BitLen() > 128 {
		return nil, ErrAmount0DeltaOverflow
	}
	return result, nil
}

func Amount1Delta(sqrtRatioA, sqrtRatioB, liquidity *uint256.Int, roundUp bool) (*uint256.Int, error) {
	if liquidity.IsZero() || sqrtRatioA.Eq(sqrtRatioB) {
		return big256.U0, nil
	}

	lower, upper := sortRatios(sqrtRatioA, sqrtRatioB)

	var tmp uint256.Int
	result := lo.Ternary(roundUp, big256.MulDivUp, big256.MulDivDown)(
		&tmp,
		liquidity,
		tmp.Sub(upper, lower),
		big256.U2Pow128,
	)

	if result.BitLen() > 128 {
		return nil, ErrAmount1DeltaOverflow
	}
	return result, nil
}
