package stonkbrokersfunv2

import (
	"github.com/holiman/uint256"
)

// StaticExtra holds per-launch data set once at arm/create and never
// mutated again. One IPoolSimulator instance == one (pad, launchId) launch;
// entity.Pool.Tokens is [token0=project token, token1=pad's quote asset].
//
// LaunchID is the ON-CHAIN launch id (StonkSafeLaunchpadV2.getLaunch's `id`
// param), never the off-chain floor API's synthetic composite id -- see
// the vendor PDF's own top-level id field, which is a frontend composite.
type StaticExtra struct {
	Pad           string `json:"pad"`
	Lens          string `json:"lens"`
	LaunchID      string `json:"launchId"` // uint256, decimal string
	IsWethLane    bool   `json:"isWethLane"`
	QuoteDecimals uint8  `json:"quoteDecimals"`

	// Tax-decay schedule (immutable once armed). currentTaxBps == BufferTaxBps
	// while now < startTime+BufferSecs, then linearly decays
	// DecayPerMinuteBps per minute elapsed past the buffer, floored at 0 (or
	// PostTaxBps if OpenEnded) once decay reaches StartTaxBps. BufferTaxBps is
	// the pad-wide BUFFER_TAX_BPS() constant (same for every launch on a pad).
	BufferTaxBps      uint16 `json:"bufferTaxBps"`
	StartTaxBps       uint16 `json:"startTaxBps"`
	DecayPerMinuteBps uint16 `json:"decayPerMinuteBps"`
	BufferSecs        uint32 `json:"bufferSecs"`
	WindowSecs        uint32 `json:"windowSecs"`
	StartTime         uint64 `json:"startTime"`
	Deadline          uint64 `json:"deadline"`
	OpenEnded         bool   `json:"openEnded"`
	PostTaxBps        uint16 `json:"postTaxBps"`

	// EoaOnly mirrors LaunchModes.eoaOnly. _tradeGates reverts NotEoa() when
	// it is set and msg.sender != tx.origin. An aggregator swap ALWAYS calls
	// buy() from the executor contract, so msg.sender can never equal
	// tx.origin on this path -- an eoaOnly launch is permanently unroutable
	// and CalcAmountOut must refuse it outright rather than quote a trade
	// that is guaranteed to revert. 74 of 283 live launches set this.
	EoaOnly bool `json:"eoaOnly"`

	// MaxBuyPpm mirrors LaunchModes.maxBuyPpm. _buy credits
	// boughtOf[id][recipient] += tokensOut and THEN reverts BuyCapExceeded()
	// if that cumulative total passes loadedSupply*maxBuyPpm/1e6. The
	// simulator cannot see the recipient's existing boughtOf balance, so it
	// enforces the single-trade upper bound only: a quote whose own tokensOut
	// already exceeds the cap can never execute, whoever the recipient is.
	// A trade under that bound may still revert if the recipient already
	// holds a position on this launch -- see the routed-recipient caveat in
	MaxBuyPpm uint32 `json:"maxBuyPpm"`

	// GradMcapUsd8 is the buy-side graduation gate compared against
	// mcapUsd8(id) computed with the POST-trade reserves. Reading that
	// comparison faithfully requires the oracle wiring below -- exactly one
	// of QuoteUsdFeed / TwapPool is set, per the pad's constructor invariant
	// (confirmed live: WETH/GME use QuoteUsdFeed direct feeds, STONK/USDG
	// use TwapPool).
	GradMcapUsd8 uint64 `json:"gradMcapUsd8"`
	LoadedSupply string `json:"loadedSupply"` // uint256, decimal string

	QuoteUsdFeed   string `json:"quoteUsdFeed,omitempty"`
	TwapPool       string `json:"twapPool,omitempty"`
	TwapWindowSecs uint32 `json:"twapWindowSecs,omitempty"`
	EthUsdFeed     string `json:"ethUsdFeed,omitempty"`
	QuoteIsToken0  bool   `json:"quoteIsToken0,omitempty"`
}

