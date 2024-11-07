package kokonutcrypto

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"math/big"
)

const (
	DexTypeKokonutCrypto = "kokonut-crypto"

	registryMethodPoolCount = "poolCount"
	registryMethodPoolList  = "poolList"

	poolMethodCoins                          = "coins"
	poolMethodDecimals                       = "decimals"
	poolMethodToken                          = "token"
	poolMethodA                              = "A"
	poolMethodBalances                       = "balances"
	poolMethodD                              = "D"
	poolMethodGamma                          = "gamma"
	poolMethodFeeGamma                       = "feeGamma"
	poolMethodMidFee                         = "midFee"
	poolMethodOutFee                         = "outFee"
	poolMethodFutureAGammaTime               = "futureAGammaTime"
	poolMethodFutureA                        = "futureA"
	poolMethodFutureGamma                    = "futureGamma"
	poolMethodInitialAGammaTime              = "initialAGammaTime"
	poolMethodInitialA                       = "initialA"
	poolMethodInitialGamma                   = "initialGamma"
	poolMethodLastPricesTimestamp            = "lastPricesTimestamp"
	poolMethodXcpProfit                      = "xcpProfit"
	poolMethodVirtualPrice                   = "virtualPrice"
	poolMethodAllowedExtraProfit             = "allowedExtraProfit"
	poolMethodAdjustmentStep                 = "adjustmentStep"
	poolMethodMaHalfTime                     = "maHalfTime"
	poolMethodPriceScale                     = "priceScale"
	poolMethodPriceOracle                    = "priceOracle"
	poolMethodLastPrices                     = "lastPrices"
	poolMethodMinRemainingPostRebalanceRatio = "minRemainingPostRebalanceRatio"

	erc20MethodTotalSupply = "totalSupply"

	zeroString    = "0"
	defaultWeight = 1
)

var (
	DefaultGas               = Gas{Exchange: 220000}
	MinGamma                 = bignumber.NewBig10("10000000000")
	MaxGamma                 = new(big.Int).Mul(big.NewInt(2), bignumber.NewBig10("10000000000000000"))
	AMultiplier              = bignumber.NewBig10("10000")
	MinA                     = new(big.Int).Div(new(big.Int).Mul(bignumber.Four, AMultiplier), big.NewInt(10)) // 4 == NCoins ** NCoins, NCoins = 2
	MaxA                     = new(big.Int).Mul(new(big.Int).Mul(bignumber.Four, AMultiplier), big.NewInt(100000))
	Precision                = bignumber.BONE
	PriceMask                = new(big.Int).Sub(new(big.Int).Lsh(bignumber.One, 128), bignumber.One)
	PriceSize           uint = 128
	PrecisionPriceScale      = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	PrecisionFee             = new(big.Int).Exp(big.NewInt(10), big.NewInt(10), nil)
)
