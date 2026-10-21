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
// The first call runs at block (nil for the latest), and every call after the first that succeeds
// at the block that one read, so the parts add up to the state of one block. The results are in
// item order. A single item that is still too large returns its error.
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
		} else if got != block.Uint64() {
			return fmt.Errorf("ephemeral: a part read block %d, not %d", got, block)
		}
		out = append(out, results...)
		return nil
	}

	if err := read(items); err != nil {
		return 0, nil, err
	}
	return block.Uint64(), out, nil
}
