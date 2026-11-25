package efficientprop

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "efficient-prop"

	// A fill's gas, less the refund it earns, by the pool's fill mode, its input (the pair's base or its quote token) and
	// the listing's class: the least first fill of that side in its block, through the adapter (executeEfficientProp),
	// measured with real transactions on a Base fork of the front's earlier storage build (its desk before the packed
	// desk; that deployment is retired) run on live Base's clock (the block count on its grid, so the front reads no
	// millisecond predeploy), the recipient holding the output token (README, Execution and gas). They are not yet
	// measured on the deployed build, whose desk is new code (the packed desk). The tokens' own transfer cost is in each
	// figure, as measured on WETH and USDC (the hook pair) and cbBTC and USDC (past it). The front's mode pays the input
	// straight to the listing's pool and calls the front's paid entry; the door's approves the door, which pulls the input
	// to the pool. Each figure is the adapter call frame's gas less the frame's refund, 19,900 in the front's mode (the
	// adapter's input balance, set and cleared in the transaction) and 39,800 in the door's (that and the door's
	// allowance): what the fill adds to its transaction's receipt. A gas limit is set from the gas before refunds, never
	// from these (README, Gas limits).
	//
	// A WORD or NONE listing on its pool's hook pair (base index 0, quote index 1), measured on a WORD WETH/USDC listing.
	gasWordFrontBaseIn  = 101_664
	gasWordFrontQuoteIn = 93_570
	gasWordDoorBaseIn   = 114_072
	gasWordDoorQuoteIn  = 109_146
	// The same on a pair past the hook pair, measured on a WORD cbBTC/USDC listing (cbBTC the base) of a three-token
	// pool.
	gasWordFrontBaseInPast  = 129_338
	gasWordFrontQuoteInPast = 129_007
	gasWordDoorBaseInPast   = 145_654
	gasWordDoorQuoteInPast  = 144_523
	// A listing whose fill reads a price source (PRINT or SESSION) adds the print difference measured on a PRINT
	// WETH/USDC listing bound to the protocol's stored price lane, against the WORD listing, the same in either mode:
	// its source word, the source's check and its slots. On the hook pair the sum is the PRINT listing's own measured
	// figure; past it the sum was not measured. A listing of an asset-module pair (the pair word's bit 15), whose fill
	// runs its module's checks, was not measured: it is quoted only with gas of its own (Front.ListingGas).
	gasPrintBaseIn  = 12_562
	gasPrintQuoteIn = 12_806
	// A listing bound to a guarded source, whose fill asks the source's lane for the side's worst held print: on the
	// hook pair, measured on a PRINT WETH/USDC listing bound to a guarded entry of the chain's hold lane, with no print
	// held at the fill (a fill inside a hold, in the post's block or the next, costs about 5,000 more); past the hook
	// pair, the WORD figure of the class plus the guarded difference measured against the WORD listing beside it (a
	// sum, not a measurement).
	gasGuardedFrontBaseIn      = 120_205
	gasGuardedFrontQuoteIn     = 112_108
	gasGuardedDoorBaseIn       = 132_360
	gasGuardedDoorQuoteIn      = 127_678
	gasGuardedFrontBaseInDiff  = 18_288
	gasGuardedFrontQuoteInDiff = 18_285
	gasGuardedDoorBaseInDiff   = 18_035
	gasGuardedDoorQuoteInDiff  = 18_279

	// Ethereum's figures in the door's mode, by the input token and by the pair's place: on its
	// pool's hook pair (the base the pool's asset, token 0, in; or the quote in); past it, with the pool's asset one of
	// its tokens (the asset in, on whichever side it is; or the other token in); or past it with neither token the
	// pool's asset (the base in; or the quote in). A listing that reads a price source (PRINT or SESSION): the least
	// first fill of its side seen across runs through the adapter, less its refund, the recipient holding the output
	// token, the fill behind the quote contracts' updates in their block, the front's clock the block's second.
	// Same-size first fills vary by up to about 500 gas from run to run, so a later run may see a fill a little under
	// its figure. The four figures past the hook pair were measured on a fork of Ethereum holding the deployed
	// build (front 0x0000000000403aeB66691063CC2fD42E71d6AcD6) and a pool of wstETH, WETH, USDC and USDT: the asset in
	// on wstETH/USDC (144,744; wstETH/USDT 145,430), the other token in on wstETH/USDT (136,333; wstETH/USDC 138,338),
	// and both sides of USDT/USDC. A class's figure is the least of its listings, so the class's other listing spends up
	// to about 2,100 more than it is charged. The two hook-pair figures were measured on the same build on a pool of
	// two tokens, wstETH and WETH, its listing with no credit mode: wstETH in and WETH in. The pool had no lender; the
	// same pool with a lender bound and unused spent the same within 16 gas. A hook-pair listing with a credit mode is
	// charged its configured figures instead (Front.CreditListings), which are higher: its fill reads its pool's room
	// and its input passes through the door.
	gasEthPrintBaseIn             = 117_708
	gasEthPrintQuoteIn            = 116_339
	gasEthPrintAssetInPast        = 144_744
	gasEthPrintOtherInPast        = 136_333
	gasEthPrintBaseInPastNoAsset  = 150_796
	gasEthPrintQuoteInPastNoAsset = 154_012
	// No listing that reads no price source (WORD or NONE) was measured on the deployed build in either mode, nor a pair
	// past the hook pair whose quote is the pool's asset: those classes have no figure, and such a listing is quoted
	// only with figures of its own (Front.ListingGas).
	//
	// Ethereum's figures in the front's mode, the same fills through the front's own entries (the paid entry where the
	// front names the listing's pool): measured on the same fork of the deployed build and the same pool as the door's
	// four above, the same runs, the same frame. Past the hook pair with the pool's asset one of the pair's tokens:
	// the asset in on wstETH/USDC (133,036; wstETH/USDT 133,722), the other token in on wstETH/USDC (125,307;
	// wstETH/USDT 126,448); with neither token the pool's asset, both sides of USDT/USDC. Each is 9,885 to 13,031
	// under the door's figure for the same fill (29,785 to 32,931 before refunds: the door's mode earns 19,900 more).
	// The hook pair's two were measured on the two-token pool of the door's hook-pair figures, the same fills.
	gasEthFrontPrintBaseIn             = 106_000
	gasEthFrontPrintQuoteIn            = 106_561
	gasEthFrontPrintAssetInPast        = 133_036
	gasEthFrontPrintOtherInPast        = 125_307
	gasEthFrontPrintBaseInPastNoAsset  = 140_911
	gasEthFrontPrintQuoteInPastNoAsset = 141_041

	// gasFirstOfInput is what a fill costs more where its pool holds none of the input token before it: the pool's
	// balance of the token is then written from zero. The least difference measured, on the wstETH/WETH listing with
	// WETH in, the pool holding no WETH against the pool holding some: 145,957 against 128,913 through the door
	// (17,068 through the front); the EVM's own difference between a slot set from zero and one changed is 17,100.
	// Charged on any listing and either mode by that rule, for a token whose balance is a plain storage word.
	gasFirstOfInput = 17_044

	// maxAge bounds quoting a sampled ladder, beside its word's end (Extra.ValidUntil). A listing's operator re-posts
	// its word every few seconds, and the front widens a word's levels with its age, so a ladder sampled longer ago
	// prices a book that has moved.
	maxAge = 10 * time.Second

	// refreshEvery is how often the tracker asks for a listing's refresh (NextUpdateAt), event or none.
	refreshEvery = time.Second
	// idleRefreshEvery is the same for a listing whose state holds no ladder (dark, retired, its desk silent or not
	// approved): nothing of it is quoted, so it is read less often until it is. It is no multiple of Ethereum's 12 s
	// slot, so an idle listing is not sampled at one moment of every slot, where a desk that publishes later in the
	// slot would never be seen. On BNB Chain both are several blocks: a refresh is asked for in whole seconds, and a
	// state whose block has passed asks for its next at once (NextUpdateAt).
	idleRefreshEvery = 5 * time.Second
	// stateShareWindow is how long one fetch of a front's /state answers serves the refreshes of its listings: they
	// come together, a listing each, and one fetch a url stands for all of them.
	stateShareWindow = 250 * time.Millisecond
	// defaultStateTimeout is the /state read's timeout where a front's titan.timeout is unset, and maxStateTimeout the
	// longest one a config may set. The fetch holds the front's lock, so a host that hangs holds every listing of the
	// front for as long as the timeout lets it.
	defaultStateTimeout = time.Second
	maxStateTimeout     = 10 * time.Second

	// defaultInclusionAllowance is the time a routed fill takes to land, s, where no allowance is configured
	// (Config.InclusionAllowance), and subSecondInclusionAllowance the same on a chain whose blocks are under a second
	// apart (subSecondChains), where a route sent lands within about a second. A shorter allowance quotes a state for
	// longer: a listing whose print or word ends a few seconds after its post stays quotable, and a fill landing later
	// than the allowance is covered by the lane's or the desk's next post (a refresh) and the route's own minimum
	// output, not by the state's window.
	defaultInclusionAllowance   = 3
	subSecondInclusionAllowance = 1
	// On a front read through /state (Front.Titan) a quote is good for its target block alone, and route finding stops
	// the chain's allowance before the block's time (builderClock.allowanceMs). A longer allowance drops more of each
	// block's quoting time, and a shorter one quotes routes that may reach the builder after it has built the block,
	// which then revert (README, Ethereum and BNB Chain).

	// scanBatch is how many listing ids the lister reads in one multicall, and defaultMaxListings how many it reads
	// from one front in one poll at most.
	scanBatch          = 64
	defaultMaxListings = 4096

	// maxLadderPoints bounds one direction's ladder, so a malformed state cannot make a quote walk far.
	maxLadderPoints = 64
	// maxPointBits bounds every sampled amount: the front takes no input or output above 2^128, and the chord's
	// products stay inside 256 bits.
	maxPointBits = 128

	// flagLive is doorView's status flag for a live listing (flags bit 0).
	flagLive = 1

	// defaultHoldMs is the hold a guarded source's lane is taken to have where its HOLD_MS cannot be read: the most a
	// stored price lane holds a replaced print.
	defaultHoldMs = 2_000
)

