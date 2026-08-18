package parityswapprop

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeParitySwapProp

	methodGetPools      = "getPools"
	methodBase          = "base"
	methodQuote         = "quote"
	methodBaseScale     = "baseScale"
	methodQuoteScale    = "quoteScale"
	methodOracle        = "oracle"
	methodPaused        = "paused"
	methodLastSwapBlock = "lastSwapBlock"
	methodBlockNotional = "blockNotional"
	methodParams        = "params"
	methodGetReserves   = "getReserves"
	methodOracleRead    = "read"

	// defaultGas is the measured gas cost of swap().
	defaultGas = 122000

	// bps is PmmPool.sol's BPS constant, the fee/spread denominator.
	bps = 10_000
)

var (
	ErrPoolPaused          = errors.New("parityswap-prop: pool paused")
	ErrInvalidToken        = errors.New("parityswap-prop: tokenIn is neither base nor quote")
	ErrZeroAmount          = errors.New("parityswap-prop: zero amount")
	ErrStalePrice          = errors.New("parityswap-prop: oracle price older than maxStaleness")
	ErrInvalidOraclePrices = errors.New("parityswap-prop: oracle bid/mid/ask out of order")
	ErrSpreadTooWide       = errors.New("parityswap-prop: oracle spread exceeds maxSpreadBps")
	ErrSwapTooLarge        = errors.New("parityswap-prop: swap notional exceeds maxSwapNotional")
	ErrBlockCapExceeded    = errors.New("parityswap-prop: accumulated block notional exceeds maxBlockNotional")
	ErrInsufficientReserve = errors.New("parityswap-prop: output balance would drop below its floor")

	// ErrOverflow guards every multiplication/addition PmmPool._quote() performs.
	// On-chain, Solidity 0.8's checked arithmetic reverts (panic 0x11) the instant
	// any individual product/sum exceeds 2**256-1 -- not just the final result --
	// so the simulator must fail the same way rather than silently wrapping or
	// computing a wider-precision answer the real contract would never reach.
	ErrOverflow = errors.New("parityswap-prop: uint256 overflow")
)
