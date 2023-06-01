package dmm

import "math/big"

const (
	DexTypeDMM         = "dmm"
	defaultTokenWeight = 50
	zeroString         = "0"
)

const (
	factoryMethodGetPool        = "allPools"
	factoryMethodAllPoolsLength = "allPoolsLength"
	poolMethodToken0            = "token0"
	poolMethodToken1            = "token1"
	poolMethodGetTradeInfo      = "getTradeInfo"
)

var (
	defaultGas = Gas{SwapBase: 65000, SwapNonBase: 104000}
	zeroBI     = big.NewInt(0)
	bONE       = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)
