package math

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var (
	ErrStableGetBalanceDidntConverge = errors.New("stable get balance didn't converge")

	_AMP_PRECISION = uint256.NewInt(1000)
)

var StableMath *stableMath

type stableMath struct{}

func init() {
	StableMath = &stableMath{}
}

func (l *stableMath) CalcOutGivenIn(
	invariant *uint256.Int,
	amp *uint256.Int,
	amountIn *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, error) {
	var err error

	balances[indexIn], err = FixedPoint.Add(balances[indexIn], amountIn)
	if err != nil {
		return nil, err
	}

	finalBalanceOut, err := l.GetTokenBalanceGivenInvariantAndAllOtherBalances(
		amp,
		balances,
		invariant,
		indexOut,
	)
	if err != nil {
		return nil, err
	}

	balances[indexIn], err = FixedPoint.Sub(balances[indexIn], amountIn)
	if err != nil {
		return nil, err
	}

	amountOut, err := FixedPoint.Sub(balances[indexOut], finalBalanceOut)
	if err != nil {
		return nil, err
	}
	amountOut, err = FixedPoint.Sub(amountOut, number.Number_1)
	if err != nil {
		return nil, err
	}

	return amountOut, nil
}

func (l *stableMath) CalculateInvariantV1(
	amp *uint256.Int,
	balances []*uint256.Int,
	roundUp bool,
) (*uint256.Int, error) {
	sum := uint256.NewInt(0)
	numTokens := uint256.NewInt(uint64(len(balances)))

	for _, b := range balances {
		var err error
		sum, err = FixedPoint.Add(sum, b)
		if err != nil {
			return nil, err
		}
	}
	if sum.IsZero() {
		return sum, nil
	}

	invariant := new(uint256.Int).Set(sum)
	ampTimesTotal := new(uint256.Int).Mul(amp, numTokens)

	for i := 0; i < 255; i++ {
		P_D := new(uint256.Int).Mul(balances[0], numTokens)
		for j := 1; j < len(balances); j++ {
			v, err := Math.Mul(P_D, balances[j])
			if err != nil {
				return nil, err
			}
			v, err = Math.Mul(v, numTokens)
			if err != nil {
				return nil, err
			}
			P_D, err = Math.Div(v, invariant, roundUp)
			if err != nil {
				return nil, err
			}
		}

		prevInvariant := invariant

		var numerator *uint256.Int
		{
			u, err := Math.Mul(numTokens, invariant)
			if err != nil {
				return nil, err
			}
			u, err = Math.Mul(u, invariant)
			if err != nil {
				return nil, err
			}

			v, err := Math.Mul(ampTimesTotal, sum)
			if err != nil {
				return nil, err
			}
			v, err = Math.Mul(v, P_D)
			if err != nil {
				return nil, err
			}
			v, err = Math.Div(v, _AMP_PRECISION, roundUp)
			if err != nil {
				return nil, err
			}

			numerator, err = FixedPoint.Add(u, v)
			if err != nil {
				return nil, err
			}
		}

		var denominator *uint256.Int
		{
			u := new(uint256.Int).Add(numTokens, number.Number_1)
			u, err := Math.Mul(u, invariant)
			if err != nil {
				return nil, err
			}

			v := new(uint256.Int).Sub(ampTimesTotal, _AMP_PRECISION)
			v, err = Math.Mul(v, P_D)
			if err != nil {
				return nil, err
			}
			v, err = Math.Div(v, _AMP_PRECISION, !roundUp)
			if err != nil {
				return nil, err
			}

			denominator, err = FixedPoint.Add(u, v)
			if err != nil {
				return nil, err
			}
		}

		var err error
		invariant, err = Math.Div(numerator, denominator, roundUp)
		if err != nil {
			return nil, err
		}

		delta := new(uint256.Int).Abs(
			new(uint256.Int).Sub(invariant, prevInvariant),
		)
		if delta.Cmp(number.Number_1) <= 0 {
			return invariant, nil
		}
	}

	return nil, ErrStableGetBalanceDidntConverge
}

