package compound

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"

const MaxLoopLimit = 256

var (
	DefaultGas     = Gas{Exchange: 285000, ExchangeUnderlying: 390000}
	FeeDenominator = bignumber.NewBig10("10000000000")
)