// OracleReading is a raw Chainlink AggregatorV3 latestRoundData snapshot,
// used both for direct-feed lanes (quote itself) and for the ETH/USD leg of
// TWAP-mode lanes.
type OracleReading struct {
	Answer    *uint256.Int `json:"answer"` // raw int256 answer; tracker validates > 0 before storing
	Decimals  uint8        `json:"decimals"`
	UpdatedAt uint64       `json:"updatedAt"`
	Ok        bool         `json:"ok"` // false if latestRoundData() reverted or answer <= 0 at fetch time
}

// TwapReading is a raw Uniswap-v3-style observe() snapshot for TWAP-mode
// lanes (STONK, USDG), plus the ETH/USD leg needed to convert the
// quote/WETH tick average into a USD mark.
type TwapReading struct {
	TickCumulativeOld int64         `json:"tickCumulativeOld"` // observe([window, 0])[0]
	TickCumulativeNow int64         `json:"tickCumulativeNow"` // observe([window, 0])[1]
	EthUsd            OracleReading `json:"ethUsd"`
	Ok                bool          `json:"ok"` // false if observe() reverted at fetch time
}

// Extra is rewritten end-to-end on each tracker refresh.
type Extra struct {
	VQuote *uint256.Int `json:"vQuote"`
	VToken *uint256.Int `json:"vToken"`

	SellsEnabled bool `json:"sellsEnabled"`
	Armed        bool `json:"armed"`
	Graduated    bool `json:"graduated"`
	Bonded       bool `json:"bonded"`
	Aborted      bool `json:"aborted"`

	// BuyCount is the launch's on-chain trade counter. The pad increments it
	// on buys AND sells alike (the name is the contract's), never decrements
	// it, so a change between two refreshes is proof the curve moved -- which
	// is what entity.Pool.Timestamp, the clock pool-service archives on, is
	// bumped from. FetchedAt is when this snapshot was read, kept separately
	// so refresh freshness stays legible once Timestamp no longer tracks it.
	BuyCount  uint64 `json:"buyCount"`
	FetchedAt int64  `json:"fetchedAt"`

	// Exactly one of DirectFeed / Twap is populated, matching
	// StaticExtra.QuoteUsdFeed / TwapPool. Backs the buy-side ErrStalePrice
	// enforcement -- ported from SafeLaunchTwapLib (docs/source/.../SafeLaunchTwapLib.sol).
	DirectFeed *OracleReading `json:"directFeed,omitempty"`
	Twap       *TwapReading   `json:"twap,omitempty"`
}

// SwapInfo carries the post-trade virtual reserves for UpdateBalance to
// apply -- CalcAmountOut must not mutate pool state itself.
type SwapInfo struct {
	NewVQuote *uint256.Int `json:"newVQuote"`
	NewVToken *uint256.Int `json:"newVToken"`
	// Graduates is true when this trade's post-trade mcap crosses
	// GradMcapUsd8 -- mirrors StonkSafeLaunchpadV2._buy's `if (mcap >=
	// l.gradMcapUsd8) _close(id, l);`. UpdateBalance sets Extra.Graduated.
	Graduates bool `json:"graduates"`
}

// PoolMeta is what the aggregator's encoder decodes from the pool extra to
// build buy() calldata -- it needs the PAD contract address (the actual
// callable target; entity.Pool.Address/StaticExtra.Pad is a synthetic
// "pad_launchId" composite key, not itself a callable contract) and the
// on-chain LaunchID to pass as buy()'s id param.
type PoolMeta struct {
	Pad      string `json:"pad"`
	LaunchID string `json:"launchId"`

	// ApprovalAddress duplicates Pad under the shared pool.ApprovalInfo JSON
	// contract, which the encoder reads to resolve a custom (non-pool)
	// approval target. The executor must approve the PAD -- not the pool
	// address, which is the synthetic "pad_launchId" composite key -- to
	// spend its quote-asset balance before calling buy().
	ApprovalAddress string `json:"approvalAddress"`
	BlockNumber     uint64 `json:"blockNumber"`
}
