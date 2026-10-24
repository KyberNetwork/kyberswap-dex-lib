package ephemeral

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// batchNode answers LensBatch calls as the contract does, running testContract reads by their x:
// 13 is a read the batch does not run, 14 one that reverts with Known(), and any other x reads
// back as (x, -x). A read made alone answers the same. It counts both kinds of call.
type batchNode struct {
	batches, alone atomic.Int64
	failBatches    bool
	// stuck makes every read a batch runs come back as x 0: a page that made no progress, as one
	// the batch starts with less than its densest step does.
	stuck bool
	// gasOf is what reading x costs in a batch; nil is 1000 gas a read.
	gasOf func(x int64) uint64
	// delay is how long a batch call takes; inFlight and peak count those running at once.
	delay          time.Duration
	inFlight, peak atomic.Int64

	mu     sync.Mutex
	blocks []uint64
	sizes  []int
}

func (n *batchNode) CallContract(_ context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
	if bytes.Equal(msg.Data, probeCode) {
		return common32(50_000_000 - probeIntrinsic), nil
	}
	if !bytes.HasPrefix(msg.Data, lensBatch.bytecode) {
		n.alone.Add(1)
		x := new(big.Int).SetBytes(msg.Data[len(testContract.bytecode):]).Int64()
		return nil, dataError{hexutil.Encode(testRead(x))}
	}
	n.batches.Add(1)
	now := n.inFlight.Add(1)
	defer n.inFlight.Add(-1)
	for peak := n.peak.Load(); now > peak && !n.peak.CompareAndSwap(peak, now); peak = n.peak.Load() {
	}
	time.Sleep(n.delay)
	n.mu.Lock()
	var at uint64 // 0 for a call at the latest block
	if block != nil {
		at = block.Uint64()
	}
	n.blocks = append(n.blocks, at)
	n.sizes = append(n.sizes, len(msg.Data))
	n.mu.Unlock()
	if n.failBatches {
		return nil, failing{"429 too many requests"}
	}

	values, err := lensBatch.abi.Constructor.Inputs.Unpack(msg.Data[len(lensBatch.bytecode):])
	if err != nil {
		return nil, err
	}
	items := *abi.ConvertType(values[1], new([]lensBatchItem)).(*[]lensBatchItem)

	type result struct {
		Ran  bool
		Gas  *big.Int
		Data []byte
	}
	results := make([]result, len(items))
	for i := range results {
		results[i].Gas = new(big.Int)
	}
	count := 0
	for i, item := range items {
		x := new(big.Int).SetBytes(item.Args).Int64()
		if x == 13 {
			continue
		}
		count = i + 1
		gas := uint64(1000)
		if n.gasOf != nil {
			gas = n.gasOf(x)
		}
		if x == 15 {
			// Started, but its page too large for the batch to copy.
			results[i] = result{Gas: new(big.Int).SetUint64(gas)}
			continue
		}
		if n.stuck {
			x = 0
		}
		results[i] = result{Ran: true, Gas: new(big.Int).SetUint64(gas), Data: testRead(x)}
	}
	// A call at the latest runs at the node's head, here 1000.
	ran := block
	if ran == nil {
		ran = big.NewInt(1000)
	}
	payload, err := lensBatch.result.Pack(ran, big.NewInt(int64(count)), results)
	if err != nil {
		return nil, err
	}
	return nil, dataError{hexutil.Encode(payload)}
}

func testRead(x int64) []byte {
	if x == 14 {
		id := testContract.abi.Errors["Known"].ID
		return id[:4]
	}
	return append(common32(uint64(x)), int32Word(-x)...)
}

// read calls b for x at block and returns what came back.
func read(t *testing.T, b *Batcher, node *batchNode, x int64, block int64) (testResult, error) {
	t.Helper()
	var got testResult
	err := b.Call(t.Context(), node, testContract, &got, 0, big.NewInt(block), big.NewInt(x))
	return got, err
}

