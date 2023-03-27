package service

import (
	"errors"
)

var (
	ErrPoolNotFound = errors.New("pool not found")
	ErrInvalidSwap  = errors.New("invalid swap")
	ErrRPCPanic     = errors.New("rpc panic")
	ErrInvalidValue = errors.New("invalid value")
)
