package ekubo

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type QuoteData struct {
	Tick           int32          `json:"tick"`
	SqrtRatioFloat *big.Int       `json:"sqrtRatio"`
	Liquidity      *big.Int       `json:"liquidity"`
	MinTick        int32          `json:"minTick"`
	MaxTick        int32          `json:"maxTick"`
	Ticks          []quoting.Tick `json:"ticks"`
}

func fetchPoolStates(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	dataFetcher string,
	poolKeys []*quoting.PoolKey,
) ([]quoting.PoolState, error) {
	if len(poolKeys) == 0 {
		return nil, nil
	}

	abiPoolKeys := make([]quoting.AbiPoolKey, 0, len(poolKeys))
	for i := range poolKeys {
		abiPoolKeys = append(abiPoolKeys, poolKeys[i].ToAbi())
	}

	poolStates := make([]quoting.PoolState, len(abiPoolKeys))

	req := ethrpcClient.R().SetContext(ctx)
	for startIdx := 0; startIdx < len(abiPoolKeys); startIdx += maxBatchSize {
		endIdx := min(startIdx+maxBatchSize, len(abiPoolKeys))

		batchQuoteData := make([]QuoteData, endIdx-startIdx)
		resp, err := req.AddCall(&ethrpc.Call{
			ABI:    dataFetcherABI,
			Target: dataFetcher,
			Method: dataFetcherMethodGetQuoteData,
			Params: []any{
				abiPoolKeys[startIdx:endIdx],
				minTickSpacingsPerPool,
			},
		}, []any{&batchQuoteData}).Call()
		if err != nil {
			logger.Errorf("failed to retrieve quote data from data fetcher: %v", err)
			return nil, err
		}

		var blockNumber uint64
		if resp.BlockNumber != nil {
			blockNumber = resp.BlockNumber.Uint64()
		}

		for i, data := range batchQuoteData {
			poolStates[i] = quoting.NewPoolState(
				data.Liquidity,
				math.FloatSqrtRatioToFixed(data.SqrtRatioFloat),
				data.Tick,
				data.Ticks,
				[2]int32{data.MinTick, data.MaxTick},
			)
			poolStates[i].SetBlockNumber(blockNumber)
		}
	}

	return poolStates, nil
}
