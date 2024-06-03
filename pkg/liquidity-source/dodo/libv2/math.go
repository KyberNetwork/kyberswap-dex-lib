package libv2

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

// GeneralIntegrate
//
//	Integrate dodo curve from V1 to V2
//	require V0>=V1>=V2>0
//	res = (1-k)i(V1-V2)+ikV0*V0(1/V2-1/V1)
//	let V1-V2=delta
//	res = i*delta*(1-k+k(V0^2/V1/V2))
//
//	i is the price of V-res trading pair
//
//	support k=1 & k=0 case
//
//	[round down]
//
// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/DODOMath.sol#L36
func GeneralIntegrate(V0, V1, V2, i, k *uint256.Int) *uint256.Int {
	if V0.Cmp(number.Zero) <= 0 {
		panic(ErrTargetIsZero)
	}

	fairAmount := SafeMul(i, SafeSub(V1, V2)) // i*delta
	if k.Cmp(number.Zero) == 0 {
		return SafeDiv(fairAmount, DecimalMathOne)
	}

	V0V0V1V2 := DecimalMathDivFloor(
		SafeDiv(
			SafeMul(V0, V0),
			V1,
		),
		V2,
	) // V0^2/V1/V2
	penalty := DecimalMathMulFloor(k, V0V0V1V2) // k(V0^2/V1/V2)

	return SafeDiv(
		SafeMul(
			fairAmount,
			SafeAdd(
				SafeSub(DecimalMathOne, k),
				penalty,
			),
		),
		DecimalMathOne2,
	)
}

// SolveQuadraticFunctionForTarget
//
//	Follow the integration function above
//	i*deltaB = (Q2-Q1)*(1-k+kQ0^2/Q1/Q2)
//	Assume Q2=Q0, Given Q1 and deltaB, solve Q0
//
//	i is the price of delta-V trading pair
//	give out target of V
//
//	support k=1 & k=0 case
//
//	[round down]
//
// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/DODOMath.sol#L65
func SolveQuadraticFunctionForTarget(V1, delta, i, k *uint256.Int) *uint256.Int {
	if k.Cmp(number.Zero) == 0 {
		return SafeAdd(V1, DecimalMathMulFloor(i, delta))
	}
	// V0 = V1*(1+(sqrt-1)/2k)
	// sqrt = √(1+4kidelta/V1)
	// premium = 1+(sqrt-1)/2k
	// uint256 sqrt = (4 * k).mul(i).mul(delta).div(V1).add(DecimalMath.ONE2).sqrt();

	if V1.Cmp(number.Zero) == 0 {
		return number.Zero
	}

	var sqrt *uint256.Int
	ki := SafeMul(SafeMul(k, number.Number_4), i)
	if ki.Cmp(number.Zero) == 0 {
		sqrt = DecimalMathOne
	} else if SafeDiv(SafeMul(ki, delta), ki).Cmp(delta) == 0 {
		sqrt = SafeSqrt(SafeAdd(SafeDiv(SafeMul(ki, delta), V1), DecimalMathOne2))
	} else {
		sqrt = SafeSqrt(SafeAdd(SafeMul(SafeDiv(ki, V1), delta), DecimalMathOne2))
	}

	premium := SafeAdd(
		DecimalMathDivFloor(
			SafeSub(sqrt, DecimalMathOne),
			SafeMul(k, number.Number_2),
		),
		DecimalMathOne,
	)

	return DecimalMathMulFloor(V1, premium)
}

