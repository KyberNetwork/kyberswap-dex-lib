package poe

import (
	"errors"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangePoe

	defaultGas int64 = 200000

	poolMethodGetTokens   = "getTokens"
	poolMethodGetBalances = "getBalances"
	poolMethodGetOracle   = "getOracle"

	factoryMethodGetPoolsLength = "getPoolsLength"
	factoryMethodGetPoolAt      = "getPoolAt"
)

// uBps is alpha's scale (10000 = 1.0x), used to validate the oracle's alpha.
var uBps = u256.UBasisPoint

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmountIn       = errors.New("invalid amount in")
	ErrInvalidAmountOut      = errors.New("invalid amount out")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrExpiredOracle         = errors.New("oracle data expired")
	ErrZeroReserve           = errors.New("zero reserve")
	ErrZeroVirtualReserve    = errors.New("zero virtual reserve")
	ErrInvalidAlpha          = errors.New("invalid alpha")
)
