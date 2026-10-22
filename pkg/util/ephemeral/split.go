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
// Every call runs at block, and the parts add up to its state; pin it first (see Pin). Given nil,
// the first call runs at the latest block and every later one at the block the first that succeeds
// reported. The results are in item order. A single item that is still too large returns its error.
func Split[T, R any](ctx context.Context, items []T, block *big.Int, tooLarge func(error) bool,
	fetch func(ctx context.Context, part []T, block *big.Int) (uint64, []R, error)) (uint64, []R, error) {
	if len(items) == 0 {
		return 0, nil, nil
	}

	out := make([]R, 0, len(items))
	var read func(part []T) error
	read = func(part []T) error {
		got, results, err := fetch(ctx, part, block)
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
		if block == nil {
			block = new(big.Int).SetUint64(got)
		} else if !ranAt(block, got) {
			return fmt.Errorf("%w: part pinned to block %d ran at %d", ErrUnpinned, block, got)
		}
		out = append(out, results...)
		return nil
	}

	if err := read(items); err != nil {
		return 0, nil, err
	}
	return block.Uint64(), out, nil
}
