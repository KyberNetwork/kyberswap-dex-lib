package swapdata

import (
	"errors"
)

var (
	ErrMarshalFailed   = errors.New("marshal failed")
	ErrUnmarshalFailed = errors.New("unmarshal failed")
)
