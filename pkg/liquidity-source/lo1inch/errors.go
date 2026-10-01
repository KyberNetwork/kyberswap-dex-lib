package lo1inch

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrTokenInNotSupported       = errors.New("tokenIn is not supported")
	ErrNoOrderAvailable          = errors.New("no order available")
	ErrCannotFulfillAmountIn     = errors.New("cannot fulfill amountIn")
	ErrAmountInBelowOrderMin     = errors.WithMessage(pool.ErrSwapAmountBelowMin, "amountIn below order min fill")
	ErrOnlyAllowFullFill         = errors.New("order only allow full fill")
	ErrFeeTakerExtensionNotFound = errors.New("fee taker extension not found")
)
