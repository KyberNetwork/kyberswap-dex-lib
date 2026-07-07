package libv1

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

// GeneralIntegrate
// Integrate dodo curve fron V1 to V2
// require V0>=V1>=V2>0
// res = (1-k)i(V1-V2)+ikV0*V0(1/V2-1/V1)
// let V1-V2=delta
// res = i*delta*(1-k+k(V0^2/V1/V2))
// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/DODOMath.sol#L30
func GeneralIntegrate(
	V0 *uint256.Int,
	V1 *uint256.Int,
	V2 *uint256.Int,
	i *uint256.Int,
	k *uint256.Int,
) *uint256.Int {
	fairAmount := DecimalMathMul(i, SafeSub(V1, V2)) // i*delta
	V0V0V1V2 := DecimalMathDivCeil(
		SafeDiv(
			SafeMul(V0, V0),
			V1,
		),
		V2,
	) // V0^2/V1/V2
	penalty := DecimalMathMul(k, V0V0V1V2) // k(V0^2/V1/V2)
	return DecimalMathMul(
		fairAmount,
		SafeAdd(
			SafeSub(DecimalMathOne, k),
			penalty,
		),
	)
}

// SolveQuadraticFunctionForTrade
// The same with integration expression above, we have:
// i*deltaB = (Q2-Q1)*(1-k+kQ0^2/Q1/Q2)
// Given Q1 and deltaB, solve Q2
// This is a quadratic function and the standard version is
// aQ2^2 + bQ2 + c = 0, where
// a=1-k
// -b=(1-k)Q1-kQ0^2/Q1+i*deltaB
// c=-kQ0^2
// and Q2=(-b+sqrt(b^2+4(1-k)kQ0^2))/2(1-k)
// note: another root is negative, abondan
// if deltaBSig=true, then Q2>Q1
// if deltaBSig=false, then Q2<Q1
// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/DODOMath.sol#L57
func SolveQuadraticFunctionForTrade(
	Q0 *uint256.Int,
	Q1 *uint256.Int,
	ideltaB *uint256.Int,
	deltaBSig bool,
	k *uint256.Int,
) *uint256.Int {
	// calculate -b value and sig
	// -b = (1-k)Q1-kQ0^2/Q1+i*deltaB
	kQ02Q1 := SafeDiv(SafeMul(DecimalMathMul(k, Q0), Q0), Q1) // kQ0^2/Q1
	b := DecimalMathMul(SafeSub(DecimalMathOne, k), Q1)       // (1-k)Q1

	minusbSig := true // nolint
	if deltaBSig {
		b = SafeAdd(b, ideltaB) // (1-k)Q1+i*deltaB
	} else {
		kQ02Q1 = SafeAdd(kQ02Q1, ideltaB) // i*deltaB+kQ0^2/Q1
	}

	if b.Cmp(kQ02Q1) >= 0 {
		b = SafeSub(b, kQ02Q1)
		minusbSig = true
	} else {
		b = SafeSub(kQ02Q1, b)
		minusbSig = false
	}

	// calculate sqrt
	squareRoot := DecimalMathMul(
		SafeMul(
			SafeSub(DecimalMathOne, k),
			number.Number_4,
		),
		SafeMul(
			DecimalMathMul(k, Q0),
			Q0,
		),
	) // 4(1-k)kQ0^2
	squareRoot = SafeSqrt(
		SafeAdd(
			SafeMul(b, b),
			squareRoot,
		),
	) // sqrt(b*b+4(1-k)kQ0*Q0)

	// final res
	denominator := new(uint256.Int).Mul(new(uint256.Int).Sub(DecimalMathOne, k), number.Number_2) // 2(1-k)
	var numerator *uint256.Int
	if minusbSig {
		numerator = SafeAdd(b, squareRoot)
	} else {
		numerator = SafeSub(squareRoot, b)
	}

	if deltaBSig {
		return DecimalMathDivFloor(numerator, denominator)
	} else {
		return DecimalMathDivCeil(numerator, denominator)
	}
}

// SolveQuadraticFunctionForTarget
// Start from the integration function
// i*deltaB = (Q2-Q1)*(1-k+kQ0^2/Q1/Q2)
// Assume Q2=Q0, Given Q1 and deltaB, solve Q0
// let fairAmount = i*deltaB
// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/DODOMath.sol#L111
func SolveQuadraticFunctionForTarget(
	V1 *uint256.Int,
	k *uint256.Int,
	fairAmount *uint256.Int,
) *uint256.Int {
	sqrt := DecimalMathDivCeil(
		SafeMul(
			DecimalMathMul(
				k,
				fairAmount,
			),
			number.Number_4,
		),
		V1,
	) // k*fairAmount*4/V1

	//sqrt = sqrt.add(DecimalMath.ONE).mul(DecimalMath.ONE).sqrt();
	sqrt = SafeSqrt(
		SafeMul(
			SafeAdd(sqrt, DecimalMathOne),
			DecimalMathOne,
		),
	) // (sqrt+1)*1

	premium := DecimalMathDivCeil(
		SafeSub(sqrt, DecimalMathOne),
		SafeMul(k, number.Number_2),
	) // (sqrt-1)/2k

	return DecimalMathMul(
		V1,
		SafeAdd(
			DecimalMathOne,
			premium,
		),
	)
}
