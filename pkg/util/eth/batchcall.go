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

	if err = batchCallWithRetry(ctx, client, batch); err != nil {
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

// batchCallWithRetry retries request-level 429/rate-limit errors with
// exponential backoff (50ms -> 200ms). Per-element errors surface to the
// caller via BatchElem.Error instead.
func batchCallWithRetry(ctx context.Context, client *rpc.Client, batch []rpc.BatchElem) error {
	const maxAttempts = 3
	delay := 50 * time.Millisecond
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err = client.BatchCallContext(ctx, batch)
		if err == nil {
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