// TestBatcherGathersConcurrentReads: reads made at once at one block go in one call, and each gets
// what it would have alone.
func TestBatcherGathersConcurrentReads(t *testing.T) {
	node := &batchNode{}
	b := NewBatcher(node, BatchConfig{WindowMs: 50, Gas: 1_000_000_000})

	var wg sync.WaitGroup
	for x := int64(1); x <= 10; x++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := read(t, b, node, x, 100)
			assert.NoError(t, err)
			assert.EqualValues(t, x, got.Value.Int64())
			assert.EqualValues(t, -x, got.Tick.Int64())
		}()
	}
	wg.Wait()

	assert.EqualValues(t, 1, node.batches.Load())
	assert.EqualValues(t, 0, node.alone.Load())
	assert.Equal(t, BatchStats{Batches: 1, Batched: 10}, b.Stats())
}

// TestBatcherFlushesOnSize: a full batch goes at once, and the rest wait for the next.
func TestBatcherFlushesOnSize(t *testing.T) {
	node := &batchNode{}
	b := NewBatcher(node, BatchConfig{MaxItems: 3, WindowMs: 1000, Gas: 1_000_000_000})

	start := time.Now()
	var wg sync.WaitGroup
	for x := int64(1); x <= 6; x++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := read(t, b, node, x, 100)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	assert.EqualValues(t, 2, node.batches.Load())
	assert.Less(t, time.Since(start), 500*time.Millisecond, "full batches do not wait for the window")
}

// TestBatcherFlushesOnTime: a lone read goes once the window ends.
func TestBatcherFlushesOnTime(t *testing.T) {
	node := &batchNode{}
	b := NewBatcher(node, BatchConfig{WindowMs: 30})

	start := time.Now()
	got, err := read(t, b, node, 5, 100)
	require.NoError(t, err)
	assert.EqualValues(t, 5, got.Value.Int64())
	assert.GreaterOrEqual(t, time.Since(start), 30*time.Millisecond)
	assert.EqualValues(t, 1, node.batches.Load())
}

// TestBatcherKeepsBlocksApart: reads at different blocks never share a call.
func TestBatcherKeepsBlocksApart(t *testing.T) {
	node := &batchNode{}
	b := NewBatcher(node, BatchConfig{WindowMs: 50, Gas: 1_000_000_000})

	var wg sync.WaitGroup
	for i := int64(0); i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := read(t, b, node, i+1, 100+i%2)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	assert.EqualValues(t, 2, node.batches.Load())
	assert.ElementsMatch(t, []uint64{100, 101}, node.blocks)
}

// TestBatcherHandsEachReadItsOwnOutcome: a read that reverts gets its error, one the batch did not
// run is made alone, and neither touches the others.
func TestBatcherHandsEachReadItsOwnOutcome(t *testing.T) {
	node := &batchNode{}
	b := NewBatcher(node, BatchConfig{WindowMs: 50, Gas: 1_000_000_000})

	errs := make([]error, 3)
	got := make([]testResult, 3)
	var wg sync.WaitGroup
	for i, x := range []int64{7, 13, 14} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got[i], errs[i] = read(t, b, node, x, 100)
		}()
	}
	wg.Wait()

	require.NoError(t, errs[0])
	assert.EqualValues(t, 7, got[0].Value.Int64())
	require.NoError(t, errs[1], "a read the batch did not run is made alone")
	assert.EqualValues(t, 13, got[1].Value.Int64())
	assert.True(t, IsContractError(errs[2], "Known"))
	assert.EqualValues(t, 1, node.batches.Load())
	assert.EqualValues(t, 1, node.alone.Load())
	assert.Equal(t, BatchStats{Batches: 1, Batched: 2, Alone: 1}, b.Stats())
}

// TestBatcherFallsBackWhenTheBatchFails: a batch the node refuses costs its reads nothing but a
// read each made alone.
func TestBatcherFallsBackWhenTheBatchFails(t *testing.T) {
	node := &batchNode{failBatches: true}
	b := NewBatcher(node, BatchConfig{WindowMs: 10})

	got, err := read(t, b, node, 9, 100)
	require.NoError(t, err)
	assert.EqualValues(t, 9, got.Value.Int64())
	assert.EqualValues(t, 1, node.alone.Load())
}

