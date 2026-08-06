package ponsv2

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// basisPointsU is basisPoints (10_000) as a *uint256.Int, matching
// PonsV2BondingCurveMath.BASIS_POINTS.
var basisPointsU = uint256.NewInt(basisPoints)

// getAmountOut ports PonsV2BondingCurveMath.getAmountOut/_amountOut exactly.
// feeBps is always 0 at PonsV2BondingCurve.buy()/sell()'s call sites (the
// curve's own feeBps/creatorTaxBps are applied separately, on the quote leg,
// outside this function) but is kept as a parameter for parity with the
// on-chain library, which also prices the curve's internal buyback swap
// through the same function with a non-zero feeBps.
func getAmountOut(amountIn, reserveIn, reserveOut *uint256.Int, feeBps uint64) (*uint256.Int, error) {
	if amountIn.IsZero() {
		return nil, ErrInsufficientInputAmount
	}
	if reserveIn.IsZero() || reserveOut.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	amountOut, err := amountOutRaw(amountIn, reserveIn, reserveOut, feeBps)
	if err != nil {
		return nil, err
	}
	if amountOut.IsZero() {
		return nil, ErrInsufficientOutputAmount
	}
	return amountOut, nil
}

// amountOutRaw ports PonsV2BondingCurveMath._amountOut:
//
//	amountInWithFee = amountIn * (BASIS_POINTS - feeBps)
//	numerator       = amountInWithFee * reserveOut
//	denominator     = reserveIn * BASIS_POINTS + amountInWithFee
//	return numerator / denominator
func amountOutRaw(amountIn, reserveIn, reserveOut *uint256.Int, feeBps uint64) (*uint256.Int, error) {
	if feeBps >= basisPoints {
		return nil, ErrInsufficientLiquidity
	}

	feeMultiplier := new(uint256.Int).SetUint64(basisPoints - feeBps)
	amountInWithFee, overflow := new(uint256.Int).MulOverflow(amountIn, feeMultiplier)
	if overflow {
		return nil, ErrOverflow
	}

	reserveInBps, overflow := new(uint256.Int).MulOverflow(reserveIn, basisPointsU)
	if overflow {
		return nil, ErrOverflow
	}
	denominator, overflow := new(uint256.Int).AddOverflow(reserveInBps, amountInWithFee)
	if overflow {
		return nil, ErrOverflow
	}
	if denominator.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	return u256.MulDivDown(new(uint256.Int), amountInWithFee, reserveOut, denominator), nil
}

// getAmountIn ports PonsV2BondingCurveMath.getAmountIn exactly:
//
//	numerator   = amountOut * reserveIn * BASIS_POINTS
//	denominator = (reserveOut - amountOut) * (BASIS_POINTS - feeBps)
//	amountIn    = numerator / denominator + 1
//
// Used only by buy()'s partial-fill clamp (pricing the input required to
// exactly exhaust sellableTokens()); the protocol has no on-chain exact-out
// entrypoint, so this is not exposed as IPoolExactOutSimulator.
func getAmountIn(amountOut, reserveIn, reserveOut *uint256.Int, feeBps uint64) (*uint256.Int, error) {
	if amountOut.IsZero() {
		return nil, ErrInsufficientOutputAmount
	}
	if reserveIn.IsZero() || reserveOut.Cmp(amountOut) <= 0 {
		return nil, ErrInsufficientLiquidity
	}
	if feeBps >= basisPoints {
		return nil, ErrInsufficientLiquidity
	}

	numeratorPart, overflow := new(uint256.Int).MulOverflow(amountOut, reserveIn)
	if overflow {
		return nil, ErrOverflow
	}

	reserveOutMinusAmountOut := new(uint256.Int).Sub(reserveOut, amountOut)
	feeMultiplier := new(uint256.Int).SetUint64(basisPoints - feeBps)
	denominator, overflow := new(uint256.Int).MulOverflow(reserveOutMinusAmountOut, feeMultiplier)
	if overflow {
		return nil, ErrOverflow
	}
	if denominator.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	amountIn := u256.MulDivDown(new(uint256.Int), numeratorPart, basisPointsU, denominator)
	return amountIn.AddUint64(amountIn, 1), nil
}

// bpsOf computes amount*bps/basisPoints rounding down, matching Solidity's
// `(spent * feeBps) / BASIS_POINTS` integer-division fee/tax calculations in
// PonsV2BondingCurve.buy()/sell().
func bpsOf(amount *uint256.Int, bps uint64) (*uint256.Int, error) {
	product, overflow := new(uint256.Int).MulOverflow(amount, new(uint256.Int).SetUint64(bps))
	if overflow {
		return nil, ErrOverflow
	}
	return product.Div(product, basisPointsU), nil
}
