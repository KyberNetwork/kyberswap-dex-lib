package fermi

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeFermi

	defaultGas int64 = 163500

	maxStaleBlocks uint64 = 2

	methodFermi         = "fermi"
	methodTraderVault   = "traderVault"
	methodGetPairs      = "getPairs"
	methodQuote         = "quoteAmounts"
	methodGetPairParams = "getPairParams"
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmountIn       = errors.New("invalid amountIn: must be positive")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity: amountIn exceeds curve max")
	ErrZeroAmountOut         = errors.New("zero amountOut")
)
