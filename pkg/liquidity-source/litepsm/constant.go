package litepsm

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

const DexTypeLitePSM = "lite-psm"

const (
	litePSMMethodTIn    = "tin"
	litePSMMethodTOut   = "tout"
	lietPSMMethodGem    = "gem"
	litePSMMethodPocket = "pocket"

	erc20MethodBalanaceOf = "balanceOf"
)

const (
	DAIAddress = "0x6b175474e89094c44da98b954eedeac495271d0f"
)

var (
	DefaultGas = Gas{SellGem: 65000, BuyGem: 65000}
	WAD        = number.Number_1e18

	// 2^256 - 1 (type(uint256).max)
	HALTED = new(uint256.Int).SetAllOne()
)