// TestBatcherWithoutABatch: a nil Batcher is Call.
func TestBatcherWithoutABatch(t *testing.T) {
	node := &batchNode{}
	var none *Batcher
	got, err := read(t, none, node, 3, 100)
	require.NoError(t, err)
	assert.EqualValues(t, 3, got.Value.Int64())
	assert.EqualValues(t, 0, node.batches.Load())
	assert.EqualValues(t, 1, node.alone.Load())
}

// TestBatcherGathersReadsAtTheLatest: reads at the latest block share a batch too, its one call
// running them all at one block, kept apart from those pinned to a block.
func TestBatcherGathersReadsAtTheLatest(t *testing.T) {
	node := &batchNode{}
	b := NewBatcher(node, BatchConfig{WindowMs: 50, Gas: 1_000_000_000})

	var wg sync.WaitGroup
	for x := int64(1); x <= 6; x++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var block *big.Int
			if x > 3 {
				block = big.NewInt(100)
			}
			var got testResult
			assert.NoError(t, b.Call(context.Background(), node, testContract, &got, 0, block, big.NewInt(x)))
			assert.EqualValues(t, x, got.Value.Int64())
		}()
	}
	wg.Wait()

	assert.EqualValues(t, 2, node.batches.Load(), "one batch at the latest, one at block 100")
	assert.EqualValues(t, 0, node.alone.Load())
	assert.ElementsMatch(t, []uint64{0, 100}, node.blocks)
}

// TestBatcherFirstPageResumesAlone: a scan whose first page comes from a batch, cut short there,
// resumes with its next pages made alone, at the same block.
func TestBatcherFirstPageResumesAlone(t *testing.T) {
	node := &batchNode{}
	b := NewBatcher(node, BatchConfig{WindowMs: 10})

	var pages []int64
	read, err := Scan(t.Context(), int64(1), Pinned{Block: big.NewInt(100)},
		func(ctx context.Context, cursor int64, block *big.Int) (Page[int64], error) {
			call := Call
			if cursor == 1 {
				call = b.Call
			}
			var got testResult
			if err := call(ctx, node, testContract, &got, 0, block, big.NewInt(cursor)); err != nil {
				return Page[int64]{}, err
			}
			pages = append(pages, got.Value.Int64())
			return Page[int64]{Block: block.Uint64(), Next: cursor + 1, Done: cursor == 3}, nil
		})
	require.NoError(t, err)
	assert.EqualValues(t, 100, read)
	assert.Equal(t, []int64{1, 2, 3}, pages)
	assert.EqualValues(t, 1, node.batches.Load())
	assert.EqualValues(t, 2, node.alone.Load())
}

// TestScanReadsAStuckBatchedFirstPageAlone: a first page that comes back from a batch with no
// progress, as one the batch starts with too little gas does, is read again alone before the scan
// calls the read stuck, and only that page, once. A page stuck alone is stuck.
func TestScanReadsAStuckBatchedFirstPageAlone(t *testing.T) {
	// A page moves the cursor by the x it reads back: 0 where it made no progress.
	scan := func(cfg *BatchConfig, node *batchNode, step int64) ([]int64, error) {
		var moved []int64
		_, err := Scan(context.Background(), int64(0), Pinned{Block: big.NewInt(100)},
			func(ctx context.Context, cursor int64, block *big.Int) (Page[int64], error) {
				var got testResult
				err := CallPage(ctx, cfg, node, cursor, int64(0), testContract, &got, 0, block, big.NewInt(step))
				if err != nil {
					return Page[int64]{}, err
				}
				moved = append(moved, got.Value.Int64())
				next := cursor + got.Value.Int64()
				return Page[int64]{Block: block.Uint64(), Next: next, Done: next >= 3}, nil
			})
		return moved, err
	}

	node := &batchNode{stuck: true}
	moved, err := scan(&BatchConfig{}, node, 1)
	require.NoError(t, err)
	assert.Equal(t, []int64{0, 1, 1, 1}, moved, "the stuck first page, then the same page and the rest alone")
	assert.EqualValues(t, 1, node.batches.Load())
	assert.EqualValues(t, 3, node.alone.Load())

	node = &batchNode{stuck: true}
	_, err = scan(&BatchConfig{}, node, 0)
	assert.ErrorIs(t, err, ErrNoProgress)
	assert.EqualValues(t, 1, node.batches.Load())
	assert.EqualValues(t, 1, node.alone.Load(), "read alone once")

	node = &batchNode{}
	_, err = scan(nil, node, 0)
	assert.ErrorIs(t, err, ErrNoProgress)
	assert.EqualValues(t, 1, node.alone.Load(), "a page read alone the first time is not read again")
}

