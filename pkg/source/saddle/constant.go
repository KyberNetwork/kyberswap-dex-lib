package saddle

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexTypeSaddle = "saddle"

	poolMethodSwapStorage     = "swapStorage"
	poolMethodGetTokenBalance = "getTokenBalance"

	erc20MethodTotalSupply = "totalSupply"

	defaultWeight = 1
	zeroSrting    = "0"

	MaxLoopLimit = 256
)

var (
	DefaultGas     = Gas{Swap: 130000, AddLiquidity: 280000, RemoveLiquidity: 150000}
	FeeDenominator = bignumber.NewBig10("10000000000")
	APrecision     = big.NewInt(100)
)
