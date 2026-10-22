package ephemeral

import (
	"bytes"
	"context"
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
	// gasOf is what reading x costs in a batch; nil is 1000 gas a read.
	gasOf func(x int64) uint64

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
	n.mu.Lock()
	n.blocks = append(n.blocks, block.Uint64())
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
		results[i] = result{Ran: true, Gas: new(big.Int).SetUint64(gas), Data: testRead(x)}
	}
	payload, err := lensBatch.result.Pack(block, big.NewInt(int64(count)), results)
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

// TestBatcherWithoutABatchOrABlock: a nil Batcher, and a read at the latest block, are Call.
func TestBatcherWithoutABatchOrABlock(t *testing.T) {
	node := &batchNode{}
	var none *Batcher
	got, err := read(t, none, node, 3, 100)
	require.NoError(t, err)
	assert.EqualValues(t, 3, got.Value.Int64())

	b := NewBatcher(node, BatchConfig{})
	require.NoError(t, b.Call(context.Background(), node, testContract, &got, 0, nil, big.NewInt(4)))
	assert.EqualValues(t, 4, got.Value.Int64())
	assert.EqualValues(t, 0, node.batches.Load())
	assert.EqualValues(t, 2, node.alone.Load())
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