// SolveQuadraticFunctionForTrade
// Follow the integration expression above, we have:
// i*deltaB = (Q2-Q1)*(1-k+kQ0^2/Q1/Q2)
// Given Q1 and deltaB, solve Q2
// This is a quadratic function and the standard version is
// aQ2^2 + bQ2 + c = 0, where
// a=1-k
// -b=(1-k)Q1-kQ0^2/Q1+i*deltaB
// c=-kQ0^2
// and Q2=(-b+sqrt(b^2+4(1-k)kQ0^2))/2(1-k)
// note: another root is negative, abondan
//
// if deltaBSig=true, then Q2>Q1, user sell Q and receive B
// if deltaBSig=false, then Q2<Q1, user sell B and receive Q
// return |Q1-Q2|
//
// as we only support sell amount as delta, the deltaB is always negative
// the input ideltaB is actually -ideltaB in the equation
//
// i is the price of delta-V trading pair
//
// support k=1 & k=0 case
//
// [round down]
// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/DODOMath.sol#L122
func SolveQuadraticFunctionForTrade(V0, V1, delta, i, k *uint256.Int) *uint256.Int {
	if V0.Cmp(number.Zero) <= 0 {
		panic(ErrTargetIsZero)
	}

	if delta.Cmp(number.Zero) == 0 {
		return number.Zero
	}

	if k.Cmp(number.Zero) == 0 {
		temp := DecimalMathMulFloor(i, delta)
		if temp.Cmp(V1) > 0 {
			return V1
		} else {
			return temp
		}
	}

	if k.Cmp(DecimalMathOne) == 0 {
		// if k==1
		// Q2=Q1/(1+ideltaBQ1/Q0/Q0)
		// temp = ideltaBQ1/Q0/Q0
		// Q2 = Q1/(1+temp)
		// Q1-Q2 = Q1*(1-1/(1+temp)) = Q1*(temp/(1+temp))
		// uint256 temp = i.mul(delta).mul(V1).div(V0.mul(V0));
		var temp *uint256.Int
		idelta := SafeMul(i, delta)
		if idelta.Cmp(number.Zero) == 0 {
			temp = number.Zero
		} else if SafeDiv(SafeMul(idelta, V1), idelta).Cmp(V1) == 0 {
			temp = SafeDiv(SafeMul(idelta, V1), SafeMul(V0, V0))
		} else {
			temp = SafeDiv(SafeMul(SafeDiv(SafeMul(delta, V1), V0), i), V0)
		}
		return SafeDiv(SafeMul(V1, temp), SafeAdd(temp, DecimalMathOne))
	}

	// calculate -b value and sig
	// b = kQ0^2/Q1-i*deltaB-(1-k)Q1
	// part1 = (1-k)Q1 >=0
	// part2 = kQ0^2/Q1-i*deltaB >=0
	// bAbs = abs(part1-part2)
	// if part1>part2 => b is negative => bSig is false
	// if part2>part1 => b is positive => bSig is true
	part2 := SafeAdd(
		SafeMul(
			SafeDiv(
				SafeMul(k, V0),
				V1,
			),
			V0,
		),
		SafeMul(i, delta),
	) // kQ0^2/Q1-i*deltaB
	bAbs := SafeMul(
		SafeSub(DecimalMathOne, k),
		V1,
	) // (1-k)Q1

	var bSig bool
	if bAbs.Cmp(part2) >= 0 {
		bAbs = SafeSub(bAbs, part2)
		bSig = false
	} else {
		bAbs = SafeSub(part2, bAbs)
		bSig = true
	}
	bAbs = SafeDiv(bAbs, DecimalMathOne)

	// calculate sqrt
	squareRoot := DecimalMathMulFloor(
		SafeMul(
			SafeSub(DecimalMathOne, k),
			number.Number_4,
		),
		SafeMul(
			DecimalMathMulFloor(k, V0),
			V0,
		),
	) // 4(1-k)kQ0^2
	squareRoot = SafeSqrt(
		SafeAdd(
			SafeMul(bAbs, bAbs),
			squareRoot,
		),
	) // sqrt(b*b+4(1-k)kQ0*Q0)

	// final res
	denominator := SafeMul(SafeSub(DecimalMathOne, k), number.Number_2) // 2(1-k)
	var numerator *uint256.Int
	if bSig {
		numerator = SafeSub(squareRoot, bAbs)
		if numerator.Cmp(number.Zero) == 0 {
			panic(ErrShouldNotBeZero)
		}
	} else {
		numerator = SafeAdd(bAbs, squareRoot)
	}

	V2 := DecimalMathDivCeil(numerator, denominator)
	if V2.Cmp(V1) > 0 {
		return number.Zero
	} else {
		return SafeSub(V1, V2)
	}
}
