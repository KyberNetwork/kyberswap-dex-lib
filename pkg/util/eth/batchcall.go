package eth

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// BatchCall is one eth_call to include in a BatchEthCall request. From/Gas/
// GasPrice are optional per-call overrides; their zero values are omitted
// from the request, letting the node apply its own defaults.
type BatchCall struct {
	To       string
	Data     []byte
	From     common.Address
	Gas      uint64
	GasPrice *big.Int
}

// BatchEthCall sends one eth_call per entry in calls as a single JSON-RPC
// batch request (rpc.BatchElem/BatchCallContext) — one HTTP round trip, but
// each call keeps its own From/Gas/GasPrice. This differs from a Multicall3
// aggregate call, which bundles every call into one on-chain call sharing a
// single gas budget across the whole batch.
//
// Per-call reverts/errors are reported positionally via the returned
// []error and do not fail the batch; only network/transport-level failures
// return a non-nil error.
func BatchEthCall(
	ctx context.Context, client *rpc.Client, calls []BatchCall, blockNumber *big.Int,
) (results [][]byte, callErrs []error, err error) {
	if len(calls) == 0 {
		return nil, nil, nil
	}

	blockArg := "latest"
	if blockNumber != nil {
		blockArg = hexutil.EncodeBig(blockNumber)
	}

	raw := make([]hexutil.Bytes, len(calls))
	batch := make([]rpc.BatchElem, len(calls))
	for i, c := range calls {
		arg := map[string]any{"to": c.To}
		if len(c.Data) > 0 {
			arg["data"] = hexutil.Bytes(c.Data)
		}
		if c.From != (common.Address{}) {
			arg["from"] = c.From
		}
		if c.Gas != 0 {
			arg["gas"] = hexutil.Uint64(c.Gas)
		}
		if c.GasPrice != nil {
			arg["gasPrice"] = (*hexutil.Big)(c.GasPrice)
		}
		batch[i] = rpc.BatchElem{
			Method: "eth_call",
			Args:   []any{arg, blockArg},
			Result: &raw[i],
		}
	}

	if err = BatchCallWithRetry(ctx, client, batch, DefaultBatchRetry); err != nil {
		return nil, nil, err
	}

	results = make([][]byte, len(calls))
	callErrs = make([]error, len(calls))
	for i, elem := range batch {
		results[i] = raw[i]
		callErrs[i] = elem.Error
	}
	return results, callErrs, nil
}

// StorageRead is one eth_getStorageAt request/result pair.
type StorageRead struct {
	Slot   common.Hash
	Result common.Hash
}

// BatchGetStorageAt reads reads' slots from address in one JSON-RPC batch,
// filling each Result in place. For contracts with no view function for the
// state -- typically unverified ones; prefer an ABI call when one exists.
//
// Fails on the first element error, unlike BatchEthCall: a storage read cannot
// revert, so an element error means transport or node trouble.
func BatchGetStorageAt(
	ctx context.Context, client *rpc.Client, address common.Address,
	reads []StorageRead, retry BatchRetry,
) error {
	if len(reads) == 0 {
		return nil
	}

	batch := make([]rpc.BatchElem, len(reads))
	for i := range reads {
		batch[i] = rpc.BatchElem{
			Method: "eth_getStorageAt",
			Args:   []any{address, reads[i].Slot, "latest"},
			Result: &reads[i].Result,
		}
	}

	if err := BatchCallWithRetry(ctx, client, batch, retry); err != nil {
		return err
	}

	for _, elem := range batch {
		if elem.Error != nil {
			return elem.Error
		}
	}
	return nil
}

// BatchRetry configures batch retry. Delay doubles after each failed attempt.
type BatchRetry struct {
	MaxAttempts  int
	InitialDelay time.Duration
}

// DefaultBatchRetry: for pool trackers, giving up inside a 2s poll cycle
// rather than stacking behind the next one.
var DefaultBatchRetry = BatchRetry{MaxAttempts: 3, InitialDelay: 50 * time.Millisecond}

// PatientBatchRetry: for one-shot callers like listers, where finishing
// matters more than finishing fast.
var PatientBatchRetry = BatchRetry{MaxAttempts: 5, InitialDelay: 500 * time.Millisecond}

// BatchCallWithRetry retries only request-level 429/rate-limit errors. Element
// errors are left in BatchElem.Error: a revert repeats identically anyway.
func BatchCallWithRetry(
	ctx context.Context, client *rpc.Client, batch []rpc.BatchElem, retry BatchRetry,
) error {
	attempts := max(retry.MaxAttempts, 1)

	delay := retry.InitialDelay
	if delay <= 0 {
		delay = DefaultBatchRetry.InitialDelay
	}

	var err error
	for range attempts {
		if err = client.BatchCallContext(ctx, batch); err == nil {
			return nil
		}

		msg := err.Error()
		if !strings.Contains(msg, "429") && !strings.Contains(msg, "rate limit") {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
	}
	return err
}
