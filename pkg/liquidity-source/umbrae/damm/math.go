package umbraedamm

import "github.com/holiman/uint256"

var feeDenom = uint256.NewInt(feeDenominator)

// cpQuote is the constant-product output, mirroring DAMMPairUpgradeable._quote:
// out = floor(in * reserveOut / (reserveIn + in)).
func cpQuote(in, reserveIn, reserveOut *uint256.Int) *uint256.Int {
	var num, den uint256.Int
	num.Mul(in, reserveOut)
	den.Add(reserveIn, in)
	return num.Div(&num, &den)
}

// getAmountOut mirrors DAMMPairUpgradeable.getAmountOut / swap exactly. The fee is ALWAYS charged
// in feeToken: on the input when feeToken is the input side, otherwise on the output.
//
//	feeOnInput:  inAfterFee = amountIn - floor(amountIn*feeBps/FEE_DENOM); out = quote(inAfterFee)
//	feeOnOutput: outFull = quote(amountIn); fee = floor(outFull*feeBps/FEE_DENOM); out = outFull - fee
//
// Returns amountOut, the fee (denominated in feeToken), and the reserve deltas: the amount that
// joins reserveIn and the amount that leaves reserveOut (fees exit reserves into accumulators, so
// K stays constant).
func getAmountOut(amountIn, reserveIn, reserveOut, feeBps *uint256.Int, feeOnInput bool) (
	amountOut, fee, reserveInDelta, reserveOutDelta *uint256.Int,
) {
	if feeOnInput {
		fee = feeOf(amountIn, feeBps)
		inAfterFee := new(uint256.Int).Sub(amountIn, fee)
		amountOut = cpQuote(inAfterFee, reserveIn, reserveOut)
		return amountOut, fee, inAfterFee, amountOut
	}
	outFull := cpQuote(amountIn, reserveIn, reserveOut)
	fee = feeOf(outFull, feeBps)
	amountOut = new(uint256.Int).Sub(outFull, fee)
	return amountOut, fee, amountIn, outFull
}

// feeOf returns floor(amount * feeBps / FEE_DENOMINATOR).
func feeOf(amount, feeBps *uint256.Int) *uint256.Int {
	var f uint256.Int
	f.Mul(amount, feeBps)
	return f.Div(&f, feeDenom)
}
