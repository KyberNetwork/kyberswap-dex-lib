package syncswapv2stable

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/holiman/uint256"
)

// https://github.com/syncswap/core-contracts/blob/5285a3a7b2b00ca8b7ffc5ae5ce6f6c6195e4aa7/contracts/pool/stable/SyncSwapStablePool.sol#L494
func getAmountOut(
	amountIn *uint256.Int,
	reserveIn *uint256.Int,
	reserveOut *uint256.Int,
	swapFee *uint256.Int,
	tokenInPrecisionMultiplier *uint256.Int,
	tokenOutPrecisionMultiplier *uint256.Int,
	A *uint256.Int,
) *big.Int {
	amountOut, _ := getExactQuote(swapFee, amountIn, reserveIn, reserveOut, tokenInPrecisionMultiplier, tokenOutPrecisionMultiplier, A)

	return amountOut.ToBig()
}

// https://github.com/syncswap/core-contracts/blob/5285a3a7b2b00ca8b7ffc5ae5ce6f6c6195e4aa7/contracts/pool/stable/SyncSwapStablePool.sol#L525
func _getAmountIn(
	swapFee *uint256.Int,
	amountOut *uint256.Int,
	reserveIn *uint256.Int,
	reserveOut *uint256.Int,
	tokenInPrecisionMultiplier *uint256.Int,
	tokenOutPrecisionMultiplier *uint256.Int,
	A *uint256.Int,
) *big.Int {
	if amountOut.Cmp(uint256.NewInt(0)) <= 0 {
		return integer.Zero()
	}

	var adjustedReserveIn = new(uint256.Int).Mul(reserveIn, tokenInPrecisionMultiplier)
	var adjustedReserveOut = new(uint256.Int).Mul(reserveOut, tokenOutPrecisionMultiplier)
	var d = computeDFromAdjustedBalances(adjustedReserveIn, adjustedReserveOut, A)

	y := new(uint256.Int).Sub(adjustedReserveOut, new(uint256.Int).Mul(amountOut, tokenOutPrecisionMultiplier))
	if y.Cmp(uint256.NewInt(1)) <= 0 {
		return integer.One()
	}

	x := getY(y, d)

	// amountIn = MAX_FEE * (x - adjustedReserveIn) / (MAX_FEE - swapFee) + 1;
	// amountIn /= tokenInPrecisionMultiplier;
	amountIn := new(uint256.Int).Add(
		new(uint256.Int).Div(
			new(uint256.Int).Mul(
				MaxFee,
				new(uint256.Int).Sub(x, adjustedReserveIn),
			),
			new(uint256.Int).Sub(MaxFee, swapFee),
		),
		uint256.NewInt(1),
	)
	amountIn.Div(amountIn, tokenInPrecisionMultiplier)

	return amountIn.ToBig()
}

func getExactQuote(
	swapFee *uint256.Int,
	amountIn *uint256.Int,
	reserveIn *uint256.Int,
	reserveOut *uint256.Int,
	tokenInPrecisionMultiplier *uint256.Int,
	tokenOutPrecisionMultiplier *uint256.Int,
	A *uint256.Int,
) (*uint256.Int, *uint256.Int) {
	var amountOut *uint256.Int
	amountOut = uint256.NewInt(0)
	fee := uint256.NewInt(0)

	if amountIn.Cmp(uint256.NewInt(0)) <= 0 {
		return amountOut, fee
	}

	var adjustedReserveIn = new(uint256.Int).Mul(reserveIn, tokenInPrecisionMultiplier)
	var adjustedReserveOut = new(uint256.Int).Mul(reserveOut, tokenOutPrecisionMultiplier)

	var feeDeductedAmountIn, feeIn = calAmountAfterFee(amountIn, swapFee)
	var d = computeDFromAdjustedBalances(adjustedReserveIn, adjustedReserveOut, A)

	var x = new(uint256.Int).Add(adjustedReserveIn, new(uint256.Int).Mul(feeDeductedAmountIn, tokenInPrecisionMultiplier))
	var y = getY(x, d)

	// (adjustedReserveOut - y - 1) / tokenOutPrecisionMultiplier
	amountOut = new(uint256.Int).Div(new(uint256.Int).Sub(new(uint256.Int).Sub(adjustedReserveOut, y), uint256.NewInt(1)), tokenOutPrecisionMultiplier)

	return amountOut, feeIn
}

func calAmountAfterFee(amountIn, swapFee *uint256.Int) (*uint256.Int, *uint256.Int) {
	// amountIn * (MaxFee - swapFee)
	var feeIn = new(uint256.Int).Div(new(uint256.Int).Mul(amountIn, swapFee), MaxFee)
	var feeDeductedAmountIn = new(uint256.Int).Sub(amountIn, feeIn)

	return feeDeductedAmountIn, feeIn
}

func computeDFromAdjustedBalances(xp0, xp1 *uint256.Int, A *uint256.Int) *uint256.Int {
	var computed = uint256.NewInt(0)

	var s = new(uint256.Int).Add(xp0, xp1)
	if s.Cmp(uint256.NewInt(0)) <= 0 {
		return computed
	}

	// var twoThousand = big.NewInt(2000)
	// var twoThousandMinusOne = big.NewInt(1999)

	var prevD *uint256.Int
	var d = s

	for i := 0; i < 256; i++ {
		//uint dP = (((d * d) / xp0) * d) / xp1 / 4;
		var dp = new(uint256.Int).Div(
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					new(uint256.Int).Div(
						new(uint256.Int).Mul(d, d), xp0), d), xp1), uint256.NewInt(4))
		prevD = d

		//d = (((2000 * s) + 2 * dP) * d) / ((2000 - 1) * d + 3 * dP);

		d = new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Add(
					new(uint256.Int).Mul(A, s),
					new(uint256.Int).Mul(uint256.NewInt(2), dp),
				), d),
			new(uint256.Int).Add(
				new(uint256.Int).Mul(new(uint256.Int).Sub(A, uint256.NewInt(1)), d),
				new(uint256.Int).Mul(uint256.NewInt(3), dp),
			),
		)

		if within1(d, prevD) {
			break
		}
	}
	computed = d

	return computed
}

func getY(x, d *uint256.Int) *uint256.Int {
	var twoThousand = uint256.NewInt(2000)
	var doubleTwoThousand = uint256.NewInt(4000)

	//uint c = (d * d) / (x * 2);
	var c = new(uint256.Int).Div(
		new(uint256.Int).Mul(d, d),
		new(uint256.Int).Mul(x, uint256.NewInt(2)),
	)
	//c = (c * d) / 4000;
	c = new(uint256.Int).Div(new(uint256.Int).Mul(c, d), doubleTwoThousand)

	//uint b = x + (d / 2000);
	var b = new(uint256.Int).Add(x, new(uint256.Int).Div(d, twoThousand))
	var yPrev *uint256.Int
	var y = d

	for i := 0; i < 256; i++ {
		yPrev = y
		//y = (y * y + c) / (y * 2 + b - d);
		y = new(uint256.Int).Div(
			new(uint256.Int).Add(new(uint256.Int).Mul(y, y), c),
			new(uint256.Int).Sub(new(uint256.Int).Add(new(uint256.Int).Mul(y, uint256.NewInt(2)), b), d),
		)

		if within1(y, yPrev) {
			break
		}
	}

	return y
}

func within1(a, b *uint256.Int) bool {
	if a.Cmp(b) > 0 {
		return new(uint256.Int).Sub(a, b).Cmp(uint256.NewInt(1)) <= 0
	}

	return new(uint256.Int).Sub(b, a).Cmp(uint256.NewInt(1)) < 0
}
