package compound

import (
	"github.com/holiman/uint256"
)

const MaxLoopLimit = 256

var (
	DefaultGas     = Gas{Exchange: 285000, ExchangeUnderlying: 390000}
	FeeDenominator = uint256.MustFromDecimal("10000000000")
	Bone           = uint256.MustFromDecimal("1000000000000000000")
)
