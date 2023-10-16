package velocorev2cpmm

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
)

// var (
// 	zeroBI         = big.NewInt(0)
// 	defaultGas     = Gas{SwapBase: 60000, SwapNonBase: 102000}
// 	defaultSwapFee = "2"
// 	bOne           = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
// 	bOneFloat, _   = new(big.Float).SetString("1000000000000000000")
// )