func (l *stableMath) CalculateInvariantV2(
	amp *uint256.Int,
	balances []*uint256.Int,
) (*uint256.Int, error) {
	sum := uint256.NewInt(0)
	numTokens := uint256.NewInt(uint64(len(balances)))

	for _, b := range balances {
		var err error
		sum, err = FixedPoint.Add(sum, b)
		if err != nil {
			return nil, err
		}
	}
	if sum.IsZero() {
		return sum, nil
	}

	invariant := new(uint256.Int).Set(sum)
	ampTimesTotal := new(uint256.Int).Mul(amp, numTokens)

	for i := 0; i < 255; i++ {
		D_P := invariant
		for j := 0; j < len(balances); j++ {
			u, err := Math.Mul(D_P, invariant)
			if err != nil {
				return nil, err
			}

			v, err := Math.Mul(balances[j], numTokens)
			if err != nil {
				return nil, err
			}

			D_P, err = Math.DivDown(u, v)
			if err != nil {
				return nil, err
			}
		}

		prevInvariant := invariant

		// numerator
		var numerator *uint256.Int
		{
			u, err := Math.Mul(ampTimesTotal, sum)
			if err != nil {
				return nil, err
			}
			u, err = Math.DivDown(u, _AMP_PRECISION)
			if err != nil {
				return nil, err
			}

			v, err := Math.Mul(D_P, numTokens)
			if err != nil {
				return nil, err
			}

			u, err = FixedPoint.Add(u, v)
			if err != nil {
				return nil, err
			}

			numerator, err = Math.Mul(u, invariant)
			if err != nil {
				return nil, err
			}
		}

		// denominator
		var denominator *uint256.Int
		{
			u := new(uint256.Int).Sub(ampTimesTotal, _AMP_PRECISION)
			u, err := Math.Mul(u, invariant)
			if err != nil {
				return nil, err
			}
			u, err = Math.DivDown(u, _AMP_PRECISION)
			if err != nil {
				return nil, err
			}

			v, err := Math.Mul(new(uint256.Int).Add(numTokens, number.Number_1), D_P)
			if err != nil {
				return nil, err
			}

			denominator, err = FixedPoint.Add(u, v)
			if err != nil {
				return nil, err
			}
		}

		var err error
		invariant, err = Math.DivDown(numerator, denominator)
		if err != nil {
			return nil, err
		}

		delta := new(uint256.Int).Abs(
			new(uint256.Int).Sub(invariant, prevInvariant),
		)
		if delta.Cmp(number.Number_1) <= 0 {
			return invariant, nil
		}
	}

	return nil, ErrStableGetBalanceDidntConverge
}

