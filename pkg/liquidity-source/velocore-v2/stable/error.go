package velocorev2stable

import "errors"

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrInvariant    = errors.New("invariant")
	ErrOverflow     = errors.New("overflow")
)
