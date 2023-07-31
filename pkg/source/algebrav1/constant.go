package algebrav1

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/ethereum/go-ethereum/common"
)

const (
	DexTypeAlgebraV1      = "algebrav1"
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
	methodGetTimepoints          = "timepoints"
	methodGetTickSpacing         = "tickSpacing"
	erc20MethodBalanceOf         = "balanceOf"

	uint16_max = 65535

	maxSwapLoop = 1000000

	timepointPageSize           = 300
	timepointWindowLimitSeconds = 3600 * 24
)

var (
	uint256_max               = bignumber.NewBig("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	pow192                    = new(big.Int).Lsh(bignumber.One, 192)
	Q128                      = bignumber.NewBig("0x100000000000000000000000000000000")
	COMMUNITY_FEE_DENOMINATOR = big.NewInt(1000)
	MAX_VOLUME_PER_LIQUIDITY  = new(big.Int).Lsh(big.NewInt(100000), 64) // maximum meaningful ratio of volume to liquidity

	slot3 = common.BigToHash(big.NewInt(3))
)
