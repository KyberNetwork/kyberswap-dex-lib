package ephemeral

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

//go:embed batch.json
var batchJSON []byte

// lensBatch is LensBatch, released by KyberNetwork/ephemeral-contracts: many lens reads in one call.
var lensBatch = MustLoad("LensBatch", batchJSON)

// BatchConfig sets how a Batcher gathers lens reads into one call.
type BatchConfig struct {
	// MaxItems flushes a batch once it holds this many reads; 0 is 64.
	MaxItems int `json:"maxItems,omitempty"`
	// WindowMs is how long the first read of a batch waits for others; 0 is 20ms.
	WindowMs int `json:"windowMs,omitempty"`
	// Gas is the gas of one batch call; 0 is the node's cap up to CallGasCeiling, as for Call.
	Gas uint64 `json:"gas,omitempty"`
	// MinItemGas is the least gas a read is started with; 0 is 6M. A lens given less than its largest
	// step, a full bitmap word, ~4.7M on the forks whose tick getter loads the most, makes no
	// progress at all.
	MinItemGas uint64 `json:"minItemGas,omitempty"`
}

const (
	defaultBatchItems      = 64
	defaultBatchWindow     = 20 * time.Millisecond
	defaultBatchMinItemGas = 6_000_000
	// maxBatchInitCode keeps a batch's call data under the 49152 bytes a node takes as init code
	// (EIP-3860), with room for LensBatch itself.
	maxBatchInitCode = 45_000
	// batchTimeout bounds a batch call, whose readers each wait on their own context.
	batchTimeout = 30 * time.Second
)

// Batcher gathers lens reads made at the same block into one LensBatch call. A read that comes
// back is what Call would have returned; one the batch did not run, or a batch that failed, is
// made alone with Call, so a Batcher only ever saves calls. A nil Batcher makes every read alone.
type Batcher struct {
	caller Caller
	cfg    BatchConfig

	mu      sync.Mutex
	pending map[uint64]*batch

	batches, batched, alone atomic.Int64
}

type batch struct {
	block    *big.Int
	reads    []*batchRead
	codes    map[*Contract]int
	initCode int
	timer    *time.Timer
}

type batchRead struct {
	contract *Contract
	args     []byte
	gas      uint64
	done     chan batchResult
}

type batchResult struct {
	ran     bool
	payload []byte
	err     error
}

// NewBatcher gathers reads made through c.
func NewBatcher(c Caller, cfg BatchConfig) *Batcher {
	if cfg.MaxItems <= 0 {
		cfg.MaxItems = defaultBatchItems
	}
	if cfg.MinItemGas == 0 {
		cfg.MinItemGas = defaultBatchMinItemGas
	}
	return &Batcher{caller: c, cfg: cfg, pending: map[uint64]*batch{}}
}

var sharedBatchers sync.Map

// SharedBatcher is the Batcher of reads made through c, one per Caller, so that the trackers of
// every exchange reading through one node fill the same batches. cfg is that of the first to ask;
// nil asks for none.
func SharedBatcher(c Caller, cfg *BatchConfig) *Batcher {
	if cfg == nil || c == nil {
		return nil
	}
	if b, ok := sharedBatchers.Load(c); ok {
		return b.(*Batcher)
	}
	b, _ := sharedBatchers.LoadOrStore(c, NewBatcher(c, *cfg))
	return b.(*Batcher)
}

// BatchStats counts what a Batcher did: batch calls made, reads they answered, and reads made
// alone because a batch did not run them.
type BatchStats struct {
	Batches, Batched, Alone int64
}

func (b *Batcher) Stats() BatchStats {
	if b == nil {
		return BatchStats{}
	}
	return BatchStats{Batches: b.batches.Load(), Batched: b.batched.Load(), Alone: b.alone.Load()}
}

// Call is Call, through the batch of block when there is a Batcher and block is pinned: a read at
// the latest block names no block for the batch to share.
func (b *Batcher) Call(ctx context.Context, c Caller, contract *Contract, out any, gas uint64, block *big.Int,
	args ...any) error {
	if b == nil || block == nil {
		return Call(ctx, c, contract, out, gas, block, args...)
	}
	packed, err := contract.packArgs(args...)
	if err != nil {
		return err
	}

	read := &batchRead{contract: contract, args: packed, gas: gas, done: make(chan batchResult, 1)}
	b.add(block, read)

	var res batchResult
	select {
	case res = <-read.done:
	case <-ctx.Done():
		return ctx.Err()
	}
	// A read the batch did not run, or ran out of gas in it with less than it would have alone, is
	// made alone.
	if res.err != nil || !res.ran || len(res.payload) == 0 {
		b.alone.Add(1)
		return Call(ctx, c, contract, out, gas, block, args...)
	}
	b.batched.Add(1)
	return decodeResult(contract, res.payload, out)
}

