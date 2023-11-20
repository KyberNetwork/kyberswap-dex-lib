package stable

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	"github.com/holiman/uint256"
)

var StableMath *stableMath

type stableMath struct{}

func init() {
	StableMath = &stableMath{}
}

func (l *stableMath) _calcOutGivenIn(
	amp *uint256.Int,
	amountIn *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, error) {
	invariant, err := l._calculateInvariant(amp, balances, true)
	if err != nil {
		return nil, err
	}

	balances[indexIn], err = math.FixedPoint.Add(balances[indexIn], amountIn)
	if err != nil {
		return nil, err
	}

	finalBalanceOut, err := l._getTokenBalanceGivenInvariantAndAllOtherBalances(
		amp,
		balances,
		invariant,
		indexOut,
	)
	if err != nil {
		return nil, err
	}

	balances[indexIn], err = math.FixedPoint.Sub(balances[indexIn], amountIn)
	if err != nil {
		return nil, err
	}

	amountOut, err := math.FixedPoint.Sub(balances[indexOut], finalBalanceOut)
	if err != nil {
		return nil, err
	}
	amountOut, err = math.FixedPoint.Sub(amountOut, number.Number_1)
	if err != nil {
		return nil, err
	}

	return amountOut, nil
}

// https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F7#L49
func (l *stableMath) _calculateInvariant(
	amp *uint256.Int,
	balances []*uint256.Int,
	roundUp bool,
) (*uint256.Int, error) {
	sum := uint256.NewInt(0)
	numTokens := uint256.NewInt(uint64(len(balances)))

	for _, b := range balances {
		var err error
		sum, err = math.FixedPoint.Add(sum, b)
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
			v, err := math.Math.Mul(P_D, balances[j])
			if err != nil {
				return nil, err
			}
			v, err = math.Math.Mul(v, numTokens)
			if err != nil {
				return nil, err
			}
			P_D, err = math.Math.Div(v, invariant, roundUp)
			if err != nil {
				return nil, err
			}
		}

		prevInvariant := invariant

		var numerator *uint256.Int
		{
			u, err := math.Math.Mul(numTokens, invariant)
			if err != nil {
				return nil, err
			}
			u, err = math.Math.Mul(u, invariant)
			if err != nil {
				return nil, err
			}

			v, err := math.Math.Mul(ampTimesTotal, sum)
			if err != nil {
				return nil, err
			}
			v, err = math.Math.Mul(v, P_D)
			if err != nil {
				return nil, err
			}
			v, err = math.Math.Div(v, _AMP_PRECISION, roundUp)
			if err != nil {
				return nil, err
			}

			numerator, err = math.FixedPoint.Add(u, v)
			if err != nil {
				return nil, err
			}
		}

		var denominator *uint256.Int
		{
			u, _ := math.FixedPoint.Add(numTokens, number.Number_1)
			u, err := math.Math.Mul(u, invariant)
			if err != nil {
				return nil, err
			}

			v, err := math.FixedPoint.Sub(ampTimesTotal, _AMP_PRECISION)
			if err != nil {
				return nil, err
			}
			v, err = math.Math.Mul(v, P_D)
			if err != nil {
				return nil, err
			}
			v, err = math.Math.Div(v, _AMP_PRECISION, !roundUp)
			if err != nil {
				return nil, err
			}

			denominator, err = math.FixedPoint.Add(u, v)
			if err != nil {
				return nil, err
			}
		}

		invariant, err := math.Math.Div(numerator, denominator, roundUp)
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

// https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F7#L465
func (l *stableMath) _getTokenBalanceGivenInvariantAndAllOtherBalances(
	amp *uint256.Int,
	balances []*uint256.Int,
	invariant *uint256.Int,
	tokenIndex int,
) (*uint256.Int, error) {
	numTokens := uint256.NewInt(uint64(len(balances)))
	ampTimesTotal, err := math.Math.Mul(amp, numTokens)
	if err != nil {
		return nil, err
	}

	sum := new(uint256.Int).Set(balances[0])
	P_D, err := math.Math.Mul(balances[0], numTokens)
	if err != nil {
		return nil, err
	}
	for j := 1; j < len(balances); j++ {
		v, err := math.Math.Mul(P_D, balances[j])
		if err != nil {
			return nil, err
		}
		v, err = math.Math.Mul(v, numTokens)
		if err != nil {
			return nil, err
		}
		P_D, err = math.Math.DivDown(v, invariant)
		if err != nil {
			return nil, err
		}

		sum, err = math.FixedPoint.Add(sum, balances[j])
		if err != nil {
			return nil, err
		}
	}

	sum, _ = math.FixedPoint.Sub(sum, balances[tokenIndex])

	inv2, err := math.Math.Mul(invariant, invariant)
	if err != nil {
		return nil, err
	}

	var c *uint256.Int
	{
		u, err := math.Math.Mul(ampTimesTotal, P_D)
		if err != nil {
			return nil, err
		}
		u, err = math.Math.DivUp(inv2, u)
		if err != nil {
			return nil, err
		}
		u, err = math.Math.Mul(u, _AMP_PRECISION)
		if err != nil {
			return nil, err
		}

		c, err = math.Math.Mul(u, balances[tokenIndex])
		if err != nil {
			return nil, err
		}
	}

	var b *uint256.Int
	{
		u, err := math.Math.DivDown(invariant, ampTimesTotal)
		if err != nil {
			return nil, err
		}
		u, err = math.Math.Mul(u, _AMP_PRECISION)
		if err != nil {
			return nil, err
		}

		b, err = math.FixedPoint.Add(sum, u)
		if err != nil {
			return nil, err
		}
	}

	var tokenBalance *uint256.Int
	{
		u, err := math.FixedPoint.Add(inv2, c)
		if err != nil {
			return nil, err
		}
		v, err := math.FixedPoint.Add(invariant, b)
		if err != nil {
			return nil, err
		}
		tokenBalance, err = math.Math.DivUp(u, v)
		if err != nil {
			return nil, err
		}
	}

	for i := 0; i < 255; i++ {
		prevTokenBalance := tokenBalance

		// calc tokenBalance
		{
			u, err := math.Math.Mul(tokenBalance, tokenBalance)
			if err != nil {
				return nil, err
			}
			u, err = math.FixedPoint.Add(u, c)
			if err != nil {
				return nil, err
			}

			v, err := math.Math.Mul(tokenBalance, number.Number_2)
			if err != nil {
				return nil, err
			}
			v, err = math.FixedPoint.Add(v, b)
			if err != nil {
				return nil, err
			}
			v, err = math.FixedPoint.Sub(v, invariant)
			if err != nil {
				return nil, err
			}

			tokenBalance, err = math.Math.DivUp(u, v)
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
