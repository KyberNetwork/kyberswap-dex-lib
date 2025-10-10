package aavev3

import (
	"context"
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
	startTime := time.Now()
	dexID := u.config.DexID
	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Warn("getOffset failed")
	}

	assets, err := u.getAssetList(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID}).
			Error("getAssetList failed")
		return nil, metadataBytes, err
	} else if len(assets) <= offset {
		return nil, metadataBytes, nil
	}

	pools, err := u.initPools(ctx, assets[offset:])
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initPools failed")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(len(assets))
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("newMetadata failed")
		return nil, metadataBytes, err
	}

	logger.
		WithFields(logger.Fields{
			"dex_id":      dexID,
			"new_pools":   len(pools),
			"duration_ms": time.Since(startTime).Milliseconds(),
		}).
		Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getAssetList(ctx context.Context) ([]common.Address, error) {
	var assets []common.Address
	_, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: u.config.AavePoolAddress,
		Method: poolMethodGetReservesList,
	}, []any{&assets}).Call()
	return assets, err
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
	aTokens, err := u.getReserveDatas(ctx, reserves)
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
		aTokenAddr := hexutil.Encode(aTokens[i].Data.ATokenAddress[:])
		assetTokenAddr := hexutil.Encode(reserve[:])
		pools = append(pools, entity.Pool{
			Address:   aTokenAddr,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: aTokenAddr, Swappable: true},
				{Address: assetTokenAddr, Swappable: true},
			},
			StaticExtra: string(extraBytes),
		})
	}

	return pools, nil
}

func (u *PoolsListUpdater) getReserveDatas(ctx context.Context, reserves []common.Address) ([]RPCReserveData, error) {
	reserveDatas := make([]RPCReserveData, len(reserves))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, reserve := range reserves {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: u.config.AavePoolAddress,
			Method: poolMethodGetReserveData,
			Params: []any{reserve},
		}, []any{&reserveDatas[i]})
	}

	_, err := req.Aggregate()
	return reserveDatas, err
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
