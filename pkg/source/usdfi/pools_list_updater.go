package usdfi

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type PoolListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	// Add timestamp to the context so that each run iteration will have something different
	ctx = util.NewContextWithTimestamp(ctx)

	var lengthBI *big.Int
	if _, err := d.ethrpcClient.NewRequest().AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: poolFactoryMethodAllPairLength,
		Params: nil,
	}, []interface{}{&lengthBI}).Call(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get number of pools from factory")

		return nil, metadataBytes, err
	}
	totalNumberOfPools := int(lengthBI.Int64())

	batchSize := d.config.NewPoolLimit
	currentOffset := metadata.Offset
	if currentOffset+batchSize > totalNumberOfPools {
		batchSize = totalNumberOfPools - currentOffset
		if batchSize <= 0 {
			return nil, metadataBytes, nil
		}
	}

	getPoolAddressRequest := d.ethrpcClient.NewRequest()
	var poolAddresses = make([]common.Address, batchSize)
	for j := 0; j < batchSize; j++ {
		getPoolAddressRequest.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: d.config.FactoryAddress,
			Method: poolFactoryMethodAllPairs,
			Params: []interface{}{big.NewInt(int64(currentOffset + j))},
		}, []interface{}{&poolAddresses[j]})
	}
	if _, err := getPoolAddressRequest.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pool address")

		return nil, metadataBytes, err
	}

	pools, err := d.processBatch(ctx, poolAddresses)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to process update new pool")

		return nil, metadataBytes, err
	}

	if len(pools) > 0 {
		logger.WithFields(logger.Fields{
			"dexID":                     d.config.DexID,
			"batchSize":                 batchSize,
			"totalNumberOfUpdatedPools": currentOffset + len(pools),
			"totalNumberOfPools":        totalNumberOfPools,
		}).Infof("scan USDFiFactory")
	}

	nextOffset := currentOffset + len(pools)
	newMetadataBytes, err := json.Marshal(Metadata{
		Offset: nextOffset,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil

}

func (d *PoolListUpdater) processBatch(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	var (
		limit        = len(poolAddresses)
		poolMetadata = make([]USDFiMetadata, limit)
		pools        = make([]entity.Pool, 0, limit)
	)

	calls := d.ethrpcClient.NewRequest()
	calls.SetContext(ctx)

	for i := 0; i < limit; i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: poolAddresses[i].Hex(),
			Method: poolMethodMetadata,
			Params: nil,
		}, []interface{}{&poolMetadata[i]})
	}

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate to get tokens from pool")

		return nil, err
	}

	for i, pAddr := range poolAddresses {
		poolAddress := strings.ToLower(pAddr.Hex())
		token0Address := strings.ToLower(poolMetadata[i].T0.Hex())
		token1Address := strings.ToLower(poolMetadata[i].T1.Hex())

		var token0 = entity.PoolToken{
			Address:   token0Address,
			Weight:    defaultTokenWeight,
			Decimals:  uint8(len(poolMetadata[i].Dec0.String()) - 1),
			Swappable: true,
		}

		var token1 = entity.PoolToken{
			Address:   token1Address,
			Weight:    defaultTokenWeight,
			Decimals:  uint8(len(poolMetadata[i].Dec1.String()) - 1),
			Swappable: true,
		}

		staticExtra := StaticExtra{
			Stable: poolMetadata[i].St,
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to marshaling the static extra data")
			return nil, err
		}

		newPool := entity.Pool{
			Address:     poolAddress,
			Exchange:    d.config.DexID,
			Type:        DexTypeUSDFi,
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{reserveZero, reserveZero},
			Tokens:      []*entity.PoolToken{&token0, &token1},
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
