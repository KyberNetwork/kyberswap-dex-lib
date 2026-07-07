package brownfiv2

import (
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func getAmountOut(parsedAmountIn, parsedReserveOut, priceIn, priceOut, k, fee *uint256.Int) *uint256.Int {
	var tmp, _amountIn, amountOut uint256.Int
	_ = v3Utils.MulDivV2(parsedAmountIn, precision, tmp.Add(precision, fee), &_amountIn, nil)

	if k.Cmp(q64x2) == 0 {
		// amountOut = FullMath.mulDiv(parsedReserveOut * _amountIn, priceIn, priceOut * parsedReserveOut + _amountIn * priceIn);
		_ = v3Utils.MulDivV2(amountOut.Mul(parsedReserveOut, &_amountIn), priceIn,
			tmp.Add(tmp.Mul(priceOut, parsedReserveOut), _amountIn.Mul(&_amountIn, priceIn)), &amountOut, nil)
	} else {
		var leftSqrt uint256.Int
		rightSqrt := &amountOut
		_ = v3Utils.MulDivV2(&_amountIn, priceIn, q64, &leftSqrt, nil)
		_ = v3Utils.MulDivV2(parsedReserveOut, priceOut, q64, &tmp, nil)
		leftSqrt.Sub(&leftSqrt, &tmp).Mul(&leftSqrt, &leftSqrt)
		_ = v3Utils.MulDivV2(tmp.Mul(priceIn, priceOut), k, q128, rightSqrt, nil)
		_ = v3Utils.MulDivV2(tmp.Mul(parsedReserveOut, &_amountIn), big256.U2, q64, &tmp, nil)
		rightSqrt.Mul(rightSqrt, &tmp)
		num := rightSqrt.Mul(q64, rightSqrt.Sqrt(rightSqrt.Add(&leftSqrt, rightSqrt)))
		num = leftSqrt.Sub(leftSqrt.Mul(priceOut, parsedReserveOut).Add(&leftSqrt, tmp.Mul(priceIn, &_amountIn)), num)
		_ = v3Utils.MulDivV2(priceOut, tmp.Sub(q64x2, k), q64, rightSqrt, nil)
		amountOut.Div(num, rightSqrt)
	}

	return &amountOut
}

func getAmountIn(parsedAmountOut, parsedReserveOut, priceIn, priceOut, k, fee *uint256.Int) *uint256.Int {
	var tmp, priceImpact, amountIn uint256.Int
	_ = v3Utils.MulDivV2(tmp.Mul(k, q64), parsedAmountOut,
		priceImpact.Mul(q64, priceImpact.Sub(parsedReserveOut, parsedAmountOut)), &priceImpact, nil)
	_ = v3Utils.MulDivV2(priceOut, priceImpact.Add(&priceImpact, q64x2), priceIn, &priceImpact, nil)
	_ = v3Utils.MulDivV2(parsedAmountOut, &priceImpact, q64x2, &amountIn, nil)
	_ = v3Utils.MulDivV2(&amountIn, tmp.Add(precision, fee), precision, &amountIn, nil)
	return &amountIn
}

func parseRawToDefaultDecimals(amount *uint256.Int, decimals uint8) *uint256.Int {
	var formattedAmount uint256.Int
	if decimals > parsedDecimals {
		formattedAmount.Div(amount, big256.TenPow(decimals-parsedDecimals))
	} else {
		formattedAmount.Mul(amount, big256.TenPow(parsedDecimals-decimals))
	}
	return &formattedAmount
}

func parseDefaultToRawDecimals(amount *uint256.Int, decimals uint8) *uint256.Int {
	var formattedAmount uint256.Int
	if decimals > parsedDecimals {
		formattedAmount.Mul(amount, big256.TenPow(decimals-parsedDecimals))
	} else {
		formattedAmount.Div(amount, big256.TenPow(parsedDecimals-decimals))
	}
	return &formattedAmount
}

func getSkewnessPrice(parsedReserveA, parsedReserveB, priceA, priceB, lambda *uint256.Int) (sPriceA, sPriceB *uint256.Int) {
	if lambda.Sign() == 0 {
		return priceA, priceB
	}
	var reserveAPrice, reserveBPrice, s uint256.Int
	reserveAPrice.Mul(parsedReserveA, priceA)
	reserveBPrice.Mul(parsedReserveB, priceB)
	s.Add(&reserveAPrice, &reserveBPrice)
	neg := reserveAPrice.Sub(&reserveAPrice, &reserveBPrice).Sign() < 0
	_ = v3Utils.MulDivV2(reserveAPrice.Abs(&reserveAPrice), lambda, &s, &s, nil)
	if neg {
		sPriceA, sPriceB = reserveAPrice.Add(q64, &s), reserveBPrice.Sub(q64, &s)
	} else {
		sPriceA, sPriceB = reserveAPrice.Sub(q64, &s), reserveBPrice.Add(q64, &s)
	}
	_ = v3Utils.MulDivV2(priceA, sPriceA, q64, sPriceA, nil)
	_ = v3Utils.MulDivV2(priceB, sPriceB, q64, sPriceB, nil)
	return sPriceA, sPriceB
}
