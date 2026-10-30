package ephemeral

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

//go:embed batch.json
var batchJSON []byte

// lensBatch is LensBatch, built from contracts/batch: many lens reads in one call.
var lensBatch = MustLoad("LensBatch", batchJSON)

// BatchConfig sets how a Batcher gathers lens reads into one call.
type BatchConfig struct {
	// MaxItems flushes a batch once it holds this many reads; 0 is 64. The batch's gas, by what each
	// read is expected to cost, flushes it first when its reads are heavy.
	MaxItems int `json:"maxItems,omitempty"`
	// WindowMs is how long the first read of a batch waits for others; 0 is 20ms.
	WindowMs int `json:"windowMs,omitempty"`
	// Gas is the gas of one batch call; 0 is the node's cap up to CallGasCeiling, as for Call.
	Gas uint64 `json:"gas,omitempty"`
	// Concurrency is how many batch calls are in flight at once; 0 is 4. Batches past it wait their
	// turn, so a backfill's reads go out several calls at a time without a call per pool.
	Concurrency int `json:"concurrency,omitempty"`
	// Rate and Burst bound the lens calls made through the node, per second: every batch is one, as
	// is every read made alone. 0 is 10 a second with bursts of 20. They take the place of the
	// lens guards' rate, which would count pools.
	Rate  float64 `json:"rate,omitempty"`
	Burst int     `json:"burst,omitempty"`
	// MinItemGas is the least gas a read is started with; 0 is 6M. A lens given less than its largest
	// step, a full bitmap word, ~4.7M on the forks whose tick getter loads the most, makes no
	// progress at all.
	MinItemGas uint64 `json:"minItemGas,omitempty"`

	// off is a config written as false: present, but asking for no batching.
	off bool
}

// UnmarshalJSON takes the settings as an object, or true for the defaults and false for none, as
// a config file writes a switch.
func (c *BatchConfig) UnmarshalJSON(data []byte) error {
	switch string(bytes.TrimSpace(data)) {
	case "true":
		*c = BatchConfig{}
		return nil
	case "false":
		*c = BatchConfig{off: true}
		return nil
	}
	type settings BatchConfig
	return json.Unmarshal(data, (*settings)(c))
}

// On reports whether c asks for batching: given, and not false.
func (c *BatchConfig) On() bool {
	return c != nil && !c.off
}

const (
	defaultBatchItems       = 64
	defaultBatchWindow      = 20 * time.Millisecond
	defaultBatchMinItemGas  = 6_000_000
	defaultBatchConcurrency = 4
	defaultBatchRate        = 10
	defaultBatchBurst       = 20
	// defaultReadGas is what a read is expected to cost before its pool or its lens has been read:
	// a typical pool, well under the ~25M of one whose bitmap read covers every word.
	defaultReadGas = 5_000_000
	// defaultBatchBudget is a batch's gas when the node's cap cannot be probed.
	defaultBatchBudget = 50_000_000
	// batchOverhead is a batch's own gas: creating its runner, and the copies of its results.
	batchOverhead = 1_000_000
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
	// inFlight holds a slot per batch call being made.
	inFlight chan struct{}
	// calls bounds the lens calls made through the node, batches and reads alone alike.
	calls *rate.Limiter

	// costs is the gas each pool's read cost when last in a batch, by lens and arguments; kinds,
	// per lens, a running average of them, for a pool not read yet.
	costs sync.Map
	kinds sync.Map

	batches, batched, alone, heavy atomic.Int64
}

type costKey struct {
	contract *Contract
	args     string
}

type batch struct {
	block    *big.Int
	reads    []*batchRead
	codes    map[*Contract]int
	initCode int
	// expected is what its reads are expected to cost, together.
	expected uint64
	timer    *time.Timer
}

type batchRead struct {
	contract *Contract
	args     []byte
	gas      uint64
	expected uint64
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
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = defaultBatchConcurrency
	}
	if cfg.Rate <= 0 {
		cfg.Rate = defaultBatchRate
	}
	if cfg.Burst <= 0 {
		cfg.Burst = defaultBatchBurst
	}
	return &Batcher{caller: c, cfg: cfg, pending: map[uint64]*batch{}, inFlight: make(chan struct{}, cfg.Concurrency),
		calls: rate.NewLimiter(rate.Limit(cfg.Rate), cfg.Burst)}
}

var sharedBatchers sync.Map

