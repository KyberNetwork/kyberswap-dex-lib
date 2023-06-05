package dmm

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

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

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

	getNumPoolsRequest := d.ethrpcClient.NewRequest()
	getNumPoolsRequest.AddCall(&ethrpc.Call{
		ABI:    dmmFactoryABI,
		Target: d.config.FactoryAddress,
		Method: factoryMethodAllPoolsLength,
		Params: nil,
	}, []interface{}{&lengthBI})

	if _, err := getNumPoolsRequest.Call(); err != nil {
		logger.Errorf("failed to get number of pairs from factory, err: %v", err)
		return nil, metadataBytes, err
	}

	totalNumberOfPools := int(lengthBI.Int64())

	currentOffset := metadata.Offset
	batchSize := d.config.NewPoolLimit
	if currentOffset+batchSize > totalNumberOfPools {
		batchSize = totalNumberOfPools - currentOffset
		if batchSize <= 0 {
			return nil, metadataBytes, nil
		}
	}

	getPairAddressRequest := d.ethrpcClient.NewRequest()

	var pairAddresses = make([]common.Address, batchSize)
	for j := 0; j < batchSize; j++ {
		getPairAddressRequest.AddCall(&ethrpc.Call{
			ABI:    dmmFactoryABI,
			Target: d.config.FactoryAddress,
			Method: factoryMethodGetPool,
			Params: []interface{}{big.NewInt(int64(currentOffset + j))},
		}, []interface{}{&pairAddresses[j]})
	}

	if _, err := getPairAddressRequest.Aggregate(); err != nil {
		logger.Errorf("failed to process aggregate, err: %v", err)
		return nil, metadataBytes, err
	}

	pools, err := d.processBatch(ctx, pairAddresses)
	if err != nil {
		logger.Errorf("failed to process update new pool, err: %v", err)
		return nil, metadataBytes, err
	}

	numPools := len(pools)

	nextOffset := currentOffset + numPools
	newMetadataBytes, err := json.Marshal(Metadata{
		Offset: nextOffset,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	if len(pools) > 0 {
		logger.WithFields(logger.Fields{
			"dexID": d.config.DexID,
		}).Infof("scan DmmFactory with batch size %v, progress: %d/%d", batchSize, currentOffset+numPools, totalNumberOfPools)
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	var limit = len(poolAddresses)
	var token0Addresses = make([]common.Address, limit)
	var token1Addresses = make([]common.Address, limit)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	for i := 0; i < limit; i++ {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    dmmPoolABI,
			Target: poolAddresses[i].Hex(),
			Method: poolMethodToken0,
			Params: nil,
		}, []interface{}{&token0Addresses[i]})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    dmmPoolABI,
			Target: poolAddresses[i].Hex(),
			Method: poolMethodToken1,
			Params: nil,
		}, []interface{}{&token1Addresses[i]})
	}

	if _, err := rpcRequest.Aggregate(); err != nil {
		logger.Errorf("failed to process aggregate to get 2 tokens from pair contract, err: %v", err)
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolAddresses))

	for i, poolAddress := range poolAddresses {
		p := strings.ToLower(poolAddress.Hex())
		token0Address := strings.ToLower(token0Addresses[i].Hex())
		token1Address := strings.ToLower(token1Addresses[i].Hex())

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

		var newPool = entity.Pool{
			Address:      p,
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			Exchange:     d.config.DexID,
			Type:         DexTypeDMM,
			Timestamp:    time.Now().Unix(),
			Reserves:     []string{zeroString, zeroString},
			Tokens:       []*entity.PoolToken{&token0, &token1},
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
