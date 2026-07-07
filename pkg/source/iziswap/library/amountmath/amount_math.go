package amountmath

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/calc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/utils"
)

func GetAmountY(
	liquidity *uint256.Int,
	sqrtPriceL96 *uint256.Int,
	sqrtPriceR96 *uint256.Int,
	sqrtRate96 *uint256.Int,
	upper bool,
) *uint256.Int {
	var amount *uint256.Int
	numerator := new(uint256.Int).Sub(sqrtPriceR96, sqrtPriceL96)
	denominator := new(uint256.Int).Sub(sqrtRate96, utils.Pow96)
	if !upper {
		// You should replace MulDivMath.mulDivFloor with equivalent Go function
		amount = calc.MulDivFloor(liquidity, numerator, denominator)
	} else {
		// You should replace MulDivMath.mulDivCeil with equivalent Go function
		amount = calc.MulDivCeil(liquidity, numerator, denominator)
	}
	return amount
}

func GetAmountX(
	liquidity *uint256.Int,
	leftPt int,
	rightPt int,
	sqrtPriceR96 *uint256.Int,
	sqrtRate96 *uint256.Int,
	upper bool,
) *uint256.Int {
	var amount *uint256.Int
	// You should replace LogPowMath.getSqrtPrice with equivalent Go function
	sqrtPricePrPl96, _ := calc.GetSqrtPrice(rightPt - leftPt)

	temp := new(uint256.Int).Mul(sqrtPriceR96, utils.Pow96)
	sqrtPricePrM196 := temp.Div(temp, sqrtRate96)

	numerator := new(uint256.Int).Sub(sqrtPricePrPl96, utils.Pow96)
	denominator := sqrtPricePrM196.Sub(sqrtPriceR96, sqrtPricePrM196)
	if !upper {
		// You should replace MulDivMath.mulDivFloor with equivalent Go function
		amount = calc.MulDivFloor(liquidity, numerator, denominator)
	} else {
		// You should replace MulDivMath.mulDivCeil with equivalent Go function
		amount = calc.MulDivCeil(liquidity, numerator, denominator)
	}
	return amount
}
