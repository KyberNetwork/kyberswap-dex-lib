package ilyris

import (
	"context"
	"encoding/json"
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

// defaultRadius is how many bins either side of active to load.
//
// 30 matches what the web app requests and comfortably covers the depth any realistic route
// crosses. It is not "all bins": a book can be thousands wide, and quoting needs the ones a
// swap can actually reach.
const defaultRadius uint32 = 30

// PoolTracker keeps a pool's book in step with the chain.
type PoolTracker struct {
	chain  chainReader
	radius uint32
}

func NewPoolTracker(chain chainReader) *PoolTracker {
	return &PoolTracker{chain: chain, radius: defaultRadius}
}

// GetNewPoolState refreshes a pool.
//
// THE COLD START IS THE WHOLE POINT OF THIS METHOD'S SHAPE. Their service hands a tracker
// RECENT LOGS, never history. A log-only implementation therefore starts with an empty book
// and stays empty forever for any pool that was created before the service started watching --
// listed, ranked, and never routed, with nothing anywhere reporting an error.
//
// So: if the stored pool has no usable book, do a full RPC refresh instead of folding logs
// into nothing.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, params pool.GetNewPoolStateParams) (entity.Pool, error) {
	if needsBootstrap(p) {
		return t.BootstrapPoolState(ctx, p, params)
	}
	// A swap moves the active bin and both sides of every bin it crossed. Reconstructing that
	// from logs alone means re-deriving the contract's traversal off-chain and hoping the two
	// agree; a full read is one call and cannot drift. Revisit only if their service starts
	// calling this per block and the RPC cost shows up.
	return t.BootstrapPoolState(ctx, p, params)
}

// BootstrapPoolState performs the full refresh: lens read, then guard read at the same block.
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
	st, err := t.chain.PoolState(ctx, p.Address, t.radius)
	if err != nil {
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
// they come from the same lens read, so there is no cheaper partial path to offer.
func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	return t.BootstrapPoolState(ctx, p, pool.GetNewPoolStateParams{})
}

// needsBootstrap reports whether the stored pool can be priced from at all.
func needsBootstrap(p entity.Pool) bool {
	if p.Extra == "" || p.StaticExtra == "" {
		return true
	}
	var ex Extra
	if err := json.Unmarshal([]byte(p.Extra), &ex); err != nil {
		return true
	}
	if len(ex.Bins) == 0 {
		return true
	}
	// Bins present but all empty is the same situation as no bins: nothing to quote from.
	for _, b := range ex.Bins {
		if b.ReserveX != "0" || b.ReserveY != "0" {
			return false
		}
	}
	return true
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
