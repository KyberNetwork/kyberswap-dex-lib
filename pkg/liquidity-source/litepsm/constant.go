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
	DefaultGas = Gas{SellGem: 115000, BuyGem: 115000}
	WAD        = number.Number_1e18

	// 2^256 - 1 (type(uint256).max)
	HALTED = new(uint256.Int).Sub(
		new(uint256.Int).Lsh(number.Number_1, 256),
		number.Number_1,
	)
)
