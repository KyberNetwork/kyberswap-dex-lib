package clipper

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const DexType = "clipper"

var defaultGas int64 = 80000

var (
	ErrInvalidTokenIn       = errors.New("invalid token in")
	ErrInvalidTokenOut      = errors.New("invalid token out")
	ErrInvalidPair          = errors.New("invalid pair")
	ErrFMVCheckFailed       = errors.New("FMV check failed")
	ErrAmountOutNaN         = errors.New("amountOut is NaN")
	ErrMinAmountInNotEnough = errors.WithMessage(pool.ErrSwapAmountBelowMin, "minAmountIn is not enough")

	basisPoint float64 = 10000
)
