package ekubo

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting/pool"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

const (
	maxBatchSize                  = 100
	minTickSpacingsPerPool uint32 = 2
)

type QuoteData struct {
	Tick      int32          `json:"tick"`
	SqrtRatio *big.Int       `json:"sqrtRatio"`
	Liquidity *big.Int       `json:"liquidity"`
	MinTick   int32          `json:"minTick"`
	MaxTick   int32          `json:"maxTick"`
	Ticks     []quoting.Tick `json:"ticks"`
}

func fetchPools(
	ctx context.Context,
	client *ethrpc.Client,
	dataFetcher string,
	poolKeys []quoting.PoolKey,
	extensions map[common.Address]pool.Extension,
	registeredPools map[string]bool,
) ([]entity.Pool, error) {
	poolKeysAbi := make([]quoting.AbiPoolKey, 0, len(poolKeys))
	for i := range poolKeys {
		poolKeysAbi = append(poolKeysAbi, (&poolKeys[i]).ToAbi())
	}

	pools := make([]entity.Pool, 0, len(poolKeysAbi))

	for startIdx := 0; startIdx < len(poolKeysAbi); startIdx += maxBatchSize {
		endIdx := min(startIdx+maxBatchSize, len(poolKeysAbi))

		var quoteData []QuoteData
		_, err := client.
			R().
			SetContext(ctx).
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
			logger.Errorf("failed to retrieve quote data from data fetcher: %w", err)
			continue
		}

		for i, data := range quoteData {
			extraJson, err := json.Marshal(Extra{
				State: quoting.NewPoolState(
					data.Liquidity,
					data.SqrtRatio,
					data.Tick,
					data.Ticks,
					[2]int32{data.MinTick, data.MaxTick},
				),
			})
			if err != nil {
				logger.WithFields(logger.Fields{
					"error": err,
				}).Error("marshalling extra failed")

				continue
			}

			poolKey := poolKeys[startIdx+i]
			extension := poolKey.Config.Extension

			var extensionId pool.Extension
			if extension.Cmp(common.Address{}) == 0 {
				extensionId = pool.Base
			} else if ext, ok := extensions[extension]; ok {
				extensionId = ext
			} else {
				logger.WithFields(logger.Fields{
					"poolKey": poolKey,
				}).Debug("skipping pool key with unknown extension")

				continue
			}

			staticExtraJson, err := json.Marshal(StaticExtra{
				PoolKey:   poolKey,
				Extension: extensionId,
			})
			if err != nil {
				logger.WithFields(logger.Fields{
					"error": err,
				}).Errorf("marshalling staticExtra failed")

				continue
			}

			pools = append(pools, entity.Pool{
				Extra:       string(extraJson),
				StaticExtra: string(staticExtraJson),
			})

			if registeredPools != nil {
				registeredPools[poolKey.StringId()] = true
			}
		}
	}

	return pools, nil
}
