package ambient

import (
	"context"
	"fmt"
	"math/big"
	"math/bits"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/sync/errgroup"
)

// readSlot(uint256)
var selectorReadSlot = [4]byte{0x02, 0xce, 0x8a, 0xf3}

// TickFetcher walks Ambient's 3-layer tick bitmap directly from readSlot().
type TickFetcher struct {
	Client  *ethclient.Client
	DexAddr common.Address
}

func NewTickFetcher(c *ethclient.Client, dexAddr common.Address) *TickFetcher {
	return &TickFetcher{Client: c, DexAddr: dexAddr}
}

// TickWindow is inclusive on both ends.
type TickWindow struct {
	MinTick int32
	MaxTick int32
}

// FullTickWindow covers the entire int24 domain.
var FullTickWindow = TickWindow{MinTick: -1 << 23, MaxTick: (1 << 23) - 1}

// MaxBatchSize bounds the per-roundtrip JSON-RPC batch size. Tenderly enforces
// a per-second rate limit on the underlying eth_call count, so very large
// batches (e.g. 260) trip 429s; chunks of 100 stay under typical free-tier
// limits while still saving ~10× roundtrips vs sequential.
var MaxBatchSize = 50

// BatchConcurrency bounds the number of chunk batches in flight at once.
// Chunks are independent eth_calls so parallelism cuts cold-load latency
// roughly linearly until the provider's per-second batch cap is hit.
var BatchConcurrency = 4

// ReadSlotsBatch issues `eth_call`s for the supplied storage slots in chunked
// JSON-RPC batches (size capped by MaxBatchSize). Chunks run in parallel up
// to BatchConcurrency workers. The result slice is positionally aligned with
// `slots`. Used to collapse the 3 stages of a cold pool load (curve+spec+
// template+lobby sweep, terminus, level+knockout) into a small number of
// roundtrips.
func (f *TickFetcher) ReadSlotsBatch(
	ctx context.Context,
	slots []common.Hash,
	blockNum *big.Int,
) ([]*big.Int, error) {
	if len(slots) == 0 {
		return nil, nil
	}

	out := make([]*big.Int, len(slots))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(BatchConcurrency)
	for start := 0; start < len(slots); start += MaxBatchSize {
		end := start + MaxBatchSize
		if end > len(slots) {
			end = len(slots)
		}
		chunkSlots := slots[start:end]
		chunkOut := out[start:end]
		g.Go(func() error {
			return f.readSlotsBatchChunk(gctx, chunkSlots, blockNum, chunkOut)
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *TickFetcher) readSlotsBatchChunk(
	ctx context.Context,
	slots []common.Hash,
	blockNum *big.Int,
	out []*big.Int,
) error {
	var blockArg string
	if blockNum == nil {
		blockArg = "latest"
	} else {
		blockArg = (*hexutil.Big)(blockNum).String()
	}

	results := make([]hexutil.Bytes, len(slots))
	batch := make([]rpc.BatchElem, len(slots))
	for i, slot := range slots {
		data := make([]byte, 0, len(selectorReadSlot)+32)
		data = append(data, selectorReadSlot[:]...)
		data = append(data, slot[:]...)
		batch[i] = rpc.BatchElem{
			Method: "eth_call",
			Args: []any{
				map[string]any{
					"to":   f.DexAddr.Hex(),
					"data": hexutil.Bytes(data).String(),
				},
				blockArg,
			},
			Result: &results[i],
		}
	}

	if err := batchCallWithRetry(ctx, f.Client.Client(), batch); err != nil {
		return fmt.Errorf("batch eth_call: %w", err)
	}

	for i, elem := range batch {
		if elem.Error != nil {
			return fmt.Errorf("batch elem %d (slot=%s): %w", i, slots[i].Hex(), elem.Error)
		}
		raw := results[i]
		if len(raw) != 32 {
			return fmt.Errorf("batch elem %d (slot=%s): unexpected len=%d", i, slots[i].Hex(), len(raw))
		}
		out[i] = new(big.Int).SetBytes(raw)
	}
	return nil
}

// batchCallWithRetry retries on HTTP 429 / "rate limit" errors with exponential
// backoff (0.5s, 1s, 2s, 4s, 8s). Per-element errors are not inspected — those
// surface to the caller. Only request-level errors are retried.
func batchCallWithRetry(ctx context.Context, client *rpc.Client, batch []rpc.BatchElem) error {
	const maxAttempts = 5
	delay := 500 * time.Millisecond
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

func (f *TickFetcher) readSlot(ctx context.Context, slot common.Hash, blockNum *big.Int) (*big.Int, error) {
	data := make([]byte, 0, len(selectorReadSlot)+32)
	data = append(data, selectorReadSlot[:]...)
	data = append(data, slot[:]...)

	res, err := f.Client.CallContract(ctx, ethereum.CallMsg{
		To:   &f.DexAddr,
		Data: data,
	}, blockNum)
	if err != nil {
		return nil, err
	}
	if len(res) != 32 {
		return nil, fmt.Errorf("unexpected readSlot return length=%d", len(res))
	}

	return new(big.Int).SetBytes(res), nil
}

func setBitPositions(word *big.Int) []uint8 {
	if word == nil || word.Sign() == 0 {
		return nil
	}

	limbs := word.Bits()
	out := make([]uint8, 0, word.BitLen())

	for limbIdx, limb := range limbs {
		u := uint(limb)
		for u != 0 {
			pos := limbIdx*bits.UintSize + bits.TrailingZeros(u)
			if pos >= 256 {
				return out
			}
			out = append(out, uint8(pos))
			u &= u - 1
		}
	}

	return out
}
