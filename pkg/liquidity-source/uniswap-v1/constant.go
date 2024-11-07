package uniswapv1

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

const (
	DexType = "uniswap-v1"

	DefaultSwapFee float64 = 0.003
)

var (
	defaultGas = Gas{Swap: 165000}

	ZERO_ADDRESS = common.Address{}

	U997  = uint256.NewInt(997)
	U1000 = uint256.NewInt(1000)
)

const (
	multicallGetEthBalanceMethod = "getEthBalance"

	erc20BalanceOfMethod = "balanceOf"

	factoryTokenCountMethod     = "tokenCount"
	factoryGetTokenWithIDMethod = "getTokenWithId"
	factoryGetExchangeMethod    = "getExchange"
)
