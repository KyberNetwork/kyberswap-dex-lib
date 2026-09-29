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

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmountIn       = errors.New("invalid amount in")
	ErrInsufficientOutput    = errors.New("insufficient output amount")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity for full swap")
	ErrSwapMovementExceeded  = errors.New("swap exceeds movement bound from entry bin")
	ErrPriceNotRepresentable = errors.New("bin price not representable")
	ErrMathOverflow          = errors.New("math overflow")
)