func (l *stableMath) GetTokenBalanceGivenInvariantAndAllOtherBalances(
	amp *uint256.Int,
	balances []*uint256.Int,
	invariant *uint256.Int,
	tokenIndex int,
) (*uint256.Int, error) {
	numTokens := uint256.NewInt(uint64(len(balances)))
	ampTimesTotal, err := Math.Mul(amp, numTokens)
	if err != nil {
		return nil, err
	}

	sum := new(uint256.Int).Set(balances[0])
	P_D, err := Math.Mul(balances[0], numTokens)
	if err != nil {
		return nil, err
	}
	for j := 1; j < len(balances); j++ {
		v, err := Math.Mul(P_D, balances[j])
		if err != nil {
			return nil, err
		}
		v, err = Math.Mul(v, numTokens)
		if err != nil {
			return nil, err
		}
		P_D, err = Math.DivDown(v, invariant)
		if err != nil {
			return nil, err
		}

		sum, err = FixedPoint.Add(sum, balances[j])
		if err != nil {
			return nil, err
		}
	}

	sum, _ = FixedPoint.Sub(sum, balances[tokenIndex])

	inv2, err := Math.Mul(invariant, invariant)
	if err != nil {
		return nil, err
	}

	var c *uint256.Int
	{
		u, err := Math.Mul(ampTimesTotal, P_D)
		if err != nil {
			return nil, err
		}
		u, err = Math.DivUp(inv2, u)
		if err != nil {
			return nil, err
		}
		u, err = Math.Mul(u, _AMP_PRECISION)
		if err != nil {
			return nil, err
		}

		c, err = Math.Mul(u, balances[tokenIndex])
		if err != nil {
			return nil, err
		}
	}

	var b *uint256.Int
	{
		u, err := Math.DivDown(invariant, ampTimesTotal)
		if err != nil {
			return nil, err
		}
		u, err = Math.Mul(u, _AMP_PRECISION)
		if err != nil {
			return nil, err
		}

		b, err = FixedPoint.Add(sum, u)
		if err != nil {
			return nil, err
		}
	}

	var tokenBalance *uint256.Int
	{
		u, err := FixedPoint.Add(inv2, c)
		if err != nil {
			return nil, err
		}
		v, err := FixedPoint.Add(invariant, b)
		if err != nil {
			return nil, err
		}
		tokenBalance, err = Math.DivUp(u, v)
		if err != nil {
			return nil, err
		}
	}

	for i := 0; i < 255; i++ {
		prevTokenBalance := tokenBalance

		// calc tokenBalance
		{
			u, err := Math.Mul(tokenBalance, tokenBalance)
			if err != nil {
				return nil, err
			}
			u, err = FixedPoint.Add(u, c)
			if err != nil {
				return nil, err
			}

			v, err := Math.Mul(tokenBalance, number.Number_2)
			if err != nil {
				return nil, err
			}
			v, err = FixedPoint.Add(v, b)
			if err != nil {
				return nil, err
			}
			v, err = FixedPoint.Sub(v, invariant)
			if err != nil {
				return nil, err
			}

			tokenBalance, err = Math.DivUp(u, v)
			if err != nil {
				return nil, err
			}
		}

		delta := new(uint256.Int).Abs(
			new(uint256.Int).Sub(tokenBalance, prevTokenBalance),
		)
		if delta.Cmp(number.Number_1) <= 0 {
			return tokenBalance, nil
		}
	}

	return nil, ErrStableGetBalanceDidntConverge
}

