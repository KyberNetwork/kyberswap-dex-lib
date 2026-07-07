package obric

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

	PoolRecord struct {
		XToken common.Address
		YToken common.Address
		Pool   common.Address
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
	logger.Info("started getting new pools")

	poolRecords, err := u.getPoolRecords(ctx)
	if err != nil {
		logger.Error("getPoolRecords failed")
		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.Warn("getOffset failed")
	}

	totalPools := len(poolRecords)
	if offset >= totalPools {
		return nil, metadataBytes, nil
	}

	batchSize := totalPools - offset
	if u.config.NewPoolLimit > 0 && batchSize > u.config.NewPoolLimit {
		batchSize = u.config.NewPoolLimit
	}

	newRecords := poolRecords[offset : offset+batchSize]

	pools, err := u.initPools(ctx, newRecords, offset)
	if err != nil {
		logger.Error("initPools failed")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + batchSize)
	if err != nil {
		logger.Error("newMetadata failed")
		return nil, metadataBytes, err
	}

	logger.Info("finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getPoolRecords(ctx context.Context) ([]PoolRecord, error) {
	var poolRecords []PoolRecord

	req := u.ethrpcClient.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    registryABI,
		Target: u.config.Factory,
		Method: "getPools",
	}, []any{&poolRecords})

	if _, err := req.Call(); err != nil {
		return nil, err
	}

	return poolRecords, nil
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

func (u *PoolsListUpdater) initPools(ctx context.Context, records []PoolRecord, offset int) ([]entity.Pool, error) {
	stateWrappers := make([]struct{ PoolState }, len(records))
	req := u.ethrpcClient.R().SetContext(ctx)
	for i, record := range records {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: record.Pool.Hex(),
			Method: "getState",
		}, []any{&stateWrappers[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(records))

	for i, record := range records {
		state := stateWrappers[i].PoolState

		tokenX := &entity.PoolToken{
			Address:   hexutil.Encode(record.XToken[:]),
			Decimals:  state.DecimalsX,
			Swappable: true,
		}
		tokenY := &entity.PoolToken{
			Address:   hexutil.Encode(record.YToken[:]),
			Decimals:  state.DecimalsY,
			Swappable: true,
		}

		staticExtra, err := json.Marshal(StaticExtra{
			PoolId:    offset + i,
			MultYBase: state.MultYBase.String(),
		})
		if err != nil {
			return nil, err
		}

		extra, err := json.Marshal(Extra{
			ReserveX:        state.ReserveX.String(),
			ReserveY:        state.ReserveY.String(),
			CurrentXK:       state.CurrentXK.String(),
			PreK:            state.PreK.String(),
			FeeMillionth:    state.FeeMillionth,
			PriceMaxAge:     state.PriceMaxAge.Uint64(),
			PriceUpdateTime: state.PriceUpdateTime.Uint64(),
			IsLocked:        state.IsLocked,
			Enable:          state.Enable,
		})
		if err != nil {
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:     hexutil.Encode(record.Pool[:]),
			Exchange:    u.config.DexId,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      []*entity.PoolToken{tokenX, tokenY},
			Extra:       string(extra),
			StaticExtra: string(staticExtra),
		})
	}

	return pools, nil
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadataBytes, err := json.Marshal(PoolsListUpdaterMetadata{Offset: newOffset})
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}
