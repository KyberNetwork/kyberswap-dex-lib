package curveplainoracle

import (
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

const MaxLoopLimit = 256

var (
	DefaultGas     = Gas{Exchange: 128000}
	Precision      = constant.TenPowInt(18)
	FeeDenominator = constant.TenPowInt(10)
)
