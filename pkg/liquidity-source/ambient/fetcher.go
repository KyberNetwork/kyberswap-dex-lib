package ambient

import (
	"context"
	"fmt"
	"math/big"
	"math/bits"
	"sort"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Public CrocSwapDex entrypoint on Ethereum mainnet.
var MainnetDexAddr = common.HexToAddress("0xaaaaaaaaa24eeeb8d57d431224f73832bc34f688")

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

// FetchActive returns every active tick whose terminus bit is set inside window.
func (f *TickFetcher) FetchActive(
	ctx context.Context,
	poolHash common.Hash,
	window TickWindow,
	blockNum *big.Int,
) ([]int32, error) {
	if window.MinTick > window.MaxTick {
		return nil, fmt.Errorf("bad window: min=%d > max=%d", window.MinTick, window.MaxTick)
	}

	minLobby := LobbyKey(window.MinTick)
	maxLobby := LobbyKey(window.MaxTick)

	var out []int32

	for lobby16 := int16(minLobby); lobby16 <= int16(maxLobby); lobby16++ {
		lobby := int8(lobby16)
		probeTick := int32(lobby) << 16

		mezzWord, err := f.readSlot(ctx, MezzSlot(poolHash, probeTick), blockNum)
		if err != nil {
			return nil, fmt.Errorf("read mezz(lobby=%d): %w", lobby, err)
		}
		if mezzWord.Sign() == 0 {
			continue
		}

		for _, mezzBit := range setBitPositions(mezzWord) {
			mezzKey := WeldLobbyMezz(lobby, mezzBit)
			probeTickTerm := int32(mezzKey) << 8

			termWord, err := f.readSlot(ctx, TerminusSlot(poolHash, probeTickTerm), blockNum)
			if err != nil {
				return nil, fmt.Errorf("read terminus(lobby=%d mezz=%d): %w", lobby, mezzBit, err)
			}
			if termWord.Sign() == 0 {
				return nil, fmt.Errorf(
					"mezz bit set but terminus empty at lobby=%d mezz=%d: slot layout mismatch",
					lobby, mezzBit,
				)
			}

			for _, termBit := range setBitPositions(termWord) {
				tick := WeldLobbyMezzTerm(lobby, mezzBit, termBit)
				if tick < window.MinTick || tick > window.MaxTick {
					continue
				}
				out = append(out, tick)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

// MaxBatchSize bounds the per-roundtrip JSON-RPC batch size. Tenderly enforces
// a per-second rate limit on the underlying eth_call count, so very large
// batches (e.g. 260) trip 429s; chunks of 50 stay under typical free-tier
// limits while still saving ~5× roundtrips vs sequential.
var MaxBatchSize = 50

// ReadSlotsBatch issues `eth_call`s for the supplied storage slots in chunked
// JSON-RPC batches (size capped by MaxBatchSize). The result slice is
// positionally aligned with `slots`. Used to collapse the 3 stages of a cold
// pool load (curve+spec+template+lobby sweep, terminus, level+knockout) into
// a small number of roundtrips.
func (f *TickFetcher) ReadSlotsBatch(
	ctx context.Context,
	slots []common.Hash,
	blockNum *big.Int,
) ([]*big.Int, error) {
	if len(slots) == 0 {
		return nil, nil
	}

	out := make([]*big.Int, len(slots))
	for start := 0; start < len(slots); start += MaxBatchSize {
		end := start + MaxBatchSize
		if end > len(slots) {
			end = len(slots)
		}
		if err := f.readSlotsBatchChunk(ctx, slots[start:end], blockNum, out[start:end]); err != nil {
			return nil, err
		}
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
