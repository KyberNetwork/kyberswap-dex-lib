package lglclob

import (
	"errors"
	"math/big"
	"time"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "lgl-clob"

	maxPriceLevels = 48

	safetyBuffer        = 0.69420
	priceLimitPrecision = 6

	orderGas         = 415_000
	orderGasPerLevel = 162_000

	// makerQuoteRefresh is how long a measured market maker's quote is kept
	// before it is measured again. It changes slowly, and measuring it costs
	// about as much gas as reading the book.
	makerQuoteRefresh = 10 * time.Minute
	// makerQuoteRetry is how soon a side the last measurement could not quote is
	// measured again once it has orders. It bounds a side whose quotes keep
	// failing to a measurement a minute.
	makerQuoteRetry = time.Minute
)

var (
	// unmeasuredMakerQuote stands for a market maker's quote not measured, above
	// the most a measured one took.
	unmeasuredMakerQuote = LevelGas{First: 700_000, Next: 105_000}

	bMaxPriceLevels       = big.NewInt(maxPriceLevels)
	uPriceLimitMultiplier = new(uint256.Int).AddUint64(big256.UBasisPoint, 12)

	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrEmptyOrders           = errors.New("empty orders")
	ErrExceededSafetyBuffer  = errors.New("exceed safety buffer")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrUnexpectedLensRevert  = errors.New("unexpected lens revert")
	ErrBookUnread            = errors.New("lens could not read the book")
)