// add queues read in the batch of block, and flushes the batch when it is full or would grow past
// what a node takes as init code.
func (b *Batcher) add(block *big.Int, read *batchRead) {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := block.Uint64()
	cur := b.pending[key]
	if cur != nil && cur.initCode+cur.cost(read) > maxBatchInitCode {
		b.detach(key, cur)
		go b.flush(cur)
		cur = nil
	}
	if cur == nil {
		cur = &batch{block: new(big.Int).Set(block), codes: map[*Contract]int{}, initCode: len(lensBatch.bytecode)}
		b.pending[key] = cur
		window := defaultBatchWindow
		if b.cfg.WindowMs > 0 {
			window = time.Duration(b.cfg.WindowMs) * time.Millisecond
		}
		cur.timer = time.AfterFunc(window, func() {
			b.mu.Lock()
			flush := b.pending[key] == cur
			if flush {
				delete(b.pending, key)
			}
			b.mu.Unlock()
			if flush {
				b.flush(cur)
			}
		})
	}

	cur.initCode += cur.cost(read)
	if _, ok := cur.codes[read.contract]; !ok {
		cur.codes[read.contract] = len(cur.codes)
	}
	cur.reads = append(cur.reads, read)
	if len(cur.reads) >= b.cfg.MaxItems {
		b.detach(key, cur)
		go b.flush(cur)
	}
}

func (b *Batcher) detach(key uint64, cur *batch) {
	cur.timer.Stop()
	delete(b.pending, key)
}

// cost is what read adds to the batch's call data: its arguments and the ABI words around them,
// and its lens's code the first time the batch sees it.
func (cur *batch) cost(read *batchRead) int {
	n := len(read.args) + 6*32
	if _, ok := cur.codes[read.contract]; !ok {
		n += len(read.contract.bytecode) + 2*32
	}
	return n
}

// lensBatchItem is LensBatch.Item.
type lensBatchItem struct {
	Code *big.Int
	Args []byte
	Gas  *big.Int
}

// lensBatchResults is what LensBatch returns.
type lensBatchResults struct {
	BlockNumber *big.Int
	Count       *big.Int
	Results     []struct {
		Ran  bool
		Data []byte
	}
}

// flush runs the batch and hands each read its result; a batch that failed hands every read the
// error, and each is made alone.
func (b *Batcher) flush(cur *batch) {
	codes := make([][]byte, len(cur.codes))
	for contract, i := range cur.codes {
		codes[i] = contract.bytecode
	}
	items := make([]lensBatchItem, len(cur.reads))
	for i, read := range cur.reads {
		items[i] = lensBatchItem{
			Code: big.NewInt(int64(cur.codes[read.contract])),
			Args: read.args,
			Gas:  new(big.Int).SetUint64(read.gas),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), batchTimeout)
	defer cancel()
	b.batches.Add(1)
	var out lensBatchResults
	err := Call(ctx, b.caller, lensBatch, &out, b.cfg.Gas, cur.block, codes, items,
		new(big.Int).SetUint64(b.cfg.MinItemGas))
	if err == nil && len(out.Results) != len(cur.reads) {
		err = fmt.Errorf("ephemeral: a batch of %d reads returned %d results", len(cur.reads), len(out.Results))
	}
	for i, read := range cur.reads {
		if err != nil {
			read.done <- batchResult{err: err}
			continue
		}
		read.done <- batchResult{ran: out.Results[i].Ran, payload: out.Results[i].Data}
	}
}

// CallPage is Call for the page of a scan at cursor: the first, at start, goes through the
// Batcher cfg asks for, shared by every tracker reading through c, so the reads of many pools at
// one block go in one call; the pages after it are read alone. A nil cfg reads every page alone.
func CallPage[C comparable](ctx context.Context, cfg *BatchConfig, c Caller, cursor, start C, contract *Contract,
	out any, gas uint64, block *big.Int, args ...any) error {
	if cfg == nil || cursor != start {
		return Call(ctx, c, contract, out, gas, block, args...)
	}
	return SharedBatcher(c, cfg).Call(ctx, c, contract, out, gas, block, args...)
}
