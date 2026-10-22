package ephemeral

import (
	"context"
	"errors"
	"fmt"
	"math/big"
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

// Head reports the chain's newest block. *ethclient.Client is one.
type Head interface {
	BlockNumber(ctx context.Context) (uint64, error)
}

// Pin returns block, or the chain's head when block is nil. A read of several calls names its
// block before the first, because the block a call reports is not always one the next call can be
// pinned to: Etherlink runs a call pinned to block N in the block after it and reports N+1, whose
// state its nodes do not have yet.
func Pin(ctx context.Context, h Head, block *big.Int) (*big.Int, error) {
	if block != nil {
		return block, nil
	}
	head, err := h.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("ephemeral: head: %w", err)
	}
	return new(big.Int).SetUint64(head), nil
}

// ErrUnpinned means a call pinned to a block reported running at another: the node does not run
// calls at the block they name, as HyperEVM's runs them at its head whatever the pin, and what it
// returned is not the state of that block.
var ErrUnpinned = errors.New("ephemeral: the node ran a call at another block than the one pinned")

// ranAt reports whether a call pinned to block that reported read ran there: the block itself, or
// the one after it on a chain that runs a call in the block after the one it names.
func ranAt(block *big.Int, read uint64) bool {
	return read == block.Uint64() || read == block.Uint64()+1
}

// Scan calls fetch from cursor start until a page is Done, every call at block, and returns block.
//
// Pin block first. Given nil, the first call runs at the latest block and the rest at the block it
// reported, which holds only where a call reports the block it was pinned to. fetch keeps the data
// each page carries.
func Scan[C comparable](ctx context.Context, start C, block *big.Int,
	fetch func(ctx context.Context, cursor C, block *big.Int) (Page[C], error)) (uint64, error) {
	cursor := start
	for {
		page, err := fetch(ctx, cursor, block)
		if err != nil {
			return 0, err
		}
		if block == nil {
			block = new(big.Int).SetUint64(page.Block)
		} else if !ranAt(block, page.Block) {
			return 0, fmt.Errorf("%w: page pinned to block %d ran at %d", ErrUnpinned, block, page.Block)
		}
		if page.Done {
			return block.Uint64(), nil
		}
		if page.Next == cursor {
			return 0, ErrNoProgress
		}
		cursor = page.Next
	}
}
