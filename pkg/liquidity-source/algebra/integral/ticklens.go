package integral

import (
	"context"
	"math/big"
	"sort"
	"strconv"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
)

func (d *PoolTracker) getPoolTicksFromSC(ctx context.Context, pool entity.Pool, param sourcePool.GetNewPoolStateParams) ([]TickResp, error) {
	changedTicks := ticklens.GetChangedTicks(param.Logs)
	if len(changedTicks) == 0 {
		// Algebra doesn't compact the tick table, so it's not feasible to fetch all for now
		return nil, ErrNotSupportFetchFullTick
	}

	logger.Infof("Fetch changed ticks (%v)", changedTicks)

	rpcRequest := d.EthrpcClient.NewRequest()
	rpcRequest.SetContext(util.NewContextWithTimestamp(ctx))
	populatedTicks := make([]Tick, len(changedTicks))
	for i, tick := range changedTicks {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    algebraIntegralPoolABI,
			Target: pool.Address,
			Method: poolTicksMethod,
			Params: []interface{}{new(big.Int).SetInt64(tick)},
		}, []interface{}{&populatedTicks[i]})
	}

	resp, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, err
	}

	ticks := make([]TickResp, 0, len(resp.Request.Calls))
	for i, result := range resp.Result {
		if !result {
			logger.Errorf("failed to try multicall with param: %v", resp.Request.Calls[i].Params)
			continue
		}

		if populatedTicks[i].LiquidityTotal.Sign() == 1 {
			ticks = append(ticks, TickResp{
				TickIdx:        strconv.FormatInt(changedTicks[i], 10),
				LiquidityGross: populatedTicks[i].LiquidityTotal.String(),
				LiquidityNet:   populatedTicks[i].LiquidityDelta.String(),
			})
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
					LiquidityNet:   t.LiquidityNet.Dec(),
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

	return ticks, nil
}
