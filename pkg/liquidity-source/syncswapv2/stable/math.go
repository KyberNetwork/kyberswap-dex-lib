package syncswapv2stable

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	ErrReserveViolation = errors.New("reserve violation")
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
) (*big.Int, *big.Int) {
	amountOut, feeDeductedAmountIn := getExactQuote(swapFee, amountIn, reserveIn, reserveOut, tokenInPrecisionMultiplier, tokenOutPrecisionMultiplier, A)

	return amountOut.ToBig(), feeDeductedAmountIn.ToBig()
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
	if amountOut.Cmp(big256.U0) <= 0 {
		return integer.Zero()
	}

	var adjustedReserveIn = new(uint256.Int).Mul(reserveIn, tokenInPrecisionMultiplier)
	var adjustedReserveOut = new(uint256.Int).Mul(reserveOut, tokenOutPrecisionMultiplier)
	var d = computeDFromAdjustedBalances(adjustedReserveIn, adjustedReserveOut, A)

	y := new(uint256.Int).Sub(adjustedReserveOut, new(uint256.Int).Mul(amountOut, tokenOutPrecisionMultiplier))
	if y.Cmp(big256.U1) <= 0 {
		return integer.One()
	}

	x := getY(y, d, A)

	// amountIn = MAX_FEE * (x - adjustedReserveIn) / (MAX_FEE - swapFee) + 1;
	// amountIn /= tokenInPrecisionMultiplier;
	amountIn := new(uint256.Int)
	amountIn.Sub(x, adjustedReserveIn).Mul(amountIn, MaxFee).Div(
		amountIn,
		new(uint256.Int).Sub(MaxFee, swapFee),
	).Add(amountIn, big256.U1).Div(amountIn, tokenInPrecisionMultiplier)

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
	amountOut := uint256.NewInt(0)
	fee := uint256.NewInt(0)

	if amountIn.Cmp(big256.U0) <= 0 {
		return amountOut, fee
	}

	var adjustedReserveIn = new(uint256.Int).Mul(reserveIn, tokenInPrecisionMultiplier)
	var adjustedReserveOut = new(uint256.Int).Mul(reserveOut, tokenOutPrecisionMultiplier)

	var feeDeductedAmountIn, _ = calAmountAfterFee(amountIn, swapFee)
	var d = computeDFromAdjustedBalances(adjustedReserveIn, adjustedReserveOut, A)

	var x = new(uint256.Int).Add(adjustedReserveIn, new(uint256.Int).Mul(feeDeductedAmountIn, tokenInPrecisionMultiplier))
	var y = getY(x, d, A)

	// (adjustedReserveOut - y - 1) / tokenOutPrecisionMultiplier
	if adjustedReserveOut.Cmp(new(uint256.Int).Add(y, big256.U1)) < 0 {
		return nil, big256.U0
	}
	amountOut.Set(adjustedReserveOut).Sub(amountOut, y).Sub(amountOut, big256.U1).Div(amountOut, tokenOutPrecisionMultiplier)
	return amountOut, feeDeductedAmountIn
}

func calAmountAfterFee(amountIn, swapFee *uint256.Int) (*uint256.Int, *uint256.Int) {
	// amountIn * (MaxFee - swapFee)
	feeIn := new(uint256.Int)
	feeIn.Set(amountIn).Mul(feeIn, swapFee).Div(feeIn, MaxFee)
	feeDeductedAmountIn := new(uint256.Int).Sub(amountIn, feeIn)

	return feeDeductedAmountIn, feeIn
}

func computeDFromAdjustedBalances(xp0, xp1 *uint256.Int, A *uint256.Int) *uint256.Int {
	nA := new(uint256.Int).Mul(A, big256.U2)
	var computed = uint256.NewInt(0)

	var s = new(uint256.Int).Add(xp0, xp1)
	if s.Cmp(big256.U0) <= 0 {
		return computed
	}

	var prevD *uint256.Int
	var d = s

	for i := 0; i < 256; i++ {
		//uint dP = (((d * d) / xp0) * d) / xp1 / 4;
		dp := new(uint256.Int)
		dp.Set(d).Mul(dp, d).Div(dp, xp0).Mul(dp, d).Div(dp, xp1).Div(dp, big256.U4)
		prevD = d

		//d = (((2000 * s) + 2 * dP) * d) / ((2000 - 1) * d + 3 * dP);
		num := new(uint256.Int)
		den := new(uint256.Int)
		d = num.Mul(nA, s).Add(
			num,
			new(uint256.Int).Mul(big256.U2, dp),
		).Mul(num, d).Div(
			num,
			den.Sub(nA, big256.U1).Mul(den, d).Add(
				den,
				new(uint256.Int).Mul(big256.U3, dp),
			),
		)
		if within1(d, prevD) {
			break
		}
	}
	computed = d

	return computed
}

func getY(x, d, A *uint256.Int) *uint256.Int {
	nA := new(uint256.Int).Mul(A, big256.U2)

	c := new(uint256.Int)
	//uint c = (d * d) / (x * 2);
	//c = (c * d) / 4000;
	c.Div(
		c.Set(d).Mul(c, d),
		new(uint256.Int).Mul(x, big256.U2),
	).Mul(c, d).Div(c, nA).Div(c, big256.U2)

	b := new(uint256.Int)
	//uint b = x + (d / 2000);
	b.Set(d).Div(b, nA).Add(b, x)
	var yPrev *uint256.Int
	var y = d

	for i := 0; i < 256; i++ {
		yPrev = y
		//y = (y * y + c) / (y * 2 + b - d);
		yNum := new(uint256.Int)
		yDen := new(uint256.Int)
		y = new(uint256.Int).Div(
			yNum.Add(yNum.Set(y).Mul(yNum, yNum), c),
			yDen.Sub(yDen.Add(yDen.Set(y).Mul(yDen, big256.U2), b), d),
		)

		if within1(y, yPrev) {
			break
		}
	}

	return y
}

func within1(a, b *uint256.Int) bool {
	if a.Cmp(b) > 0 {
		return new(uint256.Int).Sub(a, b).Cmp(big256.U1) <= 0
	}

	return new(uint256.Int).Sub(b, a).Cmp(big256.U1) < 0
}
