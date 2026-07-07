package orderbook

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrLevelsTooOld          = errors.WithMessage(pool.ErrUnsupported, "levels too old")
	ErrInvalidToken          = errors.New("invalid token")
	ErrEmptyLevels           = errors.New("empty price levels")
	ErrNoSwapLimit           = errors.New("swap limit is required for PMM pools")
	ErrSwapLimitExceeded     = errors.New("swap limit exceeded")
	ErrInvalidAmountIn       = errors.New("amountIn is less than lowest price level")
	ErrInsufficientLiquidity = errors.New("amountIn is greater than total price levels")
)
