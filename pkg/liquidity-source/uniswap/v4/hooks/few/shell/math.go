package fewshell

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	uFeeNumerator   = uint256.NewInt(feeNumerator)
	uFeeDenominator = uint256.NewInt(feeDenominator)
	uPPM            = uint256.NewInt(ppm)
)

// getAmountOut is FewV2Math.getAmountOut (FewV2Math.sol:18-30), the Uniswap V2 formula with a 997/1000 fee.
//
// amountIn is below 2^127 (swapExactIn rejects larger ones, as SafeCast.toInt128 does) and the reserves are
// below 2^112, so amountIn*997*reserveOut < 2^249 and the products below cannot overflow.
func getAmountOut(amountIn, reserveIn, reserveOut *uint256.Int) (*uint256.Int, error) {
	if amountIn.IsZero() {
		return nil, ErrInsufficientAmount
	} else if reserveIn.IsZero() || reserveOut.IsZero() {
		return nil, ErrDegeneratePair
	}
	var amountInWithFee, numerator, denominator uint256.Int
	amountInWithFee.Mul(amountIn, uFeeNumerator)
	numerator.Mul(&amountInWithFee, reserveOut)
	denominator.Mul(reserveIn, uFeeDenominator).Add(&denominator, &amountInWithFee)
	numerator.Div(&numerator, &denominator)
	if numerator.IsZero() {
		return nil, ErrInsufficientAmount
	}
	return &numerator, nil
}

// minAmountIn is the least amountIn with getAmountOut(amountIn) >= amountOut, the exact inverse of getAmountOut.
// FewV2Math.getAmountIn (FewV2Math.sol:35-46) is floor(reserveIn*amountOut*1000 / ((reserveOut-amountOut)*997)) + 1,
// which is one more than this ceiling when the division is exact (the contract's over-supply convention).
//
// amountOut and the reserves are below 2^112, so reserveIn*amountOut*1000 < 2^234 and cannot overflow.
func minAmountIn(amountOut, reserveIn, reserveOut *uint256.Int) (*uint256.Int, error) {
	if amountOut.IsZero() {
		return nil, ErrInsufficientAmount
	} else if reserveIn.IsZero() || reserveOut.IsZero() {
		return nil, ErrDegeneratePair
	} else if !amountOut.Lt(reserveOut) {
		return nil, ErrInsufficientLiquidity
	}
	var numerator, denominator uint256.Int
	numerator.Mul(reserveIn, amountOut).Mul(&numerator, uFeeDenominator)
	denominator.Sub(reserveOut, amountOut).Mul(&denominator, uFeeNumerator)
	return big256.DivUp(&numerator, &denominator), nil
}

// protocolFee is FewV2ShellHookBase._protocolFeeAmount for exact input (FewV2ShellHookBase.sol:488-505): the fee on
// the pair's output is ceil(amount*rate/1e6), rate in ppm. amount is below 2^112 here.
func protocolFee(amount *uint256.Int, rate uint32) *uint256.Int {
	var fee uint256.Int
	if rate == 0 {
		return &fee
	}
	return big256.MulDivUp(&fee, amount, uint256.NewInt(uint64(rate)), uPPM)
}

// grossBeforeFee is the least amount whose output after protocolFee is at least net:
// amount - ceil(amount*rate/1e6) = floor(amount*(1e6-rate)/1e6), so the answer is ceil(net*1e6/(1e6-rate)).
// rate is below 1e6 (at most 4095*25).
func grossBeforeFee(net *uint256.Int, rate uint32) *uint256.Int {
	if rate == 0 {
		return new(uint256.Int).Set(net)
	}
	var gross, keep uint256.Int
	return big256.MulDivUp(&gross, net, uPPM, keep.SetUint64(ppm-uint64(rate)))
}
