package syncswapv2shared

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

type PoolsListUpdater struct {
	Config       *syncswapv2.Config
	EthrpcClient *ethrpc.Client
}

func (d *PoolsListUpdater) GetPools(ctx context.Context, metadataBytes []byte, processBatch func(ctx context.Context, poolAddresses []common.Address, masterAddresses []string) ([]entity.Pool, error)) ([]entity.Pool, []byte, error) {
	var oldMetadata syncswap.Metadata
	var metadata Metadata

	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			if err1 := json.Unmarshal(metadataBytes, &oldMetadata); err1 != nil {
				return nil, metadataBytes, err1
			}
			metadata.Offset[d.Config.MasterAddress[0]] = oldMetadata.Offset
		}
	} else {
		metadata = Metadata{
			Offset: make(map[string]int),
		}
		for _, masterAddress := range d.Config.MasterAddress {
			metadata.Offset[masterAddress] = 0
		}
	}
	ctx = util.NewContextWithTimestamp(ctx)
	calls := d.EthrpcClient.NewRequest().SetContext(ctx)
	lengthBI := make([]*big.Int, len(d.Config.MasterAddress))
	for i, masterAddress := range d.Config.MasterAddress {
		calls.AddCall(&ethrpc.Call{
			ABI:    masterABI,
			Target: masterAddress,
			Method: poolMasterMethodPoolsLength,
			Params: nil,
		}, []interface{}{&lengthBI[i]})
	}
	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get number of pools from master address")

		return nil, metadataBytes, err
	}
	left := d.Config.NewPoolLimit
	batchSizes := make([]int, len(d.Config.MasterAddress))
	newPools := false
	for i, masterAddress := range d.Config.MasterAddress {
		totalNumberOfPools := int(lengthBI[i].Int64())
		currentOffset := metadata.Offset[masterAddress]
		if currentOffset >= totalNumberOfPools {
			continue
		}
		newPools = true
		if currentOffset+left > totalNumberOfPools {
			batchSizes[i] = totalNumberOfPools - currentOffset
		} else {
			batchSizes[i] = left
		}
		left -= batchSizes[i]
		if left <= 0 {
			break
		}
	}

	if !newPools {
		return nil, metadataBytes, nil
	}

	totalUpdatedPools := d.Config.NewPoolLimit - left
	getPoolAddressRequest := d.EthrpcClient.NewRequest()
	var poolAddresses = make([]common.Address, totalUpdatedPools)
	var masterAddresses = make([]string, totalUpdatedPools)
	var nextOffset = make(map[string]int)
	k := 0
	for i, batchSize := range batchSizes {
		nextOffset[d.Config.MasterAddress[i]] = metadata.Offset[d.Config.MasterAddress[i]] + batchSize
		for j := 0; j < batchSize; j++ {
			getPoolAddressRequest.AddCall(&ethrpc.Call{
				ABI:    masterABI,
				Target: d.Config.MasterAddress[i],
				Method: poolMasterMethodPools,
				Params: []interface{}{big.NewInt(int64(metadata.Offset[d.Config.MasterAddress[i]] + j))},
			}, []interface{}{&poolAddresses[k]})
			masterAddresses[k] = d.Config.MasterAddress[i]
			k++
		}
	}

	if _, err := getPoolAddressRequest.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pool addresses")

		return nil, metadataBytes, err
	}

	pools, err := processBatch(ctx, poolAddresses, masterAddresses)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to process get pool states")

		return nil, metadataBytes, err
	}

	// if len(pools) > 0 {
	// 	logger.WithFields(logger.Fields{
	// 		"dexID":                     d.Config.DexID,
	// 		"batchSize":                 batchSize,
	// 		"totalNumberOfUpdatedPools": currentOffset + batchSize,
	// 		"totalNumberOfPools":        totalNumberOfPools,
	// 	}).Info("scan SyncSwapPoolMaster")
	// }

	newMetadataBytes, err := json.Marshal(Metadata{
		Offset: nextOffset,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}
