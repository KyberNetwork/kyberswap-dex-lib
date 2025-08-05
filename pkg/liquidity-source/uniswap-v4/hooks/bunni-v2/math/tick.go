package math

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