func (l *stableMath) CalcBptOutGivenExactTokensIn(
	amp *uint256.Int,
	balances []*uint256.Int,
	amountsIn []*uint256.Int,
	bptTotalSupply *uint256.Int,
	currentInvariant *uint256.Int,
	swapFeePercentage *uint256.Int,
) (*uint256.Int, error) {
	sumBalances := uint256.NewInt(0)
	for _, b := range balances {
		var err error
		sumBalances, err = FixedPoint.Add(sumBalances, b)
		if err != nil {
			return nil, err
		}
	}

	balanceRatiosWithFee := make([]*uint256.Int, len(amountsIn))
	invariantRatioWithFee := uint256.NewInt(0)

	for i := 0; i < len(balances); i++ {
		currentWeight, err := FixedPoint.DivDown(balances[i], sumBalances)
		if err != nil {
			return nil, err
		}

		u, err := FixedPoint.Add(balances[i], amountsIn[i])
		if err != nil {
			return nil, err
		}
		balanceRatiosWithFee[i], err = FixedPoint.DivDown(u, balances[i])
		if err != nil {
			return nil, err
		}

		u, err = FixedPoint.MulDown(balanceRatiosWithFee[i], currentWeight)
		if err != nil {
			return nil, err
		}
		invariantRatioWithFee, err = FixedPoint.Add(invariantRatioWithFee, u)
		if err != nil {
			return nil, err
		}
	}

	newBalances := make([]*uint256.Int, len(balances))
	for i := 0; i < len(balances); i++ {
		var amountInWithoutFee *uint256.Int
		if balanceRatiosWithFee[i].Gt(invariantRatioWithFee) {
			var nonTaxableAmount *uint256.Int
			{
				u, err := FixedPoint.Sub(invariantRatioWithFee, FixedPoint.ONE)
				if err != nil {
					return nil, err
				}
				nonTaxableAmount, err = FixedPoint.MulDown(balances[i], u)
				if err != nil {
					return nil, err
				}
			}

			taxableAmount, err := FixedPoint.Sub(amountsIn[i], nonTaxableAmount)
			if err != nil {
				return nil, err
			}

			u, err := FixedPoint.MulDown(taxableAmount, new(uint256.Int).Sub(FixedPoint.ONE, swapFeePercentage))
			if err != nil {
				return nil, err
			}

			amountInWithoutFee, err = FixedPoint.Add(nonTaxableAmount, u)
			if err != nil {
				return nil, err
			}

		} else {
			amountInWithoutFee = amountsIn[i]
		}

		var err error
		newBalances[i], err = FixedPoint.Add(balances[i], amountInWithoutFee)
		if err != nil {
			return nil, err
		}
	}

	newInvariant, err := l.CalculateInvariantV2(amp, newBalances)
	if err != nil {
		return nil, err
	}

	invariantRatio, err := FixedPoint.DivDown(newInvariant, currentInvariant)
	if err != nil {
		return nil, err
	}

	if invariantRatio.Gt(FixedPoint.ONE) {
		return FixedPoint.MulDown(bptTotalSupply, new(uint256.Int).Sub(invariantRatio, FixedPoint.ONE))
	}

	return uint256.NewInt(0), nil
}

func (l *stableMath) CalcTokenOutGivenExactBptIn(
	amp *uint256.Int,
	balances []*uint256.Int,
	tokenIndex int,
	bptAmountIn *uint256.Int,
	bptTotalSupply *uint256.Int,
	currentInvariant *uint256.Int,
	swapFeePercentage *uint256.Int,
) (*uint256.Int, error) {
	var newInvariant *uint256.Int
	{
		u, err := FixedPoint.Sub(bptTotalSupply, bptAmountIn)
		if err != nil {
			return nil, err
		}
		u, err = FixedPoint.DivUp(u, bptTotalSupply)
		if err != nil {
			return nil, err
		}
		u, err = FixedPoint.MulUp(u, currentInvariant)
		if err != nil {
			return nil, err
		}

		newInvariant = u
	}

	newBalanceTokenIndex, err := l.GetTokenBalanceGivenInvariantAndAllOtherBalances(
		amp,
		balances,
		newInvariant,
		tokenIndex,
	)
	if err != nil {
		return nil, err
	}

	amountOutWithoutFee, err := FixedPoint.Sub(balances[tokenIndex], newBalanceTokenIndex)
	if err != nil {
		return nil, err
	}

	sumBalances := uint256.NewInt(0)
	for _, b := range balances {
		var err error
		sumBalances, err = FixedPoint.Add(sumBalances, b)
		if err != nil {
			return nil, err
		}
	}

	currentWeight, err := FixedPoint.DivDown(balances[tokenIndex], sumBalances)
	if err != nil {
		return nil, err
	}

	taxablePercentage := FixedPoint.Complement(currentWeight)

	taxableAmount, err := FixedPoint.MulUp(amountOutWithoutFee, taxablePercentage)
	if err != nil {
		return nil, err
	}
	nonTaxableAmount, err := FixedPoint.Sub(amountOutWithoutFee, taxableAmount)
	if err != nil {
		return nil, err
	}

	u, err := FixedPoint.MulDown(taxableAmount, new(uint256.Int).Sub(FixedPoint.ONE, swapFeePercentage))
	if err != nil {
		return nil, err
	}
	u, err = FixedPoint.Add(nonTaxableAmount, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}
