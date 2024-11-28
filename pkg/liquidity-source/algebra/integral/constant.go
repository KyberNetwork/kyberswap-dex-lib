package integral

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/ethereum/go-ethereum/common"
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

	methodGetLiquidity           = "liquidity"
	methodGetGlobalState         = "globalState"
	methodGetDataStorageOperator = "dataStorageOperator"
	methodGetFeeConfig           = "feeConfig"
	methodGetFeeConfigZto        = "feeConfigZto"
	methodGetFeeConfigOtz        = "feeConfigOtz"
	methodGetTimepoints          = "timepoints"
	methodGetTickSpacing         = "tickSpacing"

	maxSwapLoop         = 1000000
	maxBinarySearchLoop = 1000

	timepointPageSize = uint16(300)

	WINDOW        = 86400 // 1 day in seconds
	UINT16_MODULO = 65536

	poolLiquidityMethod   = "liquidity"
	poolGlobalStateMethod = "globalState"
	poolTickSpacingMethod = "tickSpacing"
	poolPluginMethod      = "plugin"

	basePluginV1FeeConfigMethod = "feeConfig"

	erc20BalanceOfMethod = "balanceOf"

	BEFORE_SWAP_FLAG = 1
	AFTER_SWAP_FLAG  = 1 << 1
	RESOLUTION       = 96
)

var (
	FEE_DENOMINATOR           = big.NewInt(1e6)
	COMMUNITY_FEE_DENOMINATOR = big.NewInt(1e3)

	MIN_INT256  = new(big.Int).Neg(new(big.Int).Lsh(bignumber.One, 255)) // -2^255
	MAX_UINT256 = new(big.Int).Sub(new(big.Int).Lsh(bignumber.One, 256), bignumber.One)

	slot3 = common.BigToHash(big.NewInt(3))

	EIGHT   = big.NewInt(8)
	SIXTEEN = big.NewInt(16)

	MIN_SQRT_RATIO    = big.NewInt(4295128739)
	MAX_SQRT_RATIO, _ = new(big.Int).SetString("1461446703485210103287273052203988822378723970342", 10)
	Q96               = new(big.Int).Lsh(bignumber.One, 96)
	Q128              = new(big.Int).Lsh(bignumber.One, 128)
)
