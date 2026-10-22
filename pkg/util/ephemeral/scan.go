package ephemeral

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Page is what one call of a scan reports besides its data: the block it read, and where the
// next call starts unless Done.
type Page[C comparable] struct {
	Block uint64
	Next  C
	Done  bool
}

// ErrNoProgress means a call ended where it started: one step of the scan does not fit the gas
// given, and calling again would loop.
var ErrNoProgress = errors.New("ephemeral: a call made no progress; raise the gas")

// Head reports the chain's newest block and its id. *ethclient.Client is one.
type Head interface {
	BlockNumber(ctx context.Context) (uint64, error)
	ChainID(ctx context.Context) (*big.Int, error)
}

// Pinned is the block a read of several calls is pinned to, nil for the latest, and how the
// chain's node reports a call pinned there. Pin makes one.
type Pinned struct {
	Block *big.Int
	// late: the node runs a call pinned to block N in the block after it and reports N+1, with
	// N's state. Elsewhere a report of N+1 is a node that ignored the pin.
	late bool
}

// lateChains are the chains whose nodes run a call in the block after the one it is pinned to:
// Etherlink's.
var lateChains = map[uint64]bool{uint64(valueobject.ChainIDEtherlink): true}

// lateHeads caches, per Head, whether its chain is one of lateChains: the chain of a client never
// changes, and asking costs a round trip.
var lateHeads sync.Map

// Pin returns block, or the chain's head when block is nil. A read of several calls names its
// block before the first, because the block a call reports is not always one the next call can be
// pinned to: Etherlink runs a call pinned to block N in the block after it and reports N+1, whose
// state its nodes do not have yet. Only there does the read take a report of N+1 as N.
func Pin(ctx context.Context, h Head, block *big.Int) (Pinned, error) {
	isLate, err := runsLate(ctx, h)
	if err != nil {
		return Pinned{}, err
	}
	if block != nil {
		return Pinned{Block: block, late: isLate}, nil
	}
	head, err := h.BlockNumber(ctx)
	if err != nil {
		return Pinned{}, fmt.Errorf("ephemeral: head: %w", err)
	}
	return Pinned{Block: new(big.Int).SetUint64(head), late: isLate}, nil
}

func runsLate(ctx context.Context, h Head) (bool, error) {
	// A Head that cannot be a map key is asked every time.
	cacheable := reflect.TypeOf(h).Comparable()
	if cacheable {
		if v, ok := lateHeads.Load(h); ok {
			return v.(bool), nil
		}
	}
	id, err := h.ChainID(ctx)
	if err != nil {
		return false, fmt.Errorf("ephemeral: chain id: %w", err)
	}
	isLate := id.IsUint64() && lateChains[id.Uint64()]
	if cacheable {
		lateHeads.Store(h, isLate)
	}
	return isLate, nil
}

// ErrUnpinned means a call pinned to a block reported running at another: the node does not run
// calls at the block they name, as HyperEVM's runs them at its head whatever the pin, and what it
// returned is not the state of that block.
var ErrUnpinned = errors.New("ephemeral: the node ran a call at another block than the one pinned")

// ranAt reports whether a call pinned to pin.Block that reported read ran there: the block itself,
// or the one after it on a chain that runs a call in the block after the one it names. Accepted
// elsewhere, a head that moved on between Pin and the call would pass for the pinned block.
func (pin Pinned) ranAt(read uint64) bool {
	return read == pin.Block.Uint64() || pin.late && read == pin.Block.Uint64()+1
}

// Scan calls fetch from cursor start until a page is Done, every call at pin's block, and returns
// that block.
//
// Pin the block first. Given none, the first call runs at the latest block and the rest at the
// block it reported, which holds only where a call reports the block it was pinned to. fetch keeps
// the data each page carries.
func Scan[C comparable](ctx context.Context, start C, pin Pinned,
	fetch func(ctx context.Context, cursor C, block *big.Int) (Page[C], error)) (uint64, error) {
	cursor := start
	for {
		page, err := fetch(ctx, cursor, pin.Block)
		if err != nil {
			return 0, err
		}
		if pin.Block == nil {
			pin.Block = new(big.Int).SetUint64(page.Block)
		} else if !pin.ranAt(page.Block) {
			return 0, fmt.Errorf("%w: page pinned to block %d ran at %d", ErrUnpinned, pin.Block, page.Block)
		}
		if page.Done {
			return pin.Block.Uint64(), nil
		}
		if page.Next == cursor {
			return 0, ErrNoProgress
		}
		cursor = page.Next
	}
}
