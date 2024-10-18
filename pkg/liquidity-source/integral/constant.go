package integral

import (
	"errors"
	"math/big"

	"github.com/holiman/uint256"
)

var (
	DexTypeIntegral = "integral"
	ZERO            = big.NewInt(0)
	defaultGas      = Gas{Swap: 400000}
	precision       = uint256.NewInt(1e18)

	// errors
	ErrTokenNotFound  = errors.New("tokens not found")
	ErrInvalidTokenIn = errors.New("invalid tokenIn")

	ErrTR03 = errors.New("TR03")
	ErrTR3A = errors.New("TR3A")
	ErrTR05 = errors.New("TR05")

	// pair methods
	pairToken0Method = "token0"
	pairToken1Method = "token1"

	// factory methods
	factoryAllPairsMethod       = "allPairs"
	factoryAllPairsLengthMethod = "allPairsLength"

	// relayer methods
	relayerFactoryMethod                    = "factory"
	relayerIsPairEnabledMethod              = "isPairEnabled"
	relayerGetPoolStateMethod               = "getPoolState"
	relayerGetPairByAddressMethod           = "getPriceByPairAddress"
	relayerGetTokenLimitMaxMultiplierMethod = "getTokenLimitMaxMultiplier"
)
