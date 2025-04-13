package ekubo

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"

	"github.com/KyberNetwork/logger"
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
) ([]quoting.PoolState, *big.Int, error) {
	abiPoolKeys := lo.Map(poolKeys, func(key *quoting.PoolKey, _ int) quoting.AbiPoolKey {
		return key.ToAbi()
	})

	quoteData := make([]QuoteData, 0, len(abiPoolKeys))

	req := ethrpcClient.R().SetContext(ctx)
	for startIdx := 0; startIdx < len(abiPoolKeys); startIdx += maxBatchSize {
		endIdx := min(startIdx+maxBatchSize, len(abiPoolKeys))

		req.AddCall(&ethrpc.Call{
			ABI:    dataFetcherABI,
			Target: dataFetcher,
			Method: dataFetcherMethodGetQuoteData,
			Params: []any{
				abiPoolKeys[startIdx:endIdx],
				minTickSpacingsPerPool,
			},
		}, []any{&quoteData})
	}
	resp, err := req.Aggregate()
	if err != nil {
		logger.Errorf("failed to aggregate quote data from data fetcher: %v", err)
		return nil, nil, err
	}

	poolStates := lo.Map(quoteData, func(data QuoteData, _ int) quoting.PoolState {
		return quoting.NewPoolState(
			data.Liquidity,
			math.FloatSqrtRatioToFixed(data.SqrtRatioFloat),
			data.Tick,
			data.Ticks,
			[2]int32{data.MinTick, data.MaxTick},
		)
	})

	return poolStates, resp.BlockNumber, nil
}
