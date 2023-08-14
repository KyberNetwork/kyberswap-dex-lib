package wombatmain

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat"
	"math/big"
)

var (
	DefaultGas = wombat.Gas{Swap: 125000}
	WAD        = big.NewInt(1e18)
	WADI       = big.NewInt(1e18)
)
