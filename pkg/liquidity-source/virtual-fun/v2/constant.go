package v2

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexType = valueobject.ExchangeVirtualFunV2

var (
	defaultSellGas                 int64 = 112243
	defaultBuyGas                  int64 = 163147
	defaultOpenTradingOnUniswapGas int64 = 2028549
)

var (
	ErrTokenGraduated           = errors.New("token graduated")
	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidTokenStatus       = errors.New("invalid token status")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidAmountIn          = errors.New("invalid amount in")
	ErrInsufficientInputAmount  = errors.New("insufficient input amount")
	ErrInvalidAmountOut         = errors.New("invalid amount out")
	ErrInsufficientOutputAmount = errors.New("insufficient output amount")
	ErrDivisionByZero           = errors.New("division by zero")
)
