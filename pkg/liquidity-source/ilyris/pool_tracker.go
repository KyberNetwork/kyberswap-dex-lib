package ilyris

import (
	"context"
	"math/big"
	"slices"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	chain chainReader
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, client *ethrpc.Client) *PoolTracker {
	return &PoolTracker{chain: &ethrpcChain{client: client, lens: cfg.LensAddress}}
}

// GetNewPoolState always does a full refresh: logs carry no guard or fee-decay changes, and
// one read of the book cannot drift from the chain the way replaying swaps can.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	st, err := t.chain.PoolState(ctx, p.Address)
	if err != nil {
		return p, err
	}
	if st.Fee.totalFeeRate(st.BinStepBps, st.ActiveID, st.BlockTimestamp) != st.TotalFeeRate {
		return p, ErrFeeMismatch
	}
	if st.Bins, err = t.fetchBins(ctx, p.Address, st.BlockNumber, st.ActiveID); err != nil {
		return p, err
	}

	var guard RawGuardState
	if !valueobject.IsZeroAddress(st.MarketGuard) {
		guardAddr := hexutil.Encode(st.MarketGuard[:])
		if guard, err = t.chain.GuardState(ctx, guardAddr, p.Address, st.ActiveID, st.BlockNumber); err != nil {
			guard.SwapsPaused = true // cannot prove swaps are open: fail closed
		}
	}
	return applyState(p, &st, guard)
}

// fetchBins reads every initialized bin a swap can reach. It walks the bitmap out from the
// active word; after each batch a nextNonEmptyBin probe either closes that side or jumps the
// walk to the next bin with output-side reserves, so a gap costs one probe.
// ponytail: wrong-side bins inside a skipped gap, and anything past maxScanRounds jumps, are
// not tracked; missing them under-quotes, never over-quotes.
func (t *PoolTracker) fetchBins(ctx context.Context, poolAddr string, blockNumber uint64, activeID int32) ([]RawBin, error) {
	aw := activeID >> 8
	lo, hi := aw+1, aw // scanned words are [lo, hi]
	downOpen, upOpen := true, true
	var ids []int32
	for round := 0; round < maxScanRounds && (downOpen || upOpen); round++ {
		var words []int32
		var probes []binProbe
		downProbe, upProbe := -1, -1
		if downOpen {
			newLo := max(lo-scanWords, minWord)
			for w := newLo; w < lo; w++ {
				words = append(words, w)
			}
			if lo = newLo; lo > minWord {
				downProbe = len(probes)
				probes = append(probes, binProbe{From: lo<<8 - 1, XForY: true})
			} else {
				downOpen = false
			}
		}
		if upOpen {
			newHi := min(hi+scanWords, maxWord)
			for w := hi + 1; w <= newHi; w++ {
				words = append(words, w)
			}
			if hi = newHi; hi < maxWord {
				upProbe = len(probes)
				probes = append(probes, binProbe{From: (hi + 1) << 8})
			} else {
				upOpen = false
			}
		}

		bits, next, err := t.chain.ScanBitmap(ctx, poolAddr, blockNumber, words, probes)
		if err != nil {
			return nil, err
		}
		for i, w := range words {
			for b := 0; b < bits[i].BitLen(); b++ {
				if id := w<<8 + int32(b); bits[i].Bit(b) == 1 && id >= minBinID && id <= maxBinID {
					ids = append(ids, id)
				}
			}
		}
		if downProbe >= 0 {
			if r := next[downProbe]; r.Found {
				lo = r.ID>>8 + 1 // next batch ends at the found bin's word
			} else {
				downOpen = false
			}
		}
		if upProbe >= 0 {
			if r := next[upProbe]; r.Found {
				hi = r.ID>>8 - 1 // next batch starts at the found bin's word
			} else {
				upOpen = false
			}
		}
	}
	return t.chain.BinReserves(ctx, poolAddr, blockNumber, ids)
}

func applyState(p entity.Pool, st *RawPoolState, g RawGuardState) (entity.Pool, error) {
	bins := make([]Bin, 0, len(st.Bins))
	var sumX, sumY big.Int
	for _, b := range st.Bins {
		if b.ReserveX.Sign() == 0 && b.ReserveY.Sign() == 0 {
			continue
		}
		bin := Bin{ID: b.ID}
		bin.ReserveX.SetFromBig(b.ReserveX)
		bin.ReserveY.SetFromBig(b.ReserveY)
		bins = append(bins, bin)
		sumX.Add(&sumX, b.ReserveX)
		sumY.Add(&sumY, b.ReserveY)
	}
	slices.SortFunc(bins, func(a, b Bin) int { return int(a.ID - b.ID) })

	extra, err := json.Marshal(Extra{
		ActiveID:         st.ActiveID,
		Bins:             bins,
		Fee:              &st.Fee,
		GuardSwapsPaused: g.SwapsPaused,
		GuardFreezeEnd:   g.FreezeEnd,
		BlockTimestamp:   st.BlockTimestamp,
	})
	if err != nil {
		return p, err
	}
	if p.StaticExtra == "" {
		staticExtra, err := json.Marshal(StaticExtra{
			BinStepBps: st.BinStepBps, DecimalsX: st.DecimalsX, DecimalsY: st.DecimalsY,
		})
		if err != nil {
			return p, err
		}
		p.StaticExtra = string(staticExtra)
	}
	p.Extra = string(extra)
	p.Reserves = entity.PoolReserves{sumX.String(), sumY.String()}
	p.BlockNumber = st.BlockNumber
	p.Timestamp = int64(st.BlockTimestamp)
	return p, nil
}
