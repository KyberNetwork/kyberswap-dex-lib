package syncswapv2classic

import (
	"github.com/holiman/uint256"
)

var (
	MaxFee = uint256.NewInt(100000)
)

func getAmountOut(
	amountIn *uint256.Int,
	reserveIn *uint256.Int,
	reserveOut *uint256.Int,
	swapFee *uint256.Int,
) *uint256.Int {
	amountOut, _ := getExactQuote(swapFee, amountIn, reserveIn, reserveOut)

	return amountOut
}

func getExactQuote(
	swapFee *uint256.Int,
	amountIn *uint256.Int,
	reserveIn *uint256.Int,
	reserveOut *uint256.Int,
) (*uint256.Int, *uint256.Int) {
	var amountOut, amountInWithFee, feeIn *uint256.Int
	amountOut = uint256.NewInt(0)
	fee := uint256.NewInt(0)

	if amountIn.Cmp(uint256.NewInt(0)) <= 0 {
		return amountOut, fee
	}

	amountInWithFee = calAmountAfterFee(amountIn, swapFee)
	feeIn = new(uint256.Int).Mul(amountIn, new(uint256.Int).Div(swapFee, MaxFee))

	// amountOut = (amountInWithFee * reserveOut) / (reserveIn * MAX_FEE + amountInWithFee);
	amountOut = new(uint256.Int).Div(
		new(uint256.Int).Mul(amountInWithFee, reserveOut),
		new(uint256.Int).Add(new(uint256.Int).Mul(reserveIn, MaxFee), amountInWithFee))

	return amountOut, feeIn
}

func calAmountAfterFee(amountIn, swapFee *uint256.Int) *uint256.Int {
	// amountIn * (MaxFee - swapFee)
	return new(uint256.Int).Mul(amountIn, new(uint256.Int).Sub(MaxFee, swapFee))
}

// https://github.com/syncswap/core-contracts/blob/5285a3a7b2b00ca8b7ffc5ae5ce6f6c6195e4aa7/contracts/pool/classic/SyncSwapClassicPool.sol#L506
func _getAmountIn(swapFee *uint256.Int, amountOut *uint256.Int, reserveIn *uint256.Int, reserveOut *uint256.Int) *uint256.Int {
	if amountOut.Cmp(uint256.NewInt(0)) <= 0 {
		return uint256.NewInt(0)
	}

	// amountIn = reserveIn * amountOut * MAX_FEE / ((reserveOut - amountOut) * (MAX_FEE - swapFee)) + 1
	amountIn := new(uint256.Int).Add(
		new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Mul(reserveIn, amountOut),
				MaxFee,
			),
			new(uint256.Int).Mul(
				new(uint256.Int).Sub(reserveOut, amountOut),
				new(uint256.Int).Sub(MaxFee, swapFee),
			),
		),
		uint256.NewInt(1),
	)

	return amountIn
}
