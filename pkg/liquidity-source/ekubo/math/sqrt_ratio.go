package math

import (
	"fmt"
	"math/big"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	bitMask    = bignum.NewBig("0xc00000000000000000000000")
	notBitMask = bignum.NewBig("0x3fffffffffffffffffffffff")
)

func FloatSqrtRatioToFixed(sqrtRatioFloat *big.Int) *big.Int {
	op1 := new(big.Int).And(sqrtRatioFloat, notBitMask)

	op2 := new(big.Int).And(sqrtRatioFloat, bitMask)
	op2.Rsh(op2, 89)
	op2.Add(bignum.Two, op2)

	return op1.Lsh(op1, uint(op2.Uint64()))
}

func nextSqrtRatioFromAmount0(sqrtRatio, liquidity, amount0 *big.Int) (*big.Int, error) {
	if amount0.Sign() == 0 {
		return new(big.Int).Set(sqrtRatio), nil
	}

	if liquidity.Sign() == 0 {
		return nil, ErrNoLiquidity
	}

	numerator1 := new(big.Int).Lsh(liquidity, 128)

	var (
		res *big.Int
		err error
	)

	if amount0.Sign() == -1 {
		amount0Abs := new(big.Int).Abs(amount0)

		product := amount0Abs.Mul(amount0Abs, sqrtRatio)
		if product.BitLen() > 256 {
			return nil, ErrOverflow
		}

		denominator := product.Sub(numerator1, product)
		if denominator.Sign() == -1 {
			return nil, ErrUnderflow
		}

		res, err = MulDivOverflow(numerator1, sqrtRatio, denominator, true)
	} else {
		denomP1 := new(big.Int).Div(numerator1, sqrtRatio)

		denom := denomP1.Add(denomP1, amount0)
		if denom.BitLen() > 256 {
			return nil, ErrOverflow
		}

		res, err = MulDivOverflow(numerator1, bignum.One, denom, true)
	}

	if err != nil {
		return nil, fmt.Errorf("muldiv error: %w", err)
	}

	return res, nil
}

func nextSqrtRatioFromAmount1(sqrtRatio, liquidity, amount1 *big.Int) (*big.Int, error) {
	if amount1.Sign() == 0 {
		return new(big.Int).Set(sqrtRatio), nil
	}

	if liquidity.Sign() == 0 {
		return nil, ErrNoLiquidity
	}

	amount1Abs := new(big.Int).Abs(amount1)
	roundUp := amount1.Sign() == -1

	quotient, err := MulDivOverflow(amount1Abs, TwoPow128, liquidity, roundUp)
	if err != nil {
		return nil, fmt.Errorf("muldiv error: %w", err)
	}

	var res *big.Int
	if amount1.Sign() == -1 {
		res = quotient.Sub(sqrtRatio, quotient)
		if res.Sign() == -1 {
			return nil, ErrUnderflow
		}
	} else {
		res = quotient.Add(sqrtRatio, quotient)
		if res.BitLen() > 256 {
			return nil, ErrOverflow
		}
	}

	return res, nil
}
