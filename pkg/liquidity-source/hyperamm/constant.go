package hyperamm

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeHyperAMM

	// scale18 is 1e18, used as the fixed-point denominator for fairPriceInScale18
	scale18Str = "1000000000000000000"

	// bps is the basis-point denominator (10 000)
	bpsStr = "10000"

	defaultGas = int64(200_000)
)

var (
	ErrInvalidToken         = errors.New("hyperamm: invalid token")
	ErrZeroAmountIn         = errors.New("hyperamm: zero amount in")
	ErrZeroAmountOut        = errors.New("hyperamm: zero amount out")
	ErrPoolPaused           = errors.New("hyperamm: pool is paused")
	ErrInsufficientReserve  = errors.New("hyperamm: insufficient reserve")
	ErrOverflow             = errors.New("hyperamm: overflow")
	ErrNegativeLpValue      = errors.New("hyperamm: negative lp value")
	ErrZeroFairPrice        = errors.New("hyperamm: zero fair price")
)
