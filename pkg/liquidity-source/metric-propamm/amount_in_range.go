package metricpropamm

import (
	"math/big"
	"time"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// AmountInRange implements pool.IAmountInRange: it reports how much tokenIn this pool will quote
// without being asked to quote any particular size.
//
// This pool prices against a bin book, and getRate walks it accumulating each bin's volume until
// the requested amount is covered. When the book runs out first it returns ErrInsufficientLiquidity
// -- a capacity limit, indistinguishable to a caller from a refusal on price. Reporting the book's
// total depth lets the caller sample inside it instead of discarding the pool.
//
// # Why this is exact rather than an approximation
//
// Bin.CumulativeVolume is denominated in *amountOut*, so the book's depth is not directly a maxIn.
// But getRate does not compare amountIn against those volumes either: it converts each bin's
// volume into tokenIn units first (its convertedVolume, via calculateAmountFromPrice with the
// direction flipped) and decrements a remainder initialized to amountIn. The walk therefore covers
// the request exactly when
//
//	amountIn <= sum over all bins of convertedVolume
//
// so summing the same per-bin conversion reproduces the refusal boundary rather than estimating it.
// No inversion of the price curve is involved; the conversion below is the loop's own arithmetic.
//
// There is no minimum: any positive amount within the first bin is quotable, so minIn is nil.
//
// # What this bound does not include
//
// The book's depth only. CalcAmountOut can still refuse a size inside it because the resulting
// amountOut exceeds the output reserve. That is a bound on amountOut, so folding it in would mean
// inverting the fill-price computation, and doing so approximately would *understate* the range --
// the one direction the interface forbids, because a caller detects an overstated bound by probing
// inside it and being refused, while an understated one silently shrinks what it will consider. So
// this overstates in that case on purpose and leaves it to the probe: a successful AmountInRange
// promises only that nothing above maxIn can quote.
func (s *PoolSimulator) AmountInRange(tokenIn, tokenOut string, _ pool.SwapLimit) (*big.Int, *big.Int, error) {
	if s.GetTokenIndex(tokenIn) < 0 || s.GetTokenIndex(tokenOut) < 0 || tokenIn == tokenOut {
		return nil, nil, ErrInvalidToken
	}

	// Mirror CalcAmountOut's two gates, including the noStaleCheck escape the factory sets during
	// indexing. When either holds the pool quotes nothing at any size, so a zero maximum is the
	// exact answer and spares the caller a probe that cannot succeed.
	if !s.extra.QuoteAvailable {
		return nil, new(big.Int), nil
	}
	if !s.noStaleCheck && s.poolTimestamp+s.extra.MaxAge < time.Now().Unix() {
		return nil, new(big.Int), nil
	}

	// getRate picks the side by tokenIn being token0, not by token index, so this must too.
	zeroToOne := tokenIn == s.Info.Tokens[0]
	bins := lo.Ternary(zeroToOne, s.extra.Bids, s.extra.Asks)

	maxIn, prevCumulative := new(big.Int), new(big.Int)
	for _, bin := range bins {
		if bin.CumulativeVolume == nil || bin.Price == nil || bin.Price.Sign() <= 0 {
			// calculateAmountFromPrice divides by the price, so a missing or non-positive one is
			// unusable. Report that the pool cannot say rather than panic in a routing hot path.
			return nil, nil, pool.ErrUnsupported
		}
		volume := new(big.Int).Sub(bin.CumulativeVolume, prevCumulative)
		if volume.Sign() < 0 {
			// CumulativeVolume is meant to be non-decreasing. Where it isn't, getRate's running
			// remainder stops being monotone and the sum no longer predicts where it refuses, so
			// the equivalence this method rests on does not hold. Decline instead of guessing.
			return nil, nil, pool.ErrUnsupported
		}
		maxIn.Add(maxIn, s.calculateAmountFromPrice(!zeroToOne, volume, bin.Price))
		prevCumulative = bin.CumulativeVolume
	}

	// An empty book sums to zero, which is right: getRate refuses every positive amount.
	return nil, maxIn, nil
}
