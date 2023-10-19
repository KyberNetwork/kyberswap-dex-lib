package sd59x18

import (
	"math/big"
)

func Log2(x SD59x18) (SD59x18, error) {
	xBI := Unwrap(x)
	if xBI.Cmp(bigint0) <= 0 {
		return nil, ErrMathSD59x18LogInputTooSmall
	}

	var sign *big.Int
	if xBI.Cmp(uUnit) >= 0 {
		sign = big.NewInt(1)
	} else {
		sign = big.NewInt(-1)
		xBI = new(big.Int).Div(uUnitSquared, xBI)
	}

	n := msb(new(big.Int).Div(xBI, uUnit))

	resultBI := new(big.Int).Mul(n, uUnit)

	y := new(big.Int).Rsh(xBI, uint(n.Uint64()))

	if y.Cmp(uUnit) == 0 {
		return Wrap(new(big.Int).Mul(resultBI, sign)), nil
	}

	doubleUnit := new(big.Int).Mul(unit, bigint2)
	for delta := uHalfUnit; delta.Cmp(bigint0) > 0; delta = new(big.Int).Rsh(delta, 1) {
		y = new(big.Int).Div(new(big.Int).Mul(y, y), uUnit)

		if y.Cmp(doubleUnit) >= 0 {
			resultBI = new(big.Int).Add(resultBI, delta)

			y = new(big.Int).Rsh(y, 1)
		}
	}
	resultBI = new(big.Int).Mul(resultBI, sign)
	return Wrap(resultBI), nil
}

func Exp2(x SD59x18) (SD59x18, error) {
	xBI := Unwrap(x)
	if xBI.Cmp(bigint0) < 0 {
		magicNbr, _ := new(big.Int).SetString("-59794705707972522261", 10)
		if xBI.Cmp(magicNbr) < 0 {
			return bigint0, nil
		}
		v, err := Exp2(Wrap(new(big.Int).Neg(xBI)))
		if err != nil {
			return nil, err
		}
		vBI := Unwrap(v)
		return Wrap(new(big.Int).Div(uUnitSquared, vBI)), nil
	}

	if xBI.Cmp(uExp2MaxInput) > 0 {
		return nil, ErrMathSD59x18Exp2InputTooBig
	}

	xType192x64 := new(big.Int).Div(new(big.Int).Lsh(xBI, 64), uUnit)
	return Wrap(exp2(xType192x64)), nil
}

func Pow(x SD59x18, y SD59x18) (SD59x18, error) {
	var (
		xBI = Unwrap(x)
		yBI = Unwrap(y)
	)

	if xBI.Cmp(bigint0) == 0 {
		ret := Zero()
		if yBI.Cmp(bigint0) == 0 {
			ret = unit
		}
		return Wrap(ret), nil
	}

	if xBI.Cmp(uUnit) == 0 {
		return Wrap(uUnit), nil
	}

	if yBI.Cmp(bigint0) == 0 {
		return Wrap(uUnit), nil
	}

	if yBI.Cmp(uUnit) == 0 {
		return x, nil
	}

	a, err := Log2(x)
	if err != nil {
		return nil, err
	}
	a, err = Mul(a, y)
	if err != nil {
		return nil, err
	}
	return Exp2(a)
}

func Mul(x SD59x18, y SD59x18) (SD59x18, error) {
	var (
		xBI = Unwrap(x)
		yBI = Unwrap(y)
	)

	if xBI.Cmp(uMinSD59x18) == 0 || yBI.Cmp(uMinSD59x18) == 0 {
		return nil, ErrMathSD59x18MulInputTooSmall
	}

	xAbs := new(big.Int).Abs(xBI)
	yAbs := new(big.Int).Abs(yBI)

	resultAbs, err := mulDiv18(xAbs, yAbs)
	if err != nil {
		return nil, err
	}

	if resultAbs.Cmp(uMaxSD59x18) > 0 {
		return nil, ErrMathSD59x18MulOverflow
	}

	sameSign := xBI.Sign() == yBI.Sign()
	result := resultAbs
	if !sameSign {
		result = new(big.Int).Neg(resultAbs)
	}
	return Wrap(result), nil
}

func Div(x SD59x18, y SD59x18) (SD59x18, error) {
	var (
		xBI = Unwrap(x)
		yBI = Unwrap(y)
	)

	if xBI.Cmp(uMinSD59x18) == 0 || yBI.Cmp(uMinSD59x18) == 0 {
		return nil, ErrMathSD59x18DivInputTooSmall
	}

	var (
		xAbs = new(big.Int).Abs(xBI)
		yAbs = new(big.Int).Abs(yBI)
	)

	resultAbs, err := mulDiv(xAbs, uUnit, yAbs)
	if err != nil {
		return nil, err
	}

	if resultAbs.Cmp(uMaxSD59x18) > 0 {
		return nil, ErrMathSD59x18DivOverflow
	}

	sameSign := xBI.Sign() == yBI.Sign()
	result := resultAbs
	if !sameSign {
		result = new(big.Int).Neg(resultAbs)
	}

	return Wrap(result), nil
}
