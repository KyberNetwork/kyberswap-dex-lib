package printr

import (
	"github.com/holiman/uint256"
)

var bipsScalar = uint256.NewInt(10000)

// BuyCostResult holds the output of CalcBuyCost.
type BuyCostResult struct {
	AvailableAmount *uint256.Int
	Cost            *uint256.Int
	Fee             *uint256.Int
}

// CalcBuyCost replicates Solidity _estimateTokenCost (without priceLimit).
// Given a tokenAmount to buy, returns the cost including fee.
//
// Solidity reference (PrintrTrading.sol:333-409):
//
//	initialTokenReserve = maxTokenSupply / totalCurves
//	curveConstant       = virtualReserve * initialTokenReserve
//	tokenReserve        = curveConstant / (virtualReserve + reserve)
//	curveCost           = curveConstant / (tokenReserve - tokenAmount) - virtualReserve - reserve
//	fee                 = curveCost * tradingFee / BIPS_SCALAR  (min 1 if tradingFee > 0)
func CalcBuyCost(
	maxTokenSupply *uint256.Int,
	totalCurves uint16,
	virtualReserve *uint256.Int,
	reserve *uint256.Int,
	completionThreshold *uint256.Int,
	tradingFee uint16,
	tokenAmount *uint256.Int,
) *BuyCostResult {
	if totalCurves == 0 {
		return &BuyCostResult{AvailableAmount: new(uint256.Int), Cost: new(uint256.Int), Fee: new(uint256.Int)}
	}

	// initialTokenReserve = maxTokenSupply / totalCurves
	initialTokenReserve := new(uint256.Int).Div(maxTokenSupply, uint256.NewInt(uint64(totalCurves)))

	// curveConstant = virtualReserve * initialTokenReserve
	curveConstant := new(uint256.Int).Mul(virtualReserve, initialTokenReserve)

	// tokenReserve = curveConstant / (virtualReserve + reserve)
	vPlusR := new(uint256.Int).Add(virtualReserve, reserve)
	tokenReserve := new(uint256.Int).Div(curveConstant, vPlusR)

	// issuedSupply = initialTokenReserve - tokenReserve
	issuedSupply := new(uint256.Int).Sub(initialTokenReserve, tokenReserve)

	// Cap tokenAmount by completionThreshold
	// completionAmount = completionThreshold (already in token units from getCurve)
	availableAmount := new(uint256.Int).Set(tokenAmount)
	if new(uint256.Int).Add(issuedSupply, tokenAmount).Gt(completionThreshold) {
		if completionThreshold.Gt(issuedSupply) {
			availableAmount.Sub(completionThreshold, issuedSupply)
		} else {
			availableAmount.Clear()
		}
		tokenAmount = new(uint256.Int).Set(availableAmount)
	}

	if tokenAmount.IsZero() {
		return &BuyCostResult{
			AvailableAmount: availableAmount,
			Cost:            new(uint256.Int),
			Fee:             new(uint256.Int),
		}
	}

	// curveCost = curveConstant / (tokenReserve - tokenAmount) - virtualReserve - reserve
	newTokenReserve := new(uint256.Int).Sub(tokenReserve, tokenAmount)
	quotient := new(uint256.Int).Div(curveConstant, newTokenReserve)
	curveCost := new(uint256.Int).Sub(quotient, virtualReserve)
	curveCost.Sub(curveCost, reserve)

	// Round up check: if (curveConstant / newTokenReserve) * newTokenReserve < curveConstant
	checkProduct := new(uint256.Int).Mul(quotient, newTokenReserve)
	if checkProduct.Lt(curveConstant) {
		curveCost.AddUint64(curveCost, 1)
	}

	// fee = (curveCost * tradingFee) / BIPS_SCALAR
	fee := new(uint256.Int)
	if tradingFee > 0 {
		fee.Mul(curveCost, uint256.NewInt(uint64(tradingFee)))
		fee.Div(fee, bipsScalar)
		if fee.IsZero() {
			fee.SetUint64(1)
		}
	}

	// cost = curveCost + fee
	cost := new(uint256.Int).Add(curveCost, fee)

	return &BuyCostResult{
		AvailableAmount: availableAmount,
		Cost:            cost,
		Fee:             fee,
	}
}

