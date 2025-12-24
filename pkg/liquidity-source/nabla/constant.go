package nabla

import (
	"errors"

	"github.com/KyberNetwork/int256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeNabla

	decimals = 18

	nablaPriceAPI = "https://antenna.nabla.fi/v1/updates/price/latest"
)

var (
	priceScalingFactor = int256.NewInt(1e8)
	pricePrecision     = int256.NewInt(1e8)
	feePrecision       = int256.NewInt(1e6)

	mantissa = int256.NewInt(1e18)

	i1990 = int256.NewInt(1990)
	i1e3  = int256.NewInt(1000)
	i1e4  = int256.NewInt(10000)
	i1e6  = int256.NewInt(1000000)
)

var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrInsufficientReserves = errors.New("insufficient reserves")
	ErrZeroSwap             = errors.New("zero swap")
	ErrStalePrice           = errors.New("stale price")
)
