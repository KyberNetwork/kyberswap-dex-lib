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
	var tempFeeAmount, totalFeeAmount uint256.Int

	// protocolFee
	tempFeeAmount.Mul(costs, protocolFeeBasisPoint).Add(&tempFeeAmount, U5000).Div(&tempFeeAmount, u256.UBasisPoint)
	totalFeeAmount.Add(&totalFeeAmount, &tempFeeAmount)

	// creatorFee
	tempFeeAmount.Mul(costs, creatorFeeBasisPoints).Add(&tempFeeAmount, U5000).Div(&tempFeeAmount, u256.UBasisPoint)
	totalFeeAmount.Add(&totalFeeAmount, &tempFeeAmount)

	// referralFee
	tempFeeAmount.Mul(costs, referralFeeBasisPoint).Add(&tempFeeAmount, U5000).Div(&tempFeeAmount, u256.UBasisPoint)
	totalFeeAmount.Add(&totalFeeAmount, &tempFeeAmount)

	return &totalFeeAmount
}

func getMaxTokensForSale(allowedTotalSupply, salePercentage *uint256.Int) *uint256.Int {
	maxTokensForSale, _ := new(uint256.Int).MulDivOverflow(allowedTotalSupply, salePercentage, u256.U100)
	return maxTokensForSale
}

func getBuyLimit(totalSupply, allowedTotalSupply, salePercentage *uint256.Int) *uint256.Int {
	buyLimit, _ := new(uint256.Int).MulDivOverflow(allowedTotalSupply, salePercentage, u256.U100)
	if !buyLimit.Lt(totalSupply) {
		buyLimit.Sub(buyLimit, totalSupply)
	}

	return buyLimit
}

func getSellLimit(totalSupply, a, b, curveScaler, protocolFeeBasisPoint, creatorFeeBasisPoints, referralFeeBasisPoint *uint256.Int) *uint256.Int {
	sellLimit := integralFloor(
		new(uint256.Int).Div(totalSupply, granularityScaler),
		new(uint256.Int),
		a, b, curveScaler,
	)
	fee := getFee(
		sellLimit,
		protocolFeeBasisPoint,
		creatorFeeBasisPoints,
		referralFeeBasisPoint,
	)

	return sellLimit.Sub(sellLimit, fee)
}
