package math

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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

// MetaStable: https://etherscan.io/address/0x063c624672e390363b25f0c6c68ad9067c34595b#code#F30#L109
//
// Stable Version 1: https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F8#L109
//
// Stable Version 2: https://etherscan.io/address/0x13f2f70a951fb99d48ede6e25b0bdf06914db33f#code#F5#L125
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

// MetaStable: https://etherscan.io/address/0x063c624672e390363b25f0c6c68ad9067c34595b#code#F30#L152
//
// Stable Version 1: https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F8#L152
//
// Stable Version 2: https://etherscan.io/address/0x13f2f70a951fb99d48ede6e25b0bdf06914db33f#code#F5#L166
func (l *stableMath) CalcInGivenOut(
	invariant *uint256.Int,
	amp *uint256.Int,
	amountOut *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, error) {
	var err error

	balances[indexOut], err = FixedPoint.Sub(balances[indexOut], amountOut)
	if err != nil {
		return nil, err
	}

	finalBalanceIn, err := l.GetTokenBalanceGivenInvariantAndAllOtherBalances(
		amp,
		balances,
		invariant,
		indexIn,
	)
	if err != nil {
		return nil, err
	}

	balances[indexOut], err = FixedPoint.Add(balances[indexOut], amountOut)
	if err != nil {
		return nil, err
	}

	// return finalBalanceIn.sub(balances[tokenIndexIn]).add(1);
	amountIn, err := FixedPoint.Sub(finalBalanceIn, balances[indexIn])
	if err != nil {
		return nil, err
	}

	amountIn, err = FixedPoint.Add(amountIn, number.Number_1)
	if err != nil {
		return nil, err
	}

	return amountIn, nil
}

func (l *stableMath) CalculateInvariantV1(
	amp *uint256.Int,
	balances []*uint256.Int,
	roundUp bool,
) (*uint256.Int, error) {
	var sum uint256.Int
	var numTokens uint256.Int
	numTokens.SetUint64(uint64(len(balances)))
	sumPtr := &sum

	for _, b := range balances {
		var err error
		sumPtr, err = FixedPoint.Add(sumPtr, b)
		if err != nil {
			return nil, err
		}
	}
	if sumPtr.IsZero() {
		return sumPtr, nil
	}

	var invariant uint256.Int
	invariant.Set(sumPtr)
	invariantPtr := &invariant

	var ampTimesTotal uint256.Int
	ampTimesTotal.Mul(amp, &numTokens)

	for i := 0; i < 255; i++ {
		var pD uint256.Int
		pD.Mul(balances[0], &numTokens)
		P_D := &pD
		for j := 1; j < len(balances); j++ {
			v, err := Math.Mul(P_D, balances[j])
			if err != nil {
				return nil, err
			}
			v, err = Math.Mul(v, &numTokens)
			if err != nil {
				return nil, err
			}
			P_D, err = Math.Div(v, invariantPtr, roundUp)
			if err != nil {
				return nil, err
			}
		}

		prevInvariant := invariantPtr

		var numerator *uint256.Int
		{
			u, err := Math.Mul(&numTokens, invariantPtr)
			if err != nil {
				return nil, err
			}
			u, err = Math.Mul(u, invariantPtr)
			if err != nil {
				return nil, err
			}

			v, err := Math.Mul(&ampTimesTotal, sumPtr)
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
			var u uint256.Int
			u.Add(&numTokens, number.Number_1)
			uPtr, err := Math.Mul(&u, invariantPtr)
			if err != nil {
				return nil, err
			}

			var v uint256.Int
			v.Sub(&ampTimesTotal, _AMP_PRECISION)
			vPtr, err := Math.Mul(&v, P_D)
			if err != nil {
				return nil, err
			}
			vPtr, err = Math.Div(vPtr, _AMP_PRECISION, !roundUp)
			if err != nil {
				return nil, err
			}

			denominator, err = FixedPoint.Add(uPtr, vPtr)
			if err != nil {
				return nil, err
			}
		}

		var err error
		invariantPtr, err = Math.Div(numerator, denominator, roundUp)
		if err != nil {
			return nil, err
		}

		var diff uint256.Int
		diff.Sub(invariantPtr, prevInvariant)
		var delta uint256.Int
		delta.Abs(&diff)
		if delta.Cmp(number.Number_1) <= 0 {
			return invariantPtr, nil
		}
	}

	return nil, ErrStableGetBalanceDidntConverge
}

