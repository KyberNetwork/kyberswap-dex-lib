package btrprop

import (
	"context"
	"math/big"
	"sync"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	utileth "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

// batchRead is one phase-1 eth_call: its packed input, the method whose output it decodes,
// where the decoded value lands, and the flag set to whether the call succeeded.
type batchRead struct {
	abi    abi.ABI
	method string
	target common.Address
	data   []byte
	out    any
	ok     *bool
}

// batchReads collects phase-1 reads so they go out as one JSON-RPC batch.
type batchReads struct {
	calls []batchRead
}

// add queues one call. out receives the decoded result; ok reports whether the call succeeded.
func (b *batchReads) add(a abi.ABI, target string, method string, params []any, out any, ok *bool) {
	data, err := a.Pack(method, params...)
	if err != nil {
		panic(err) // params are fixed types built in this package
	}
	b.calls = append(b.calls, batchRead{
		abi: a, method: method, target: common.HexToAddress(target), data: data, out: out, ok: ok,
	})
}

// maxBatchSize is the most requests one JSON-RPC batch may carry. The Monad node rejects a
// larger batch with a single error object, which the client reports as a decode error.
const maxBatchSize = 100

// run sends every queued call pinned to bn, in batches of at most maxBatchSize sent
// concurrently. A reverted or undecodable call leaves its ok flag false; only a transport error
// fails the run.
func (b *batchReads) run(ctx context.Context, client *ethrpc.Client, bn *big.Int) error {
	var wg sync.WaitGroup
	errc := make(chan error, (len(b.calls)+maxBatchSize-1)/maxBatchSize)
	for start := 0; start < len(b.calls); start += maxBatchSize {
		chunk := b.calls[start:min(start+maxBatchSize, len(b.calls))]
		wg.Add(1)
		go func() {
			defer wg.Done()
			errc <- runChunk(ctx, client, chunk, bn)
		}()
	}
	wg.Wait()
	close(errc)
	for err := range errc {
		if err != nil {
			return err
		}
	}
	return nil
}

// runChunk sends one batch of calls and decodes each result into its own output.
func runChunk(ctx context.Context, client *ethrpc.Client, chunk []batchRead, bn *big.Int) error {
	calls := make([]utileth.BatchCall, len(chunk))
	for i, c := range chunk {
		calls[i] = utileth.BatchCall{To: c.target.Hex(), Data: c.data}
	}
	results, errs, err := utileth.BatchEthCall(ctx, client.GetETHClient().Client(), calls, bn)
	if err != nil {
		return err
	}
	for i, c := range chunk {
		*c.ok = false
		if errs[i] != nil {
			continue
		}
		if err := c.abi.UnpackIntoInterface(c.out, c.method, results[i]); err != nil {
			continue
		}
		*c.ok = true
	}
	return nil
}
