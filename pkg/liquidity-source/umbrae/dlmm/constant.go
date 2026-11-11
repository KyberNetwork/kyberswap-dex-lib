package umbraedlmm

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeUmbraeDlmm

	// defaultGas is a base cost; per-bin traversal adds gasPerBin.
	defaultGas int64 = 180000
	gasPerBin  int64 = 12000

	// activeBinID is the 1:1 centre bin (2^23) — the price exponent's zero point.
	activeBinID uint32 = 8388608

	// minActiveBin is SwapCalculator.MIN_ACTIVE_BIN (2^22). A volatilityReference below it means
	// "unset" — the anchored-volatility window starts fresh at the entry bin.
	minActiveBin uint32 = 4194304

	// Fee/price precision constants (mirror FeeHelper / BinHelper).
	basisPoints = 10000
	maxFeeBps   = 500 // FeeHelper.MAX_FEE — 5% total fee cap

	// Swap movement bound (SwapMovement.isAllowed, V2 #278): a swap may not reach a bin further
	// than MAX_BIN_ID_DISTANCE ids from the entry bin, nor further than MAX_CUMULATIVE_STEP_BPS
	// worth of binStep. Execution reverts past the bound, so the simulator must refuse to quote
	// through it.
	maxBinIDDistance     uint64 = 2048
	maxCumulativeStepBps uint64 = 10000

	// maxSwapBinsTraversed is SwapMovement.MAX_SWAP_BINS_TRAVERSED (revision 1 only): a swap may
	// move to a new bin at most 100 times; execution reverts on the 101st move.
	maxSwapBinsTraversed = 100

	factoryMethodAllPairs          = "allPairs"
	factoryMethodAllPairsLength    = "allPairsLength"
	factoryMethodGetVariableFeeCap = "getVariableFeeCap"

	pairMethodTokenX        = "tokenX"
	pairMethodTokenY        = "tokenY"
	pairMethodBinStep       = "binStep"
	pairMethodGetDecimals   = "getDecimals"
	pairMethodGetActiveID   = "getActiveId"
	pairMethodGetQuoteState = "getQuoteState"
	pairMethodGetReserves   = "getReserves"

	viewerMethodActiveBins = "getActiveBinsWithReserves"
	viewerMethodQuoteSwap  = "quoteSwap"
)

// Swap revisions: which deployed pair build the simulator mirrors. A deployment sets it in
// Config.SwapRevision; the lister persists it per pool in StaticExtra.
const (
	// SwapRevisionV2 is the V2 stack deployed on Base on 2026-08-24.
	SwapRevisionV2 uint8 = 0
	// SwapRevisionV2Oct2026 is the 2026-10 audit build (Robinhood Chain, deployed 2026-10-15). It
	// adds, over SwapRevisionV2:
	//   - #1212 volatility carry: once a new filter window opens, a swap starts from
	//     reductionFactor/10000 of the decayed volatility (FeeHelper.getCarriedVolatility);
	//   - #1211 dust: when the input left after emptying a bin buys nothing (0 out at that bin, or
	//     1 there and 0 at the adjacent bin), that bin takes the whole remaining input;
	//   - the 100-move traversal limit (maxSwapBinsTraversed).
	SwapRevisionV2Oct2026 uint8 = 1
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmountIn       = errors.New("invalid amount in")
	ErrInsufficientOutput    = errors.New("insufficient output amount")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity for full swap")
	ErrSwapMovementExceeded  = errors.New("swap exceeds movement bound from entry bin")
	ErrSwapTraversalLimit    = errors.New("swap exceeds the bin traversal limit")
	ErrPriceNotRepresentable = errors.New("bin price not representable")
	ErrMathOverflow          = errors.New("math overflow")
)