// CalcBuyTokenAmount replicates Solidity _quoteTokenAmount.
// Given a baseSpend, returns the number of tokens receivable.
//
// Solidity reference (PrintrTrading.sol:421-444):
//
//	curveBudget = (BIPS_SCALAR * baseSpend) / (BIPS_SCALAR + tradingFee)
//	tokenAmount = tokenReserve - curveConstant / (virtualReserve + reserve + curveBudget)
//	Round down if remainder, to prevent precision attacks.
func CalcBuyTokenAmount(
	maxTokenSupply *uint256.Int,
	totalCurves uint16,
	virtualReserve *uint256.Int,
	reserve *uint256.Int,
	tradingFee uint16,
	baseSpend *uint256.Int,
) *uint256.Int {
	if totalCurves == 0 {
		return new(uint256.Int)
	}

	// initialTokenReserve = maxTokenSupply / totalCurves
	initialTokenReserve := new(uint256.Int).Div(maxTokenSupply, uint256.NewInt(uint64(totalCurves)))

	// curveConstant = virtualReserve * initialTokenReserve
	curveConstant := new(uint256.Int).Mul(virtualReserve, initialTokenReserve)

	// tokenReserve = curveConstant / (virtualReserve + reserve)
	vPlusR := new(uint256.Int).Add(virtualReserve, reserve)
	tokenReserve := new(uint256.Int).Div(curveConstant, vPlusR)

	// curveBudget = (BIPS_SCALAR * baseSpend) / (BIPS_SCALAR + tradingFee)
	curveBudget := new(uint256.Int).Mul(bipsScalar, baseSpend)
	bipsTotal := new(uint256.Int).Add(bipsScalar, uint256.NewInt(uint64(tradingFee)))
	curveBudget.Div(curveBudget, bipsTotal)

	// tokenAmount = tokenReserve - curveConstant / (virtualReserve + reserve + curveBudget)
	denominator := new(uint256.Int).Add(vPlusR, curveBudget)
	quotient := new(uint256.Int).Div(curveConstant, denominator)
	tokenAmount := new(uint256.Int).Sub(tokenReserve, quotient)

	// Round down check: if (curveConstant / (tokenReserve - tokenAmount)) * (tokenReserve - tokenAmount) < curveConstant && tokenAmount != 0
	if !tokenAmount.IsZero() {
		newTokenReserve := new(uint256.Int).Sub(tokenReserve, tokenAmount)
		q2 := new(uint256.Int).Div(curveConstant, newTokenReserve)
		checkProduct := new(uint256.Int).Mul(q2, newTokenReserve)
		if checkProduct.Lt(curveConstant) {
			tokenAmount.SubUint64(tokenAmount, 1)
		}
	}

	return tokenAmount
}

// SellRefundResult holds the output of CalcSellRefund.
type SellRefundResult struct {
	TokenAmountIn *uint256.Int
	Refund        *uint256.Int
	Fee           *uint256.Int
}

// CalcSellRefund replicates Solidity _estimateTokenRefund (without priceLimit).
// Given a tokenAmount to sell, returns the refund after fee.
//
// Solidity reference (PrintrTrading.sol:460-536):
//
//	curveRefund = virtualReserve + reserve - curveConstant / (tokenReserve + tokenAmount)
//	Round down if remainder, to prevent precision attacks.
//	fee = (curveRefund * tradingFee) / BIPS_SCALAR  (min 1 if tradingFee > 0 and curveRefund > 0)
//	refund = curveRefund - fee
func CalcSellRefund(
	maxTokenSupply *uint256.Int,
	totalCurves uint16,
	virtualReserve *uint256.Int,
	reserve *uint256.Int,
	tradingFee uint16,
	tokenAmount *uint256.Int,
) *SellRefundResult {
	if totalCurves == 0 {
		return &SellRefundResult{TokenAmountIn: new(uint256.Int), Refund: new(uint256.Int), Fee: new(uint256.Int)}
	}

	// initialTokenReserve = maxTokenSupply / totalCurves
	initialTokenReserve := new(uint256.Int).Div(maxTokenSupply, uint256.NewInt(uint64(totalCurves)))

	// curveConstant = virtualReserve * initialTokenReserve
	curveConstant := new(uint256.Int).Mul(virtualReserve, initialTokenReserve)

	// tokenReserve = curveConstant / (virtualReserve + reserve)
	vPlusR := new(uint256.Int).Add(virtualReserve, reserve)
	tokenReserve := new(uint256.Int).Div(curveConstant, vPlusR)

	// Cap tokenAmount by currentIssuedSupply
	currentIssuedSupply := new(uint256.Int).Sub(initialTokenReserve, tokenReserve)
	tokenAmountIn := new(uint256.Int).Set(tokenAmount)
	if tokenAmountIn.Gt(currentIssuedSupply) {
		tokenAmountIn.Set(currentIssuedSupply)
	}

	if tokenAmountIn.IsZero() {
		return &SellRefundResult{
			TokenAmountIn: tokenAmountIn,
			Refund:        new(uint256.Int),
			Fee:           new(uint256.Int),
		}
	}

	// curveRefund = virtualReserve + reserve - curveConstant / (tokenReserve + tokenAmount)
	newTokenReserve := new(uint256.Int).Add(tokenReserve, tokenAmountIn)
	quotient := new(uint256.Int).Div(curveConstant, newTokenReserve)
	curveRefund := new(uint256.Int).Add(virtualReserve, reserve)
	curveRefund.Sub(curveRefund, quotient)

	// Round down check: if (curveConstant / newTokenReserve) * newTokenReserve < curveConstant && curveRefund != 0
	checkProduct := new(uint256.Int).Mul(quotient, newTokenReserve)
	if checkProduct.Lt(curveConstant) && !curveRefund.IsZero() {
		curveRefund.SubUint64(curveRefund, 1)
	}

	// fee = (curveRefund * tradingFee) / BIPS_SCALAR
	fee := new(uint256.Int)
	if tradingFee > 0 && !curveRefund.IsZero() {
		fee.Mul(curveRefund, uint256.NewInt(uint64(tradingFee)))
		fee.Div(fee, bipsScalar)
		if fee.IsZero() {
			fee.SetUint64(1)
		}
	}

	// refund = curveRefund - fee
	refund := new(uint256.Int).Sub(curveRefund, fee)

	return &SellRefundResult{
		TokenAmountIn: tokenAmountIn,
		Refund:        refund,
		Fee:           fee,
	}
}