// SharedBatcher is the Batcher of reads made through c, one per Caller, so that the trackers of
// every exchange reading through one node fill the same batches. cfg is that of the first to ask;
// nil or false asks for none, as does a Caller that cannot be a map key, which cannot be shared.
func SharedBatcher(c Caller, cfg *BatchConfig) *Batcher {
	if !cfg.On() || c == nil || !reflect.TypeOf(c).Comparable() {
		return nil
	}
	if b, ok := sharedBatchers.Load(c); ok {
		return b.(*Batcher)
	}
	b, _ := sharedBatchers.LoadOrStore(c, NewBatcher(c, *cfg))
	return b.(*Batcher)
}

// BatchStats counts what a Batcher did: batch calls made, reads they answered, reads made alone
// because a batch did not run them, and reads made alone because they would fill half a batch.
type BatchStats struct {
	Batches, Batched, Alone, Heavy int64
}

func (b *Batcher) Stats() BatchStats {
	if b == nil {
		return BatchStats{}
	}
	return BatchStats{Batches: b.batches.Load(), Batched: b.batched.Load(), Alone: b.alone.Load(),
		Heavy: b.heavy.Load()}
}

// budget is the gas of one batch call: the gas a call is sent (see callGas), or defaultBatchBudget
// when the node's cap cannot be learnt.
func (b *Batcher) budget(ctx context.Context) uint64 {
	if gas := callGas(ctx, b.caller, b.cfg.Gas); gas != 0 {
		return gas
	}
	return defaultBatchBudget
}

// expected is what reading args through contract is expected to cost: what that pool cost last
// time, else the running average of the lens's reads, else defaultReadGas.
func (b *Batcher) expected(contract *Contract, args []byte) uint64 {
	if gas, ok := b.costs.Load(costKey{contract, string(args)}); ok {
		return gas.(uint64)
	}
	if kind, ok := b.kinds.Load(contract); ok {
		return kind.(*atomic.Uint64).Load()
	}
	return defaultReadGas
}

// learn records what a read cost in a batch.
func (b *Batcher) learn(read *batchRead, gas uint64) {
	b.costs.Store(costKey{read.contract, string(read.args)}, gas)
	kind, loaded := b.kinds.LoadOrStore(read.contract, new(atomic.Uint64))
	avg := kind.(*atomic.Uint64)
	if !loaded {
		avg.Store(gas)
		return
	}
	// A quarter of the newest read: pools of one lens vary, and an average that followed each one
	// would size every batch by the last pool.
	old := avg.Load()
	avg.Store(old - old/4 + gas/4)
}

// Call is Call, through the batch of block when there is a Batcher and block is pinned: a read at
// the latest block names no block for the batch to share.
func (b *Batcher) Call(ctx context.Context, c Caller, contract *Contract, out any, gas uint64, block *big.Int,
	args ...any) error {
	if b == nil {
		return Call(ctx, c, contract, out, gas, block, args...)
	}
	packed, err := contract.packArgs(args...)
	if err != nil {
		return err
	}

	budget := b.budget(ctx)
	expected := b.expected(contract, packed)
	if expected > budget/2 {
		// A pool known to fill half a batch gains little from one and would push the others out of
		// it: reading every bitmap word of a spacing-1 pool costs ~25M.
		b.heavy.Add(1)
		return b.callAlone(ctx, c, contract, out, gas, block, args...)
	}

	read := &batchRead{contract: contract, args: packed, gas: gas, expected: expected,
		done: make(chan batchResult, 1)}
	b.add(block, read, budget)

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
		return b.callAlone(ctx, c, contract, out, gas, block, args...)
	}
	b.batched.Add(1)
	if first, ok := ctx.Value(firstPageKey{}).(*firstPage); ok {
		first.batched = true
	}
	return decodeResult(contract, res.payload, out)
}

// add queues read in the batch of block, and flushes the batch when it is full, or would grow past
// what a node takes as init code or past the gas its reads are expected to need.
func (b *Batcher) add(block *big.Int, read *batchRead, budget uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := latestBatch
	if block != nil {
		key = block.Uint64()
	}
	cur := b.pending[key]
	if cur != nil && (cur.initCode+cur.cost(read) > maxBatchInitCode ||
		batchOverhead+cur.expected+read.expected > budget) {
		b.detach(key, cur)
		go b.flush(cur)
		cur = nil
	}
	if cur == nil {
		cur = &batch{block: block, codes: map[*Contract]int{}, initCode: len(lensBatch.bytecode)}
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
	cur.expected += read.expected
	if _, ok := cur.codes[read.contract]; !ok {
		cur.codes[read.contract] = len(cur.codes)
	}
	cur.reads = append(cur.reads, read)
	if len(cur.reads) >= b.cfg.MaxItems {
		b.detach(key, cur)
		go b.flush(cur)
	}
}

// latestBatch keys the batch of reads at the latest block: one call runs all its reads at one
// block, whichever the node is at, and each read takes the block its lens reports.
const latestBatch = ^uint64(0)

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
		Gas  *big.Int
		Data []byte
	}
}

