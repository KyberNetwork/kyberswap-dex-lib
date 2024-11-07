package nomiswap

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}
	ctx = util.NewContextWithTimestamp(ctx)
	stableFactoryABI, _ := NomiStableFactoryMetaData.GetAbi()
	var lengthBI *big.Int
	if _, err := d.ethrpcClient.NewRequest().AddCall(&ethrpc.Call{
		ABI:    *stableFactoryABI,
		Target: d.config.FactoryAddress,
		Method: "allPairsLength",
		Params: nil,
	}, []interface{}{&lengthBI}).Call(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get number of pools from master address")

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
	for i := 0; i < batchSize; i++ {
		getPoolAddressRequest.AddCall(&ethrpc.Call{
			ABI:    *stableFactoryABI,
			Target: d.config.FactoryAddress,
			Method: "allPairs",
			Params: []interface{}{big.NewInt(int64(currentOffset + i))},
		}, []interface{}{&poolAddresses[i]})
	}
	if _, err := getPoolAddressRequest.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pool addresses")

		return nil, metadataBytes, err
	}

	pools, err := d.processBatch(ctx, poolAddresses)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to process get pool states")

		return nil, metadataBytes, err
	}

	if len(pools) > 0 {
		logger.WithFields(logger.Fields{
			"dexID":                     d.config.DexID,
			"batchSize":                 batchSize,
			"totalNumberOfUpdatedPools": currentOffset + batchSize,
			"totalNumberOfPools":        totalNumberOfPools,
		}).Info("scan NomiSwapStablePoolMaster")
	}

	nextOffset := currentOffset + batchSize
	if nextOffset > totalNumberOfPools {
		nextOffset = totalNumberOfPools
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		Offset: nextOffset,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	var (
		tokens = make([][2]common.Address, len(poolAddresses))
		// reserves = make([][2]*big.Int, len(poolAddresses))
	)

	stablePoolABI, _ := NomiStablePoolMetaData.GetAbi()
	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < len(poolAddresses); i++ {
		// reservesData, err1 := stablePoolABI.Pack("getReserves", nil)
		// calls.AddCall(&ethrpc.Call{CallData: reservesData}, []interface{}{&reserves[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    *stablePoolABI,
			Target: poolAddresses[i].Hex(),
			Method: "token0",
			Params: nil,
		}, []interface{}{&tokens[i][0]})
		calls.AddCall(&ethrpc.Call{
			ABI:    *stablePoolABI,
			Target: poolAddresses[i].Hex(),
			Method: "token1",
			Params: nil,
		}, []interface{}{&tokens[i][1]})
	}

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pool type and assets")

		return nil, err
	}

	var pools = make([]entity.Pool, 0, len(poolAddresses))
	for i := 0; i < len(poolAddresses); i++ {
		poolAddress := strings.ToLower(poolAddresses[i].Hex())
		token0Address := strings.ToLower(tokens[i][0].Hex())
		token1Address := strings.ToLower(tokens[i][1].Hex())
		var token0 = entity.PoolToken{
			Address:   token0Address,
			Weight:    defaultTokenWeight,
			Swappable: true,
		}
		var token1 = entity.PoolToken{
			Address:   token1Address,
			Weight:    defaultTokenWeight,
			Swappable: true,
		}

		newPool := entity.Pool{
			Address:   poolAddress,
			Exchange:  d.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{reserveZero, reserveZero},
			Tokens:    []*entity.PoolToken{&token0, &token1},
		}

		pools = append(pools, newPool)
	}
	return pools, nil
}
