package nabla

import (
	"errors"

	"github.com/KyberNetwork/int256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeNabla

	decimals = 18

	defaultGas = 244218
)

var (
	feePrecision                     = int256.NewInt(1e6)
	defaultMaxCoverageRatioForSwapIn = int256.NewInt(200)

	mantissa = int256.NewInt(1e18)

	i100 = int256.NewInt(100)
	i1e4 = int256.NewInt(10000)
	i1e6 = int256.NewInt(1000000)
)

var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrInsufficientReserves = errors.New("insufficient reserves")
	ErrZeroSwap             = errors.New("zero swap")
	ErrStalePrice           = errors.New("stale price")
)
