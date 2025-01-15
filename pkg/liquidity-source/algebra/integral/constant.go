package integral

import (
	"github.com/holiman/uint256"
)

const (
	DexType              = "algebra-integral"
	graphSkipLimit       = 5000
	graphFirstLimit      = 1000
	defaultTokenDecimals = 18
	defaultTokenWeight   = 50
	zeroString           = "0"
	emptyString          = ""

	timepointPageSize = uint16(300)
	maxSwapLoop       = 1000000

	WINDOW        = 86400 // 1 day in seconds
	UINT16_MODULO = 65536

	poolLiquidityMethod   = "liquidity"
	poolGlobalStateMethod = "globalState"
	poolTickSpacingMethod = "tickSpacing"
	poolPluginMethod      = "plugin"
	poolTicksMethod       = "ticks"

	dynamicFeeManagerPluginFeeConfigMethod = "feeConfig"

	slidingFeePluginFeeFactorsMethod = "s_feeFactors"

	votalityOraclePluginTimepointsMethod             = "timepoints"
	votalityOraclePluginTimepointIndexMethod         = "timepointIndex"
	votalityOraclePluginLastTimepointTimestampMethod = "lastTimepointTimestamp"
	votalityOraclePluginIsInitializedMethod          = "isInitialized"

	erc20BalanceOfMethod = "balanceOf"

	BEFORE_SWAP_FLAG = 1
	RESOLUTION       = 96

	s_priceChangeFactor = 1000
	s_baseFee           = 500

	FACTOR_DENOMINATOR = 1000
	FEE_FACTOR_SHIFT   = 96
)

var (
	FEE_DENOMINATOR           = uint256.NewInt(1e6)
	COMMUNITY_FEE_DENOMINATOR = uint256.NewInt(1e3)

	uZERO       = uint256.NewInt(0)
	uONE        = uint256.NewInt(1)
	uTWO        = uint256.NewInt(2)
	uSIX        = uint256.NewInt(6)
	uFIFTEEN    = uint256.NewInt(15)
	uTWENTYFOUR = uint256.NewInt(24)

	MIN_SQRT_RATIO    = uint256.NewInt(4295128739)
	MAX_SQRT_RATIO, _ = uint256.FromDecimal("1461446703485210103287273052203988822378723970342")

	Q96 = new(uint256.Int).Lsh(uONE, 96) // 1 << 96

	BASE_FEE_MULTIPLIER   = new(uint256.Int).Lsh(uONE, FEE_FACTOR_SHIFT)   // 1 << 96
	DOUBLE_FEE_MULTIPLIER = new(uint256.Int).Lsh(uONE, 2*FEE_FACTOR_SHIFT) // 1 << 2*96

	// Predefined e values multiplied by 10^20 as constants
	CLOSEST_VALUE_0, _       = uint256.FromDecimal("100000000000000000000")   // 1
	CLOSEST_VALUE_1, _       = uint256.FromDecimal("271828182845904523536")   // ~= e
	CLOSEST_VALUE_2, _       = uint256.FromDecimal("738905609893065022723")   // ~= e^2
	CLOSEST_VALUE_3, _       = uint256.FromDecimal("2008553692318766774092")  // ~= e^3
	CLOSEST_VALUE_4, _       = uint256.FromDecimal("5459815003314423907811")  // ~= e^4
	CLOSEST_VALUE_DEFAULT, _ = uint256.FromDecimal("14841315910257660342111") // ~= e^5

	E_HALF_MULTIPLIER, _ = uint256.FromDecimal("164872127070012814684") // e^0.5
	E_MULTIPLIER_BIG, _  = uint256.FromDecimal("100000000000000000000") // 1e20
)
