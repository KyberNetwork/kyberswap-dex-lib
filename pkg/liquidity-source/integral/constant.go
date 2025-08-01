package integral

import (
	"errors"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexTypeIntegral = "integral"

	pairToken0Method = "token0"
	pairToken1Method = "token1"

	factoryAllPairsMethod       = "allPairs"
	factoryAllPairsLengthMethod = "allPairsLength"

	relayerFactoryMethod                    = "factory"
	relayerIsPairEnabledMethod              = "isPairEnabled"
	relayerGetPoolStateMethod               = "getPoolState"
	relayerGetPairByAddressMethod           = "getPriceByPairAddress"
	relayerGetTokenLimitMaxMultiplierMethod = "getTokenLimitMaxMultiplier"
)

var (
	defaultGas = Gas{Swap: 400000}

	precision = u256.TenPow(18)
)

var (
	ErrTokenNotFound = errors.New("tokens not found")
	ErrNoSwapLimit   = errors.New("no swap limit")

	ErrTR03 = errors.New("TR03")
	ErrTR3A = errors.New("TR3A")
	ErrTR05 = errors.New("TR05")
)
