package liquiditybookv21

import "errors"

var (
	ErrInvalidAmount      = errors.New("invalid amount")
	ErrInvalidReserve     = errors.New("invalid reserve")
	ErrInvalidToken       = errors.New("invalid token")
	ErrPowUnderflow       = errors.New("pow underflow")
	ErrMulDivOverflow     = errors.New("mul div overflow")
	ErrMulShiftOverflow   = errors.New("mul shift overflow")
	ErrNotFoundBinID      = errors.New("not found bin id")
	ErrFeeTooLarge        = errors.New("fee too large")
	ErrMultiplierTooLarge = errors.New("multiplier too large")
)
