package twamm

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var exponentConstant = uint256.NewInt(12392656037)

func computeSqrtSaleRatioX128(saleRateToken0, saleRateToken1 *uint256.Int) *uint256.Int {
	var saleRatio uint256.Int
	saleRatio.Div(
		saleRatio.Lsh(saleRateToken1, 128),
		saleRateToken0,
	)

	if bitLen := saleRatio.BitLen(); bitLen <= 128 {
		// Full precision
		return saleRatio.Sqrt(saleRatio.Lsh(&saleRatio, 128))
	} else if bitLen <= 192 {
		// We know it only has 192 bits, so we can shift it 64 before rooting to get more precision
		return saleRatio.Lsh(saleRatio.Sqrt(saleRatio.Lsh(&saleRatio, 64)), 32)
	}

	return saleRatio.Lsh(saleRatio.Sqrt(saleRatio.Lsh(&saleRatio, 16)), 56)
}

func CalculateNextSqrtRatio(sqrtRatio, liquidity, saleRateToken0, saleRateToken1 *uint256.Int, timeElapsed uint32,
	fee uint64) *uint256.Int {
	sqrtSaleRatio := computeSqrtSaleRatioX128(saleRateToken0, saleRateToken1)
	if liquidity.IsZero() {
		return sqrtSaleRatio
	}

	var saleRate, tmp uint256.Int
	saleRate.MulDivOverflow(
		saleRate.Sqrt(saleRate.Mul(saleRateToken0, saleRateToken1)),
		tmp.SubUint64(big256.U2Pow64, fee),
		big256.U2Pow64,
	)
	exponent, _ := tmp.MulDivOverflow(
		tmp.Mul(&saleRate, tmp.SetUint64(uint64(timeElapsed))),
		exponentConstant,
		liquidity,
	)

	twoPowExponentX128 := exp2(exponent)
	if twoPowExponentX128 == nil {
		return sqrtSaleRatio
	}
	twoPowExponentX128.Lsh(twoPowExponentX128, 64)

	var num *uint256.Int
	sign := sqrtRatio.Gt(sqrtSaleRatio)
	if sign {
		num = exponent.Sub(sqrtRatio, sqrtSaleRatio)
	} else {
		num = exponent.Sub(sqrtSaleRatio, sqrtRatio)
	}

	c, _ := num.MulDivOverflow(num, big256.U2Pow128, saleRate.Add(sqrtSaleRatio, sqrtRatio))
	term1, term2 := saleRate.Sub(twoPowExponentX128, c), twoPowExponentX128.Add(twoPowExponentX128, c)

	if sign {
		sqrtSaleRatio.MulDivOverflow(sqrtSaleRatio, term2, term1)
	} else {
		sqrtSaleRatio.MulDivOverflow(sqrtSaleRatio, term1, term2)
	}
	return sqrtSaleRatio
}