// subSecondChains are the chains whose blocks are under a second apart, where the inclusion allowance defaults to
// subSecondInclusionAllowance.
var subSecondChains = map[valueobject.ChainID]bool{valueobject.ChainIDBase: true}

// defaultMinTipCapWei is each chain's tip cap floor where the config names none (Config.MinTipCapWei): a listing
// whose cap is under it reverts fills an ordinary transaction would send, at the taker's gas, so it is not quoted.
// Ethereum's is 2 gwei, from eth_feeHistory over the 7,168 blocks to 26,486,699 (one day to 24 Nov 2026, a public
// node): the 90th-percentile priority fee of a block, by gas, was 1.0 gwei at the median block, at most 2.0 gwei in
// 90% of blocks and at most 2.1 gwei in 99% (the median transaction's: 0.011 gwei at the median block). So a cap of 2
// gwei takes nine transactions in ten in nine blocks in ten, and a cap under it turns ordinary wallet tips away. BNB
// Chain's is 0.1 gwei: its base fee is 0, its validators' floor 0.05 gwei, and its tips read on 25 Nov 2026 were 0.05
// gwei at the 10th percentile, about 0.05 at the median and 0.1 at the 90th. No other chain has a default: a chain
// read at the head sets its own.
var defaultMinTipCapWei = map[valueobject.ChainID]uint64{valueobject.ChainIDEthereum: 2_000_000_000,
	valueobject.ChainIDBSC: 100_000_000}

