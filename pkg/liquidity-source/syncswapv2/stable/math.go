package syncswapv2stable

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
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
	if amountOut.Cmp(constant.ZeroBI) <= 0 {
		return integer.Zero()
	}

	var adjustedReserveIn = new(uint256.Int).Mul(reserveIn, tokenInPrecisionMultiplier)
	var adjustedReserveOut = new(uint256.Int).Mul(reserveOut, tokenOutPrecisionMultiplier)
	var d = computeDFromAdjustedBalances(adjustedReserveIn, adjustedReserveOut, A)

	y := new(uint256.Int).Sub(adjustedReserveOut, new(uint256.Int).Mul(amountOut, tokenOutPrecisionMultiplier))
	if y.Cmp(constant.One) <= 0 {
		return integer.One()
	}

	x := getY(y, d)

	// amountIn = MAX_FEE * (x - adjustedReserveIn) / (MAX_FEE - swapFee) + 1;
	// amountIn /= tokenInPrecisionMultiplier;
	amountIn := new(uint256.Int)
	amountIn.Sub(x, adjustedReserveIn).Mul(amountIn, MaxFee).Div(
		amountIn,
		new(uint256.Int).Sub(MaxFee, swapFee),
	).Add(amountIn, constant.One).Div(amountIn, tokenInPrecisionMultiplier)

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

	if amountIn.Cmp(constant.ZeroBI) <= 0 {
		return amountOut, fee
	}

	var adjustedReserveIn = new(uint256.Int).Mul(reserveIn, tokenInPrecisionMultiplier)
	var adjustedReserveOut = new(uint256.Int).Mul(reserveOut, tokenOutPrecisionMultiplier)

	var feeDeductedAmountIn, _ = calAmountAfterFee(amountIn, swapFee)
	var d = computeDFromAdjustedBalances(adjustedReserveIn, adjustedReserveOut, A)

	var x = new(uint256.Int).Add(adjustedReserveIn, new(uint256.Int).Mul(feeDeductedAmountIn, tokenInPrecisionMultiplier))
	var y = getY(x, d)

	// (adjustedReserveOut - y - 1) / tokenOutPrecisionMultiplier
	if adjustedReserveOut.Cmp(new(uint256.Int).Add(y, constant.One)) < 0 {
		return nil, constant.ZeroBI
	}
	amountOut.Set(adjustedReserveOut).Sub(amountOut, y).Sub(amountOut, constant.One).Div(amountOut, tokenOutPrecisionMultiplier)
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
	var computed = uint256.NewInt(0)

	var s = new(uint256.Int).Add(xp0, xp1)
	if s.Cmp(constant.ZeroBI) <= 0 {
		return computed
	}

	var prevD *uint256.Int
	var d = s

	for i := 0; i < 256; i++ {
		//uint dP = (((d * d) / xp0) * d) / xp1 / 4;
		dp := new(uint256.Int)
		dp.Set(d).Mul(dp, d).Div(dp, xp0).Mul(dp, d).Div(dp, xp1).Div(dp, constant.Four)
		prevD = d

		//d = (((2000 * s) + 2 * dP) * d) / ((2000 - 1) * d + 3 * dP);
		num := new(uint256.Int)
		den := new(uint256.Int)
		d = num.Mul(A, s).Add(
			num,
			new(uint256.Int).Mul(constant.Two, dp),
		).Mul(num, d).Div(
			num,
			den.Sub(A, constant.One).Mul(den, d).Add(
				den,
				new(uint256.Int).Mul(constant.Three, dp),
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

	c := new(uint256.Int)
	//uint c = (d * d) / (x * 2);
	//c = (c * d) / 4000;
	c.Div(
		c.Set(d).Mul(c, d),
		new(uint256.Int).Mul(x, constant.Two),
	).Mul(c, d).Div(c, doubleTwoThousand)

	b := new(uint256.Int)
	//uint b = x + (d / 2000);
	b.Set(d).Div(b, twoThousand).Add(b, x)
	var yPrev *uint256.Int
	var y = d

	for i := 0; i < 256; i++ {
		yPrev = y
		//y = (y * y + c) / (y * 2 + b - d);
		yNum := new(uint256.Int)
		yDen := new(uint256.Int)
		y = new(uint256.Int).Div(
			yNum.Add(yNum.Set(y).Mul(yNum, yNum), c),
			yDen.Sub(yDen.Add(yDen.Set(y).Mul(yDen, constant.Two), b), d),
		)

		if within1(y, yPrev) {
			break
		}
	}

	return y
}

func within1(a, b *uint256.Int) bool {
	if a.Cmp(b) > 0 {
		return new(uint256.Int).Sub(a, b).Cmp(constant.One) <= 0
	}

	return new(uint256.Int).Sub(b, a).Cmp(constant.One) < 0
}
