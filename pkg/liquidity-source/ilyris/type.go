package ilyris

import "math/big"

// StaticExtra is the part of a pool that never changes after creation. Serialised once by the
// lister and never rewritten by the tracker, so anything that CAN move must not live here.
type StaticExtra struct {
	BinStepBps uint32 `json:"binStepBps"`
	DecimalsX  uint8  `json:"decimalsX"`
	DecimalsY  uint8  `json:"decimalsY"`
}

// Extra is the mutable snapshot the tracker refreshes.
type Extra struct {
	ActiveID int32     `json:"activeId"`
	Bins     []BinJSON `json:"bins"`

	// TotalFeeRate is in FEE_PRECISION units (1e9) and ALREADY includes the volatility
	// component. Stored resolved rather than as its parts because the contract owns the fee
	// model: re-deriving it here would be a second source of truth, and the way that fails is
	// a fee that disagrees with the chain by a few units and misprices every quote.
	TotalFeeRate uint64 `json:"totalFeeRate"`

	// Guard state, read at the SAME BLOCK as the book above.
	//
	// Carried in Extra rather than fetched at quote time because a simulator has no network:
	// it prices from this snapshot. And it must be here at all because the pool does not
	// consult the guard when quoting -- _guardSwap runs in swapExactIn/swapExactOut but not in
	// quoteExactIn/quoteExactOut -- so a quote-derived model is blind to a closed market.
	MarketGuard      string `json:"marketGuard"`
	GuardSwapsPaused bool   `json:"guardSwapsPaused"`
	GuardFreezeEnd   uint64 `json:"guardFreezeEnd"`
	BlockTimestamp   uint64 `json:"blockTimestamp"`
}

// BinJSON is one bin on the wire. Reserves are decimal strings: they are uint128 on chain and
// JSON numbers are float64, which silently loses precision above 2^53. A wei figure that
// round-trips wrong is not a display bug here -- it is a wrong quote.
type BinJSON struct {
	ID       int32  `json:"id"`
	ReserveX string `json:"x"`
	ReserveY string `json:"y"`
}

// Metadata is the lister's cursor between rounds. Their framework hands back whatever bytes we
// returned last time, so this is how a scan resumes rather than restarting.
type Metadata struct {
	Offset int `json:"offset"`
}

func (b BinJSON) reserves() (*big.Int, *big.Int, bool) {
	x, okx := new(big.Int).SetString(b.ReserveX, 10)
	y, oky := new(big.Int).SetString(b.ReserveY, 10)
	if !okx || !oky || x.Sign() < 0 || y.Sign() < 0 {
		return nil, nil, false
	}
	return x, y, true
}
