package uniswapv3

import (
	"context"
	"sort"
	"strconv"

	"github.com/daoleno/uniswapv3-sdk/constants"
	"github.com/daoleno/uniswapv3-sdk/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

const (
	multicallBatchSize = 500
	maxWordSize        = 256
)

var (
	minWordIndex = utils.MinTick / maxWordSize
)

// getPoolTicksFromSC get all ticks of a pool from TickLens smart-contract
func (d *PoolTracker) getPoolTicksFromSC(ctx context.Context, pool entity.Pool) ([]TickResp, error) {
	tickSpace := getTickSpacing(pool.SwapFee)
	poolMinWordIdx := int16(minWordIndex/tickSpace - 1)
	poolMaxWordIdx := -poolMinWordIdx

	// Prepare the list of wordIndexes, the total number of indexes is poolMaxWordIdx-poolMinWordIdx+1
	wordIndexes := make([]int16, 0, poolMaxWordIdx-poolMinWordIdx+1)
	for idx := poolMinWordIdx; idx <= poolMaxWordIdx; idx++ {
		wordIndexes = append(wordIndexes, idx)
	}

	// We will process 500 word indexes at a time
	chunkedWordIndexes := lo.Chunk[int16](wordIndexes, multicallBatchSize)

	var ticks []TickResp

	for _, chunk := range chunkedWordIndexes {
		rpcRequest := d.ethrpcClient.NewRequest()
		rpcRequest.SetContext(util.NewContextWithTimestamp(ctx))

		// In each word index, there will be 256 populatedTick, that's why the type is [][]populatedTick
		populatedTicks := make([][]populatedTick, multicallBatchSize)
		for i, wordIndex := range chunk {
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    tickLensABI,
				Target: d.config.TickLensAddress,
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

	// Sort the ticks because function NewTickListDataProvider needs
	sort.SliceStable(ticks, func(i, j int) bool {
		iTick, _ := strconv.Atoi(ticks[i].TickIdx)
		jTick, _ := strconv.Atoi(ticks[j].TickIdx)

		return iTick < jTick
	})

	return ticks, nil
}

func getTickSpacing(swapFee float64) int {
	return constants.TickSpacings[constants.FeeAmount(swapFee)]
}
