package mantisswap

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexTypeMantisSwap = "mantisswap"

	mainPoolMethodLpList           = "lpList"
	mainPoolMethodLpRatio          = "lpRatio"
	mainPoolMethodBaseFee          = "baseFee"
	mainPoolMethodSwapAlloed       = "swapAllowed"
	mainPoolMethodPaused           = "paused"
	mainPoolMethodSlippageA        = "slippageA"
	mainPoolMethodSlippageN        = "slippageN"
	mainPoolMethodSlippageK        = "slippageK"
	mainPoolMethodTokenOraclePrice = "tokenOraclePrice"

	lpMethodDecimals       = "decimals"
	lpMethodAsset          = "asset"
	lpMethodLiability      = "liability"
	lpMethodLiabilityLimit = "liabilityLimit"
	lpMethodUnderlier      = "underlier"

	defaultWeight = 1
	zeroString    = "0"
)

var (
	DefaultGas = Gas{Swap: 400000}
	One18      = bignumber.TenPowInt(18)
	One        = bignumber.NewBig("0x10000000000000000")
	Ln2        = bignumber.NewBig("0xb17217f7d1cf79ac")
)
