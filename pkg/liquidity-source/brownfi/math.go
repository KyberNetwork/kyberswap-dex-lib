package brownfi

import (
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func calcDelta(amountIn, reserveOut, kappa, oPrice *uint256.Int, isSell bool) *uint256.Int {
	var tmp1, tmp2, tmp3 uint256.Int
	if isSell { // tmp1 = (P * dx - y)^2
		_ = v3Utils.MulDivV2(oPrice, amountIn, q128, &tmp1, nil)
		tmp1.Abs(tmp1.Sub(reserveOut, &tmp1))
	} else { // tmp1 = (P * x - dy)^2
		_ = v3Utils.MulDivV2(oPrice, reserveOut, q128, &tmp1, nil)
		tmp1.Abs(tmp1.Sub(amountIn, &tmp1))
	}
	tmp1.Mul(&tmp1, &tmp1)
	// tmp2 = 2 * P * K * y * dx
	_ = v3Utils.MulDivV2(oPrice, amountIn, q128, &tmp2, nil)
	_ = v3Utils.MulDivV2(kappa, reserveOut, q128, &tmp3, nil)
	tmp2.Mul(&tmp2, &tmp3).Mul(&tmp2, big256.U2)
	return tmp1.Add(&tmp1, &tmp2)
}

func getAmountOut(amountIn, reserveOut, kappa, oPrice, fee, feePrecision *uint256.Int, zeroForOne bool) *uint256.Int {
	var numerator, denominator uint256.Int
	if kappa.Cmp(q128x2) == 0 {
		if zeroForOne {
			// dy = P * y * dx / (P * dx + y)
			_ = v3Utils.MulDivV2(oPrice, reserveOut, q128, &numerator, nil)
			_ = v3Utils.MulDivRoundingUpV2(oPrice, amountIn, q128, &denominator)
			numerator.Mul(&numerator, amountIn)
			denominator.Add(&denominator, reserveOut)
		} else {
			// dx = (x * dy) / (P * x + dy)
			numerator.Mul(amountIn, reserveOut)
			_ = v3Utils.MulDivRoundingUpV2(oPrice, reserveOut, q128, &denominator)
			denominator.Add(&denominator, amountIn)
		}
		numerator.Div(&numerator, &denominator)
	} else {
		delta := calcDelta(amountIn, reserveOut, kappa, oPrice, zeroForOne)
		if zeroForOne {
			// P * dx + y - sqrt(delta)
			_ = v3Utils.MulDivV2(oPrice, amountIn, q128, &numerator, nil)
			numerator.Add(&numerator, reserveOut).Sub(&numerator, delta.Sqrt(delta))
			// (2 - K)
			denominator.Sub(q128x2, kappa)
		} else {
			// P * x + dy - sqrt(delta)
			_ = v3Utils.MulDivV2(oPrice, reserveOut, q128, &numerator, nil)
			numerator.Add(&numerator, amountIn).Sub(&numerator, delta.Sqrt(delta))
			// P * (2 - K)
			_ = v3Utils.MulDivV2(oPrice, denominator.Sub(q128x2, kappa), q128, &denominator, nil)
		}
		_ = v3Utils.MulDivV2(&numerator, q128, &denominator, &numerator, nil)
	}

	// Apply fee: amountOut * (FEE_DENOMINATOR - fee) / FEE_DENOMINATOR
	_ = v3Utils.MulDivV2(&numerator, denominator.Sub(feePrecision, fee), feePrecision, &numerator, nil)
	return &numerator
}
