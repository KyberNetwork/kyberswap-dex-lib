package axima

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType    = "axima"
	defaultGas = 175000

	// gasPerExtension is the swap gas of one Metric pool extension. Measured on Base: hook frames cost
	// 8k-39k (priceVelocity 13k, stopLoss 27-39k, price-provider hook 8k), about 20k on average.
	gasPerExtension = 20000
)

var Q64BI = new(big.Int).Lsh(bignumber.One, 64)
