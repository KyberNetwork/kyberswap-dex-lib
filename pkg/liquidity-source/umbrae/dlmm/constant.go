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

	// Fee/price precision constants (mirror FeeHelper / BinHelper).
	basisPoints = 10000
	maxFeeBps   = 500 // FeeHelper.MAX_FEE — 5% total fee cap

	factoryMethodAllPairs          = "allPairs"
	factoryMethodAllPairsLength    = "allPairsLength"
	factoryMethodGetVariableFeeCap = "getVariableFeeCap"

	pairMethodTokenX            = "tokenX"
	pairMethodTokenY            = "tokenY"
	pairMethodBinStep           = "binStep"
	pairMethodGetDecimals       = "getDecimals"
	pairMethodGetActiveID       = "getActiveId"
	pairMethodGetQuoteState     = "getQuoteState"
	pairMethodGetPairStatistics = "getPairStatistics"

	viewerMethodActiveBins = "getActiveBinsWithReserves"
	viewerMethodQuoteSwap  = "quoteSwap"
)

var (
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidReserve     = errors.New("invalid reserve")
	ErrInvalidAmountIn    = errors.New("invalid amount in")
	ErrInsufficientOutput = errors.New("insufficient output amount")
	ErrInvalidPrice       = errors.New("invalid bin price")
)