// TestScanWordsReadsAStuckBatchedFirstPageAlone: as for Scan, a bitmap scan's first page that a
// batch ran with no progress is read again alone.
func TestScanWordsReadsAStuckBatchedFirstPageAlone(t *testing.T) {
	node := &batchNode{stuck: true}
	cfg := &BatchConfig{}
	_, words, err := ScanWords(t.Context(), 0, 10, Pinned{Block: big.NewInt(100)}, WordsParallel,
		func(ctx context.Context, lower, upper int16, block *big.Int) (Page[int16], []int16, error) {
			var got testResult
			// Read alone, a page reads every word: x 11 moves the cursor past them all.
			err := CallPage(ctx, cfg, node, lower, int16(0), testContract, &got, 0, block, big.NewInt(11))
			if err != nil {
				return Page[int16]{}, nil, err
			}
			next := lower + int16(got.Value.Int64())
			var read []int16
			for w := lower; w < next && w <= upper; w++ {
				read = append(read, w)
			}
			return Page[int16]{Block: block.Uint64(), Next: next, Done: next > upper}, read, nil
		})
	require.NoError(t, err)
	assert.Len(t, words, 11)
	assert.EqualValues(t, 1, node.batches.Load())
	assert.EqualValues(t, 1, node.alone.Load())
}

// TestSharedBatcherIsOnePerCaller: the trackers reading through one node share its batches.
func TestSharedBatcherIsOnePerCaller(t *testing.T) {
	node, other := &batchNode{}, &batchNode{}
	assert.Nil(t, SharedBatcher(node, nil))
	a := SharedBatcher(node, &BatchConfig{})
	assert.Same(t, a, SharedBatcher(node, &BatchConfig{MaxItems: 5}))
	assert.NotSame(t, a, SharedBatcher(other, &BatchConfig{}))
}

// TestBatcherSendsALensCodeOnce: the reads of one lens share its code in the batch, so the call
// grows by their arguments alone.
func TestBatcherSendsALensCodeOnce(t *testing.T) {
	node := &batchNode{}
	b := NewBatcher(node, BatchConfig{WindowMs: 50, Gas: 1_000_000_000})

	var wg sync.WaitGroup
	for x := int64(100); x < 140; x++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := read(t, b, node, x, 100)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	require.Len(t, node.sizes, 1)
	perRead := (node.sizes[0] - len(lensBatch.bytecode) - len(testContract.bytecode)) / 40
	assert.LessOrEqual(t, perRead, 6*32+32, "a read adds its arguments and their ABI words, not the code")
}

// readAll reads xs at once at block 100 and waits for them.
func readAll(t *testing.T, b *Batcher, node *batchNode, xs ...int64) {
	t.Helper()
	var wg sync.WaitGroup
	for _, x := range xs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := read(t, b, node, x, 100)
			assert.NoError(t, err)
			assert.EqualValues(t, x, got.Value.Int64())
		}()
	}
	wg.Wait()
}

// TestBatcherPacksByGas: once the reads' costs are known, a batch takes as many as its gas holds,
// not as many as its count allows.
func TestBatcherPacksByGas(t *testing.T) {
	node := &batchNode{gasOf: func(int64) uint64 { return 20_000_000 }}
	b := NewBatcher(node, BatchConfig{WindowMs: 50, Gas: 50_000_000})

	readAll(t, b, node, 20, 21, 22, 23)
	require.EqualValues(t, 1, node.batches.Load(), "unknown reads are expected to be light")

	readAll(t, b, node, 20, 21, 22, 23)
	assert.EqualValues(t, 3, node.batches.Load(), "two reads of 20M fill a 50M batch")
}