// builderChains are the chains whose builders sell a block's placement, where a desk's update lands only in a block a
// builder built with it ahead of a fill: a front there is read through /state (Front.Titan) and filled through its
// doors. Each is read on its own block clock (builderClocks).
var builderChains = map[valueobject.ChainID]bool{valueobject.ChainIDEthereum: true, valueobject.ChainIDBSC: true}

// builderClock is how a builder chain's blocks fall in time, as its /state read follows them, and the read's own
// timings there.
type builderClock struct {
	// blockMs is the time from one block to the next, ms
	blockMs uint64
	// beacon: an answer's slot is the beacon chain's, whose second is its target block's (Ethereum). Else its target
	// block's time is no field of the answer: it is the head's milliseconds (BNB Chain's milliTimestamp, else the start
	// of the head's second) plus one block, and the block may be late, so the second after it is sampled too
	beacon bool
	// allowanceMs is the default inclusion allowance (Config.InclusionAllowanceMs): how long before its target block
	// a route quoted from a state must reach the block's builder
	allowanceMs uint64
	// shareWindow is how long one /state fetch and one head probe serve the refreshes of a front's listings
	// (stateShareWindow on Ethereum), mergeWindow how long a fetch waits for the other urls after the first answer that
	// holds an account (titan.DefaultMergeWindow on Ethereum), and timeout the fetch's default timeout
	// (defaultStateTimeout on Ethereum): on BNB Chain each is a fraction of its 450 ms block, past which a state is
	// for a block already built
	shareWindow, mergeWindow, timeout time.Duration
	// classGas: the chain's builder build was measured, so a listing of a measured class is charged its class's
	// figure (gasOfBuilder). Elsewhere a listing is quoted only with figures of its own (Front.ListingGas,
	// Front.CreditListings)
	classGas bool
	// msPrecompile: the chain's millisecond precompile (BEP-706, msPrecompile) is read beside the head read, and a
	// front whose clock reads it (mode 3) is quoted only while it answers nothing, when the front reads the block's
	// second (its seal time); once it answers, the clock reads the block's own milliseconds, which no block override
	// sets, so nothing is quoted
	msPrecompile bool
}

