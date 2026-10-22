package ephemeral

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"reflect"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"

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
// changes, and asking costs a round trip. The reads that start together, as a backfill's pools do,
// share one question rather than each asking before any answer is kept.
var lateHeads sync.Map

type lateHead struct {
	once sync.Once
	late bool
	err  error
}

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
	if !reflect.TypeOf(h).Comparable() {
		return askLate(ctx, h)
	}
	v, _ := lateHeads.LoadOrStore(h, &lateHead{})
	l := v.(*lateHead)
	l.once.Do(func() { l.late, l.err = askLate(ctx, h) })
	if l.err != nil {
		// A failed question is not kept: the next read asks again.
		lateHeads.CompareAndDelete(h, l)
	}
	return l.late, l.err
}

func askLate(ctx context.Context, h Head) (bool, error) {
	id, err := h.ChainID(ctx)
	if err != nil {
		return false, fmt.Errorf("ephemeral: chain id: %w", err)
	}
	return id.IsUint64() && lateChains[id.Uint64()], nil
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

// WordsParallel is how many calls ScanWords has in flight for one read. It bounds the calls one
// pool's read makes at once, which a lens guard counts as one read.
const WordsParallel = 4

// ScanWords reads the words [lower, upper] of a bitmap lens, every call at pin's block, and returns
// that block and the items of every call in word order.
//
// A call reads from its lower word as far as its gas allows. Each time one stops short, the words
// it left are cut in two, and the upper half goes to another call when fewer than parallel are in
// flight: the ticks of a pool sit in few words, which a plain cut of the range would leave in one
// window read call after call, while halving whatever is left splits wherever the ticks are. fetch
// reads words [lower, upper] in one call; it is called concurrently.
func ScanWords[T any](ctx context.Context, lower, upper int16, pin Pinned, parallel int,
	fetch func(ctx context.Context, lower, upper int16, block *big.Int) (Page[int16], []T, error)) (uint64,
	[]T, error) {
	first, items, err := fetch(ctx, lower, upper, pin.Block)
	if err != nil {
		return 0, nil, err
	}
	if pin.Block == nil {
		pin.Block = new(big.Int).SetUint64(first.Block)
	} else if !pin.ranAt(first.Block) {
		return 0, nil, fmt.Errorf("%w: page pinned to block %d ran at %d", ErrUnpinned, pin.Block, first.Block)
	}
	if first.Done {
		return pin.Block.Uint64(), items, nil
	}
	if first.Next <= lower {
		return 0, nil, ErrNoProgress
	}

	// Each call's items, by the word it started at: the calls start at distinct words, and those
	// words order them.
	var mu sync.Mutex
	parts := map[int16][]T{lower: items}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(max(parallel, 1))
	var read func(lower, upper int16) error
	read = func(lower, upper int16) error {
		for {
			page, got, err := fetch(gctx, lower, upper, pin.Block)
			if err != nil {
				return err
			}
			if !pin.ranAt(page.Block) {
				return fmt.Errorf("%w: page pinned to block %d ran at %d", ErrUnpinned, pin.Block, page.Block)
			}
			mu.Lock()
			parts[lower] = got
			mu.Unlock()
			if page.Done {
				return nil
			}
			if page.Next <= lower {
				return ErrNoProgress
			}
			lower = page.Next
			// Never waits for a free call: a goroutine that did would hold one, and with every one
			// held the read would stop. Without one, this goroutine reads both halves in turn.
			if mid := int16((int(lower) + int(upper)) / 2); mid < upper {
				hi := upper
				if g.TryGo(func() error { return read(mid+1, hi) }) {
					upper = mid
				}
			}
		}
	}
	g.Go(func() error { return read(first.Next, upper) })
	if err := g.Wait(); err != nil {
		return 0, nil, err
	}

	starts := slices.Sorted(maps.Keys(parts))
	items = nil
	for _, start := range starts {
		items = append(items, parts[start]...)
	}
	return pin.Block.Uint64(), items, nil
}
