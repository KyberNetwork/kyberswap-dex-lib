package meta

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const MaxLoopLimit = 255

var (
	DefaultGas     = curve.Gas{Exchange: 145000, ExchangeUnderlying: 260000}
	FeeDenominator = bignumber.NewBig10("10000000000")
	Precision      = bignumber.NewBig10("1000000000000000000")
)
