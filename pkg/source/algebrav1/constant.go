package algebrav1

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

const (
	DexTypeAlgebraV1      = "algebra-v1"
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
	erc20MethodBalanceOf         = "balanceOf"

	maxSwapLoop         = 1000000
	maxBinarySearchLoop = 1000

	timepointPageSize = uint16(300)

	WINDOW        = 86400 // 1 day in seconds
	UINT16_MODULO = 65536

	BaseGas          = int64(242000)
	CrossInitTickGas = int64(21000)
)

var (
	COMMUNITY_FEE_DENOMINATOR        = uint256.NewInt(1000)
	COMMUNITY_FEE_DENOMINATOR_BIGINT = big.NewInt(1000)

	slot3 = common.BigToHash(big.NewInt(3))
)
