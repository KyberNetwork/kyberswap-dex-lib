package stonkbrokersfunv2

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// DexType covers only the Smart Launch V2 pads (mint & standard launches) on
// Robinhood Chain (chainId 4663) -- StonkSafeLaunchpadV2 + SafeLaunchLensV2.
// Out of scope, deliberately: V3 pads (external-token/BYO launches), the
// post-bond DEX leg, and the separate Stonk Launcher factory (unverified on
// Blockscout, never called here).
const (
	DexType = valueobject.ExchangeStonkbrokersFunV2

	// Scope decision -- BUY ONLY (quote -> token). Investigated during
	// the scope decision recorded below ("Scope
	// decision: buy-only" for full evidence):
	//
	//   sell(id, tokensIn, minQuoteOut, ref, account, ethOut) requires
	//   EITHER account == msg.sender, OR operatorOf[account][id][msg.sender]
	//   == true. operatorOf is set exclusively by
	//   setOperator(id, operator, approved) which hardcodes
	//   `operatorOf[msg.sender][id][operator] = approved` -- only the
	//   position holder can grant it, for themselves, per launch, with no
	//   permit/signature variant to bundle into a router's swap calldata.
	//   A stateless aggregator route never becomes the position holder
	//   (boughtOf[id][executor] is 0 unless the executor itself bought on
	//   this exact launch earlier), so it can neither satisfy
	//   account == msg.sender with real credit, nor obtain a same-tx
	//   operator grant. Per the integration doc's own terminal note,
	//   "Transferred bags cannot be sold through the pad" -- boughtOf
	//   credit is non-transferable, so this holds even for tokens the
	//   end user already holds from elsewhere. Sell is not implemented.
	//
	// buy(id, quoteIn, minTokensOut, ref, recipient) has none of this
	// friction: quoteIn is pulled from msg.sender (the router), tokens and
	// boughtOf credit land on `recipient` (the end user) -- exactly
	// KyberSwap's standard executor shape.

	methodLaunchCount     = "launchCount"
	methodGetLaunch       = "getLaunch"
	methodModesOf         = "modesOf"
	methodBufferSecsOf    = "bufferSecsOf"
	methodCurrentTaxBps   = "currentTaxBps"
	methodLaunchIdOfToken = "launchIdOfToken"
	methodQuote           = "quote"
	methodQuoteDecimals   = "quoteDecimals"
	methodIsWethLane      = "isWethLane"
	methodQuoteIsToken0   = "quoteIsToken0"
	methodQuoteUsdFeed    = "quoteUsdFeed"
	methodTwapPool        = "twapPool"
	methodTwapWindowSecs  = "twapWindowSecs"
	methodEthUsdFeed      = "ethUsdFeed"
	methodBufferTaxBps    = "BUFFER_TAX_BPS"

	// MethodBuy is exported because the swap is BUILT elsewhere: this package
	// only ever quotes, while the aggregator's encoder packs the calldata off
	// PadABI. Sharing the name keeps the two in step.
	//
	// buyEth() (payable, WETH lane only) is deliberately not used: the executor
	// already holds wrapped WETH by the time it reaches this pool, so buy()
	// avoids an unwrap-then-rewrap round trip inside the pad.
	MethodBuy = "buy"

	// The lens exposes viewLaunch/viewLaunches too; we read launch state off the
	// pad directly and only use the lens as an independent quote oracle in tests.
	methodQuoteBuy = "quoteBuy"

	bps = 10_000

	// defaultGas is the measured cost of a real buyEth() on the WETH pad: a
	// Tenderly fork of Robinhood Chain at block 46382449 executed
	// buyEth(176, 0, 0x0, recipient) with 0.01 ETH for 265,009 gas
	// That trade took the
	// ordinary path -- no graduation close, launch already had a prior buy --
	// so a first buy into a launch, or one that trips the graduation branch,
	// costs more than this.
	defaultGas = 265_000
)

var (
	ErrPoolNotArmed     = errors.New("launch not armed")
	ErrPoolGraduated    = errors.New("launch graduated, trade on pad closed")
	ErrPoolBonded       = errors.New("launch bonded, trade via the locked DEX pool instead")
	ErrPoolAborted      = errors.New("launch aborted")
	ErrWindowClosed     = errors.New("curve window closed")
	ErrZeroTrade        = errors.New("zero trade amount")
	ErrSlippageExceeded = errors.New("slippage exceeded")
	ErrBuyCapExceeded   = errors.New("per-recipient buy cap exceeded (StonkSafeLaunchpadV2 BuyCapExceeded)")
	ErrNoSnapshotBlock  = errors.New("multicall returned no block number to pin the refresh to")
	ErrEoaOnly          = errors.New("launch is eoaOnly: buy() reverts NotEoa when msg.sender != tx.origin, so it is unreachable from an aggregator executor")

	// ErrStalePrice mirrors the on-chain revert: buy() calls mcapUsd8(id)
	// unconditionally (no try/catch) to check the graduation gate, and
	// mcapUsd8 reverts if the pad's oracle (direct Chainlink feed or TWAP
	// pool, per-pad -- see StaticExtra) is stale. Stock lanes
	// (GME/NVDA/AAPL/SPCX/USO) go stale over weekends (documented worst
	// case 79.7h gap). The simulator ports SafeLaunchTwapLib's
	// staleness window and return this error rather than a quote whenever
	// the tracked oracle snapshot is stale -- ported from AGENTS.md's "track
	// every flag/check on the swap path so the simulator can reject swaps
	// that would revert on-chain."
	ErrStalePrice        = errors.New("quote-asset oracle stale, buy blocked (graduation mcap unreadable on-chain)")
	ErrBadOracleAnswer   = errors.New("oracle answer invalid (<= 0, unscalable, or overflow)")
	ErrInvalidToken      = errors.New("invalid tokenIn/tokenOut for this pool")
	ErrInvalidPoolTokens = errors.New("pool must have exactly 2 tokens")

	// ErrSellNotSupported documents the buy-only scope decision above for
	// any future caller that tries CalcAmountIn or a token->quote direction.
	ErrSellNotSupported = errors.New("sell not supported by this integration: requires a caller-owned boughtOf credit or a pre-granted per-launch operator approval, unreachable from a generic aggregator swap")
)
