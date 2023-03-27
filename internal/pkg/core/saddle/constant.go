package saddle

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
)

const MaxLoopLimit = 256

var (
	DefaultGas     = Gas{Swap: 130000, AddLiquidity: 280000, RemoveLiquidity: 150000}
	FeeDenominator = utils.NewBig10("10000000000")
	APrecision     = big.NewInt(100)
)
