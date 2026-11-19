package ephemeral

import (
	"context"
	"fmt"
	"math/big"
)

// Split reads items in as few calls as the node takes, for a contract that reads many items in
// one call. fetch reads a part in one call and returns the block it read and one result per item;
// a part whose error tooLarge accepts is halved, and both halves are read again.
//
// Every call runs at pin's block, and the parts add up to its state. A read pinned to none runs its
// first call at the latest block and every later one at the block the first that succeeds ran at
// (see Pin); a part that made no call of its own, and so reports block 0, pins nothing. The results
// are in item order. A single item that is still too large returns its error.
func Split[T, R any](ctx context.Context, items []T, pin Pinned, tooLarge func(error) bool,
	fetch func(ctx context.Context, part []T, block *big.Int) (uint64, []R, error)) (uint64, []R, error) {
	if len(items) == 0 {
		return 0, nil, nil
	}

	out := make([]R, 0, len(items))
	var read func(part []T) error
	read = func(part []T) error {
		got, results, err := fetch(ctx, part, pin.Block)
		if err != nil {
			if len(part) < 2 || !tooLarge(err) {
				return err
			}
			mid := len(part) / 2
			if err := read(part[:mid]); err != nil {
				return err
			}
			return read(part[mid:])
		}
		if len(results) != len(part) {
			return fmt.Errorf("ephemeral: a call returned %d results for %d items", len(results), len(part))
		}
		if pin.Block == nil {
			if got != 0 {
				pin.take(got)
			}
		} else if !pin.ranAt(got) {
			return fmt.Errorf("%w: part pinned to block %d ran at %d", ErrUnpinned, pin.Block, got)
		}
		out = append(out, results...)
		return nil
	}

	if err := read(items); err != nil {
		return 0, nil, err
	}
	if pin.Block == nil {
		return 0, out, nil
	}
	return pin.Block.Uint64(), out, nil
}
