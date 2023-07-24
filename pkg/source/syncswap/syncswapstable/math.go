package syncswapstable

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func getAmountOut(
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	swapFee *big.Int,
	tokenInPrecisionMultiplier *big.Int,
	tokenOutPrecisionMultiplier *big.Int,
) *big.Int {
	amountOut, _ := getExactQuote(swapFee, amountIn, reserveIn, reserveOut, tokenInPrecisionMultiplier, tokenOutPrecisionMultiplier)

	return amountOut
}

func getExactQuote(
	swapFee *big.Int,
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	tokenInPrecisionMultiplier *big.Int,
	tokenOutPrecisionMultiplier *big.Int,
) (*big.Int, *big.Int) {
	var amountOut *big.Int
	amountOut = big.NewInt(0)
	fee := big.NewInt(0)

	if amountIn.Cmp(bignumber.ZeroBI) <= 0 {
		return amountOut, fee
	}

	var adjustedReserveIn = new(big.Int).Mul(reserveIn, tokenInPrecisionMultiplier)
	var adjustedReserveOut = new(big.Int).Mul(reserveOut, tokenOutPrecisionMultiplier)

	var feeDeductedAmountIn, feeIn = calAmountAfterFee(amountIn, swapFee)
	var d = computeDFromAdjustedBalances(adjustedReserveIn, adjustedReserveOut)

	var x = new(big.Int).Add(adjustedReserveIn, new(big.Int).Mul(feeDeductedAmountIn, tokenInPrecisionMultiplier))
	var y = getY(x, d)

	// (adjustedReserveOut - y - 1) / tokenOutPrecisionMultiplier
	amountOut = new(big.Int).Div(new(big.Int).Sub(new(big.Int).Sub(adjustedReserveOut, y), bignumber.One), tokenOutPrecisionMultiplier)

	return amountOut, feeIn
}

func calAmountAfterFee(amountIn, swapFee *big.Int) (*big.Int, *big.Int) {
	// amountIn * (MaxFee - swapFee)
	var feeIn = new(big.Int).Div(new(big.Int).Mul(amountIn, swapFee), MaxFee)
	var feeDeductedAmountIn = new(big.Int).Sub(amountIn, feeIn)

	return feeDeductedAmountIn, feeIn
}

func computeDFromAdjustedBalances(xp0, xp1 *big.Int) *big.Int {
	var computed = big.NewInt(0)

	var s = new(big.Int).Add(xp0, xp1)
	if s.Cmp(bignumber.ZeroBI) <= 0 {
		return computed
	}

	var twoThousand = big.NewInt(2000)
	var twoThousandMinusOne = big.NewInt(1999)

	var prevD *big.Int
	var d = s

	for i := 0; i < 256; i++ {
		//uint dP = (((d * d) / xp0) * d) / xp1 / 4;
		var dp = new(big.Int).Div(
			new(big.Int).Div(
				new(big.Int).Mul(
					new(big.Int).Div(
						new(big.Int).Mul(d, d), xp0), d), xp1), bignumber.Four)
		prevD = d

		//d = (((2000 * s) + 2 * dP) * d) / ((2000 - 1) * d + 3 * dP);

		d = new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Add(
					new(big.Int).Mul(twoThousand, s),
					new(big.Int).Mul(bignumber.Two, dp),
				), d),
			new(big.Int).Add(
				new(big.Int).Mul(twoThousandMinusOne, d),
				new(big.Int).Mul(bignumber.Three, dp),
			),
		)

		if within1(d, prevD) {
			break
		}
	}
	computed = d

	return computed
}

func getY(x, d *big.Int) *big.Int {
	var twoThousand = big.NewInt(2000)
	var doubleTwoThousand = big.NewInt(4000)

	//uint c = (d * d) / (x * 2);
	var c = new(big.Int).Div(
		new(big.Int).Mul(d, d),
		new(big.Int).Mul(x, bignumber.Two),
	)
	//c = (c * d) / 4000;
	c = new(big.Int).Div(new(big.Int).Mul(c, d), doubleTwoThousand)

	//uint b = x + (d / 2000);
	var b = new(big.Int).Add(x, new(big.Int).Div(d, twoThousand))
	var yPrev *big.Int
	var y = d

	for i := 0; i < 256; i++ {
		yPrev = y
		//y = (y * y + c) / (y * 2 + b - d);
		y = new(big.Int).Div(
			new(big.Int).Add(new(big.Int).Mul(y, y), c),
			new(big.Int).Sub(new(big.Int).Add(new(big.Int).Mul(y, bignumber.Two), b), d),
		)

		if within1(y, yPrev) {
			break
		}
	}

	return y
}

func within1(a, b *big.Int) bool {
	if a.Cmp(b) > 0 {
		return new(big.Int).Sub(a, b).Cmp(bignumber.One) <= 0
	}

	return new(big.Int).Sub(b, a).Cmp(bignumber.One) < 0
}
