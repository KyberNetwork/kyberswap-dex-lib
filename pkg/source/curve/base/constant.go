package base

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const MaxLoopLimit = 256

var (
	DefaultGas     = curve.Gas{Exchange: 128000}
	Precision      = bignumber.NewBig10("1000000000000000000")
	FeeDenominator = bignumber.NewBig10("10000000000")
)
