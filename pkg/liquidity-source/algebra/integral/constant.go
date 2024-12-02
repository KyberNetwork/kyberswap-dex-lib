package integral

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType               = "algebra-integral"
	graphSkipLimit        = 5000
	graphFirstLimit       = 1000
	defaultTokenDecimals  = 18
	defaultTokenWeight    = 50
	zeroString            = "0"
	emptyString           = ""
	graphQLRequestTimeout = 20 * time.Second

	timepointPageSize = uint16(300)
	maxSwapLoop       = 1000000

	WINDOW        = 86400 // 1 day in seconds
	UINT16_MODULO = 65536

	poolTicksMethod       = "ticks"
	poolLiquidityMethod   = "liquidity"
	poolGlobalStateMethod = "globalState"
	poolTickSpacingMethod = "tickSpacing"
	poolPluginMethod      = "plugin"

	dynamicFeeManagerPluginFeeConfigMethod = "feeConfig"

	slidingFeePluginFeeFactorsMethod = "s_feeFactors"

	votalityOraclePluginTimepointsMethod             = "timepoints"
	votalityOraclePluginTimepointIndexMethod         = "timepointIndex"
	votalityOraclePluginLastTimepointTimestampMethod = "lastTimepointTimestamp"
	votalityOraclePluginIsInitializedMethod          = "isInitialized"

	erc20BalanceOfMethod = "balanceOf"

	ticklensGetPopulatedTicksInWordMethod = "getPopulatedTicksInWord"

	BEFORE_SWAP_FLAG = 1
	AFTER_SWAP_FLAG  = 1 << 1
	RESOLUTION       = 96

	s_priceChangeFactor = 1000
	s_baseFee           = 500

	FACTOR_DENOMINATOR = 1000
	FEE_FACTOR_SHIFT   = 96
)

var (
	FEE_DENOMINATOR           = big.NewInt(1e6)
	COMMUNITY_FEE_DENOMINATOR = big.NewInt(1e3)

	MIN_INT256  = new(big.Int).Neg(new(big.Int).Lsh(bignumber.One, 255)) // -2^255
	MAX_UINT256 = new(big.Int).Sub(new(big.Int).Lsh(bignumber.One, 256), bignumber.One)

	MAX_UINT16 = new(big.Int).SetUint64(1<<16 - 1)
	MIN_UINT16 = new(big.Int).SetUint64(1)

	EIGHT      = big.NewInt(8)
	FIFTEEN    = big.NewInt(15)
	SIXTEEN    = big.NewInt(16)
	TWENTYFOUR = big.NewInt(24)

	MIN_SQRT_RATIO    = big.NewInt(4295128739)
	MAX_SQRT_RATIO, _ = new(big.Int).SetString("1461446703485210103287273052203988822378723970342", 10)
	Q96               = new(big.Int).Lsh(bignumber.One, 96)
	Q128              = new(big.Int).Lsh(bignumber.One, 128)

	BASE_FEE_MULTIPLIER   = new(big.Int).Lsh(bignumber.One, FEE_FACTOR_SHIFT)
	DOUBLE_FEE_MULTIPLIER = new(big.Int).Lsh(bignumber.One, 2*FEE_FACTOR_SHIFT)

	// Predefined e values multiplied by 10^20 as constants
	CLOSEST_VALUE_0, _       = new(big.Int).SetString("100000000000000000000", 10)   // 1
	CLOSEST_VALUE_1, _       = new(big.Int).SetString("271828182845904523536", 10)   // ~= e
	CLOSEST_VALUE_2, _       = new(big.Int).SetString("738905609893065022723", 10)   // ~= e^2
	CLOSEST_VALUE_3, _       = new(big.Int).SetString("2008553692318766774092", 10)  // ~= e^3
	CLOSEST_VALUE_4, _       = new(big.Int).SetString("5459815003314423907811", 10)  // ~= e^4
	CLOSEST_VALUE_DEFAULT, _ = new(big.Int).SetString("14841315910257660342111", 10) // ~= e^5

	E_HALF_MULTIPLIER, _ = new(big.Int).SetString("164872127070012814684", 10) // e^0.5
	E_MULTIPLIER_BIG, _  = new(big.Int).SetString("100000000000000000000", 10) // 1e20
)
