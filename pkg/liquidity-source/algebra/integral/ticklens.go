package integral

import (
	"context"
	"sort"
	"strconv"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
)

const (
	batchSize   = 500
	maxWordSize = 256
)

func (d *PoolTracker) getPoolTicksFromSC(ctx context.Context, pool entity.Pool, param sourcePool.GetNewPoolStateParams) ([]TickResp, error) {
	var wordIndexes []int16

	changedTicks := ticklens.GetChangedTicks(param.Logs)
	if len(changedTicks) > 0 {
		// only refetch changed tick if possible
		wordIndexes = lo.Uniq(lo.Map(changedTicks, func(t int64, _ int) int16 { return int16(t >> 8) }))
	} else {
		// Algebra doesn't compact the tick table, so it's not feasible to fetch all for now
		return nil, ErrNotSupportFetchFullTick
	}

	logger.Infof("Fetch tick from wordPosition %v to %v (%v)", wordIndexes[0], wordIndexes[len(wordIndexes)-1], changedTicks)

	chunkedWordIndexes := lo.Chunk(wordIndexes, batchSize)

	var ticks []TickResp

	for _, chunk := range chunkedWordIndexes {
		rpcRequest := d.ethrpcClient.NewRequest()
		rpcRequest.SetContext(util.NewContextWithTimestamp(ctx))

		var populatedTicks []ticklens.PopulatedTick
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    ticklensABI,
			Target: d.config.TickLensAddress,
			Method: ticklensGetPopulatedTicksInWordMethod,
			Params: []interface{}{common.HexToAddress(pool.Address), chunk},
		}, []interface{}{&populatedTicks})

		_, err := rpcRequest.Call()
		if err != nil {
			return nil, err
		}

		if len(populatedTicks) > 0 {
			for _, pt := range populatedTicks {
				ticks = append(ticks, TickResp{
					TickIdx:        pt.Tick.String(),
					LiquidityGross: pt.LiquidityGross.String(),
					LiquidityNet:   pt.LiquidityNet.String(),
				})
			}
		}
	}

	// if we only fetched some ticks, then update them to the original ticks and return
	if len(changedTicks) > 0 {
		var extra Extra
		if err := json.Unmarshal([]byte(pool.Extra), &extra); err != nil {
			return nil, err
		}

		// ticklens contract might return unchanged tick (in the same word), so need to filter them out
		changedTickSet := mapset.NewThreadUnsafeSet(changedTicks...)
		changedTickMap := make(map[int]TickResp, len(changedTicks))
		for _, t := range ticks {
			tIdx, err := strconv.ParseInt(t.TickIdx, 10, 64)
			if err == nil && changedTickSet.ContainsOne(tIdx) {
				changedTickMap[int(tIdx)] = t
			}
		}

		combined := make([]TickResp, 0, len(changedTicks)+len(extra.Ticks))
		for _, t := range extra.Ticks {
			if tick, ok := changedTickMap[t.Index]; ok {
				// changed, use new value
				combined = append(combined, tick)
				delete(changedTickMap, t.Index)
			} else if changedTickSet.ContainsOne(int64(t.Index)) {
				// some changed ticks might be consumed entirely and are not in `changedTickMap`, delete them
				logger.Debugf("deleted tick %v %v", pool.Address, t)
			} else {
				// use old value
				combined = append(combined, TickResp{
					TickIdx:        strconv.Itoa(t.Index),
					LiquidityGross: t.LiquidityGross.String(),
					LiquidityNet:   t.LiquidityNet.String(),
				})
			}
		}

		// remaining (newly created ticks)
		for _, tick := range changedTickMap {
			combined = append(combined, tick)
		}
		ticks = combined
	}

	sort.SliceStable(ticks, func(i, j int) bool {
		iTick, _ := strconv.Atoi(ticks[i].TickIdx)
		jTick, _ := strconv.Atoi(ticks[j].TickIdx)

		return iTick < jTick
	})

	return nil, nil
}
