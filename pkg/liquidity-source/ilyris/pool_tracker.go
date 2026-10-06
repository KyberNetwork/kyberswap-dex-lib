package ilyris

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	_ pool.IPoolTracker           = (*PoolTracker)(nil)
	_ pool.ITicksBasedPoolTracker = (*PoolTracker)(nil)
)

// Bitmap walk bounds. Each round reads scanWords bitmap words (256 bins each) per open
// direction plus one nextNonEmptyBin probe past them, in a single multicall.
const (
	scanWords = 8
	// ponytail: a book with more than maxScanRounds separate gaps wider than 2048 bins is
	// only partly tracked; swaps past that run out of bins (no misprice). Raise if one appears.
	maxScanRounds = 16
	minWord       = MinBinID >> 8
	maxWord       = MaxBinID >> 8
)

// PoolTracker keeps a pool's book in step with the chain.
type PoolTracker struct {
	chain chainReader
}

func NewPoolTracker(chain chainReader) *PoolTracker {
	return &PoolTracker{chain: chain}
}

// GetNewPoolState does a full refresh every time. Their service hands a tracker RECENT logs,
// never history, so folding logs would leave a pre-existing pool's book empty; and a swap
// moves the active bin, every crossed bin and the volatility state, which one read captures.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, params pool.GetNewPoolStateParams) (entity.Pool, error) {
	return t.BootstrapPoolState(ctx, p, params)
}

// BootstrapPoolState performs the full refresh: pool state, bins and guard at one block.
func (t *PoolTracker) BootstrapPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	st, guard, err := t.FetchRPCData(ctx, p)
	if err != nil {
		// Return the pool UNCHANGED. Returning a zeroed one would replace a good book with an
		// empty one and quietly delist us on a transient RPC error.
		return p, err
	}
	return applyState(p, st, guard), nil
}

// FetchRPCData reads bins and the market guard at one block. The guard has no
// on-chain block tag of its own, so the pin is the call's block number.
func (t *PoolTracker) FetchRPCData(ctx context.Context, p entity.Pool) (RawPoolState, RawGuardState, error) {
	st, err := t.chain.PoolState(ctx, p.Address)
	if err != nil {
		return RawPoolState{}, RawGuardState{}, err
	}
	// The simulator derives the fee from st.Fee so it can move after a swap. Refuse a snapshot
	// where that derivation disagrees with the contract at the same block.
	if got := st.Fee.totalFeeRate(st.BinStepBps, st.ActiveID, st.BlockTimestamp); got != st.TotalFeeRate {
		return RawPoolState{}, RawGuardState{}, fmt.Errorf("ilyris: derived fee rate %d != getTotalFeeRate %d",
			got, st.TotalFeeRate)
	}
	if st.Bins, err = t.fetchBins(ctx, p.Address, st.BlockNumber, st.ActiveID); err != nil {
		return RawPoolState{}, RawGuardState{}, err
	}

	guard := RawGuardState{BlockNumber: st.BlockNumber}
	if st.MarketGuard != "" && !isZeroAddress(st.MarketGuard) {
		// Read every refresh, never cached: setMarketGuard can repoint it at any time
		// (BinPool.sol:322), so a guard address captured at pool creation can be stale.
		g, gerr := t.chain.GuardState(ctx, st.MarketGuard, st.BlockNumber)
		if gerr != nil {
			// Guard unreadable means we cannot prove swaps are open. Fail CLOSED by leaving
			// swapsPaused set: refusing to quote costs a route, quoting into a reverting swap
			// costs the integration's credibility.
			guard.SwapsPaused = true
			guard.BlockNumber = st.BlockNumber
		} else {
			guard = g
			if guard.BlockNumber == 0 {
				guard.BlockNumber = st.BlockNumber
			}
		}
	}
	if st.BlockNumber != 0 && guard.BlockNumber != 0 && guard.BlockNumber != st.BlockNumber {
		// Mixed-block snapshot: refuse to quote rather than pair reserves with a
		// later/earlier gate.
		guard.SwapsPaused = true
		guard.BlockNumber = st.BlockNumber
	}
	return st, guard, nil
}

// FetchPoolTicks re-reads the book. Same call as the bootstrap for us: our "ticks" are bins and
// they come from the same full read, so there is no cheaper partial path to offer.
func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	return t.BootstrapPoolState(ctx, p, pool.GetNewPoolStateParams{})
}

