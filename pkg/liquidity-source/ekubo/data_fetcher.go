package ekubo

import (
	"context"
	"encoding/json"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	quoting2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

const (
	maxBatchSize                  = 100
	minTickSpacingsPerPool uint32 = 2
)

func fetchPools(
	ctx context.Context,
	client *ethrpc.Client,
	dataFetcher string,
	poolKeys []quoting2.PoolKey,
	registeredPools map[string]bool,
) ([]entity.Pool, error) {
	poolKeysAbi := make([]quoting2.AbiPoolKey, 0, len(poolKeys))
	for i := range poolKeys {
		poolKeysAbi = append(poolKeysAbi, (&poolKeys[i]).ToAbi())
	}

	pools := make([]entity.Pool, 0, len(poolKeysAbi))

	for startIdx := 0; startIdx < len(poolKeysAbi); startIdx += maxBatchSize {
		endIdx := min(startIdx+maxBatchSize, len(poolKeysAbi))

		var quoteData []QuoteData
		_, err := client.R().SetContext(ctx).
			AddCall(&ethrpc.Call{
				ABI:    dataFetcherABI,
				Target: dataFetcher,
				Method: "getQuoteData",
				Params: []any{
					poolKeysAbi[startIdx:endIdx],
					minTickSpacingsPerPool,
				},
			}, []any{&quoteData}).
			Call()

		if err != nil {
			logger.Errorf("failed to retrieve quote data from data fetcher: %v", err)
			continue
		}

		for i, data := range quoteData {
			extraJson, err := json.Marshal(data)
			if err != nil {
				return nil, err
			}

			poolKey := poolKeys[startIdx+i]

			pools = append(pools, entity.Pool{
				Extra: string(extraJson),
			})

			if registeredPools != nil {
				registeredPools[poolKey.StringId()] = true
			}
		}
	}

	return pools, nil
}
