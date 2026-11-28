package efficientprop

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// backoff is how long a failed read of a hash, or a hash that is not the config's, is kept before the next read.
const backoff = 5 * time.Second

// codeHashes holds the creation-code hashes a front and its door factory answer, each read once from the deployed
// contract, the first time it is wanted, and kept. No hash is taken from config: a listing door's address moves with
// its creation code, and so does a desk contract's, so a reader that predicted one from a configured hash would name
// an address nothing lives at as soon as the code changed. A read that fails, or a front's hash that is not the one
// the config names, is kept as that error for backoff and not read again before: a front that answers wrongly costs
// its readers no call a refresh. The lock covers the maps alone, never a read, so one front's slow or failing read
// holds no other front back.
type codeHashes struct {
	client *ethrpc.Client
	now    func() time.Time
	mu     sync.Mutex
	kept   map[hashKey]common.Hash
	failed map[hashKey]hashFailure
}

type hashKey struct {
	cell   bool
	target common.Address
}

type hashFailure struct {
	err   error
	until time.Time
}

func newCodeHashes(client *ethrpc.Client) *codeHashes {
	return &codeHashes{client: client, now: time.Now, kept: map[hashKey]common.Hash{},
		failed: map[hashKey]hashFailure{}}
}

// readHash is the one word target answers to method().
func (h *codeHashes) readHash(ctx context.Context, a abi.ABI, target, method string) (common.Hash, error) {
	var out common.Hash
	resp, err := h.client.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: a, Target: target, Method: method}, []any{&out}).TryAggregate()
	if err != nil {
		return common.Hash{}, err
	}
	if len(resp.Result) != 1 || !resp.Result[0] || out == (common.Hash{}) {
		return common.Hash{}, fmt.Errorf("efficient-prop: %s.%s() did not answer a hash", target, method)
	}
	return out, nil
}

// get is the hash for key, kept, or read by read (without the lock) and kept; a failure is kept for backoff.
func (h *codeHashes) get(key hashKey, read func() (common.Hash, error)) (common.Hash, error) {
	h.mu.Lock()
	if v, ok := h.kept[key]; ok {
		h.mu.Unlock()
		return v, nil
	}
	if f, ok := h.failed[key]; ok && h.now().Before(f.until) {
		h.mu.Unlock()
		return common.Hash{}, f.err
	}
	h.mu.Unlock()
	v, err := read()
	h.mu.Lock()
	defer h.mu.Unlock()
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return common.Hash{}, err
		}
		h.failed[key] = hashFailure{err: err, until: h.now().Add(backoff)}
		return common.Hash{}, err
	}
	delete(h.failed, key)
	h.kept[key] = v
	return v, nil
}

// door is the creation-code hash of f's listing doors, as its door factory's DOOR_CODEHASH() answers it.
func (h *codeHashes) door(ctx context.Context, f Front) (common.Hash, error) {
	return h.get(hashKey{target: common.HexToAddress(f.DoorFactory)}, func() (common.Hash, error) {
		v, err := h.readHash(ctx, factoryABI, f.DoorFactory, "DOOR_CODEHASH")
		if err != nil {
			return common.Hash{}, fmt.Errorf("%w: %w", ErrDoorCodehash, err)
		}
		return v, nil
	})
}

// cell is the creation-code hash of f's desk contracts, as the front's CELL_CODEHASH() answers it. On a builder chain
// it must be the hash the config names: a front of another desk build is refused. On a chain read at the head the
// config names none, and the front's own answer is taken.
func (h *codeHashes) cell(ctx context.Context, f Front) (common.Hash, error) {
	return h.get(hashKey{cell: true, target: common.HexToAddress(f.Address)}, func() (common.Hash, error) {
		v, err := h.readHash(ctx, frontABI, f.Address, "CELL_CODEHASH")
		if err != nil {
			return common.Hash{}, fmt.Errorf("%w: %w", ErrCellCodehash, err)
		}
		if f.CellCodehash != "" && v != common.HexToHash(f.CellCodehash) {
			return common.Hash{}, fmt.Errorf("%w: front %s answers %s, the config names %s", ErrCellCodehash, f.Address,
				hexutil.Encode(v[:]), f.CellCodehash)
		}
		return v, nil
	})
}

// slotRuleOf is the slot rule of front f's listings: the per-desk rule, under the front's CELL_CODEHASH.
func slotRuleOf(ctx context.Context, hashes *codeHashes, f *Front) (deskRule, error) {
	cell, err := hashes.cell(ctx, *f)
	if err != nil {
		return deskRule{}, err
	}
	return deskRule{cellHash: cell}, nil
}
