package curveTricrypto

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

var (
	DefaultGas       = Gas{Exchange: 240000}
	MinGamma         = utils.NewBig10("10000000000")
	MaxGamma         = new(big.Int).Mul(big.NewInt(5), utils.NewBig10("10000000000000000"))
	AMultiplier      = utils.NewBig10("10000")
	Precision        = constant.BONE
	PriceMask        = new(big.Int).Sub(new(big.Int).Lsh(constant.One, 128), constant.One)
	PriceSize   uint = 128
)