// fetchBins reads every initialized bin a swap can reach, at blockNumber. It walks the
// bitmap outward from the active word; after each batch a nextNonEmptyBin probe either
// closes that direction (no output-side liquidity beyond) or jumps the walk to the next
// populated bin, so a gap costs one probe instead of a word read per 256 bins.
//
// Skipped gaps hold no output-side reserves. X-only bins below active (or Y-only above) in a
// gap are left out; a swap reaches them only after crossing the gap, and missing them
// under-quotes rather than over-quotes.
func (t *PoolTracker) fetchBins(ctx context.Context, poolAddr string, blockNumber uint64, activeID int32) ([]RawBin, error) {
	aw := activeID >> 8
	lo, hi := aw+1, aw // scanned words are [lo, hi]; next batches end at lo-1 / start at hi+1
	downOpen, upOpen := true, true
	var ids []int32
	for round := 0; round < maxScanRounds && (downOpen || upOpen); round++ {
		var words []int32
		var probes []BinProbe
		var downProbe, upProbe = -1, -1
		if downOpen {
			newLo := max(lo-scanWords, minWord)
			for w := newLo; w < lo; w++ {
				words = append(words, w)
			}
			lo = newLo
			if downOpen = lo > minWord; downOpen {
				downProbe = len(probes)
				probes = append(probes, BinProbe{From: lo<<8 - 1, XForY: true})
			}
		}
		if upOpen {
			newHi := min(hi+scanWords, maxWord)
			for w := hi + 1; w <= newHi; w++ {
				words = append(words, w)
			}
			hi = newHi
			if upOpen = hi < maxWord; upOpen {
				upProbe = len(probes)
				probes = append(probes, BinProbe{From: (hi + 1) << 8, XForY: false})
			}
		}

		bits, next, err := t.chain.ScanBitmap(ctx, poolAddr, blockNumber, words, probes)
		if err != nil {
			return nil, err
		}
		if len(bits) != len(words) || len(next) != len(probes) {
			return nil, fmt.Errorf("ilyris: bitmap scan returned %d/%d results", len(bits)+len(next), len(words)+len(probes))
		}
		for i, w := range words {
			for b := 0; bits[i] != nil && b < bits[i].BitLen(); b++ {
				if id := w<<8 + int32(b); bits[i].Bit(b) == 1 && id >= MinBinID && id <= MaxBinID {
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
	if len(ids) == 0 {
		return nil, nil
	}
	return t.chain.BinReserves(ctx, poolAddr, blockNumber, ids)
}

// applyState writes a chain read back into their entity.
func applyState(p entity.Pool, st RawPoolState, g RawGuardState) entity.Pool {
	bins := make([]BinJSON, 0, len(st.Bins))
	sumX, sumY := new(big.Int), new(big.Int)
	for _, b := range st.Bins {
		if b.ReserveX == nil || b.ReserveY == nil {
			continue
		}
		if b.ReserveX.Sign() == 0 && b.ReserveY.Sign() == 0 {
			continue
		}
		// Decimal strings, not JSON numbers: these are uint128 and float64 loses precision
		// above 2^53, which would corrupt a quote rather than raise an error.
		bins = append(bins, BinJSON{ID: b.ID, ReserveX: b.ReserveX.String(), ReserveY: b.ReserveY.String()})
		sumX.Add(sumX, b.ReserveX)
		sumY.Add(sumY, b.ReserveY)
	}
	sort.Slice(bins, func(i, j int) bool { return bins[i].ID < bins[j].ID })

	ex, _ := json.Marshal(Extra{
		ActiveID:         st.ActiveID,
		Bins:             bins,
		TotalFeeRate:     st.TotalFeeRate,
		Fee:              &st.Fee,
		MarketGuard:      strings.ToLower(st.MarketGuard),
		GuardSwapsPaused: g.SwapsPaused,
		GuardFreezeEnd:   g.FreezeEnd,
		BlockTimestamp:   st.BlockTimestamp,
		BlockNumber:      st.BlockNumber,
	})
	p.Extra = string(ex)

	// StaticExtra is written only if absent. It is immutable by definition, and rewriting it
	// every refresh would let a bad read silently redefine the pool's decimals.
	if p.StaticExtra == "" {
		se, _ := json.Marshal(StaticExtra{
			BinStepBps: st.BinStepBps, DecimalsX: st.DecimalsX, DecimalsY: st.DecimalsY,
		})
		p.StaticExtra = string(se)
	}

	p.Reserves = entity.PoolReserves{sumX.String(), sumY.String()}
	p.BlockNumber = st.BlockNumber
	p.Timestamp = int64(st.BlockTimestamp)
	if len(p.Tokens) == 2 {
		if p.Tokens[0].Address == "" {
			p.Tokens[0].Address = strings.ToLower(st.TokenX)
		}
		if p.Tokens[1].Address == "" {
			p.Tokens[1].Address = strings.ToLower(st.TokenY)
		}
		p.Tokens[0].Decimals, p.Tokens[1].Decimals = st.DecimalsX, st.DecimalsY
		p.Tokens[0].Swappable, p.Tokens[1].Swappable = true, true
	}
	return p
}

func isZeroAddress(a string) bool {
	s := strings.TrimPrefix(strings.ToLower(a), "0x")
	return s == "" || strings.Trim(s, "0") == ""
}