// msPrecompile is BNB Chain's millisecond timestamp precompile (BEP-706), which a mode-3 front clock reads.
var msPrecompile = common.HexToAddress("0x0000000000000000000000000000000000000070")

// builderClocks are the builder chains' clocks. Ethereum: 12 s slots, each block at its beacon slot's second, a 1 s
// allowance. BNB Chain: 450 ms blocks (450.4 ms on average over the 30,000 blocks to 135,693,393, 25 Nov 2026), each
// header's milliseconds in its milliTimestamp, and an allowance of 100 ms, the time before a block's end by which its
// builder (48 Club) takes the bundles it places a desk's update ahead of.
var builderClocks = map[valueobject.ChainID]builderClock{
	valueobject.ChainIDEthereum: {blockMs: 12_000, beacon: true, allowanceMs: 1_000, shareWindow: stateShareWindow,
		mergeWindow: titan.DefaultMergeWindow, timeout: defaultStateTimeout, classGas: true},
	valueobject.ChainIDBSC: {blockMs: 450, allowanceMs: 100, shareWindow: 100 * time.Millisecond,
		mergeWindow: 100 * time.Millisecond, timeout: 400 * time.Millisecond, msPrecompile: true},
}

var (
	ErrInvalidConfig = errors.New("efficient-prop: invalid config")
	ErrInvalidState  = errors.New("efficient-prop: invalid pool state")
	ErrFrontMismatch = errors.New("efficient-prop: door factory serves another front")
	ErrDoorCodehash  = errors.New("efficient-prop: door factory does not answer its door code hash")
	ErrCellCodehash  = errors.New("efficient-prop: front does not answer the desk code hash the config names")
	ErrNoBlockNumber = errors.New("efficient-prop: multicall returned no block number")
	ErrSecondFill    = errors.New("efficient-prop: direction already filled in this route, no further fill is proven")
)
