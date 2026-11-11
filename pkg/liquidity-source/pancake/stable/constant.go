package pancakestable

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
)

const (
	DexType = curve.PoolTypePancakeStable

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
