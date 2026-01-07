package clear

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}
func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	metadata := Metadata{
		Offset: make(map[string]int),
	}
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, nil, err
		}
	}
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	lengthBI := make([]*big.Int, len(d.config.FactoryAddresses))
	for i, factoryAddress := range d.config.FactoryAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    clearFactoryABI,
			Target: factoryAddress,
			Method: methodVaultsLength,
		}, []any{&lengthBI[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get number of pools from master address")
		return nil, metadataBytes, err
	}
	left := d.config.NewPoolLimit
	batchSizes := lo.Map(d.config.FactoryAddresses, func(factoryAddress string, i int) int {
		totalNumberOfPools := int(lengthBI[i].Int64())
		currentOffset := metadata.Offset[factoryAddress]
		if currentOffset >= totalNumberOfPools || left <= 0 {
			return 0
		}
		newPools := lo.Ternary(currentOffset+left > totalNumberOfPools, totalNumberOfPools-currentOffset, left)
		left -= newPools
		return newPools
	})
	newPools := lo.Sum(batchSizes)
	if newPools == 0 {
		return nil, metadataBytes, nil
	}

	req = d.ethrpcClient.NewRequest().SetContext(ctx)
	nextOffset := make(map[string]int)
	poolAddresses := make([]common.Address, newPools)
	factoryAddresses := make([]string, newPools)
	k := 0
	for i, batchSize := range batchSizes {
		nextOffset[d.config.FactoryAddresses[i]] = metadata.Offset[d.config.FactoryAddresses[i]] + batchSize
		for j := 0; j < batchSize; j++ {
			req.AddCall(&ethrpc.Call{
				ABI:    clearFactoryABI,
				Target: d.config.FactoryAddresses[i],
				Method: methodVaults,
				Params: []any{big.NewInt(int64(metadata.Offset[d.config.FactoryAddresses[i]] + j))},
			}, []any{&poolAddresses[k]})
			factoryAddresses[k] = d.config.FactoryAddresses[i]
			k++
		}
	}

	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pool addresses")
		return nil, metadataBytes, err
	}

	req = d.ethrpcClient.NewRequest().SetContext(ctx)
	tokens := make([][]common.Address, newPools)

	for i := range poolAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    clearVaultABI,
			Target: poolAddresses[i].Hex(),
			Method: "tokens",
		}, []any{&tokens[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pool tokens")
		return nil, metadataBytes, err
	}

	req = d.ethrpcClient.NewRequest().SetContext(ctx)
	iouTokens := make([][]common.Address, newPools)
	for i := range tokens {
		iouTokens[i] = make([]common.Address, len(tokens[i]))
		for j := range tokens[i] {
			req.AddCall(&ethrpc.Call{
				ABI:    clearVaultABI,
				Target: poolAddresses[i].Hex(),
				Method: "iouOf",
				Params: []any{tokens[i][j]},
			}, []any{&iouTokens[i][j]})
		}
	}

	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pool iou tokens")
		return nil, metadataBytes, err
	}

	pools := lo.Map(poolAddresses, func(poolAddress common.Address, i int) entity.Pool {
		extra := Extra{
			SwapAddress: strings.ToLower(d.config.SwapAddress),
			IOUs: lo.Map(iouTokens[i], func(iouToken common.Address, _ int) string {
				return strings.ToLower(iouToken.Hex())
			}),
		}

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("[Clear] failed to marshal extra data")
			return entity.Pool{}
		}
		return entity.Pool{
			Address:  strings.ToLower(poolAddresses[i].Hex()),
			Exchange: d.config.DexID,
			Type:     DexType,
			Reserves: lo.Map(tokens[i], func(_ common.Address, _ int) string {
				return defaultReserves
			}),
			Tokens: lo.Map(tokens[i], func(token common.Address, j int) *entity.PoolToken {
				return &entity.PoolToken{
					Address:   token.Hex(),
					Swappable: true,
				}
			}),
			Extra: string(extraBytes),
		}
	})

	newMetadataBytes, err := json.Marshal(Metadata{
		Offset: nextOffset,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}
