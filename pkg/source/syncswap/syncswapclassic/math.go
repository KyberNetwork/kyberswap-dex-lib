package syncswapclassic

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func getAmountOut(
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	swapFee *big.Int,
) *big.Int {
	amountOut, _ := getExactQuote(swapFee, amountIn, reserveIn, reserveOut)

	return amountOut
}

func getExactQuote(
	swapFee *big.Int,
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
) (*big.Int, *big.Int) {
	var amountOut, amountInWithFee, feeIn *big.Int
	amountOut = big.NewInt(0)
	fee := big.NewInt(0)

	if amountIn.Cmp(bignumber.ZeroBI) <= 0 {
		return amountOut, fee
	}

	amountInWithFee = calAmountAfterFee(amountIn, swapFee)
	feeIn = new(big.Int).Mul(amountIn, new(big.Int).Div(swapFee, MaxFee))

	// amountOut = (amountInWithFee * reserveOut) / (reserveIn * MAX_FEE + amountInWithFee);
	amountOut = new(big.Int).Div(
		new(big.Int).Mul(amountInWithFee, reserveOut),
		new(big.Int).Add(new(big.Int).Mul(reserveIn, MaxFee), amountInWithFee))

	return amountOut, feeIn
}

func calAmountAfterFee(amountIn, swapFee *big.Int) *big.Int {
	// amountIn * (MaxFee - swapFee)
	return new(big.Int).Mul(amountIn, new(big.Int).Sub(MaxFee, swapFee))
}
