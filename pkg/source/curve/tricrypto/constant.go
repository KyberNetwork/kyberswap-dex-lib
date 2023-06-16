package tricrypto

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	DefaultGas       = Gas{Exchange: 240000}
	MinGamma         = bignumber.NewBig10("10000000000")
	MaxGamma         = new(big.Int).Mul(big.NewInt(5), bignumber.NewBig10("10000000000000000"))
	AMultiplier      = bignumber.NewBig10("10000")
	MinA             = new(big.Int).Div(new(big.Int).Mul(big.NewInt(27), AMultiplier), big.NewInt(100)) // 27 = NCoins ** NCoins, NCoins = 3
	MaxA             = new(big.Int).Mul(new(big.Int).Mul(big.NewInt(27), AMultiplier), big.NewInt(1000))
	Precision        = bignumber.BONE
	PriceMask        = new(big.Int).Sub(new(big.Int).Lsh(bignumber.One, 128), bignumber.One)
	PriceSize   uint = 128
)
