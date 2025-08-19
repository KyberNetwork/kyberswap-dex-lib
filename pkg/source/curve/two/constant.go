package two

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	DefaultGas       = Gas{Exchange: 220000}
	MinGamma         = bignumber.NewBig10("10000000000")
	MaxGamma         = new(big.Int).Mul(big.NewInt(2), bignumber.NewBig10("10000000000000000"))
	AMultiplier      = bignumber.NewBig10("10000")
	MinA             = new(big.Int).Div(new(big.Int).Mul(bignumber.Four, AMultiplier), big.NewInt(10)) // 4 == NCoins ** NCoins, NCoins = 2
	MaxA             = new(big.Int).Mul(new(big.Int).Mul(bignumber.Four, AMultiplier), big.NewInt(100000))
	Precision        = bignumber.BONE
	PriceMask        = new(big.Int).Sub(new(big.Int).Lsh(bignumber.One, 128), bignumber.One)
	PriceSize   uint = 128
)
