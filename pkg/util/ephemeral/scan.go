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

// Scan calls fetch from cursor start until a page is Done, and returns the block every page read.
//
// The first call runs at block (nil for the latest), and every later one at the block the first
// read, so the pages add up to the state of one block. fetch keeps the data each page carries.
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
		} else if page.Block != block.Uint64() {
			return 0, fmt.Errorf("ephemeral: a page read block %d, not %d", page.Block, block)
		}
		if page.Done {
			return page.Block, nil
		}
		if page.Next == cursor {
			return 0, ErrNoProgress
		}
		cursor = page.Next
	}
}
