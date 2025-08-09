package ldf

// ShiftMode represents the shift mode for the distribution
type ShiftMode uint8

const (
	ShiftModeBoth ShiftMode = iota
	ShiftModeLeft
	ShiftModeRight
	ShiftModeStatic
)

// EnforceShiftMode enforces the shift mode based on the current tick and last tick
func EnforceShiftMode(tick, lastTick int, shiftMode ShiftMode) int {
	if (shiftMode == ShiftModeLeft && tick > lastTick) || (shiftMode == ShiftModeRight && tick < lastTick) {
		return lastTick
	}
	return tick
}
