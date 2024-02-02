package base

import (
	"github.com/holiman/uint256"
)

const (
	MaxLoopLimit = 256

	MaxTokenCount = 10
)

var (
	DefaultGas     = Gas{Exchange: 128000}
	Precision      = uint256.MustFromDecimal("1000000000000000000")
	FeeDenominator = uint256.MustFromDecimal("10000000000")
)
