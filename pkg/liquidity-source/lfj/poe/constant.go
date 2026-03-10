package poe

import (
	"errors"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangePoe

	defaultGas int64 = 150000
)

var (
	bps            = u256.UBasisPoint
	pricePrecision = u256.TenPow(24)
	feePrecision   = u256.TenPow(6)
)

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
