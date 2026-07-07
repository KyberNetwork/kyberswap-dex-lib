package genericarm

import "errors"

const (
	DexType         = "generic-arm"
	defaultReserves = "100000000000000000000000"
	// priceScale4626 is AbstractARM's PRICE_SCALE (1e36). The upgraded ARM contract used by
	// Pricable4626 pools no longer exposes PRICE_SCALE() as a getter (it's now an internal constant).
	priceScale4626 = "1000000000000000000000000000000000000"
)

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity   = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrUnsupportedSwap         = errors.New("unsupported swap")
	ErrUnsupportedArmType      = errors.New("unsupported arm type")
	ErrWithdrawalQueueState    = errors.New("failed to fetch withdrawal queue state")
	ErrFailedToFetchPoolState  = errors.New("failed to fetch pool state")
)
