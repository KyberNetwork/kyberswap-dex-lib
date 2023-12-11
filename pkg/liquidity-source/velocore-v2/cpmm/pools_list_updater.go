package cpmm

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var ErrWETHNotFound = errors.New("WETH not found")

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

func (d *PoolsListUpdater) InitPool(_ context.Context) error {
	return nil
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexId": d.config.DexID,
		"type":  DexType,
	}).Info("start get new pools")

	var offset, newPools int

	defer func(s time.Time) {
		logger.WithFields(logger.Fields{
			"dexId":    d.config.DexID,
			"type":     DexType,
			"offset":   offset,
			"newPools": newPools,
			"duration": time.Since(s).String(),
		}).Info("finish get new pools")
	}(time.Now())

	ctx = util.NewContextWithTimestamp(ctx)

	totalNumberOfPools, err := d.getPoolsLength(ctx)
	if err != nil {
		return nil, metadataBytes, err
	}

	offset, err = d.getOffset(metadataBytes)
	if err != nil {
		return nil, metadataBytes, err
	}

	batchSize := getBatchSize(totalNumberOfPools, d.config.NewPoolLimit, offset)
	if batchSize == 0 {
		return nil, metadataBytes, nil
	}

	poolAddresses, err := d.queryPoolAddresses(ctx, offset, batchSize)
	if err != nil {
		return nil, metadataBytes, err
	}

	pools, err := d.processBatch(ctx, poolAddresses)
	if err != nil {
		return nil, metadataBytes, err
	}
	newPools = len(pools)

	newMetadataBytes, err := d.newMetadata(offset + batchSize)
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	var (
		limit = len(poolAddresses)
		pools = make([]entity.Pool, 0, len(poolAddresses))

		tokens  = make([][maxPoolTokenNumber]bytes32, limit)
		weights = make([][maxPoolTokenNumber]*big.Int, limit)
	)

	req := d.ethrpcClient.R()
	for i := 0; i < limit; i++ {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddresses[i].Hex(),
			Method: poolMethodRelevantTokens,
			Params: nil,
		}, []interface{}{&tokens[i]})

		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddresses[i].Hex(),
			Method: poolMethodTokenWeights,
			Params: nil,
		}, []interface{}{&weights[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dexId": d.config.DexID,
			"type":  DexType,
		}).Error(err.Error())
		return nil, err
	}

	for i, poolAddress := range poolAddresses {
		var (
			p                = strings.ToLower(poolAddress.Hex())
			nativeTokenIndex = unknownInt

			poolTokens   = []*entity.PoolToken{}
			reserves     = []string{}
			tokenWeights = []*big.Int{}
		)

		for j := 0; j < maxPoolTokenNumber; j++ {
			t := tokens[i][j].unwrapToken()
			w := weights[i][j]
			if t == valueobject.ZeroAddress {
				break
			}

			if strings.EqualFold(t, valueobject.EtherAddress) {
				nativeTokenIndex = j
				weth, ok := valueobject.WETHByChainID[d.config.ChainID]
				if !ok {
					return nil, ErrWETHNotFound
				}
				t = strings.ToLower(weth)
			}

			poolTokens = append(poolTokens, &entity.PoolToken{
				Address:   t,
				Swappable: true,
			})
			tokenWeights = append(tokenWeights, w)
			reserves = append(reserves, reserveZero)
		}

		staticExtra := StaticExtra{
			Weights:          tokenWeights,
			PoolTokenNumber:  uint(len(poolTokens)),
			NativeTokenIndex: nativeTokenIndex,
			Vault:            d.config.VaultAddress,
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexId": d.config.DexID,
				"type":  DexType,
			}).Error(err.Error())
			return nil, err
		}

		newPool := entity.Pool{
			Address:      p,
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			Exchange:     d.config.DexID,
			Type:         DexType,
			Timestamp:    time.Now().Unix(),
			Reserves:     reserves,
			Tokens:       poolTokens,
			StaticExtra:  string(staticExtraBytes),
		}
		pools = append(pools, newPool)
	}

	return pools, nil
}

func (d *PoolsListUpdater) queryPoolAddresses(ctx context.Context, offset int, batchSize int) ([]common.Address, error) {
	poolAddresses := make([]common.Address, batchSize)
	req := d.ethrpcClient.R()
	for j := 0; j < batchSize; j++ {
		req.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: d.config.FactoryAddress,
			Method: factoryMethodPoolList,
			Params: []interface{}{big.NewInt(int64(offset + j))},
		}, []interface{}{&poolAddresses[j]})
	}

	resp, err := req.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId": d.config.DexID,
			"type":  DexType,
		}).Error(err.Error())
		return nil, err
	}

	var ret []common.Address
	for i, isSuccess := range resp.Result {
		if isSuccess {
			ret = append(ret, poolAddresses[i])
		}
	}

	return ret, nil
}

func (d *PoolsListUpdater) getPoolsLength(ctx context.Context) (int, error) {
	var l *big.Int
	req := d.ethrpcClient.R()
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: factoryMethodPoolsLength,
		Params: nil,
	}, []interface{}{&l})
	if _, err := req.Call(); err != nil {
		logger.WithFields(
			logger.Fields{
				"dexId": d.config.DexID,
				"type":  DexType,
			}).Error(err.Error())
		return 0, err
	}
	return int(l.Uint64()), nil
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadata := Metadata{
		Offset: newOffset,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId": u.config.DexID,
			"type":  DexType,
		}).Error(err.Error())
		return nil, err
	}

	return metadataBytes, nil
}

func (d *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}

	var metadata Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		logger.WithFields(logger.Fields{
			"dexId": d.config.DexID,
			"type":  DexType,
		}).Error(err.Error())
		return 0, err
	}

	return metadata.Offset, nil
}

func getBatchSize(length int, limit int, offset int) int {
	if offset == length {
		return 0
	}

	if offset+limit > length {
		return length - offset
	}

	return limit
}
