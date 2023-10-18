package velocorev2cpmm

import "errors"

var (
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidTokenGrowth = errors.New("invalid token growth")

	ErrInvalidR  = errors.New("invalid r")
	ErrNotFoundR = errors.New("not found r")
)
