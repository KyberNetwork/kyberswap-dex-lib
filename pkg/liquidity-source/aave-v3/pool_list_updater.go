package aavev3

import (
	"context"
	"math/big"
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
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

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

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = u.config.DexID
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Warn("getOffset failed")
	}

	totalAssets, err := u.getTotalAssets(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID}).
			Error("getTotalAssets failed")

		return nil, metadataBytes, err
	}

	if offset >= totalAssets {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "offset": offset, "count": totalAssets}).
			Info("No new pools to fetch")

		return []entity.Pool{}, metadataBytes, nil
	}

	var assets []common.Address
	if offset == 0 {
		assets, err = u.getAssetList(ctx, totalAssets)
		if err != nil {
			logger.
				WithFields(logger.Fields{"dex_id": dexID}).
				Error("getAssetList failed")

			return nil, metadataBytes, err
		}
	} else {
		assets, err = u.getNewAssets(ctx, offset, totalAssets)
		if err != nil {
			logger.
				WithFields(logger.Fields{"dex_id": dexID}).
				Error("getNewAssets failed")

			return nil, metadataBytes, err
		}
	}

	pools, err := u.initPools(ctx, assets)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initPools failed")

		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + len(assets))
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("newMetadata failed")

		return nil, metadataBytes, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      dexID,
				"new_pools":   len(pools),
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getAssetList(ctx context.Context, totalAssets int) ([]common.Address, error) {
	assets := make([]common.Address, totalAssets)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: u.config.AavePoolAddress,
		Method: poolMethodGetReservesList,
		Params: nil,
	}, []any{&assets})

	if _, err := req.Call(); err != nil {
		return nil, err
	}

	return assets, nil
}

func (u *PoolsListUpdater) getTotalAssets(ctx context.Context) (int, error) {
	var reservesCount *big.Int

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: u.config.AavePoolAddress,
		Method: poolMethodGetReservesCount,
		Params: nil,
	}, []any{&reservesCount})

	if _, err := req.Call(); err != nil {
		return 0, err
	}

	return int(reservesCount.Uint64()), nil
}

func (u *PoolsListUpdater) getNewAssets(ctx context.Context, offset, count int) ([]common.Address, error) {
	assets := make([]common.Address, count-offset)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := offset; i < count; i++ {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: u.config.AavePoolAddress,
			Method: poolMethodGetReserveAddressById,
			Params: []any{uint16(i)},
		}, []any{&assets[i-offset]})
	}

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	return assets, nil
}

func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}

	return metadata.Offset, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, reserves []common.Address) ([]entity.Pool, error) {
	aTokens, err := u.getATokens(ctx, reserves)
	if err != nil {
		return nil, err
	}

	extraBytes, err := json.Marshal(&StaticExtra{
		AavePoolAddress: u.config.AavePoolAddress,
	})
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(reserves))

	for i, reserve := range reserves {
		assetTokenAddr := hexutil.Encode(reserve[:])
		aTokenAddr := hexutil.Encode(aTokens[i][:])

		aToken := &entity.PoolToken{
			Address:   aTokenAddr,
			Swappable: true,
		}

		assetToken := &entity.PoolToken{
			Address:   assetTokenAddr,
			Swappable: true,
		}

		var newPool = entity.Pool{
			Address:     aTokenAddr,
			Exchange:    u.config.DexID,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{"0", "0"},
			Tokens:      []*entity.PoolToken{aToken, assetToken},
			StaticExtra: string(extraBytes),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}

func (u *PoolsListUpdater) getATokens(ctx context.Context, reserves []common.Address) ([]common.Address, error) {
	aTokens := make([]common.Address, len(reserves))

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i, reserve := range reserves {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: u.config.AavePoolAddress,
			Method: poolMethodGetReserveAToken,
			Params: []any{reserve},
		}, []any{&aTokens[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	return aTokens, nil
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadata := PoolsListUpdaterMetadata{
		Offset: newOffset,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}
