package capricornpamm

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeCapricornPamm

	methodToken0        = "token0"
	methodToken1        = "token1"
	methodFactory       = "factory"
	methodOracleId      = "oracleId"
	methodGetReserves   = "getReserves"
	methodFeeBps        = "feeBps"
	methodPricingEngine = "pricingEngine"
	methodPaused        = "paused"
	methodQuoteExactIn  = "quoteExactIn"

	methodMaxInputAmount       = "maxInputAmount"
	methodEngineOracleRegistry = "oracleRegistry"

	methodGetPrice             = "getPrice"
	methodMaxPushPriceAge      = "maxPushPriceAge"
	methodPythValidTimePeriod  = "pythValidTimePeriod"
	methodOracleRegistryPaused = "paused"

	defaultGas     = 154999
	feeDenominator = 10000

	pushAgeSafetyBufferSec = uint64(120)
)

var (
	feeDenominatorU256 = uint256.NewInt(feeDenominator)

	ErrInvalidToken     = errors.New("invalid token")
	ErrZeroAmount       = errors.New("zero amount in")
	ErrPaused           = errors.New("pool paused")
	ErrAmountInTooLarge = errors.New("amount in exceeds snapshot ladder")
	ErrNoQuote          = errors.New("no quote available for direction")
	ErrPoolUnavailable  = errors.New("pool not quoteable at snapshot time")
)
