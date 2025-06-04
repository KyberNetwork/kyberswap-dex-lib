package litepsm

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
)

const DexTypeLitePSM = "lite-psm"

const (
	litePSMMethodTIn    = "tin"
	litePSMMethodTOut   = "tout"
	litePSMMethodPocket = "pocket"
)

var (
	DefaultGas = Gas{SellGem: 65000, BuyGem: 65000}
	WAD        = number.Number_1e18

	HALTED = number.MaxU256
)
