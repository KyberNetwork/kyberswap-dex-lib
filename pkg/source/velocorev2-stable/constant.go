package velocorev2stable

import "math/big"

const (
	DexTypeVelocoreV2Stable = "velocorev2-stable"

	registryMethodGetPools = "getPools"
	lensMethodQueryPool    = "queryPool"
	poolMethodTokenInfo    = "tokenInfo"

	defaultWeight = 1
	defaultGas    = 1000000
)

var (
	maxUint128 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
)
