package fourmeme

import (
	"github.com/holiman/uint256"
)

var (
	defaultGas = Gas{Swap: 250000}

	ZERO      = uint256.NewInt(0)
	PRECISION = uint256.NewInt(10000)
	EXP18     = new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(18))
	EXP9      = new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(9))
)

const (
	DexType = "four-meme"

	erc20BalanceOfMethod = "balanceOf"

	pairTokenAMethod      = "tokenA"
	pairTokenBMethod      = "tokenB"
	pairGetReservesMethod = "getReserves"
	pairKLastMethod       = "kLast"

	tokenManager2TokenCountMethod = "_tokenCount"
	tokenManager2TokensMethod     = "_tokens"

	tokenManagerHelperGetTokenInfoMethod = "getTokenInfo"

	ZERO_ADDRESS = "0x0000000000000000000000000000000000000000"
)

var ()
