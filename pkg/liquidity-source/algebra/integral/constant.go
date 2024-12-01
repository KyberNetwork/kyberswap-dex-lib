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

	EIGHT   = big.NewInt(8)
	SIXTEEN = big.NewInt(16)

	MIN_SQRT_RATIO    = big.NewInt(4295128739)
	MAX_SQRT_RATIO, _ = new(big.Int).SetString("1461446703485210103287273052203988822378723970342", 10)
	Q96               = new(big.Int).Lsh(bignumber.One, 96)
	Q128              = new(big.Int).Lsh(bignumber.One, 128)

	BASE_FEE_MULTIPLIER   = new(big.Int).Lsh(bignumber.One, FEE_FACTOR_SHIFT)
	DOUBLE_FEE_MULTIPLIER = new(big.Int).Lsh(bignumber.One, 2*FEE_FACTOR_SHIFT)
)
