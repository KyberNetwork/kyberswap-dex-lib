package uniswapv1

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

const (
	DexType = "uniswap-v1"

	DefaultSwapFee = 0.003

	multicallGetEthBalanceMethod = "getEthBalance"

	erc20BalanceOfMethod = "balanceOf"

	factoryTokenCountMethod     = "tokenCount"
	factoryGetTokenWithIDMethod = "getTokenWithId"
	factoryGetExchangeMethod    = "getExchange"

	defaultGas = 165000
)

var (
	ZeroAddress = common.Address{}

	U997  = uint256.NewInt(997)
	U1000 = uint256.NewInt(1000)

	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidAmountIn          = errors.New("invalid amount in")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInvalidAmountOut         = errors.New("invalid amount out")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("INSUFFICIENT_LIQUIDITY")
)
