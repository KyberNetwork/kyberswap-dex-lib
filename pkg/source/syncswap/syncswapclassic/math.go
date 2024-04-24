package syncswapclassic

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"

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

// https://github.com/syncswap/core-contracts/blob/5285a3a7b2b00ca8b7ffc5ae5ce6f6c6195e4aa7/contracts/pool/classic/SyncSwapClassicPool.sol#L506
func _getAmountIn(swapFee *big.Int, amountOut *big.Int, reserveIn *big.Int, reserveOut *big.Int) *big.Int {
	if amountOut.Cmp(bignumber.ZeroBI) <= 0 {
		return integer.Zero()
	}

	// amountIn = reserveIn * amountOut * MAX_FEE / ((reserveOut - amountOut) * (MAX_FEE - swapFee)) + 1
	amountIn := new(big.Int).Add(
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Mul(reserveIn, amountOut),
				MaxFee,
			),
			new(big.Int).Mul(
				new(big.Int).Sub(reserveOut, amountOut),
				new(big.Int).Sub(MaxFee, swapFee),
			),
		),
		integer.One(),
	)

	return amountIn
}
