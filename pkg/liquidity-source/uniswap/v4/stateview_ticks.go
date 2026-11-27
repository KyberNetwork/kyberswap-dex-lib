package uniswapv4

import (
	"cmp"
	"context"
	"fmt"
	"math/big"
	"slices"
	"strconv"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
)

const (
	minTick = -887272
	maxTick = 887272

	// tickBitmapOffset is Pool.State.tickBitmap (mapping(int16 => uint256)) after ticks (+4).
	tickBitmapOffset = 4 + 1

	// tickReadBatch caps the calls in one multicall of a full tick read: a pool with tickSpacing 1 has 6932
	// bitmap words.
	tickReadBatch = 500
)

// tickBitmapSlot is the storage slot of tickBitmap[word] for the pool whose state starts at base.
func tickBitmapSlot(base common.Hash, word int16) common.Hash {
	return mappingSlot(base, tickBitmapOffset, int64(word))
}

// readTicksFromStateView reads every initialized tick of the pool: the tick bitmap words covering
// [minTick, maxTick], then the tick info of each set bit, all pinned to the block of the first multicall.
func (t *PoolTracker) readTicksFromStateView(ctx context.Context, p entity.Pool) ([]ticklens.TickResp, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool static extra: %w", err)
	}
	spacing := int64(staticExtra.TickSpacing)
	if spacing <= 0 {
		return nil, fmt.Errorf("invalid tick spacing %d", spacing)
	}
	poolID := common.HexToHash(p.Address)
	base := stateSlot(poolID)

	// Usable ticks are multiples of spacing inside [minTick, maxTick]; Go's / truncates toward zero, which is
	// the compressed tick of the outermost usable tick on both sides. >> floors, as TickBitmap.position does.
	minWord, maxWord := int16((minTick/spacing)>>8), int16((maxTick/spacing)>>8)
	words := make([]int16, 0, int(maxWord)-int(minWord)+1)
	for w := int(minWord); w <= int(maxWord); w++ {
		words = append(words, int16(w))
	}

	var block *big.Int
	bitmaps := make([]*big.Int, len(words))
	slots := make([][32]byte, len(words))
	if err := t.readInBatches(ctx, len(words), &block, func(req *ethrpc.Request, i int) {
		if t.useExtsload() {
			req.AddCall(t.extsloadCall(tickBitmapSlot(base, words[i])), []any{&slots[i]})
			return
		}
		req.AddCall(&ethrpc.Call{ABI: stateViewABI, Target: t.config.StateViewAddress, Method: "getTickBitmap",
			Params: []any{poolID, words[i]}}, []any{&bitmaps[i]})
	}); err != nil {
		return nil, fmt.Errorf("failed to read tick bitmap: %w", err)
	}

	var tickIdxs []int64
	for i, w := range words {
		bm := bitmaps[i]
		if t.useExtsload() {
			bm = new(big.Int).SetBytes(slots[i][:])
		}
		if bm == nil {
			continue
		}
		for bit := range bm.BitLen() {
			if bm.Bit(bit) == 1 {
				tickIdxs = append(tickIdxs, (int64(w)*256+int64(bit))*spacing)
			}
		}
	}

	infos := make([]stateViewTick, len(tickIdxs))
	tickWords := make([][32]byte, len(tickIdxs))
	if err := t.readInBatches(ctx, len(tickIdxs), &block, func(req *ethrpc.Request, i int) {
		if t.useExtsload() {
			req.AddCall(t.extsloadCall(tickSlot(base, tickIdxs[i])), []any{&tickWords[i]})
			return
		}
		req.AddCall(&ethrpc.Call{ABI: stateViewABI, Target: t.config.StateViewAddress, Method: "getTickInfo",
			Params: []any{poolID, big.NewInt(tickIdxs[i])}}, []any{&infos[i]})
	}); err != nil {
		return nil, fmt.Errorf("failed to read tick info of %d ticks: %w", len(tickIdxs), err)
	}

	ticks := make([]ticklens.TickResp, 0, len(tickIdxs)) // ascending: words and bits are walked in order
	for i, idx := range tickIdxs {
		if t.useExtsload() {
			infos[i].LiquidityGross, infos[i].LiquidityNet = decodeTickInfo(tickWords[i])
		}
		if infos[i].LiquidityGross == nil || infos[i].LiquidityGross.Sign() == 0 || infos[i].LiquidityNet == nil {
			continue
		}
		ticks = append(ticks, ticklens.TickResp{
			TickIdx:        strconv.FormatInt(idx, 10),
			LiquidityGross: infos[i].LiquidityGross.String(),
			LiquidityNet:   infos[i].LiquidityNet.String(),
		})
	}
	return ticks, nil
}

// readInBatches runs n calls in multicalls of at most tickReadBatch calls. The first multicall's block pins the
// rest through *block.
func (t *PoolTracker) readInBatches(ctx context.Context, n int, block **big.Int,
	add func(req *ethrpc.Request, i int)) error {
	for lo := 0; lo < n; lo += tickReadBatch {
		req := t.ethrpcClient.NewRequest().SetContext(ctx)
		if *block != nil {
			req.SetBlockNumber(*block)
		}
		for i := lo; i < min(lo+tickReadBatch, n); i++ {
			add(req, i)
		}
		resp, err := req.Aggregate()
		if err != nil {
			return err
		}
		if *block == nil && resp.BlockNumber != nil && resp.BlockNumber.Sign() > 0 {
			*block = resp.BlockNumber
		}
	}
	return nil
}

// changedTicksFromLogs returns the ticks the logs change for poolID: those of Burn-shaped logs (ticks in topics) and
// of this pool's v4 ModifyLiquidity logs (ticks in data).
func changedTicksFromLogs(poolID common.Hash, logs []types.Log) ([]int64, error) {
	ticks := ticklens.GetChangedTicks(logs)
	modifyLiquidity := poolManagerABI.Events["ModifyLiquidity"].ID
	for _, log := range logs {
		if len(log.Topics) < 2 || log.Topics[0] != modifyLiquidity || log.Topics[1] != poolID {
			continue
		}
		ev, err := poolManagerFilterer.ParseModifyLiquidity(log)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ModifyLiquidity log: %w", err)
		}
		ticks = append(ticks, ev.TickLower.Int64(), ev.TickUpper.Int64())
	}
	return lo.Uniq(ticks), nil
}

// storedTickResps returns the stored ticks, ascending, as the tracker's tick responses.
func storedTickResps(stored []Tick) []ticklens.TickResp {
	ticks := make([]ticklens.TickResp, 0, len(stored))
	for _, t := range stored {
		ticks = append(ticks, ticklens.TickResp{
			TickIdx:        strconv.Itoa(t.Index),
			LiquidityGross: t.LiquidityGross.String(),
			LiquidityNet:   t.LiquidityNet.String(),
		})
	}
	slices.SortStableFunc(ticks, func(a, b ticklens.TickResp) int {
		ai, _ := strconv.Atoi(a.TickIdx)
		bi, _ := strconv.Atoi(b.TickIdx)
		return cmp.Compare(ai, bi)
	})
	return ticks
}
