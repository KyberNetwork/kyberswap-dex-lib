package zkswapfinance

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func getAmountOut(
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	tokenWeightIn uint,
	tokenWeightOut uint,
	swapFee *big.Int,
) (*big.Int, error) {
	amountAfterFee := calcAmountAfterFee(amountIn, swapFee)
	if amountAfterFee.Cmp(bignumber.ZeroBI) <= 0 {
		return bignumber.ZeroBI, ErrInsufficientInputAmount
	}

	if reserveIn.Cmp(bignumber.ZeroBI) <= 0 || reserveOut.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	numerator := new(big.Int).Mul(amountAfterFee, reserveOut)
	denominator := new(big.Int).Add(reserveIn, amountAfterFee)
	return new(big.Int).Div(numerator, denominator), nil
}

func calcAmountAfterFee(amountIn, swapFee *big.Int) *big.Int {
	// In - (fee*In)/Bone
	return new(big.Int).Sub(amountIn, new(big.Int).Div(new(big.Int).Mul(swapFee, amountIn), bignumber.BONE))
}
