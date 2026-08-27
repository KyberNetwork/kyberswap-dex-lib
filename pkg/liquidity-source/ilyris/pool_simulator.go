package ilyris

import (
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Compile-time proof that this satisfies their interface.
//
// This single line is why the module exists. IPoolSimulator has 14 methods; embedding
// pool.Pool supplies 11 and we write the rest. If a signature drifts in a future version of
// theirs, `go build` says so here rather than a reviewer discovering it in a public PR.
var _ pool.IPoolSimulator = (*PoolSimulator)(nil)

var (
	// ErrInsufficientLiquidity lives in math.go (the kernel returns it). A quote that cannot
	// be filled is an ERROR, never a zero amount. A zero would be routed as "this pool offers
	// nothing" -- indistinguishable from a real quote of zero -- and the aggregator would
	// silently rank us last instead of skipping us.
	ErrInvalidToken = errors.New("ilyris: token not in pool")
	// The guard is checked here because the POOL DOES NOT CHECK IT WHEN QUOTING.
	// _guardSwap is called from swapExactIn (BinPool.sol:658) and swapExactOut (:715) but NOT
	// from quoteExactIn (:740) or quoteExactOut (:748). A simulator built from quotes alone is
	// structurally blind to the gate and would route into a reverting swap -- the worst
	// failure mode for an integration, because it reads as our pool being broken rather than
	// closed.
	ErrSwapsPaused           = errors.New("ilyris: swaps paused by market guard")
	ErrCorporateActionFreeze = errors.New("ilyris: corporate-action freeze")
)

// bin is one discrete price level. X sits at and above the active bin, Y at and below.
type bin struct {
	ID       int32
	ReserveX *big.Int
	ReserveY *big.Int
}

// PoolSimulator prices swaps against a local copy of the bin book.
//
// Embeds pool.Pool for the 11 methods that need no Ilyris-specific behaviour. Info.Reserves
// is the SUM of bin reserves, which is what their routing uses for coarse ranking; the
// per-bin detail below is what actually prices a swap.
type PoolSimulator struct {
	pool.Pool

	binStepBps uint32
	activeID   int32
	decimalsX  int
	decimalsY  int
	bins       []bin // ascending by ID, populated bins only

	// Fee rate in FEE_PRECISION units (1e9), already including any volatility component.
	// Read per refresh rather than derived here: the contract owns the fee model and a
	// re-implementation of it is a second source of truth that can disagree.
	totalFeeRate uint64

	// Guard state, pinned to the same block as the book above. See ErrSwapsPaused.
	guardSwapsPaused bool
	guardFreezeEnd   uint64
	blockTimestamp   uint64
}

// GetMetaInfo is one of the three we must write. ApprovalAddress is deliberately left empty:
// we ship an on-chain adapter (ks-dex-adapter-lib) which holds the tokens and approves the
// pool itself, so there is no separate spender for the router to approve.
func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{BlockNumber: p.Info.BlockNumber}
}

// CloneState MUST be overridden. pool.Pool's version returns nil, and a nil clone silently
// breaks split routing and multi-hop: the aggregator cannot restore pre-swap state, so it
// cannot evaluate a second path through this pool.
//
// Clones exactly what UpdateBalance mutates -- reserves, the bin book, and the active id --
// and nothing else. A deeper copy would be waste on a hot path; a shallower one would let a
// speculative route corrupt the state the next route reads.
func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p

	cloned.Info.Reserves = make([]*big.Int, len(p.Info.Reserves))
	for i, r := range p.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}

	cloned.bins = make([]bin, len(p.bins))
	for i, b := range p.bins {
		cloned.bins[i] = bin{
			ID:       b.ID,
			ReserveX: new(big.Int).Set(b.ReserveX),
			ReserveY: new(big.Int).Set(b.ReserveY),
		}
	}
	return &cloned
}

// blocked reports whether the market guard would revert a swap right now.
//
// Conservative on the freeze window because the simulator has no wall clock -- only the block
// timestamp of the last refresh. Quoting a swap that reverts is worse than declining to quote
// one that would have succeeded: the first looks like a broken pool, the second like a thin
// one.
func (p *PoolSimulator) blocked() error {
	if p.guardSwapsPaused {
		return ErrSwapsPaused
	}
	if p.guardFreezeEnd != 0 && p.guardFreezeEnd > p.blockTimestamp {
		return ErrCorporateActionFreeze
	}
	return nil
}

// tokenIndex resolves an address to 0 (X) or 1 (Y), case-insensitively.
//
// Their PoolInfo.GetTokenIndex compares with ==. pool.FromEntity lowercases, but a simulator
// constructed directly does not, and our own manifest carries checksummed addresses -- so an
// exact compare works for one source and silently fails for the other. Fold the case here and
// the ambiguity cannot reach the routing.
func (p *PoolSimulator) tokenIndex(addr string) int {
	for i, t := range p.Info.Tokens {
		if strings.EqualFold(t, addr) {
			return i
		}
	}
	return -1
}
