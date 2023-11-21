package stablemeta

import "github.com/holiman/uint256"

var _AMP_PRECISION = uint256.NewInt(1000)

var (
	defaultGas = Gas{Swap: 10}
)

const (
	poolTypeStable     = "Stable"
	poolTypeMetaStable = "MetaStable"

	poolTypeVersion1 = 1
	poolTypeVersion2 = 2
)
