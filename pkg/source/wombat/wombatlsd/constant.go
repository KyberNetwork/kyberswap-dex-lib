package wombatlsd

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat"
)

var (
	DefaultGas = wombat.Gas{Swap: 88000}
	WAD        = big.NewInt(1e18)
	WADI       = big.NewInt(1e18)
)
