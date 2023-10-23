package liquiditybookv21

import "errors"

var (
	ErrInvalidBinID   = errors.New("invalid bin id")
	ErrInvalidReserve = errors.New("invalid reserve")
	ErrInvalidToken   = errors.New("invalid token")
)
