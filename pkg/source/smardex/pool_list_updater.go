package smardex

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

type (
	PoolListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	PoolListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

func NewPoolListUpdater(config *Config, client *ethrpc.Client) *PoolListUpdater {
	return &PoolListUpdater{
		config:       config,
		ethrpcClient: client,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Debug("Start getting new pools")
	defer func() {
		logger.
			WithFields(
				logger.Fields{
					"dex_id":      u.config.DexID,
					"duration_ms": time.Since(startTime).Milliseconds(),
				}).
			Debug("Finish getting new pools")
	}()

	var metadata PoolListUpdaterMetadata
	if len(metadataBytes) > 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	var pairsLength *big.Int
	if _, err := u.ethrpcClient.NewRequest().AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryAllPairsLengthMethod,
		Params: nil,
	}, []interface{}{&pairsLength}).Call(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("%s: failed to get number of pools from factory", u.config.DexID)

		return nil, metadataBytes, err
	}
	totalNumberOfPools := int(pairsLength.Int64())

	pagingSize := u.config.PoolPagingSize
	currentOffset := metadata.Offset
	if currentOffset+pagingSize > totalNumberOfPools {
		pagingSize = totalNumberOfPools - currentOffset
		if pagingSize <= 0 {
			return nil, metadataBytes, nil
		}
	}

	getPoolAddressReq := u.ethrpcClient.NewRequest()
	var poolAddresses = make([]common.Address, pagingSize)
	for i := 0; i < pagingSize; i++ {
		getPoolAddressReq.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryAllPairsMethod,
			Params: []interface{}{big.NewInt(int64(currentOffset + i))},
		}, []interface{}{&poolAddresses[i]})
	}
	if _, err := getPoolAddressReq.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("%s: failed to get pool address list", u.config.DexID)

		return nil, metadataBytes, err
	}

	pools, err := u.initPairs(ctx, poolAddresses)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("%s: failed to init pool's info", u.config.DexID)

		return nil, metadataBytes, err
	}

	if len(pools) > 0 {
		logger.WithFields(logger.Fields{
			"dexID":                     u.config.DexID,
			"poolPagingSize":            u.config.PoolPagingSize,
			"totalNumberOfUpdatedPools": currentOffset + len(pools),
			"totalNumberOfPools":        totalNumberOfPools,
		}).Infof("%s: scan factory", u.config.DexID)
	}

	nextOffset := currentOffset + len(pools)
	newMetadataBytes, err := json.Marshal(PoolListUpdaterMetadata{
		Offset: nextOffset,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolListUpdater) initPairs(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	var (
		poolsLength     = len(poolAddresses)
		token0Addresses = make([]common.Address, poolsLength)
		token1Addresses = make([]common.Address, poolsLength)
	)

	rpcRequest := u.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	for i := 0; i < poolsLength; i++ {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: poolAddresses[i].Hex(),
			Method: pairToken0Method,
			Params: nil,
		}, []interface{}{&token0Addresses[i]})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: poolAddresses[i].Hex(),
			Method: pairToken1Method,
			Params: nil,
		}, []interface{}{&token1Addresses[i]})
	}

	if _, err := rpcRequest.Aggregate(); err != nil {
		logger.Errorf("%s: failed to process aggregate to get 2 tokens from pair contract, err: %v", u.config.DexID, err)
		return nil, err
	}

	pools := make([]entity.Pool, 0, poolsLength)

	for i, pairAddress := range poolAddresses {
		p := strings.ToLower(pairAddress.Hex())
		token0Address := strings.ToLower(token0Addresses[i].Hex())
		token1Address := strings.ToLower(token1Addresses[i].Hex())

		var poolToken0 = entity.PoolToken{
			Address:   token0Address,
			Swappable: true,
		}
		var poolToken1 = entity.PoolToken{
			Address:   token1Address,
			Swappable: true,
		}

		var newPool = entity.Pool{
			Address:      p,
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			SwapFee:      0,
			Exchange:     u.config.DexID,
			Type:         DexTypeSmardex,
			Timestamp:    time.Now().Unix(),
			Reserves:     []string{reserveZero, reserveZero},
			Tokens:       []*entity.PoolToken{&poolToken0, &poolToken1},
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
