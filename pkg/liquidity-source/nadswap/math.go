package nadswap

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// getAmountOutGeneral computes the V2 constant-product formula with LP_FEE_RATE = 25 BPS.
// Used for general (non-meme) pairs. MulDivDown uses 512-bit internal arithmetic, so
// the intermediate product amountInWithFee * reserveOut cannot overflow.
func getAmountOutGeneral(amountIn, reserveIn, reserveOut *uint256.Int) (*uint256.Int, error) {
	if amountIn.IsZero() {
		return nil, ErrInsufficientInput
	}
	if reserveIn.IsZero() || reserveOut.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	// amountInWithFee = amountIn * (BPS - LP_FEE_RATE)
	var amountInWithFee, bpsMinusLp uint256.Int
	bpsMinusLp.Sub(uBPS, uLpFeeRate)
	if _, overflow := amountInWithFee.MulOverflow(amountIn, &bpsMinusLp); overflow {
		return nil, ErrOverflow
	}

	// denominator = reserveIn * BPS + amountInWithFee
	var denom, reserveInBPS uint256.Int
	if _, overflow := reserveInBPS.MulOverflow(reserveIn, uBPS); overflow {
		return nil, ErrOverflow
	}
	if _, overflow := denom.AddOverflow(&reserveInBPS, &amountInWithFee); overflow {
		return nil, ErrOverflow
	}

	var out uint256.Int
	big256.MulDivDown(&out, &amountInWithFee, reserveOut, &denom)
	return &out, nil
}

// getAmountOutMemeBuy: quote token in -> base token out.
// totalFeeRate = LP_FEE_RATE + feeRate, applied entirely on input.
func getAmountOutMemeBuy(amountIn, reserveQuote, reserveBase *uint256.Int, feeRate uint16) (*uint256.Int, error) {
	if amountIn.IsZero() {
		return nil, ErrInsufficientInput
	}
	if reserveQuote.IsZero() || reserveBase.IsZero() {
		return nil, ErrInsufficientLiquidity
	}
	totalFeeRate := uint64(LpFeeRate) + uint64(feeRate)
	if totalFeeRate >= BPS {
		return nil, ErrInvalidFeeRate
	}

	uTotal := uint256.NewInt(totalFeeRate)
	var bpsMinusTotal uint256.Int
	bpsMinusTotal.Sub(uBPS, uTotal)

	var amountInWithFee uint256.Int
	if _, overflow := amountInWithFee.MulOverflow(amountIn, &bpsMinusTotal); overflow {
		return nil, ErrOverflow
	}

	var reserveQuoteBPS, denom uint256.Int
	if _, overflow := reserveQuoteBPS.MulOverflow(reserveQuote, uBPS); overflow {
		return nil, ErrOverflow
	}
	if _, overflow := denom.AddOverflow(&reserveQuoteBPS, &amountInWithFee); overflow {
		return nil, ErrOverflow
	}

	var out uint256.Int
	big256.MulDivDown(&out, &amountInWithFee, reserveBase, &denom)
	return &out, nil
}
