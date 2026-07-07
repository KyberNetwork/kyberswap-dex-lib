package meta

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const MaxLoopLimit = 255

var (
	DefaultGas     = curve.Gas{Exchange: 145000, ExchangeUnderlying: 260000}
	FeeDenominator = big256.New("10000000000")
	Precision      = big256.New("1000000000000000000")
)