func (l *stableMath) CalculateInvariantV2(
	amp *uint256.Int,
	balances []*uint256.Int,
) (*uint256.Int, error) {
	var sum uint256.Int
	var numTokens uint256.Int
	numTokens.SetUint64(uint64(len(balances)))
	sumPtr := &sum

	for _, b := range balances {
		var err error
		sumPtr, err = FixedPoint.Add(sumPtr, b)
		if err != nil {
			return nil, err
		}
	}
	if sumPtr.IsZero() {
		return sumPtr, nil
	}

	var invariant uint256.Int
	invariant.Set(sumPtr)
	invariantPtr := &invariant

	var ampTimesTotal uint256.Int
	ampTimesTotal.Mul(amp, &numTokens)

	for i := 0; i < 255; i++ {
		D_P := invariantPtr
		for j := 0; j < len(balances); j++ {
			u, err := Math.Mul(D_P, invariantPtr)
			if err != nil {
				return nil, err
			}

			v, err := Math.Mul(balances[j], &numTokens)
			if err != nil {
				return nil, err
			}

			D_P, err = Math.DivDown(u, v)
			if err != nil {
				return nil, err
			}
		}

		prevInvariant := invariantPtr

		var numerator *uint256.Int
		{
			u, err := Math.Mul(&ampTimesTotal, sumPtr)
			if err != nil {
				return nil, err
			}
			u, err = Math.DivDown(u, _AMP_PRECISION)
			if err != nil {
				return nil, err
			}

			v, err := Math.Mul(D_P, &numTokens)
			if err != nil {
				return nil, err
			}

			u, err = FixedPoint.Add(u, v)
			if err != nil {
				return nil, err
			}

			numerator, err = Math.Mul(u, invariantPtr)
			if err != nil {
				return nil, err
			}
		}

		var denominator *uint256.Int
		{
			var u uint256.Int
			u.Sub(&ampTimesTotal, _AMP_PRECISION)
			uPtr, err := Math.Mul(&u, invariantPtr)
			if err != nil {
				return nil, err
			}
			uPtr, err = Math.DivDown(uPtr, _AMP_PRECISION)
			if err != nil {
				return nil, err
			}

			var v uint256.Int
			v.Add(&numTokens, number.Number_1)
			vPtr, err := Math.Mul(&v, D_P)
			if err != nil {
				return nil, err
			}

			denominator, err = FixedPoint.Add(uPtr, vPtr)
			if err != nil {
				return nil, err
			}
		}

		var err error
		invariantPtr, err = Math.DivDown(numerator, denominator)
		if err != nil {
			return nil, err
		}

		var diff uint256.Int
		diff.Sub(invariantPtr, prevInvariant)
		var delta uint256.Int
		delta.Abs(&diff)
		if delta.Cmp(number.Number_1) <= 0 {
			return invariantPtr, nil
		}
	}

	return nil, ErrStableGetBalanceDidntConverge
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F17#L399
// This function calculates the balance of a given token (tokenIndex)
// given all the other balances and the invariant
func (l *stableMath) GetTokenBalanceGivenInvariantAndAllOtherBalances(
	amp *uint256.Int,
	balances []*uint256.Int,
	invariant *uint256.Int,
	tokenIndex int,
) (*uint256.Int, error) {
	var numTokens uint256.Int
	numTokens.SetUint64(uint64(len(balances)))
	ampTimesTotal, err := Math.Mul(amp, &numTokens)
	if err != nil {
		return nil, err
	}

	var sum uint256.Int
	sum.Set(balances[0])
	sumPtr := &sum
	P_D, err := Math.Mul(balances[0], &numTokens)
	if err != nil {
		return nil, err
	}
	for j := 1; j < len(balances); j++ {
		v, err := Math.Mul(P_D, balances[j])
		if err != nil {
			return nil, err
		}
		v, err = Math.Mul(v, &numTokens)
		if err != nil {
			return nil, err
		}
		P_D, err = Math.DivDown(v, invariant)
		if err != nil {
			return nil, err
		}

		sumPtr, err = FixedPoint.Add(sumPtr, balances[j])
		if err != nil {
			return nil, err
		}
	}

	sumPtr, _ = FixedPoint.Sub(sumPtr, balances[tokenIndex])

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

		b, err = FixedPoint.Add(sumPtr, u)
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

		var diff uint256.Int
		diff.Sub(tokenBalance, prevTokenBalance)
		var delta uint256.Int
		delta.Abs(&diff)
		if delta.Cmp(number.Number_1) <= 0 {
			return tokenBalance, nil
		}
	}

	return nil, ErrStableGetBalanceDidntConverge
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F17#L201
func (l *stableMath) CalcBptOutGivenExactTokensIn(
	amp *uint256.Int,
	balances []*uint256.Int,
	amountsIn []*uint256.Int,
	bptTotalSupply *uint256.Int,
	currentInvariant *uint256.Int,
	swapFeePercentage *uint256.Int,
) (*uint256.Int, error) {
	var sumBalances uint256.Int
	sumBalancesPtr := &sumBalances
	for _, b := range balances {
		var err error
		sumBalancesPtr, err = FixedPoint.Add(sumBalancesPtr, b)
		if err != nil {
			return nil, err
		}
	}

	balanceRatiosWithFee := make([]*uint256.Int, len(amountsIn))
	var invariantRatioWithFee uint256.Int
	invariantRatioWithFeePtr := &invariantRatioWithFee

	for i := 0; i < len(balances); i++ {
		currentWeight, err := FixedPoint.DivDown(balances[i], sumBalancesPtr)
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
		invariantRatioWithFeePtr, err = FixedPoint.Add(invariantRatioWithFeePtr, u)
		if err != nil {
			return nil, err
		}
	}

	newBalances := make([]*uint256.Int, len(balances))
	for i := 0; i < len(balances); i++ {
		var amountInWithoutFee *uint256.Int
		if balanceRatiosWithFee[i].Gt(invariantRatioWithFeePtr) {
			var nonTaxableAmount *uint256.Int
			{
				u, err := FixedPoint.Sub(invariantRatioWithFeePtr, FixedPoint.ONE)
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

			var feeComplement uint256.Int
			feeComplement.Sub(FixedPoint.ONE, swapFeePercentage)
			u, err := FixedPoint.MulDown(taxableAmount, &feeComplement)
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
		var invariantRatioDelta uint256.Int
		invariantRatioDelta.Sub(invariantRatio, FixedPoint.ONE)
		return FixedPoint.MulDown(bptTotalSupply, &invariantRatioDelta)
	}

	return u256.U0, nil
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F17#L257
func (l *stableMath) CalcTokenInGivenExactBptOut(
	amp *uint256.Int,
	balances []*uint256.Int,
	tokenIndex int,
	bptAmountOut *uint256.Int,
	bptTotalSupply *uint256.Int,
	currentInvariant *uint256.Int,
	swapFeePercentage *uint256.Int,
) (*uint256.Int, error) {
	// Token in, so we round up overall.

	newInvariant, err := FixedPoint.Add(bptTotalSupply, bptAmountOut)
	if err != nil {
		return nil, err
	}

	newInvariant, err = FixedPoint.DivUp(newInvariant, bptTotalSupply)
	if err != nil {
		return nil, err
	}

	newInvariant, err = FixedPoint.MulUp(newInvariant, currentInvariant)
	if err != nil {
		return nil, err
	}

	// Calculate amount in without fee.
	newBalanceTokenIndex, err := l.GetTokenBalanceGivenInvariantAndAllOtherBalances(
		amp,
		balances,
		newInvariant,
		tokenIndex,
	)
	if err != nil {
		return nil, err
	}

	amountInWithoutFee, err := FixedPoint.Sub(newBalanceTokenIndex, balances[tokenIndex])
	if err != nil {
		return nil, err
	}

	// First calculate the sum of all token balances, which will be used to calculate
	// the current weight of each token
	var sumBalances uint256.Int
	sumBalancesPtr := &sumBalances
	for _, b := range balances {
		var err error
		sumBalancesPtr, err = FixedPoint.Add(sumBalancesPtr, b)
		if err != nil {
			return nil, err
		}
	}

	// We can now compute how much extra balance is being deposited and used in virtual swaps, and charge swap fees
	// accordingly.
	currentWeight, err := FixedPoint.DivDown(balances[tokenIndex], sumBalancesPtr)
	if err != nil {
		return nil, err
	}

	taxablePercentage := FixedPoint.Complement(currentWeight)

	taxableAmount, err := FixedPoint.MulUp(amountInWithoutFee, taxablePercentage)
	if err != nil {
		return nil, err
	}

	nonTaxableAmount, err := FixedPoint.Sub(amountInWithoutFee, taxableAmount)
	if err != nil {
		return nil, err
	}

	temp, err := FixedPoint.Sub(FixedPoint.ONE, swapFeePercentage)
	if err != nil {
		return nil, err
	}

	temp, err = FixedPoint.DivUp(taxableAmount, temp)
	if err != nil {
		return nil, err
	}

	return FixedPoint.Add(nonTaxableAmount, temp)
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F17#L302
/*
   Flow of calculations:
   amountsTokenOut -> amountsOutProportional ->
   amountOutPercentageExcess -> amountOutBeforeFee -> newInvariant -> amountBPTIn
*/
func (l *stableMath) CalcBptInGivenExactTokensOut(
	amp *uint256.Int,
	balances []*uint256.Int,
	amountsOut []*uint256.Int,
	bptTotalSupply *uint256.Int,
	currentInvariant *uint256.Int,
	swapFeePercentage *uint256.Int,
) (*uint256.Int, error) {
	// BPT in, so we round up overall.

	// First loop calculates the sum of all token balances, which will be used to calculate
	// the current weights of each token relative to this sum
	var sumBalances uint256.Int
	sumBalancesPtr := &sumBalances
	for _, b := range balances {
		var err error
		sumBalancesPtr, err = FixedPoint.Add(sumBalancesPtr, b)
		if err != nil {
			return nil, err
		}
	}

	// Calculate the weighted balance ratio without considering fees
	balanceRatiosWithoutFee := make([]*uint256.Int, len(amountsOut))
	var invariantRatioWithoutFees uint256.Int
	invariantRatioWithoutFeesPtr := &invariantRatioWithoutFees

	for i := 0; i < len(balances); i++ {
		currentWeight, err := FixedPoint.DivUp(balances[i], sumBalancesPtr)
		if err != nil {
			return nil, err
		}

		u, err := FixedPoint.Sub(balances[i], amountsOut[i])
		if err != nil {
			return nil, err
		}
		balanceRatiosWithoutFee[i], err = FixedPoint.DivUp(u, balances[i])
		if err != nil {
			return nil, err
		}

		u, err = FixedPoint.MulUp(balanceRatiosWithoutFee[i], currentWeight)
		if err != nil {
			return nil, err
		}

		invariantRatioWithoutFeesPtr, err = FixedPoint.Add(invariantRatioWithoutFeesPtr, u)
		if err != nil {
			return nil, err
		}
	}

	// Second loop calculates new amounts in, taking into account the fee on the percentage excess
	newBalances := make([]*uint256.Int, len(balances))
	for i := 0; i < len(balances); i++ {
		// Swap fees are typically charged on 'token in', but there is no 'token in' here, so we apply it to
		// 'token out'. This results in slightly larger price impact.

		var amountOutWithFee *uint256.Int
		if invariantRatioWithoutFeesPtr.Gt(balanceRatiosWithoutFee[i]) {
			var nonTaxableAmount *uint256.Int

			nonTaxableAmount, err := FixedPoint.MulDown(balances[i], FixedPoint.Complement(invariantRatioWithoutFeesPtr))
			if err != nil {
				return nil, err
			}

			taxableAmount, err := FixedPoint.Sub(amountsOut[i], nonTaxableAmount)
			if err != nil {
				return nil, err
			}

			var feeComplement uint256.Int
			feeComplement.Sub(FixedPoint.ONE, swapFeePercentage)
			u, err := FixedPoint.DivUp(taxableAmount, &feeComplement)
			if err != nil {
				return nil, err
			}

			amountOutWithFee, err = FixedPoint.Add(nonTaxableAmount, u)
			if err != nil {
				return nil, err
			}

		} else {
			amountOutWithFee = amountsOut[i]
		}

		var err error
		newBalances[i], err = FixedPoint.Sub(balances[i], amountOutWithFee)
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

	// return amountBPTIn
	return FixedPoint.MulUp(bptTotalSupply, FixedPoint.Complement(invariantRatio))
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F17#L354
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

	var sumBalances uint256.Int
	sumBalancesPtr := &sumBalances
	for _, b := range balances {
		var err error
		sumBalancesPtr, err = FixedPoint.Add(sumBalancesPtr, b)
		if err != nil {
			return nil, err
		}
	}

	currentWeight, err := FixedPoint.DivDown(balances[tokenIndex], sumBalancesPtr)
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

	var feeComplement uint256.Int
	feeComplement.Sub(FixedPoint.ONE, swapFeePercentage)
	u, err := FixedPoint.MulDown(taxableAmount, &feeComplement)
	if err != nil {
		return nil, err
	}
	u, err = FixedPoint.Add(nonTaxableAmount, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (l *stableMath) CalcDueTokenProtocolSwapFeeAmount(
	amplificationParameter *uint256.Int,
	balances []*uint256.Int,
	lastInvariant *uint256.Int,
	tokenIndex int,
	protocolSwapFeePercentage *uint256.Int,
) (*uint256.Int, error) {
	finalBalanceFeeToken, err := l.GetTokenBalanceGivenInvariantAndAllOtherBalances(
		amplificationParameter,
		balances,
		lastInvariant,
		tokenIndex,
	)
	if err != nil {
		return nil, err
	}

	if balances[tokenIndex].Cmp(finalBalanceFeeToken) <= 0 {
		return number.Zero, nil
	}

	var accumulatedTokenSwapFees uint256.Int
	accumulatedTokenSwapFees.Sub(balances[tokenIndex], finalBalanceFeeToken)

	return FixedPoint.MulDown(&accumulatedTokenSwapFees, protocolSwapFeePercentage)
}
