package efficientprop

import (
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "efficient-prop"

	// A fill's gas, less the refund it earns, by the pool's fill mode, its input (the pair's base or its quote token)
	// and the listing's class: the least first fill of that side in its block, through the adapter (executeEfficientProp),
	// measured with real transactions on a Base fork of the deployed front run on live Base's clock (the block count
	// on its grid, so the front reads no millisecond predeploy), the recipient holding the output token (README,
	// Execution and gas). The tokens' own transfer cost is in each figure, as measured on WETH and USDC (the hook
	// pair) and cbBTC and USDC (past it). The front's mode pays the input straight to the listing's pool and calls the
	// front's paid entry; the door's approves the door, which pulls the input to the pool. Each figure is the adapter
	// call frame's gas less the frame's refund, 19,900 in the front's mode (the adapter's input balance, set and
	// cleared in the transaction) and 39,800 in the door's (that and the door's allowance): what the fill adds to its
	// transaction's receipt. A gas limit is set from the gas before refunds, never from these (README, Gas limits).
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
	// its figure. The four figures past the hook pair were measured again on a fork of Ethereum holding the deployed
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
	// A WORD or NONE listing on Ethereum, one that reads no price source, by the same classes and inputs: the least
	// first fill of its side seen across runs through the adapter, under the same conditions as the print figures above.
	// None was measured on the deployed build: the pools the print figures were measured on list no pair of this kind,
	// so measure them before such a listing is served. The four past the hook pair are the earlier build's, whose print
	// figures past the hook pair stood 2,088 to 2,834 under the deployed build's. The two on the hook pair are a sum,
	// not a measurement: the deployed build's print figures there less the earlier build's print difference (10,087
	// with the base in, 10,287 with the quote in). The earlier build's own figures there were 115,769 and 109,098; the
	// deployed build's print figures on that pair came out 8,148 and 3,046 under the earlier build's, and a charge must
	// not stand above a fill's gas.
	gasEthWordBaseIn             = 107_621
	gasEthWordQuoteIn            = 106_052
	gasEthWordAssetInPast        = 132_553
	gasEthWordOtherInPast        = 123_212
	gasEthWordBaseInPastNoAsset  = 138_621
	gasEthWordQuoteInPastNoAsset = 141_125
	// Ethereum's figures in the front's mode, the same fills through the front's own entries (the paid entry where the
	// front names the listing's pool): measured on the same fork of the deployed build and the same pool as the door's
	// four above, the same runs, the same frame. Past the hook pair with the pool's asset one of the pair's tokens:
	// the asset in on wstETH/USDC (133,036; wstETH/USDT 133,722), the other token in on wstETH/USDC (125,307;
	// wstETH/USDT 126,448); with neither token the pool's asset, both sides of USDT/USDC. Each is 9,885 to 13,031
	// under the door's figure for the same fill (29,785 to 32,931 before refunds: the door's mode earns 19,900 more).
	// The hook pair's two
	// were measured on the two-token pool of the door's hook-pair figures, the same fills. No listing that reads no
	// price source was measured in this mode, so none has a figure: such a listing is quoted in the front's mode only
	// with figures of its own (Front.ListingGas).
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

	// defaultInclusionAllowance is the time a routed fill takes to land, s, where no allowance is configured
	// (Config.InclusionAllowance), and subSecondInclusionAllowance the same on a chain whose blocks are under a second
	// apart (subSecondChains), where a route sent lands within about a second. A shorter allowance quotes a state for
	// longer: a listing whose print or word ends a few seconds after its post stays quotable, and a fill landing later
	// than the allowance is covered by the lane's or the desk's next post (a refresh) and the route's own minimum
	// output, not by the state's window.
	defaultInclusionAllowance   = 3
	subSecondInclusionAllowance = 1
	// builderInclusionAllowance is the same on a front read through /state (Front.Titan), where a quote is good for
	// its target block alone: route finding stops that long before the block's second. A longer allowance drops more
	// of each block's quoting time, and a shorter one quotes routes that may reach the builder after it has built the
	// block, which then revert (README, Ethereum).
	builderInclusionAllowance = 1

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

// builderChains are the chains whose builders sell a block's placement, where a desk's update lands only in a block a
// builder built with it ahead of a fill: a front there is read through /state (Front.Titan) and filled through its
// doors.
var builderChains = map[valueobject.ChainID]bool{valueobject.ChainIDEthereum: true}

var (
	ErrInvalidConfig = errors.New("efficient-prop: invalid config")
	ErrInvalidState  = errors.New("efficient-prop: invalid pool state")
	ErrFrontMismatch = errors.New("efficient-prop: door factory serves another front")
	ErrDoorCodehash  = errors.New("efficient-prop: door factory does not answer its door code hash")
	ErrCellCodehash  = errors.New("efficient-prop: front does not answer the desk code hash the config names")
	ErrNoBlockNumber = errors.New("efficient-prop: multicall returned no block number")
	ErrSecondFill    = errors.New("efficient-prop: direction already filled in this route")
)
