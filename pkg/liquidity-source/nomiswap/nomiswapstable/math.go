package nomiswapstable

import (
	"time"

	"github.com/holiman/uint256"
)

func getAmountOut(
	amountIn *uint256.Int,
	reserveIn *uint256.Int,
	reserveOut *uint256.Int,
	swapFee *uint256.Int,
	tokenInPrecisionMultiplier *uint256.Int,
	tokenOutPrecisionMultiplier *uint256.Int,
	A *uint256.Int,
	// futureATime int64,
	// futureA *uint256.Int,
	// initialATime int64,
	// initialA *uint256.Int,
) *uint256.Int {
	// A := getA(futureATime, futureA, initialATime, initialA)
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

	if amountIn.Cmp(Zero) <= 0 {
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
	N_A := new(uint256.Int).Mul(A, Four)

	// c = (D * D) / (x * 2)
	var c = new(uint256.Int).Div(
		new(uint256.Int).Mul(d, d),
		new(uint256.Int).Mul(x, Two),
	)

	// c = (c * D) / ((N_A * 2) / A_PRECISION)
	c = new(uint256.Int).Div(
		new(uint256.Int).Mul(c, d),
		new(uint256.Int).Div(
			new(uint256.Int).Mul(N_A, Two),
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
		den := new(uint256.Int).Sub(new(uint256.Int).Add(new(uint256.Int).Mul(y, Two), b), d)

		y = new(uint256.Int).Div(num, den)
		if new(uint256.Int).Mod(num, den).Cmp(Zero) != 0 {
			y.Add(y, One)
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

func getA(
	futureATime int64,
	futureA *uint256.Int,
	initialATime int64,
	initialA *uint256.Int,
) *uint256.Int {

	var t1 = futureATime
	var a1 = futureA
	var now = time.Now().Unix()
	if t1 > now {
		var t0 = initialATime
		var a0 = initialA

		uint256.NewInt(uint64(now - t0))
		if a1.Cmp(a0) > 0 {
			return new(uint256.Int).Add(
				a0,
				new(uint256.Int).Div(
					new(uint256.Int).Mul(
						new(uint256.Int).Sub(a1, a0),
						uint256.NewInt(uint64(now-t0)),
					),
					uint256.NewInt(uint64(t1-t0)),
				),
			)
		} else {
			return new(uint256.Int).Sub(
				a0,
				new(uint256.Int).Div(
					new(uint256.Int).Mul(
						new(uint256.Int).Sub(a0, a1),
						uint256.NewInt(uint64(now-t0)),
					),
					uint256.NewInt(uint64(t1-t0)),
				),
			)
		}
	}
	return a1
}

func computeDFromAdjustedBalances(xp0, xp1, A *uint256.Int) *uint256.Int {
	var computed = uint256.NewInt(0)
	var s = new(uint256.Int).Add(xp0, xp1)
	N_A := new(uint256.Int).Mul(A, Four)
	if s.Cmp(Zero) == 0 {
		return computed
	}
	var prevD *uint256.Int
	var d = s

	for i := 0; i < MAX_LOOP_LIMIT; i++ {
		var dp = new(uint256.Int).Div(
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					new(uint256.Int).Div(
						new(uint256.Int).Mul(d, d), xp0), d), xp1), Four)
		prevD = d

		// D = (((N_A * s) / A_PRECISION + 2 * dP) * D) / ((N_A / A_PRECISION - 1) * D + 3 * dP);
		d = new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Add(
					new(uint256.Int).Div(new(uint256.Int).Mul(N_A, s), A_PRECISION),
					new(uint256.Int).Mul(Two, dp),
				), d),
			new(uint256.Int).Add(
				new(uint256.Int).Mul(
					new(uint256.Int).Sub(new(uint256.Int).Div(N_A, A_PRECISION), One),
					d,
				),
				new(uint256.Int).Mul(Three, dp),
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
	if a.Cmp(b) > 0 {
		return new(uint256.Int).Sub(a, b).Cmp(One) <= 0
	}

	return new(uint256.Int).Sub(b, a).Cmp(One) < 0
}
