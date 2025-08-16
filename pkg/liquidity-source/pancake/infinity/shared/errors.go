package shared

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrUnsupportedHook   = errors.WithMessage(pool.ErrUnsupported, "unsupported hook")
	ErrUninitializedPool = errors.New("pool is uninitialized")
	ErrInvalidToken      = errors.New("invalid token")
	ErrInvalidAmountIn   = errors.New("invalid amount in")
)
