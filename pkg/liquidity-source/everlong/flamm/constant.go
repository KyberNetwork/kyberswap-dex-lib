package everlongflamm

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// DexType is the exchange and pool type of the Everlong FLAMM integration.
const DexType = valueobject.ExchangeEverlongFlamm

// Shared read-only words. They are never a destination, and no function returns or stores one where a caller
// could write through it: results that would be one of these are fresh copies. The big256 words are copied
// too, because big256.BONE, TenPow and U2Pow96 are shared by every package in the library.
var (
	uZero       = uint256.NewInt(0)
	uOne        = uint256.NewInt(1)
	uWad        = new(uint256.Int).Set(big256.BONE)       // 1e18
	uWadSquared = new(uint256.Int).Set(big256.TenPow(36)) // WAD*WAD
	uQ96        = new(uint256.Int).Set(big256.U2Pow96)    // 2**96
	uPpm        = uint256.NewInt(1_000_000)
	uBps        = uint256.NewInt(10_000)
)

// Venues of one pool, as the adapter's `data` word 1 selects them (EverlongFlammAdapter.sol VENUE_SWAP /
// VENUE_LEVERAGE).
const (
	VenueSwap  uint8 = 0
	VenueLever uint8 = 1
)

// Default gas per venue and direction: the largest receipt gas of executeEverlongFlamm measured through the adapter on
// anvil forks of Base, plus 25%, rounded up to 10,000 (fork_gas_test.go TestForkGas at blocks 51313000 and
// 51330060, fork_parity_test.go TestForkParity, and fork_edges_test.go TestForkSequence / TestForkGasGrid at
// 51302915, 51324800 and 51330060).
//
// Gas is state-dependent. A leverage fill is priced on the frozen leverage curve, whose anchor solve is closed-form
// on the half-law piece and a ~60-step bisection over a quartic Bezier on the Hermite piece
// (CollRebalancerMath.sol:190, :333). The plan and the execution each run the hook's quote, four solves up (frame,
// leverageQuote, post-anchor, band assert) and five down (plus _cvRequiredOnAnchor), so the most expensive leverage
// fills start and end on the Hermite piece: every solve bisects. The recovery piece runs fewer (one wall solve
// replaces the strict one, and its y-bisection is two mulDivs a step). The first fills after arming, on the
// half-law piece, cost ~1.9M; Hermite-to-Hermite fills cost 2.89M-2.93M up and 3.30M-3.32M down at every size.
// The rest varies with the Router legs through Morpho, the same bundles the swap venue settles with.
//
// Measured maxima: swap sell 1,085,310 (a notional-capped partial sell, whose hook fill bisects), swap buy 1,202,312,
// lever-up 2,929,633 and lever-down 3,318,066 (a dust lever-down on the Hermite piece right after a lever-up).
// Overridable through Config.
//
// A swap sell the pool clips (notional cap, room, a venue or asset debt cap, Morpho liquidity, the funding ceiling)
// also grows with its input, so its estimate adds two terms to defaultGasSwapSell, from the counts the simulator's
// own settlement ran:
//   - defaultGasSwapSellCapEval per curve solve of EverlongHook._maxInForGrossCap (EverlongHook.sol:583), a bisection
//     over [0, amountIn] of min(64, ~log2(amountIn)) solves, run by every previewExactIn of the plan and again by the
//     execution's executeExactIn (FLAMMSwapLib.sol:87, :197): up to 320 solves;
//   - defaultGasSwapSellPass per funding pass after the first (FLAMMSwapLib.sol:163, at most four): the re-plan's
//     Router.fundingCeiling calls, fee and fill.
//
// A buy runs neither (its fill never bisects, hook.go capFill; its plan has no funding loop). Measured by
// fork_gas_test.go TestForkGasSwapSellClip at 51333700 over 645 mined clipped sells (notional cap, venue debt cap,
// Morpho liquidity, and funding ceilings bound by a lowered pin; 1-4 passes, 24-256 solves, inputs 2^9-2^72): every
// receipt is within 950,917 + 8,201 per solve + 161,247 per extra pass (the steepest per-solve slope of a scenario,
// then the largest step between pass counts). The two slopes plus 25%, rounded up to 100 and 10,000, are the terms;
// the intercept plus 25% (1,188,646) is within defaultGasSwapSell. Estimates were 1.31x-1.56x the receipts across the two recorded fork runs, the highest 1.554x.
const (
	defaultGasSwapSell        int64 = 1_360_000
	defaultGasSwapSellCapEval int64 = 10_300
	defaultGasSwapSellPass    int64 = 210_000
	defaultGasSwapBuy         int64 = 1_510_000
	defaultGasLeverUp         int64 = 3_670_000
	defaultGasLeverDown       int64 = 4_150_000
)

