package liquiditybookv20

import (
	"math/big"
	"time"
)

const (
	DexTypeLiquidityBookV20 = "liquiditybook-v20"
)

const (
	factoryMethodGetNumberOfLBPairs = "getNumberOfLBPairs"
	factoryMethodAllLBPairs         = "allLBPairs"

	pairMethodTokenX           = "tokenX"
	pairMethodTokenY           = "tokenY"
	pairMethodFeeParameters    = "feeParameters"
	pairMethodGetReservesAndID = "getReservesAndId"
)
const (
	defaultTokenWeight = 50

	graphQLRequestTimeout = 20 * time.Second
	graphFirstLimit       = 1000

	basisPointMax = 10000

	scaleOffset = 128

	realIDShift = 1 << 23

	defaultGas = 125000
)

var (
	scale    = new(big.Int).Lsh(big.NewInt(1), scaleOffset)
	precison = big.NewInt(1e18)

	maxUint256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
)
