package integral

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
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

var _ = poollist.RegisterFactoryCE(DexTypeIntegral, NewPoolListUpdater)

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

	var factory common.Address
	if _, err := u.ethrpcClient.NewRequest().AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerFactoryMethod,
	}, []any{&factory}).Call(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("%s: failed to get factory address", u.config.DexID)

		return nil, metadataBytes, err
	}

	var pairsLength *big.Int
	if _, err := u.ethrpcClient.NewRequest().AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: factory.Hex(),
		Method: factoryAllPairsLengthMethod,
	}, []any{&pairsLength}).Call(); err != nil {
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
			Target: factory.Hex(),
			Method: factoryAllPairsMethod,
			Params: []any{big.NewInt(int64(currentOffset + i))},
		}, []any{&poolAddresses[i]})
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
	type pair struct {
		poolAddress string
		token0      common.Address
		token1      common.Address
	}

	poolsLength := len(poolAddresses)
	pairs := make([]pair, poolsLength)

	rpcRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i := 0; i < poolsLength; i++ {
		poolAddressHex := poolAddresses[i].Hex()

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: poolAddressHex,
			Method: pairToken0Method,
		}, []any{&pairs[i].token0}).AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: poolAddressHex,
			Method: pairToken1Method,
		}, []any{&pairs[i].token1})

		pairs[i].poolAddress = strings.ToLower(poolAddressHex)
	}

	if _, err := rpcRequest.Aggregate(); err != nil {
		logger.Errorf("%s: failed to process aggregate to get 2 tokens from pair contract, err: %v", u.config.DexID, err)
		return nil, err
	}

	extra, _ := json.Marshal(&Extra{
		RelayerAddress: u.config.RelayerAddress,
	})

	pools := make([]entity.Pool, 0, poolsLength)

	for _, pair := range pairs {
		newPool := entity.Pool{
			Address:   pair.poolAddress,
			SwapFee:   0,
			Exchange:  u.config.DexID,
			Type:      DexTypeIntegral,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   hexutil.Encode(pair.token0[:]),
					Swappable: true,
				},
				{
					Address:   hexutil.Encode(pair.token1[:]),
					Swappable: true,
				},
			},
			Extra: string(extra),
		}
		pools = append(pools, newPool)
	}

	return pools, nil
}
