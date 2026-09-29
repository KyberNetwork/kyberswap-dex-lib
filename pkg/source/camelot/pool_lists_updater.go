package camelot

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolListsUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexTypeCamelot, NewPoolsListUpdater)

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

	pairCount, blockNumber, err := d.getPairCount(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get pair count")
		return nil, metadataBytes, err
	}

	pairAddresses, _, err := d.getPairAddresses(ctx, metadata.Offset, pairCount, blockNumber)
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

	pools, _, err := d.getNewPools(ctx, pairAddresses, blockNumber)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get new pools")
		return nil, metadataBytes, err
	}

	metadata.Offset = metadata.Offset + uint64(len(pairAddresses))
	if blockNumber != nil {
		for i := range pools {
			pools[i].BlockNumber = blockNumber.Uint64()
		}
	}
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

func (d *PoolListsUpdater) getNewPools(ctx context.Context, pairAddresses []common.Address, blockNumber *big.Int) ([]entity.Pool, *big.Int, error) {
	var (
		token0Addresses = make([]common.Address, len(pairAddresses))
		token1Addresses = make([]common.Address, len(pairAddresses))
		feeDenominators = make([]*big.Int, len(pairAddresses))
	)

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil {
		req.SetBlockNumber(blockNumber)
	}
	for i, pairAddr := range pairAddresses {
		req.
			AddCall(&ethrpc.Call{
				ABI:    camelotPairABI,
				Target: pairAddr.Hex(),
				Method: pairMethodToken0,
				Params: nil,
			}, []any{&token0Addresses[i]}).
			AddCall(&ethrpc.Call{
				ABI:    camelotPairABI,
				Target: pairAddr.Hex(),
				Method: pairMethodToken1,
				Params: nil,
			}, []any{&token1Addresses[i]}).
			AddCall(&ethrpc.Call{
				ABI:    camelotPairABI,
				Target: pairAddr.Hex(),
				Method: pairMethodFeeDenominator,
				Params: nil,
			}, []any{&feeDenominators[i]})
	}

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get new pools")
		return nil, nil, err
	}

	pools := make([]entity.Pool, 0, len(pairAddresses))
	for i, pairAddr := range pairAddresses {
		token0 := entity.PoolToken{
			Address:   hexutil.Encode(token0Addresses[i][:]),
			Swappable: true,
		}
		token1 := entity.PoolToken{
			Address:   hexutil.Encode(token1Addresses[i][:]),
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
			return nil, nil, err
		}

		pool := entity.Pool{
			Address:     hexutil.Encode(pairAddr[:]),
			Exchange:    d.cfg.DexID,
			Type:        DexTypeCamelot,
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      []*entity.PoolToken{&token0, &token1},
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, pool)
	}

	return pools, resp.BlockNumber, nil
}

func (d *PoolListsUpdater) getPairAddresses(ctx context.Context, offset uint64, pairCount uint64, blockNumber *big.Int) ([]common.Address, *big.Int, error) {
	start := offset
	end := min(offset+uint64(d.cfg.NewPoolLimit), pairCount)

	if start >= end {
		return []common.Address{}, blockNumber, nil
	}

	pairAddresses := make([]common.Address, end-start)
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil {
		req.SetBlockNumber(blockNumber)
	}
	for i := start; i < end; i++ {
		req.AddCall(&ethrpc.Call{
			ABI:    camelotFactoryABI,
			Target: d.cfg.FactoryAddress,
			Method: factoryMethodAllPairs,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&pairAddresses[i-start]})
	}

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get pair addresses")
		return nil, nil, err
	}

	return pairAddresses, resp.BlockNumber, nil
}

func (d *PoolListsUpdater) getPairCount(ctx context.Context) (uint64, *big.Int, error) {
	var pairCount *big.Int

	req := d.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    camelotFactoryABI,
			Target: d.cfg.FactoryAddress,
			Method: factoryMethodAllPairsLength,
			Params: nil,
		}, []any{&pairCount})

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get pair count")
		return 0, nil, err
	}

	return pairCount.Uint64(), resp.BlockNumber, nil
}
