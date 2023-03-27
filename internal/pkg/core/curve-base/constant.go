package curveBase

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
)

const MaxLoopLimit = 256

var (
	DefaultGas     = Gas{Exchange: 128000}
	Precision      = utils.NewBig10("1000000000000000000")
	FeeDenominator = utils.NewBig10("10000000000")
)
