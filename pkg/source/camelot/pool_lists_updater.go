package camelot

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolListsUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolListsUpdater {
	return &PoolListsUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolListsUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) > 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexID": d.cfg.DexID,
				"error": err,
			}).Error("can not unmarshal metadata")
			return nil, metadataBytes, err
		}
	}

	logger.WithFields(logger.Fields{
		"dexID":  d.cfg.DexID,
		"offset": metadata.Offset,
	}).Info("get new pools")

	pairCount, err := d.getPairCount(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get pair count")
		return nil, metadataBytes, err
	}

	pairAddresses, err := d.getPairAddresses(ctx, metadata.Offset, pairCount)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get pair addresses")
		return nil, metadataBytes, err
	}

	if len(pairAddresses) == 0 {
		return nil, metadataBytes, nil
	}

	pools, err := d.getNewPools(ctx, pairAddresses)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get new pools")
		return nil, metadataBytes, err
	}

	metadata.Offset = metadata.Offset + uint64(len(pairAddresses))
	newMetadataBytes, err := json.Marshal(metadata)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not marshal metadata")
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolListsUpdater) getNewPools(ctx context.Context, pairAddresses []common.Address) ([]entity.Pool, error) {
	var (
		token0Addresses = make([]common.Address, len(pairAddresses))
		token1Addresses = make([]common.Address, len(pairAddresses))
		feeDenominators = make([]*big.Int, len(pairAddresses))
	)

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i, pairAddr := range pairAddresses {
		req.
			AddCall(&ethrpc.Call{
				ABI:    camelotPairABI,
				Target: pairAddr.Hex(),
				Method: pairMethodToken0,
				Params: nil,
			}, []interface{}{&token0Addresses[i]}).
			AddCall(&ethrpc.Call{
				ABI:    camelotPairABI,
				Target: pairAddr.Hex(),
				Method: pairMethodToken1,
				Params: nil,
			}, []interface{}{&token1Addresses[i]}).
			AddCall(&ethrpc.Call{
				ABI:    camelotPairABI,
				Target: pairAddr.Hex(),
				Method: pairMethodFeeDenominator,
				Params: nil,
			}, []interface{}{&feeDenominators[i]})
	}

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get new pools")
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(pairAddresses))
	for i, pairAddr := range pairAddresses {
		token0 := entity.PoolToken{
			Address:   strings.ToLower(token0Addresses[i].Hex()),
			Weight:    defaultTokenWeight,
			Swappable: true,
		}
		token1 := entity.PoolToken{
			Address:   strings.ToLower(token1Addresses[i].Hex()),
			Weight:    defaultTokenWeight,
			Swappable: true,
		}

		staticExtra := StaticExtra{
			FeeDenominator: feeDenominators[i],
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexID": d.cfg.DexID,
				"error": err,
			}).Error("can not marshal static extra")
			return nil, err
		}

		pool := entity.Pool{
			Address:     strings.ToLower(pairAddr.Hex()),
			Exchange:    d.cfg.DexID,
			Type:        DexTypeCamelot,
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      []*entity.PoolToken{&token0, &token1},
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, pool)
	}

	return pools, nil
}

func (d *PoolListsUpdater) getPairAddresses(ctx context.Context, offset uint64, pairCount uint64) ([]common.Address, error) {
	start := offset
	end := offset + uint64(d.cfg.NewPoolLimit)
	if end > pairCount {
		end = pairCount
	}

	if start >= end {
		return []common.Address{}, nil
	}

	pairAddresses := make([]common.Address, end-start)
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i := start; i < end; i++ {
		req.AddCall(&ethrpc.Call{
			ABI:    camelotFactoryABI,
			Target: d.cfg.FactoryAddress,
			Method: factoryMethodAllPairs,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&pairAddresses[i-start]})
	}

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get pair addresses")
		return nil, err
	}

	return pairAddresses, nil
}

func (d *PoolListsUpdater) getPairCount(ctx context.Context) (uint64, error) {
	var pairCount *big.Int

	req := d.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    camelotFactoryABI,
			Target: d.cfg.FactoryAddress,
			Method: factoryMethodAllPairsLength,
			Params: nil,
		}, []interface{}{&pairCount})

	_, err := req.Call()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get pair count")
		return 0, err
	}

	return pairCount.Uint64(), nil
}
