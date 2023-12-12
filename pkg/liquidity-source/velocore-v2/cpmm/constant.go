package cpmm

import "math/big"

const (
	DexType = "velocore-v2-cpmm"

	factoryMethodPoolsLength = "poolsLength"
	factoryMethodPoolList    = "poolList"

	poolMethodPoolBalances   = "poolBalances"
	poolMethodRelevantTokens = "relevantTokens"
	poolMethodTokenWeights   = "tokenWeights"
	poolMethodFee1e9         = "fee1e9"
	poolMethodFeeMultiplier  = "feeMultiplier"

	reserveZero = "0"
)

const (
	// `maxPoolTokenNumber` is equal to `_MAX_TOKENS` (fixed in smart contract, which is 4) add 1 lp token.
	// https://github.com/velocore/velocore-contracts/blob/master/src/pools/constant-product/ConstantProductPool.sol#L47
	maxPoolTokenNumber = 5

	lpTokenNumber = 1

	unknownInt = -1
)

var (
	// (1 << 127) - 1
	unknownBI = new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(127), nil), big.NewInt(1))

	bigint1e9 = big.NewInt(1e9)
)

var (
	defaultGas = Gas{Swap: 145000}
)
