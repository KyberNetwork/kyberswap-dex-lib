package two

import (
	"math/big"

	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	DefaultGas       = Gas{Exchange: 220000}
	MinGamma         = utils.NewBig10("10000000000")
	MaxGamma         = new(big.Int).Mul(big.NewInt(2), utils.NewBig10("10000000000000000"))
	AMultiplier      = utils.NewBig10("10000")
	MinA             = new(big.Int).Div(new(big.Int).Mul(constant.Four, AMultiplier), big.NewInt(10)) // 4 == NCoins ** NCoins, NCoins = 2
	MaxA             = new(big.Int).Mul(new(big.Int).Mul(constant.Four, AMultiplier), big.NewInt(100000))
	Precision        = constant.BONE
	PriceMask        = new(big.Int).Sub(new(big.Int).Lsh(constant.One, 128), constant.One)
	PriceSize   uint = 128
)
