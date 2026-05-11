package canonic

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeCanonic

	defaultGas int64 = 625_092

	defaultRungCount uint16 = 64

	marketStateUnwindOnly = 1
	marketStatePaused     = 2

	maobMethodBaseToken      = "baseToken"
	maobMethodQuoteToken     = "quoteToken"
	maobMethodBaseDecimals   = "baseDecimals"
	maobMethodQuoteDecimals  = "quoteDecimals"
	maobMethodBaseScale      = "baseScale"
	maobMethodQuoteScale     = "quoteScale"
	maobMethodGetMidPrice    = "getMidPrice"
	maobMethodTakerFee       = "takerFee"
	maobMethodFeeDenom       = "FEE_DENOM"
	maobMethodMinQuoteTaker  = "minQuoteTaker"
	maobMethodMarketState    = "marketState"
	maobMethodStateExpiresAt = "stateExpiresAt"
	maobMethodRungDenom      = "RUNG_DENOM"
	maobMethodPriceSigfigs   = "PRICE_SIGFIGS"
	maobMethodGetDepth       = "getDepth"
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmountIn       = errors.New("invalid amount in")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrMarketPaused          = errors.New("market is paused")
	ErrMarketUnwindOnly      = errors.New("market is unwind only")
	ErrOracleStale           = errors.New("oracle price is stale")
	ErrQuoteAmountTooLow     = errors.New("below min quote taker")
	ErrInvalidAmountOut      = errors.New("zero amount out")
)