// flush runs the batch and hands each read its result; a batch that failed hands every read the
// error, and each is made alone. A batch refused for its size is cut in two first (see run).
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

	b.inFlight <- struct{}{}
	defer func() { <-b.inFlight }()
	ctx, cancel := context.WithTimeout(context.Background(), batchTimeout)
	defer cancel()
	b.run(ctx, cur.block, codes, cur.reads, items)
}

// run makes one batch call of reads, whose LensBatch items are items, and hands each read its
// result. Half the reads of a batch whose response the node refuses for its size (see
// refusedForSize) give about half the response: each half is then a call of its own, with every
// lens code of the batch, which its items index, and a read refused alone gets the error, to be
// made alone with less gas.
func (b *Batcher) run(ctx context.Context, block *big.Int, codes [][]byte, reads []*batchRead,
	items []lensBatchItem) {
	b.batches.Add(1)
	var out lensBatchResults
	err := b.calls.Wait(ctx)
	if err == nil {
		err = callOnce(ctx, b.caller, lensBatch, &out, callGas(ctx, b.caller, b.cfg.Gas), block, codes, items,
			new(big.Int).SetUint64(b.cfg.MinItemGas))
	}
	if len(reads) > 1 && refusedForSize(err) {
		mid := len(reads) / 2
		b.run(ctx, block, codes, reads[:mid], items[:mid])
		b.run(ctx, block, codes, reads[mid:], items[mid:])
		return
	}
	if err == nil && len(out.Results) != len(reads) {
		err = fmt.Errorf("ephemeral: a batch of %d reads returned %d results", len(reads), len(out.Results))
	}
	for i, read := range reads {
		if err != nil {
			read.done <- batchResult{err: err}
			continue
		}
		result := out.Results[i]
		if result.Gas.IsUint64() && result.Gas.Sign() > 0 {
			if result.Ran {
				b.learn(read, result.Gas.Uint64())
			} else {
				// Started and its gas spent, but its result too large for the batch to copy: it costs
				// at least that, and is read alone once that fills half a batch.
				b.costs.Store(costKey{read.contract, string(read.args)}, result.Gas.Uint64())
			}
		}
		read.done <- batchResult{ran: result.Ran, payload: result.Data}
	}
}

// callAlone is Call, each call it makes counted against the calls the Batcher allows.
func (b *Batcher) callAlone(ctx context.Context, c Caller, contract *Contract, out any, gas uint64, block *big.Int,
	args ...any) error {
	if err := b.calls.Wait(ctx); err != nil {
		return err
	}
	return callSmaller(ctx, c, b.calls, contract, out, gas, block, args...)
}

// CallPage is Call for the page of a scan at cursor: the first, at start, goes through the
// Batcher cfg asks for, shared by every tracker reading through c, so the reads of many pools at
// one block go in one call; the pages after it are read alone, counted against the Batcher's
// calls. A nil cfg reads every page alone, as Call.
func CallPage[C comparable](ctx context.Context, cfg *BatchConfig, c Caller, cursor, start C, contract *Contract,
	out any, gas uint64, block *big.Int, args ...any) error {
	if !cfg.On() {
		return Call(ctx, c, contract, out, gas, block, args...)
	}
	b := SharedBatcher(c, cfg)
	if cursor != start || ctx.Value(aloneKey{}) != nil {
		return b.callAlone(ctx, c, contract, out, gas, block, args...)
	}
	return b.Call(ctx, c, contract, out, gas, block, args...)
}

// A batch starts a read with what it has left, MinItemGas at the least, which on a chain that prices
// storage above Ethereum can be less than a pool's densest step: such a first page comes back with
// no progress where a call of its own, with the node's whole cap, makes some. Scan and ScanWords
// learn through firstPageKey that their first page came from a batch, and read it again alone,
// through aloneKey, before they call the read stuck.
type (
	firstPageKey struct{}
	aloneKey     struct{}
	firstPage    struct{ batched bool }
)
