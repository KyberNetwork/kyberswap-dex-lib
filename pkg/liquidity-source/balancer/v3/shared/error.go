package shared

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrUnsupportedHook    = errors.WithMessage(pool.ErrUnsupported, "unsupported hook")
	ErrInvalidExtra       = errors.New("invalid extra data")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidAmountIn    = errors.New("invalid amount in")
	ErrInvalidAmountOut   = errors.New("invalid amount out")
	ErrPoolIsPaused       = errors.New("pool is paused")
	ErrMaxDepositExceeded = errors.New("max deposit exceeded")
	ErrMaxRedeemExceeded  = errors.New("max redeem exceeded")
)
