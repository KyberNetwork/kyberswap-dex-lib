package bancorv3

import "github.com/holiman/uint256"

const (
	DexType = "bancor-v3"

	bancorNetworkMethodLiquidityPools   = "liquidityPools"
	bancorNetworkMethodCollectionByPool = "collectionByPool"

	poolCollectionMethodPoolData      = "poolData"
	poolCollectionMethodNetworkFeePPM = "networkFeePPM"
)

var (
	pmmResolution = uint256.NewInt(1_000_000)

	defaultGas = Gas{Swap: 150000}
)
