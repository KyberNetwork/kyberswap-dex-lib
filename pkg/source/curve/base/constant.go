package base

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const MaxLoopLimit = 256

var (
	DefaultGas     = curve.Gas{Exchange: 128000}
	Precision      = big256.New("1000000000000000000")
	FeeDenominator = big256.New("10000000000")
)
