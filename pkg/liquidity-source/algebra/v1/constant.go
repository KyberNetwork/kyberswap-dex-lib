package algebrav1

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

const (
	DexTypeAlgebraV1     = "algebra-v1"
	graphSkipLimit       = 5000
	graphFirstLimit      = 1000
	defaultTokenDecimals = 18
	zeroString           = "0"
	emptyString          = ""

	methodGetLiquidity           = "liquidity"
	methodGetGlobalState         = "globalState"
	methodGetDataStorageOperator = "dataStorageOperator"
	methodGetFeeConfig           = "feeConfig"
	methodGetFeeConfigZto        = "feeConfigZto"
	methodGetFeeConfigOtz        = "feeConfigOtz"
	methodGetTimepoints          = "timepoints"
	methodGetTickSpacing         = "tickSpacing"
	methodGetTicks               = "ticks"

	maxSwapLoop         = 1000000
	maxBinarySearchLoop = 1000

	tickChunkSize = 100

	WINDOW        = 86400 // 1 day in seconds
	UINT16_MODULO = 65536

	BaseGas          = int64(242334)
	CrossInitTickGas = int64(21492)
)

var (
	COMMUNITY_FEE_DENOMINATOR        = uint256.NewInt(1000)
	COMMUNITY_FEE_DENOMINATOR_BIGINT = big.NewInt(1000)

	slot3 = common.BigToHash(big.NewInt(3))
)