// TestBatcherReadsAHeavyPoolAlone: a read known to fill half a batch is made alone, and the others
// keep their batch.
func TestBatcherReadsAHeavyPoolAlone(t *testing.T) {
	node := &batchNode{gasOf: func(x int64) uint64 {
		if x == 30 {
			return 30_000_000
		}
		return 1_000_000
	}}
	b := NewBatcher(node, BatchConfig{WindowMs: 50, Gas: 50_000_000})

	readAll(t, b, node, 30, 31, 32)
	readAll(t, b, node, 30, 31, 32)

	assert.EqualValues(t, 2, node.batches.Load())
	assert.EqualValues(t, 1, node.alone.Load(), "the heavy pool, read alone the second time")
	assert.EqualValues(t, 1, b.Stats().Heavy)
}

// TestBatcherLearnsAPageTooLargeForItsBatch: a read the batch started but could not copy back is
// made alone, and once its cost fills half a batch it is read alone without one.
func TestBatcherLearnsAPageTooLargeForItsBatch(t *testing.T) {
	node := &batchNode{gasOf: func(int64) uint64 { return 30_000_000 }}
	b := NewBatcher(node, BatchConfig{WindowMs: 50, Gas: 50_000_000})

	got, err := read(t, b, node, 15, 7)
	require.NoError(t, err)
	assert.EqualValues(t, 15, got.Value.Int64())
	assert.EqualValues(t, 1, node.batches.Load())
	assert.EqualValues(t, 1, node.alone.Load(), "made alone after the batch")

	_, err = read(t, b, node, 15, 8)
	require.NoError(t, err)
	assert.EqualValues(t, 1, node.batches.Load(), "no second batch")
	assert.EqualValues(t, 2, node.alone.Load())
	assert.EqualValues(t, 1, b.Stats().Heavy)
}

// TestBatcherLearnsALensFromItsPools: a pool not read yet is expected to cost what its lens'sss
// pools have cost.
func TestBatcherLearnsALensFromItsPools(t *testing.T) {
	node := &batchNode{gasOf: func(int64) uint64 { return 8_000_000 }}
	b := NewBatcher(node, BatchConfig{WindowMs: 50, Gas: 50_000_000})

	readAll(t, b, node, 40)
	packed, err := testContract.packArgs(big.NewInt(41))
	require.NoError(t, err)
	assert.EqualValues(t, 8_000_000, b.expected(testContract, packed))
}

// TestBatcherBoundsTheBatchesInFlight: full batches go out together up to the bound, and no
// further.
func TestBatcherBoundsTheBatchesInFlight(t *testing.T) {
	node := &batchNode{delay: 50 * time.Millisecond}
	b := NewBatcher(node, BatchConfig{MaxItems: 1, Concurrency: 3, Gas: 1_000_000_000})

	start := time.Now()
	readAll(t, b, node, 50, 51, 52, 53, 54, 55, 56, 57, 58)

	assert.EqualValues(t, 9, node.batches.Load())
	assert.EqualValues(t, 3, node.peak.Load(), "three batches at once, never more")
	assert.Less(t, time.Since(start), 9*50*time.Millisecond, "and not one at a time")
}

// TestBatcherCountsItsCalls: the rate counts batch calls, not the reads in them.
func TestBatcherCountsItsCalls(t *testing.T) {
	node := &batchNode{}
	b := NewBatcher(node, BatchConfig{WindowMs: 20, Gas: 1_000_000_000, Rate: 5, Burst: 1})

	start := time.Now()
	readAll(t, b, node, 60, 61, 62, 63, 64, 65, 66, 67)
	assert.EqualValues(t, 1, node.batches.Load(), "eight reads, one call")
	assert.Less(t, time.Since(start), 150*time.Millisecond, "one call is within the burst")

	b = NewBatcher(node, BatchConfig{MaxItems: 1, Gas: 1_000_000_000, Rate: 5, Burst: 1})
	start = time.Now()
	readAll(t, b, node, 70, 71, 72)
	assert.GreaterOrEqual(t, time.Since(start), 350*time.Millisecond, "three calls at five a second")
}

