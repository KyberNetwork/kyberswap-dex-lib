package pancakestable

import "math/big"

const (
	DexType = "pancake-stable"

	factoryMethodPairLength       = "pairLength"
	factoryMethodSwapPairContract = "swapPairContract"

	poolMethodNCoins       = "N_COINS"
	poolMethodRates        = "RATES"
	poolMethodPrecisionMul = "PRECISION_MUL"
	poolMethodCoins        = "coins"
	poolMethodToken        = "token"
)

var (
	Zero = big.NewInt(0)
	One  = big.NewInt(1)
)
