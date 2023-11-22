package composable

import "github.com/holiman/uint256"

var (
	_AMP_PRECISION = uint256.NewInt(1000)

	DefaultGas = Gas{Swap: 80000}
)

const (
	unknownInt = -1
)

const (
	poolTypeVersion1 = 1
	poolTypeVersion5 = 5
)
