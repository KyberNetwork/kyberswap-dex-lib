package nomiswapstable

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func getAmountOut(
	amountIn *uint256.Int,
	reserveIn *uint256.Int,
	reserveOut *uint256.Int,
	swapFee *uint256.Int,
	tokenInPrecisionMultiplier *uint256.Int,
	tokenOutPrecisionMultiplier *uint256.Int,
	A *uint256.Int,
) *uint256.Int {
	amountOut, _ := getExactQuote(swapFee, amountIn, reserveIn, reserveOut, tokenInPrecisionMultiplier, tokenOutPrecisionMultiplier, A)
	return amountOut
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

	if amountIn.Sign() <= 0 {
		return amountOut, fee
	}

	var adjustedReserveIn = new(uint256.Int).Mul(reserveIn, tokenInPrecisionMultiplier)
	var adjustedReserveOut = new(uint256.Int).Mul(reserveOut, tokenOutPrecisionMultiplier)
	var feeDeductedAmountIn, feeIn = calAmountAfterFee(amountIn, swapFee)
	var d = computeDFromAdjustedBalances(adjustedReserveIn, adjustedReserveOut, A)
	var x = new(uint256.Int).Add(adjustedReserveIn, new(uint256.Int).Mul(feeDeductedAmountIn, tokenInPrecisionMultiplier))
	var y = getY(x, d, A)
	// (adjustedReserveOut - y) / tokenOutPrecisionMultiplier
	amountOut = new(uint256.Int).Div(new(uint256.Int).Sub(adjustedReserveOut, y), tokenOutPrecisionMultiplier)
	return amountOut, feeIn
}

func getY(x, d, A *uint256.Int) *uint256.Int {
	// N_A = A * 4
	N_A := new(uint256.Int).Mul(A, u256.U4)

	// c = (D * D) / (x * 2)
	var c = new(uint256.Int).Div(
		new(uint256.Int).Mul(d, d),
		new(uint256.Int).Mul(x, u256.U2),
	)

	// c = (c * D) / ((N_A * 2) / A_PRECISION)
	c = new(uint256.Int).Div(
		new(uint256.Int).Mul(c, d),
		new(uint256.Int).Div(
			new(uint256.Int).Mul(N_A, u256.U2),
			A_PRECISION,
		),
	)

	// b = x + ((D * A_PRECISION) / N_A);
	var b = new(uint256.Int).Add(
		x,
		new(uint256.Int).Div(new(uint256.Int).Mul(d, A_PRECISION), N_A),
	)

	var yPrev *uint256.Int
	var y = d

	for i := 0; i < MAX_LOOP_LIMIT; i++ {
		yPrev = y
		//y = (y * y + c) / (y * 2 + b - d);
		num := new(uint256.Int).Add(new(uint256.Int).Mul(y, y), c)
		den := new(uint256.Int).Sub(new(uint256.Int).Add(new(uint256.Int).Mul(y, u256.U2), b), d)

		y = new(uint256.Int).Div(num, den)
		if new(uint256.Int).Mod(num, den).Sign() != 0 {
			y.Add(y, u256.U1)
		}
		if within1(y, yPrev) {
			break
		}
	}
	return y

}
func calAmountAfterFee(amountIn, swapFee *uint256.Int) (*uint256.Int, *uint256.Int) {
	// amountIn * (MaxFee - swapFee)
	var feeIn = new(uint256.Int).Div(new(uint256.Int).Mul(amountIn, swapFee), MaxFee)
	var feeDeductedAmountIn = new(uint256.Int).Sub(amountIn, feeIn)

	return feeDeductedAmountIn, feeIn
}

func computeDFromAdjustedBalances(xp0, xp1, A *uint256.Int) *uint256.Int {
	var computed = uint256.NewInt(0)
	var s = new(uint256.Int).Add(xp0, xp1)
	N_A := new(uint256.Int).Mul(A, u256.U4)
	if s.Sign() == 0 {
		return computed
	}
	var prevD *uint256.Int
	var d = s

	for i := 0; i < MAX_LOOP_LIMIT; i++ {
		var dp = new(uint256.Int).Div(
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					new(uint256.Int).Div(
						new(uint256.Int).Mul(d, d), xp0), d), xp1), u256.U4)
		prevD = d

		// D = (((N_A * s) / A_PRECISION + 2 * dP) * D) / ((N_A / A_PRECISION - 1) * D + 3 * dP);
		d = new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Add(
					new(uint256.Int).Div(new(uint256.Int).Mul(N_A, s), A_PRECISION),
					new(uint256.Int).Mul(u256.U2, dp),
				), d),
			new(uint256.Int).Add(
				new(uint256.Int).Mul(
					new(uint256.Int).Sub(new(uint256.Int).Div(N_A, A_PRECISION), u256.U1),
					d,
				),
				new(uint256.Int).Mul(u256.U3, dp),
			),
		)
		if within1(d, prevD) {
			break
		}
	}

	computed = d

	return computed

}

func within1(a, b *uint256.Int) bool {
	if a.Gt(b) {
		return new(uint256.Int).Sub(a, b).Cmp(u256.U1) <= 0
	}

	return new(uint256.Int).Sub(b, a).Cmp(u256.U1) < 0
}
