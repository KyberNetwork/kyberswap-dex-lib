package ticklens

import (
	"context"
	"errors"
	"math/big"
	"sort"
	"strconv"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/daoleno/uniswapv3-sdk/utils"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type TickResp struct {
	TickIdx        string `json:"tickIdx"`
	LiquidityGross string `json:"liquidityGross"`
	LiquidityNet   string `json:"liquidityNet"`
}

type PopulatedTick struct {
	Tick           *big.Int
	LiquidityNet   *big.Int
	LiquidityGross *big.Int
}

type commonExtra struct {
	TickSpacing int
	Ticks       []struct {
		Index          int      `json:"index"`
		LiquidityGross *big.Int `json:"liquidityGross"`
		LiquidityNet   *big.Int `json:"liquidityNet"`
	} `json:"ticks"`
}

const (
	multicallBatchSize = 500
	maxWordSize        = 256

	tickLensMethodGetPopulatedTicksInWord = "getPopulatedTicksInWord"
)

var (
	minWordIndex = utils.MinTick / maxWordSize
)

// GetPoolTicksFromSC get all ticks of a pool from TickLens smart-contract
func GetPoolTicksFromSC(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	tickLensAddress string,
	pool entity.Pool,
	param pool.GetNewPoolStateParams,
) ([]TickResp, error) {
	var extra commonExtra
	if err := json.Unmarshal([]byte(pool.Extra), &extra); err != nil {
		return nil, errors.New("failed to unmarshal pool extra")
	}
	tickSpace := extra.TickSpacing

	var wordIndexes []int16

	changedTicks := GetChangedTicks(param.Logs)

	if len(changedTicks) > 0 {
		// only refetch changed tick if possible
		wordIndexes = lo.Uniq(lo.Map(changedTicks, func(t int64, _ int) int16 { return int16((t / int64(tickSpace)) >> 8) }))
	} else {
		// fetch all
		poolMinWordIdx := int16(minWordIndex/tickSpace - 1)
		poolMaxWordIdx := -poolMinWordIdx

		// Prepare the list of wordIndexes, the total number of indexes is poolMaxWordIdx-poolMinWordIdx+1
		wordIndexes = make([]int16, 0, poolMaxWordIdx-poolMinWordIdx+1)
		for idx := poolMinWordIdx; idx <= poolMaxWordIdx; idx++ {
			wordIndexes = append(wordIndexes, idx)
		}
	}

	logger.Infof("Fetch tick from wordPosition %v to %v (%v)", wordIndexes[0], wordIndexes[len(wordIndexes)-1], changedTicks)

	// We will process 500 word indexes at a time
	chunkedWordIndexes := lo.Chunk[int16](wordIndexes, multicallBatchSize)

	var ticks []TickResp

	for _, chunk := range chunkedWordIndexes {
		rpcRequest := ethrpcClient.NewRequest()
		rpcRequest.SetContext(util.NewContextWithTimestamp(ctx))

		// In each word index, there will be 256 populatedTick, that's why the type is [][]populatedTick
		populatedTicks := make([][]PopulatedTick, multicallBatchSize)
		for i, wordIndex := range chunk {
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    tickLensABI,
				Target: tickLensAddress,
				Method: tickLensMethodGetPopulatedTicksInWord,
				Params: []interface{}{common.HexToAddress(pool.Address), wordIndex},
			}, []interface{}{&populatedTicks[i]})
		}

		resp, err := rpcRequest.TryAggregate()
		if err != nil {
			return nil, err
		}

		resTicks := make([]TickResp, 0, len(resp.Request.Calls))
		for j, result := range resp.Result {
			if !result {
				logger.Errorf("failed to try multicall with param: %v", resp.Request.Calls[j].Params)
				continue
			}

			for _, pt := range populatedTicks[j] {
				resTicks = append(resTicks, TickResp{
					TickIdx:        pt.Tick.String(),
					LiquidityGross: pt.LiquidityGross.String(),
					LiquidityNet:   pt.LiquidityNet.String(),
				})
			}
		}

		if len(resTicks) > 0 {
			ticks = append(ticks, resTicks...)
		}
	}

	// if we only fetched some ticks, then update them to the original ticks and return
	if len(changedTicks) > 0 {
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
				logger.Infof("deleted tick %v %v", pool.Address, t)
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

	// Sort the ticks because function NewTickListDataProvider needs
	sort.SliceStable(ticks, func(i, j int) bool {
		iTick, _ := strconv.Atoi(ticks[i].TickIdx)
		jTick, _ := strconv.Atoi(ticks[j].TickIdx)

		return iTick < jTick
	})

	return ticks, nil
}

// only support Burn event for now
func GetChangedTicks(logs []types.Log) []int64 {
	var ticks []int64
	for _, log := range logs {
		if len(log.Topics) < 4 || log.Topics[0] != burnEvent.ID {
			continue
		}
		bottomTick := log.Topics[2].Big().Int64()
		topTick := log.Topics[3].Big().Int64()
		ticks = append(ticks, bottomTick, topTick)
	}
	return lo.Uniq(ticks)
}
