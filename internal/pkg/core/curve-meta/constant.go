package curveMeta

import (
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

const MaxLoopLimit = 255

var (
	DefaultGas     = Gas{Exchange: 145000, ExchangeUnderlying: 260000}
	FeeDenominator = utils.NewBig10("10000000000")
	Precision      = utils.NewBig10("1000000000000000000")
)
