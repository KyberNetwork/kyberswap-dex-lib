package ticklens

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
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

type TickResp struct {
	TickIdx        string `json:"tickIdx"`
	LiquidityGross string `json:"liquidityGross"`
	LiquidityNet   string `json:"liquidityNet"`
}

type populatedTick struct {
	Tick           *big.Int
	LiquidityNet   *big.Int
	LiquidityGross *big.Int
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
) ([]TickResp, error) {
	tickSpace := getTickSpacing(pool.SwapFee)
	if tickSpace == 0 {
		// non-standard pool fee, try again from pool extra
		var extra struct{ TickSpacing int }
		if err := json.Unmarshal([]byte(pool.Extra), &extra); err != nil {
			return nil, errors.New("failed to get pool tick spacing")
		}
		tickSpace = extra.TickSpacing
	}
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
		rpcRequest := ethrpcClient.NewRequest()
		rpcRequest.SetContext(util.NewContextWithTimestamp(ctx))

		// In each word index, there will be 256 populatedTick, that's why the type is [][]populatedTick
		populatedTicks := make([][]populatedTick, multicallBatchSize)
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