// Default margins (Config): the fill-time robustness a listing gets unless its configuration sets a margin, 0
// included. Each only refuses, so a default trades a sliver of coverage for fewer reverted or re-priced fills:
//   - a minute of feed life and of spread life;
//   - 30 seconds of Morpho accrual a swap must survive with the same amounts (a clipped fill's cap can move);
//   - 50 bps of pool asset feed move, which is one ordinary cbBTC/USD round (below);
//   - a refresh at most 10 minutes old: a refresh that keeps failing leaves pool-service with an entity that sees
//     neither the fills nor the feed rounds since.
//
// defaultPriceBandMarginBps is sized on the pool asset feed's own round sizes, because the quote has to survive
// one round the snapshot has not seen. Measured over Base blocks 51066959-51369359 (167.6 h, 567 cbBTC/USD rounds
// on aggregator 0x51cE3091, the AnswerUpdated logs): consecutive rounds move a median 9.7 bps, p75 28.8, p90 33.6,
// p95 37.8, p99 52.1, max 72.5, with the mass between 28 and 34 bps -- the feed's own deviation threshold -- and
// 49.3% of rounds above 10 bps. A margin of m bps covers a round of m bps: on the committed grids, at 50 bps a
// round of +-10, +-20, +-35 or +-50 bps reverts and re-prices no accepted swap quote at any drift of the snapshot
// from the book that was tried (0, +200, +400 and +600 bps pinned), where at 10 bps the band-pinned state loses 8,
// 47 and 57 of its 610 accepted quotes to +20, +35 and +50 bps rounds. The cover costs 23 of 1,463 accepted swap
// quotes and 23 of 374 leverage quotes, and 50 bps is 1/16th of the deployed band (swapPriceBandWad 8e16). Rounds
// above the margin are still bounded by the refresh: the feed aggregators are in the dependency set
// (pool_tracker.go), so a round triggers one.
//
// defaultLeverMinEdgeBps keeps the routed quote on the swap venue unless the leverage venue pays more than this
// many bps above it (pickVenue). The leverage venue costs 2.31M more gas up and 2.64M more down than the swap
// venue (the gas defaults above), which at Base's observed gas price of 0.006 gwei and ETH at $2,407 is $0.033
// and $0.038: 10 bps covers it from about $40 of route value up, and below that the whole difference is under
// four cents. A bps edge is not a gas cost, so it removes the thin-gain tail rather than making the choice
// gas-aware; gas-aware venue selection stays a condition on enabling LeverRouting (README).
const (
	defaultPriceBandMarginBps uint64 = 50
	defaultPriceAgeMarginSec  uint64 = 60
	defaultSpreadAgeMarginSec uint64 = 60
	defaultDebtDriftSec       uint64 = 30
	defaultMaxSnapshotAgeSec  uint64 = 600
	defaultLeverMinEdgeBps    uint64 = 10
)

// maxVenues caps the Router venue set a pool may be listed and quoted with: every venue costs a fixed number of
// reads per refresh and a leg per settlement pass, and a pool past this is outside what the port was measured on
// and is refused. It is a port-side ceiling above the Router's own, which is eight and binding for every pool
// forever (MMRouterLib.sol:227 `if (n >= MAX_VENUES) revert TooManyVenues();` over IMMRouter.sol:13
// `MAX_VENUES = 8`, on the one site that grows the array; retirement flags a venue and never pops it, so the
// length is monotonic). On a Router of the registered code, therefore, this refusal cannot fire: it bounds a
// Router that raises its own cap, and such a Router is new code, to be registered only after the port is measured
// on it.
const maxVenues = 64

// scheduledChangeLeadSec is how long before a scheduled implementation, hook-set, venue or loan-asset change
// becomes executable the simulator stops quoting (Extra.ScheduledChangeAt, tracker_reads.go scheduledChangeAt): a
// fill quoted now can land that much later, after anyone executed the upgrade in an earlier block.
const scheduledChangeLeadSec uint64 = 1800

// Refusals of the integration layer. The pool's own reverts are the sentinels in errors.go.
var (
	ErrInvalidProfile   = errors.New("everlong-flamm: pool is not a supported profile or its listing disagrees with it")
	ErrNotAttested      = errors.New("everlong-flamm: snapshot was not attested against the pool's own previews")
	ErrProfileDrift     = errors.New("everlong-flamm: on-chain wiring drifted from the listed profile")
	ErrPoolRefused      = errors.New("everlong-flamm: pool state is outside the quotable envelope")
	ErrInvalidToken     = errors.New("everlong-flamm: invalid token pair")
	ErrZeroAmountOut    = errors.New("everlong-flamm: zero amount out")
	ErrFeedAgeMargin    = errors.New("everlong-flamm: a price feed round expires within the configured margin")
	ErrSpreadNotLive    = errors.New("everlong-flamm: leverage spread is not live through the configured margin")
	ErrBandMargin       = errors.New("everlong-flamm: fill is within the configured margin of the price band")
	ErrFeedMoveMargin   = errors.New("everlong-flamm: fill does not survive a pool asset feed move of the configured margin")
	ErrDebtDrift        = errors.New("everlong-flamm: fill does not survive the configured accrual drift")
	ErrOracleDrift      = errors.New("everlong-flamm: fill does not survive the market oracle's end-of-window answer")
	ErrSwapInfoMismatch = errors.New("everlong-flamm: swap info was quoted on a different state")
	ErrSnapshotStale    = errors.New("everlong-flamm: snapshot is older than the configured maximum age")
	ErrScheduledChange  = errors.New(
		"everlong-flamm: a scheduled implementation, hook-set, venue or loan-asset change is executable")
)
