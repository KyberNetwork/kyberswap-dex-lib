package pool

import (
	"errors"
)

type ExtensionType int

const (
	Base ExtensionType = iota
	Oracle
)

var (
	ErrZeroAmount = errors.New("zero amount")
)
