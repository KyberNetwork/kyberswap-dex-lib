package hashflow

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ZeroBF = big.NewFloat(0)
)

func calcReserves(pair Pair) entity.PoolReserves {
	return entity.PoolReserves{
		calcReserve0(pair).String(),
		calcReserve1(pair).String(),
	}
}

func calcReserve0(pair Pair) *big.Int {
	if len(pair.OneToZeroPriceLevels) == 0 {
		return ZeroBI
	}

	maxAmount1In := pair.OneToZeroPriceLevels[len(pair.OneToZeroPriceLevels)-1].Level

	amountOutAfterDecimals := getMaxLiquidity(maxAmount1In, pair.OneToZeroPriceLevels)

	amountOut, _ := new(big.Float).Mul(
		amountOutAfterDecimals,
		bignumber.TenPowDecimals(pair.Decimals[0]),
	).Int(nil)

	return amountOut
}

func calcReserve1(pair Pair) *big.Int {
	if len(pair.ZeroToOnePriceLevels) == 0 {
		return ZeroBI
	}

	maxAmount0In := pair.ZeroToOnePriceLevels[len(pair.ZeroToOnePriceLevels)-1].Level

	amountOutAfterDecimals := getMaxLiquidity(maxAmount0In, pair.ZeroToOnePriceLevels)

	amountOut, _ := new(big.Float).Mul(
		amountOutAfterDecimals,
		bignumber.TenPowDecimals(pair.Decimals[1]),
	).Int(nil)

	return amountOut
}

func getMaxLiquidity(maxAmountIn *big.Float, priceLevels []PriceLevel) *big.Float {
	if len(priceLevels) == 0 {
		return ZeroBF
	}

	if maxAmountIn.Cmp(priceLevels[0].Level) < 0 {
		return ZeroBF
	}

	if maxAmountIn.Cmp(priceLevels[len(priceLevels)-1].Level) > 0 {
		return ZeroBF
	}

	amountOut := ZeroBF
	amountLeft := maxAmountIn
	currentLevelIdx := 0

	for {
		previousLevel := ZeroBF
		if currentLevelIdx > 0 {
			previousLevel = priceLevels[currentLevelIdx-1].Level
		}

		currentLevelAmount := new(big.Float).Sub(priceLevels[currentLevelIdx].Level, previousLevel)
		if currentLevelAmount.Cmp(amountLeft) > 0 {
			currentLevelAmount = amountLeft
		}

		amountOut = new(big.Float).Add(amountOut, new(big.Float).Mul(currentLevelAmount, priceLevels[currentLevelIdx].Price))
		amountLeft = new(big.Float).Sub(amountLeft, currentLevelAmount)
		currentLevelIdx++

		if amountLeft.Cmp(ZeroBF) == 0 {
			break
		}

		if currentLevelIdx > len(priceLevels)-1 {
			break
		}
	}

	return amountOut
}
