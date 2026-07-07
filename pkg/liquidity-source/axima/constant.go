package axima

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType    = "axima"
	defaultGas = 175000
)

var Q64BI = new(big.Int).Lsh(bignumber.One, 64)
