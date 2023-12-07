package wombatstable

import "math/big"

const (
	DexType = "velocore-v2-wombat-stable"

	registryMethodGetPools = "getPools"
	lensMethodQueryPool    = "queryPool"
	poolMethodTokenInfo    = "tokenInfo"

	defaultWeight = 1
)

var (
	// (1 << 128) - 1
	maxUint128 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))

	defaultGas = Gas{Swap: 125000}
)
