package plainoracle

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
)

const MaxLoopLimit = 256

var (
	DefaultGas     = curve.Gas{Exchange: 128000}
	Precision      = uint256.MustFromDecimal("1000000000000000000")
	FeeDenominator = uint256.MustFromDecimal("10000000000")
)
