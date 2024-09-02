package integral

import (
	"log"

	"github.com/holiman/uint256"
)

func (p *PoolSimulator) tradeY(yAfter, xBefore, yBefore *uint256.Int, data []byte) (*uint256.Int, error) {
	yAfterInt := ToInt256(yAfter)
	xBeforeInt := ToInt256(xBefore)
	yBeforeInt := ToInt256(yBefore)
	averagePriceInt := ToInt256(decodePriceInfo(data))

	xTradedInt := MulInt256(SubInt256(yAfterInt, yBeforeInt), averagePriceInt)

	xAfterInt := SubInt256(xBeforeInt, NegFloorDiv(xTradedInt, p.DecimalsConverter))

	if xAfterInt.Cmp(ZERO) < 0 {
		return nil, ErrT028
	}

	return ToUint256(xAfterInt), nil
}

func (p *PoolSimulator) tradeX(xAfter, xBefore, yBefore *uint256.Int, data []byte) (*uint256.Int, error) {
	xAfterInt := ToInt256(xAfter)
	xBeforeInt := ToInt256(xBefore)
	yBeforeInt := ToInt256(yBefore)
	averagePriceInt := ToInt256(decodePriceInfo(data))

	yTradedInt := MulInt256(SubInt256(xAfterInt, xBeforeInt), averagePriceInt)

	yAfterInt := SubInt256(yBeforeInt, NegFloorDiv(yTradedInt, p.DecimalsConverter))

	log.Fatalf("----- %+v--- %+v\n", yBeforeInt, NegFloorDiv(yTradedInt, p.DecimalsConverter))

	if yAfterInt.Cmp(ZERO) < 0 {
		return nil, ErrT027
	}

	return ToUint256(yAfterInt), nil
}

func decodePriceInfo(data []byte) *uint256.Int {
	return new(uint256.Int).SetBytes(data)
}

// func getAveragePrice() {
// 	secondsAgo := twapInterval
// 	secondsAgos := []uint32{secondsAgo, 0}

// 	// Call Uniswap V3 Pool's observe function
// 	tickCumulatives, err := uniswapPair.Observe(secondsAgos)
// 	if err != nil {
// 		return 0, err
// 	}

// 	// Calculate the tick cumulatives delta
// 	tickCumulativesDelta := tickCumulatives[1].Sub(tickCumulatives[0])
// 	arithmeticMeanTick := int24(tickCumulativesDelta.Div(int64(secondsAgo)))

// 	if tickCumulativesDelta.Cmp(big.NewInt(0)) < 0 && tickCumulativesDelta.Mod(big.NewInt(int64(secondsAgo))).Cmp(big.NewInt(0)) != 0 {
// 		arithmeticMeanTick--
// 	}

// 	sqrtRatioX96 := TickMathGetSqrtRatioAtTick(arithmeticMeanTick)

// 	// If sqrtRatioX96 <= type(uint128).max
// 	if sqrtRatioX96.Cmp(big.NewInt(math.MaxUint128)) <= 0 {
// 		ratioX192 := new(big.Int).Mul(sqrtRatioX96, sqrtRatioX96)
// 		return FullMathMulDiv(ratioX192, decimalsConverter, big.NewInt(1<<192))
// 	} else {
// 		ratioX128 := FullMathMulDiv(sqrtRatioX96, sqrtRatioX96, big.NewInt(1<<64))
// 		return FullMathMulDiv(ratioX128, decimalsConverter, big.NewInt(1<<128))
// 	}
// }
