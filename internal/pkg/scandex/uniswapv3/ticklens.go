package uniswapv3

import (
	"context"
	"sort"
	"strconv"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/pkg/logger"

	"github.com/daoleno/uniswapv3-sdk/constants"
	"github.com/daoleno/uniswapv3-sdk/utils"
	"github.com/ethereum/go-ethereum/common"
)

const (
	getPopulatedTicksInWord = "getPopulatedTicksInWord"
	multicallBatchSize      = 500
	maxWordSize             = 256
	minWordIndex            = utils.MinTick / maxWordSize
)

// getPoolTicksFromSC get all ticks of a pool from TickLens smart-contract
func (t *UniSwapV3) getPoolTicksFromSC(ctx context.Context, pool entity.Pool) ([]TickResp, error) {
	defer func() {
		logger.WithFields(logger.Fields{
			"pool": pool.Address,
		}).Debug("done fetching pool ticks from smart contract")
	}()

	tickSpace := getTickSpacing(pool.SwapFee)
	poolMinWordIdx := int16(minWordIndex/tickSpace - 1)
	poolMaxWordIdx := -poolMinWordIdx
	var ticks []TickResp

	batchIdx := 0
	var calls = make([]*repository.TryCallParams, 0, multicallBatchSize)
	var populatedTicks = make([][]populatedTick, multicallBatchSize)

	for i := poolMinWordIdx; i <= poolMaxWordIdx; i++ {
		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.UniV3TickLens,
			Target: t.properties.TickLensAddress,
			Method: getPopulatedTicksInWord,
			Params: []interface{}{common.HexToAddress(pool.Address), i},
			Output: &populatedTicks[batchIdx],
		})

		batchIdx += 1
		if batchIdx != multicallBatchSize && i != poolMaxWordIdx {
			continue
		}

		resTicks, err := t.processTickLenCall(ctx, populatedTicks, calls)
		if err != nil {
			return nil, err
		}

		batchIdx = 0
		calls = make([]*repository.TryCallParams, 0, multicallBatchSize)
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

func (t *UniSwapV3) processTickLenCall(ctx context.Context, populatedTicks [][]populatedTick, calls []*repository.TryCallParams) ([]TickResp, error) {
	if err := t.scanService.TryAggregate(ctx, false, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, err
	}

	ticks := make([]TickResp, 0, len(calls))
	for i, call := range calls {
		if !*call.Success {
			logger.Errorf("failed to try multicall with param %v", call.Params)
			continue
		}

		for _, pt := range populatedTicks[i] {
			ticks = append(ticks, TickResp{
				TickIdx:        pt.Tick.String(),
				LiquidityGross: pt.LiquidityGross.String(),
				LiquidityNet:   pt.LiquidityNet.String(),
			})
		}
	}

	return ticks, nil
}

func getTickSpacing(swapFee float64) int {
	return constants.TickSpacings[constants.FeeAmount(swapFee)]
}
