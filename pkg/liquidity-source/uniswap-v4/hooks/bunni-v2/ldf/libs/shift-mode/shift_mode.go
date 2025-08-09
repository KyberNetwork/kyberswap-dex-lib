package shiftmode

// ShiftMode represents the shift mode for the distribution
type ShiftMode uint8

const (
	Both ShiftMode = iota
	Left
	Right
	Static
)

// EnforceShiftMode enforces the shift mode based on the current tick and last tick
func EnforceShiftMode(tick, lastTick int, shiftMode ShiftMode) int {
	if (shiftMode == Left && tick > lastTick) || (shiftMode == Right && tick < lastTick) {
		return lastTick
	}
	return tick
}
