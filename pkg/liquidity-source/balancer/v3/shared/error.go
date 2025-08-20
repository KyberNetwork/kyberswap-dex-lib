package shared

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrUnsupportedHook  = errors.WithMessage(pool.ErrUnsupported, "unsupported hook")
	ErrInvalidExtra     = errors.New("invalid extra data")
	ErrEmptyBalances    = errors.New("empty balances")
	ErrInvalidToken     = errors.New("invalid token")
	ErrInvalidAmountIn  = errors.New("invalid amount in")
	ErrInvalidAmountOut = errors.New("invalid amount out")
)