// TestLensGuardWithABatchKeepsTheCooldownOnly: with batched reads, a guard lets every pool read at
// once, the Batcher bounding the calls, and still holds a pool back for its cooldown.
func TestLensGuardWithABatchKeepsTheCooldownOnly(t *testing.T) {
	g := NewLensGuard(GuardConfig{}, &BatchConfig{})
	release := make(chan struct{})
	var ran atomic.Int64
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := g.Try(context.Background(), fmt.Sprint("pool", i), func(context.Context) error {
				ran.Add(1)
				<-release
				return nil
			})
			assert.True(t, ok)
			assert.NoError(t, err)
		}()
	}
	require.Eventually(t, func() bool { return ran.Load() == 10 }, time.Second, time.Millisecond,
		"ten pools at once, where a guard counting pools lets two")
	close(release)
	wg.Wait()

	ok, err := g.Try(t.Context(), "pool0", func(context.Context) error { return nil })
	require.NoError(t, err)
	assert.False(t, ok, "a pool read moments ago waits for its cooldown")

	assert.NotNil(t, NewLensGuard(GuardConfig{}, nil).limiter, "without a batch, the guard is NewGuard's")
}

// countingChain is a Chain that counts the chain ids it is asked.
type countingChain struct {
	asked atomic.Int64
}

func (h *countingChain) ChainID(context.Context) (*big.Int, error) {
	h.asked.Add(1)
	time.Sleep(20 * time.Millisecond)
	return big.NewInt(1), nil
}

// TestPinAsksTheChainOnce: reads that start together share one chain id question.
func TestPinAsksTheChainOnce(t *testing.T) {
	h := &countingChain{}
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := Pin(context.Background(), h, big.NewInt(7))
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 1, h.asked.Load())
}

// TestBatchConfigSwitch: a config file may switch batching on with true or off with false, as well
// as give its settings.
func TestBatchConfigSwitch(t *testing.T) {
	var cfg struct {
		LensBatch *BatchConfig `json:"lensBatch"`
	}
	for raw, want := range map[string]bool{
		`{}`:                             false,
		`{"lensBatch": true}`:            true,
		`{"lensBatch": false}`:           false,
		`{"lensBatch": {}}`:              true,
		`{"lensBatch": {"maxItems": 8}}`: true,
	} {
		cfg.LensBatch = nil
		require.NoError(t, json.Unmarshal([]byte(raw), &cfg), raw)
		assert.Equal(t, want, cfg.LensBatch.On(), raw)
	}
	require.NoError(t, json.Unmarshal([]byte(`{"lensBatch": {"maxItems": 8}}`), &cfg))
	assert.Equal(t, 8, cfg.LensBatch.MaxItems)
	assert.Error(t, json.Unmarshal([]byte(`{"lensBatch": "yes"}`), &cfg))
}

// TestLensGuardWithABatchAdmitsManyPoolsQuickly: with batched reads no rate holds keys back, so a
// backfill's pools all stay within their cooldown; admitting them does not walk every key each time.
func TestLensGuardWithABatchAdmitsManyPoolsQuickly(t *testing.T) {
	g := NewLensGuard(GuardConfig{}, &BatchConfig{})
	start := time.Now()
	for i := range 100_000 {
		ok, err := g.Try(t.Context(), fmt.Sprint("pool", i), func(context.Context) error { return nil })
		require.NoError(t, err)
		require.True(t, ok)
	}
	assert.Less(t, time.Since(start), 5*time.Second, "100,000 pools, where walking every key each time takes minutes")
}

// TestUncomparableCallerIsNotKeyed: a Caller that cannot be a map key, such as a func, gets no
// shared Batcher and is warned of its gas every time, rather than panicking as a sync.Map key.
func TestUncomparableCallerIsNotKeyed(t *testing.T) {
	c := callerFunc(func(ethereum.CallMsg, *big.Int) ([]byte, error) {
		return common32(10_000_000 - probeIntrinsic), nil
	})
	assert.NotPanics(t, func() { assert.Nil(t, SharedBatcher(c, &BatchConfig{})) })
	assert.NotPanics(t, func() { callGas(t.Context(), c, 50_000_000) })
}
