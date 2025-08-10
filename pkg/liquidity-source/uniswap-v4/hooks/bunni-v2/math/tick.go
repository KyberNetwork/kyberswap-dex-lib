package math

import "github.com/holiman/uint256"

var (
	MIN_TICK = -887272
	MAX_TICK = -MIN_TICK
)

func MinUsableTick(tickSpacing int) int {
	return (MIN_TICK / tickSpacing) * tickSpacing
}

func MaxUsableTick(tickSpacing int) int {
	return (MAX_TICK / tickSpacing) * tickSpacing
}

func RoundTick(currentTick int, tickSpacing int) (roundedTick, nextRoundedTick int) {
	compressed := currentTick / tickSpacing

	if currentTick < 0 && currentTick%tickSpacing != 0 {
		compressed--
	}

	roundedTick = compressed * tickSpacing
	nextRoundedTick = roundedTick + tickSpacing
	return
}

func RoundTickSingle(currentTick int, tickSpacing int) int {
	compressed := currentTick / tickSpacing

	if currentTick < 0 && currentTick%tickSpacing != 0 {
		compressed--
	}

	return compressed * tickSpacing
}

func WeightedSum(value0, weight0, value1, weight1 *uint256.Int) *uint256.Int {
	// (value0 * weight0 + value1 * weight1) / (weight0 + weight1)

	var result uint256.Int
	result.Mul(value0, weight0)

	var temp uint256.Int
	temp.Mul(value1, weight1)

	result.Add(&result, &temp)

	temp.Add(weight0, weight1)

	result.Div(&result, &temp)

	return &result
}
