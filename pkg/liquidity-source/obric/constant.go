package obric

import (
	"errors"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeObric

	priceBufferSeconds = 20

	defaultGas int64 = 200000
)

var (
	millionth = u256.TenPow(6)
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmountIn       = errors.New("invalid amount in")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrPoolLocked            = errors.New("pool is locked")
	ErrPriceStale            = errors.New("price is stale")
	ErrZeroCurrentXK         = errors.New("currentXK is zero")
	ErrInvalidAmountOut      = errors.New("invalid amount out")
	ErrPoolDisabled          = errors.New("pool is disabled")
)
