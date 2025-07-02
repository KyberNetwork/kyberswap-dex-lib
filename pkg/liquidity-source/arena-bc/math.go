package arenabc

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func integralCeil(upperBound, lowerBound, a, b, curveScaler *uint256.Int) *uint256.Int {
	var upperSum, lowerSum, temp uint256.Int
	temp.Exp(upperBound, u256.U3).Mul(&temp, u256.U2).Mul(&temp, a)
	upperSum.Set(&temp)
	temp.Exp(upperBound, u256.U2).Mul(&temp, u256.U3).Mul(&temp, b)
	upperSum.Add(&upperSum, &temp)

	temp.Exp(lowerBound, u256.U3).Mul(&temp, u256.U2).Mul(&temp, a)
	lowerSum.Set(&temp)
	temp.Exp(lowerBound, u256.U2).Mul(&temp, u256.U3).Mul(&temp, b)
	lowerSum.Add(&lowerSum, &temp)

	temp.Mul(curveScaler, u256.U6)

	// ((upperSum - lowerSum) + (params.curveScaler * 6 - 1)) / (params.curveScaler * 6)
	return upperSum.Sub(&upperSum, &lowerSum).Add(&upperSum, &temp).Sub(&upperSum, u256.U1).Div(&upperSum, &temp)
}

func integralFloor(upperBound, lowerBound, a, b, curveScaler *uint256.Int) *uint256.Int {
	var upperSum, lowerSum, temp uint256.Int
	temp.Exp(upperBound, u256.U3).Mul(&temp, u256.U2).Mul(&temp, a)
	upperSum.Set(&temp)
	temp.Exp(upperBound, u256.U2).Mul(&temp, u256.U3).Mul(&temp, b)
	upperSum.Add(&upperSum, &temp)

	temp.Exp(lowerBound, u256.U3).Mul(&temp, u256.U2).Mul(&temp, a)
	lowerSum.Set(&temp)
	temp.Exp(lowerBound, u256.U2).Mul(&temp, u256.U3).Mul(&temp, b)
	lowerSum.Add(&lowerSum, &temp)

	temp.Mul(curveScaler, u256.U6)

	// (upperSum - lowerSum) / (curveScaler * 6)
	return upperSum.Sub(&upperSum, &lowerSum).Div(&upperSum, &temp)
}

func getFee(costs, protocolFeeBasisPoint, creatorFeeBasisPoints, referralFeeBasisPoint *uint256.Int) *uint256.Int {
	var protocolFee, creatorFee, referralFee, totalFeeAmount uint256.Int
	protocolFee.Mul(costs, protocolFeeBasisPoint).Add(&protocolFee, U5000).Div(&protocolFee, u256.UBasisPoint)
	creatorFee.Mul(costs, creatorFeeBasisPoints).Add(&creatorFee, U5000).Div(&creatorFee, u256.UBasisPoint)
	referralFee.Mul(costs, referralFeeBasisPoint).Add(&referralFee, U5000).Div(&referralFee, u256.UBasisPoint)
	// no referrer
	protocolFee.Add(&protocolFee, &referralFee)
	referralFee.Clear()
	totalFeeAmount.Add(&protocolFee, &creatorFee)

	return &totalFeeAmount
}
