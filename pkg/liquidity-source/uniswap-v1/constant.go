package uniswapv1

import "github.com/holiman/uint256"

const (
	DexType = "uniswap-v1"
)

var (
	defaultGas = Gas{Swap: 60000}

	DefaultSwapFee float64 = 0.003
	MinTokenBought         = uint256.NewInt(1)
	ZERO_ADDRESS           = "0x0000000000000000000000000000000000000000"
)

const (
	multicallGetEthBalanceMethod = "getEthBalance"

	erc20BalanceOfMethod = "balanceOf"

	factoryTokenCountMethod     = "tokenCount"
	factoryGetTokenWithIDMethod = "getTokenWithId"
	factoryGetExchangeMethod    = "getExchange"
)
