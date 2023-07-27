package algebrav1

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

	methodGetLiquidity   = "liquidity"
	methodGetGlobalState = "globalState"
	erc20MethodBalanceOf = "balanceOf"

	tickSpacing = 60
)

var (
	Q128                      = bignumber.NewBig("0x100000000000000000000000000000000")
	COMMUNITY_FEE_DENOMINATOR = big.NewInt(1000)
)
