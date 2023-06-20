package plainoracle

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const MaxLoopLimit = 256

var (
	DefaultGas     = Gas{Exchange: 128000}
	Precision      = bignumber.TenPowInt(18)
	FeeDenominator = bignumber.TenPowInt(10)
)
