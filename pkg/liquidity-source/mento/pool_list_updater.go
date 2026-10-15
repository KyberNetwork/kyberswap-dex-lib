package mento

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

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type PoolsListUpdaterMetadata struct {
	Offset int `json:"offset"`
}

// poolMetadata mirrors FPMM.metadata()'s named outputs.
type poolMetadata struct {
	Dec0 *big.Int
	Dec1 *big.Int
	R0   *big.Int
	R1   *big.Int
	T0   common.Address
	T1   common.Address
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools enumerates FPMMFactory.deployedFPMMAddresses() (append-only)
// and returns the pools past the persisted offset, at most NewPoolLimit per
// call. Every error path returns the caller's metadata unchanged so the
// cursor never skips a failed batch.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	deployed, err := u.getDeployedPools(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{"dex": u.config.DexID, "err": err}).Error("deployedFPMMAddresses failed")
		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.WithFields(logger.Fields{"dex": u.config.DexID, "err": err}).Warn("invalid metadata, restarting from 0")
	}

	total := len(deployed)
	if offset >= total {
		return nil, metadataBytes, nil
	}

	batchSize := total - offset
	if u.config.NewPoolLimit > 0 && batchSize > u.config.NewPoolLimit {
		batchSize = u.config.NewPoolLimit
	}

	pools, err := u.initPools(ctx, deployed[offset:offset+batchSize])
	if err != nil {
		logger.WithFields(logger.Fields{"dex": u.config.DexID, "err": err}).Error("initPools failed")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := json.Marshal(PoolsListUpdaterMetadata{Offset: offset + batchSize})
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getDeployedPools(ctx context.Context) ([]common.Address, error) {
	var deployed []common.Address
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: methodDeployedFPMMAddresses,
	}, []any{&deployed})
	if _, err := req.Call(); err != nil {
		return nil, err
	}
	return deployed, nil
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

// initPools reads metadata() for each new pool in one multicall. Reserves are
// left at zero and token decimals unset; the tracker and pool-service fill
// those in. The pool's own decimals0/decimals1 scales are persisted in
// StaticExtra because the quote math uses them verbatim.
func (u *PoolsListUpdater) initPools(ctx context.Context, addresses []common.Address) ([]entity.Pool, error) {
	metas := make([]poolMetadata, len(addresses))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, addr := range addresses {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: hexutil.Encode(addr[:]),
			Method: methodMetadata,
		}, []any{&metas[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(addresses))
	for i, addr := range addresses {
		meta := metas[i]
		if meta.Dec0 == nil || meta.Dec1 == nil || meta.Dec0.Sign() == 0 || meta.Dec1.Sign() == 0 {
			logger.WithFields(logger.Fields{"dex": u.config.DexID, "pool": addr.Hex()}).Warn("metadata() returned zero decimals, skipping")
			continue
		}

		staticExtra, err := json.Marshal(StaticExtra{
			Decimals0: meta.Dec0.String(),
			Decimals1: meta.Dec1.String(),
		})
		if err != nil {
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(addr[:]),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(meta.T0[:]), Swappable: true},
				{Address: hexutil.Encode(meta.T1[:]), Swappable: true},
			},
			StaticExtra: string(staticExtra),
		})
	}

	return pools, nil
}
