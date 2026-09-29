package flap

import "errors"

const DexType = "flap"

const (
	// defaultGas is a placeholder until measured against a real swap; the swap flows through the
	// Portal contract's swapExactInput/swapExactInputV3 entrypoints, dispatched internally to
	// PortalTradeV2 for bonding-curve tokens.
	defaultGas = 200000

	// bpsDenominator is the basis-point denominator used by Portal.getFeeRate() (buyFeeRate=100,
	// sellFeeRate=100 observed on-chain == 1%), confirmed by magnitude against a bps scale rather
	// than a WAD (1e18) scale.
	bpsDenominator = 10_000
)

// graduationProgress is the "progress" value the board API reports once a token has migrated off the
// bonding curve onto DEX liquidity (100.00%). Pools at/after this point are no longer tradable through
// the Portal bonding curve, so the list updater skips them.
const graduationProgress = "100.00"

// TokenStatus mirrors the status field decoded from Portal.getTokenV8 (TokenStateV8.status), reverse
// engineered from live calls: word[0]==1 for a token still on the curve (progress < 100, matches the
// graduatinghot board), word[0]==2 observed conceptually for status==DEX per Portal._getSwapImplementation
// (status == TokenStatus.DEX routes to PORTAL_DEX_ROUTER instead of PORTAL_TRADE_V2).
type TokenStatus uint8

const (
	TokenStatusInvalid  TokenStatus = 0
	TokenStatusTradable TokenStatus = 1
	TokenStatusDEX      TokenStatus = 2
)

var (
	ErrTokenNotListed     = errors.New("flap: token not listed")
	ErrPoolNotTradable    = errors.New("flap: token is not tradable on the bonding curve (graduated or invalid)")
	ErrZeroAmountOut      = errors.New("flap: zero amount out")
	ErrInsufficientSupply = errors.New("flap: sell amount exceeds circulating supply")

	ErrInvalidEvent = errors.New("flap: invalid TokenCreated event")

	ErrDivByZero                = errors.New("flap: division by zero")
	ErrCurveUnderflow           = errors.New("flap: curve computation underflow")
	ErrSupplyExceedsTotalSupply = errors.New("flap: supply exceeds total supply")
	ErrUnsupportedDecimals      = errors.New("flap: unsupported reserve token decimals (>18)")
	ErrInvalidFeeBps            = errors.New("flap: fee/tax bps out of range")
)
