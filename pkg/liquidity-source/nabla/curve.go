package nabla

import (
	"github.com/KyberNetwork/int256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/int256"
)

type Curve struct {
	beta *int256.Int
	c    *int256.Int
}

func NewCurve(beta, c *int256.Int) *Curve {
	return &Curve{
		beta: beta,
		c:    c,
	}
}

func mul(a, b *int256.Int) *int256.Int {
	result := new(int256.Int).Mul(a, b)
	return result.Quo(result, mantissa)
}

func div(a, b *int256.Int) *int256.Int {
	result := new(int256.Int).Mul(a, mantissa)
	return result.Quo(result, b)
}

func abs(a *int256.Int) *int256.Int {
	if a.Sign() < 0 {
		return new(int256.Int).Neg(a)
	}
	return new(int256.Int).Set(a)
}

func log2(value *int256.Int) uint {
	if value.Sign() <= 0 {
		return 0
	}
	return uint(value.ToBig().BitLen() - 1)
}

func sqrt(_a *int256.Int) *int256.Int {
	a := new(int256.Int).Mul(_a, mantissa)

	if a.Sign() == 0 {
		return i256.Zero.Clone()
	}

	log2Val := log2(a)
	result := new(int256.Int).Lsh(i256.One, log2Val>>1)

	for i := 0; i < 7; i++ {
		quotient := new(int256.Int).Quo(a, result)
		sum := new(int256.Int).Add(result, quotient)
		result = new(int256.Int).Rsh(sum, 1)
	}

	quotient := new(int256.Int).Quo(a, result)
	return i256.Min(result, quotient)
}

func (c *Curve) convertToInternalDecimals(value *int256.Int, dec int64) *int256.Int {
	convertedValue := new(int256.Int).Set(value)

	if dec > decimals {
		divisor := i256.TenPow(uint64(dec - decimals))
		convertedValue.Quo(convertedValue, divisor)
	} else if dec < decimals {
		multiplier := i256.TenPow(uint64(decimals - dec))
		convertedValue.Mul(convertedValue, multiplier)
	}

	return convertedValue
}

func (c *Curve) convertToExternalDecimals(value *int256.Int, dec int64) *int256.Int {
	convertedValue := new(int256.Int).Set(value)

	if dec > decimals {
		multiplier := i256.TenPow(uint64(dec - decimals))
		convertedValue.Mul(convertedValue, multiplier)
	} else if dec < decimals {
		divisor := i256.TenPow(uint64(decimals - dec))
		convertedValue.Quo(convertedValue, divisor)
	}

	return convertedValue
}

func (c *Curve) solveQuadratic(a, b, cVal *int256.Int) *int256.Int {
	bSquared := mul(b, b)
	fourA := new(int256.Int).Mul(i256.Four, a)
	fourAC := mul(fourA, cVal)
	discriminant := new(int256.Int).Sub(bSquared, fourAC)

	if discriminant.Sign() < 0 {
		discriminant = i256.Zero
	}

	sqrtDiscriminant := sqrt(discriminant)
	negB := new(int256.Int).Neg(b)
	numerator := new(int256.Int).Add(negB, sqrtDiscriminant)
	twoA := new(int256.Int).Mul(i256.Two, a)
	almostSolution := div(numerator, twoA)

	if almostSolution.Sign() < 0 {
		return i256.Zero
	}

	return almostSolution
}

func (c *Curve) Psi(b, l *int256.Int, dec int64) *int256.Int {
	iB := c.convertToInternalDecimals(b, dec)
	iL := c.convertToInternalDecimals(l, dec)

	var psi *int256.Int

	if iB.Sign() == 0 && iL.Sign() == 0 {
		psi = i256.Zero.Clone()
	} else {
		diff := new(int256.Int).Sub(iB, iL)
		diff = abs(diff)

		diffSquared := mul(diff, diff)

		betaDiffSquared := mul(c.beta, diffSquared)
		cTimesIL := mul(c.c, iL)
		denominator := new(int256.Int).Add(iB, cTimesIL)
		psi = div(betaDiffSquared, denominator)
		psi.Add(psi, iB)
	}

	return c.convertToExternalDecimals(psi, dec)
}

func (c *Curve) InverseDiagonal(b, l, capitalB *int256.Int, dec int64) *int256.Int {
	iB := c.convertToInternalDecimals(b, dec)
	iL := c.convertToInternalDecimals(l, dec)
	iCapitalB := c.convertToInternalDecimals(capitalB, dec)

	quadraticA := new(int256.Int).Add(mantissa, c.c)

	cTimesIL := mul(c.c, iL)
	term1 := new(int256.Int).Add(iB, cTimesIL)

	capitalBMinusB := new(int256.Int).Sub(iCapitalB, iB)
	term2 := mul(capitalBMinusB, quadraticA)

	quadraticB := new(int256.Int).Sub(term1, term2)

	diff := new(int256.Int).Sub(iB, iL)
	factor := mul(diff, diff)

	betaFactor := mul(c.beta, factor)
	term3 := mul(capitalBMinusB, term1)

	quadraticC := new(int256.Int).Sub(betaFactor, term3)

	t := c.solveQuadratic(quadraticA, quadraticB, quadraticC)

	return c.convertToExternalDecimals(t, dec)
}

func (c *Curve) InverseHorizontal(b, l, capitalB *int256.Int, dec int64) *int256.Int {
	iB := c.convertToInternalDecimals(b, dec)
	iL := c.convertToInternalDecimals(l, dec)
	iCapitalB := c.convertToInternalDecimals(capitalB, dec)

	quadraticA := new(int256.Int).Add(mantissa, c.beta)

	diff := new(int256.Int).Sub(iB, iL)
	twoBeta := new(int256.Int).Mul(i256.Two, c.beta)
	term1 := mul(twoBeta, diff)

	twoIB := new(int256.Int).Mul(i256.Two, iB)
	cTimesIL := mul(c.c, iL)

	quadraticB := new(int256.Int).Add(term1, twoIB)
	quadraticB.Add(quadraticB, cTimesIL)
	quadraticB.Sub(quadraticB, iCapitalB)

	factor := mul(diff, diff)

	betaFactor := mul(c.beta, factor)
	capitalBMinusB := new(int256.Int).Sub(iCapitalB, iB)
	term2 := new(int256.Int).Add(iB, cTimesIL)
	term3 := mul(capitalBMinusB, term2)

	quadraticC := new(int256.Int).Sub(betaFactor, term3)

	t := c.solveQuadratic(quadraticA, quadraticB, quadraticC)

	return c.convertToExternalDecimals(t, dec)
}
