package twamm

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
)

func computeSqrtSaleRatioX128(saleRateToken0, saleRateToken1 *big.Int) *big.Int {
	saleRatio := new(big.Int)
	saleRatio.Div(
		saleRatio.Lsh(saleRateToken1, 128),
		saleRateToken0,
	)

	bitLen := saleRatio.BitLen()
	if bitLen <= 128 {
		// Full precision
		saleRatio.Sqrt(saleRatio.Lsh(saleRatio, 128))
	} else if bitLen <= 192 {
		// We know it only has 192 bits, so we can shift it 64 before rooting to get more precision
		saleRatio.Lsh(saleRatio.Sqrt(saleRatio.Lsh(saleRatio, 64)), 32)
	} else {
		saleRatio.Lsh(saleRatio.Sqrt(saleRatio.Lsh(saleRatio, 16)), 56)
	}

	return saleRatio
}

var exponentConstant = big.NewInt(12392656037)

func CalculateNextSqrtRatio(sqrtRatio, liquidity, saleRateToken0, saleRateToken1 *big.Int, timeElapsed uint32, fee uint64) *big.Int {
	sqrtSaleRatio := computeSqrtSaleRatioX128(saleRateToken0, saleRateToken1)

	if liquidity.Sign() == 0 {
		return sqrtSaleRatio
	}

	saleRate := new(big.Int)
	feeBig := new(big.Int).SetUint64(fee)

	saleRate.Div(
		saleRate.Mul(
			saleRate.Sqrt(saleRate.Mul(saleRateToken0, saleRateToken1)),
			feeBig.Sub(math.TwoPow64, feeBig),
		),
		math.TwoPow64,
	)

	exponent := feeBig.Div(
		feeBig.Mul(
			feeBig.Mul(
				saleRate,
				feeBig.SetUint64(uint64(timeElapsed)),
			),
			exponentConstant,
		),
		liquidity,
	)

	twoPowExponentX128 := exp2(exponent)
	if twoPowExponentX128 == nil {
		return sqrtSaleRatio
	}

	twoPowExponentX128.Lsh(twoPowExponentX128, 64)

	var (
		num  *big.Int
		sign bool
	)
	if sqrtRatio.Cmp(sqrtSaleRatio) == 1 {
		num, sign = exponent.Sub(sqrtRatio, sqrtSaleRatio), true
	} else {
		num, sign = exponent.Sub(sqrtSaleRatio, sqrtRatio), false
	}

	c := num.Div(
		num.Lsh(num, 128),
		saleRate.Add(sqrtSaleRatio, sqrtRatio),
	)

	term1, term2 := saleRate.Sub(twoPowExponentX128, c), twoPowExponentX128.Add(twoPowExponentX128, c)

	if sign {
		return sqrtSaleRatio.Div(
			sqrtSaleRatio.Mul(
				sqrtSaleRatio,
				term2,
			),
			term1,
		)
	} else {
		return sqrtSaleRatio.Div(
			sqrtSaleRatio.Mul(
				sqrtSaleRatio,
				term1,
			),
			term2,
		)
	}
}
