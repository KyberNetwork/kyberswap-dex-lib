package slyngfun

import "errors"

const DexType = "slyng-fun"

// bps is the launchpad's fee denominator (Launchpad.BPS).
const bps = 10_000

const (
	launchpadMethodCurves      = "curves"
	launchpadMethodTradeFeeBps = "TRADE_FEE_BPS"
	launchpadMethodSnipeBps    = "SNIPE_BPS"
	launchpadMethodSnipeWindow = "SNIPE_WINDOW_SECONDS"
	launchpadMethodTokenCount  = "tokenCount"
	launchpadMethodAllTokens   = "allTokens"
	launchpadEventTokenCreated = "TokenCreated"
	launchpadEventBought       = "Bought"
	launchpadEventSold         = "Sold"
	launchpadEventGraduated    = "Graduated"
)

// Gas measured through ks-dex-adapter-lib's SlyngFunAdapter on a Robinhood Chain mainnet fork,
// cold storage. An ETH-quoted trade is the cheap case (142k buy, 126k sell on SLYNG's live curve);
// an ERC-20 quote pays for the transferFrom and a proxied token on top (330k buy on a fresh USDG
// curve, 167k sell). The buy that fills the curve also mints and locks the Uniswap v4 position
// in the same transaction, which cost 779k against 142k for the plain buy.
var defaultGas = Gas{
	BuyNative:  150_000,
	SellNative: 130_000,
	BuyERC20:   340_000,
	SellERC20:  175_000,
	Graduation: 640_000,
}

var (
	ErrInvalidToken   = errors.New("invalid token")
	ErrInvalidAmount  = errors.New("invalid amount")
	ErrZeroAmount     = errors.New("zero amount")
	ErrGraduated      = errors.New("curve graduated, it trades on its Uniswap v4 pool now")
	ErrUnknownCurve   = errors.New("the launchpad has no curve for this token")
	ErrArithmetic     = errors.New("curve arithmetic error")
	ErrMalformedLog   = errors.New("malformed event log")
	ErrFailedCall     = errors.New("launchpad state call failed")
	ErrInvalidReserve = errors.New("invalid reserve")
)
