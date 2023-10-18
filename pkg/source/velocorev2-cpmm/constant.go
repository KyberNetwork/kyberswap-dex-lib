package velocorev2cpmm

import (
	"math/big"
)

const (
	DexTypeVelocoreV2CPMM = "velocorev2-cpmm"

	reserveZero = "0"
)

const (
	factoryMethodPoolsLength = "poolsLength"
	factoryMethodPoolList    = "poolList"

	poolMethodPoolBalances   = "poolBalances"
	poolMethodRelevantTokens = "relevantTokens"
	poolMethodTokenWeights   = "tokenWeights"
	poolMethodFee1e9         = "fee1e9"
	poolMethodFeeMultiplier  = "feeMultiplier"
)

const (
	// `maxPoolTokenNumber` is equal to `_MAX_TOKENS` (fixed in smart contract, which is 4) add 1 lp token.
	// https://github.com/velocore/velocore-contracts/blob/master/src/pools/constant-product/ConstantProductPool.sol#L47
	maxPoolTokenNumber = 5

	lpTokenNumber = 1

	unknownInt = -1
)

var (
	unknownBI = new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))

	bigint0    = big.NewInt(0)
	bigint1    = big.NewInt(1)
	bigint2    = big.NewInt(2)
	bigint1e4  = big.NewInt(1e4)
	bigint1e5  = big.NewInt(1e5)
	bigint1e9  = big.NewInt(1e9)
	bigint1e18 = big.NewInt(1e18)
)
