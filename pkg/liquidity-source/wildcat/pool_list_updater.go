package wildcat

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}
	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
	Pair struct {
		Pair   common.Address
		Token0 common.Address
		Token1 common.Address
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
		return nil, metadataBytes, err
	}
	batchSize := u.config.NewPoolLimit
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	var pairs []Pair
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: "getPairs",
		Params: []any{big.NewInt(int64(offset)), big.NewInt(int64(offset + batchSize))},
	}, []any{&pairs})

	_, err = req.Aggregate()
	if err != nil {
		return nil, metadataBytes, err
	}

	pools := lo.Map(pairs, func(pair Pair, _ int) entity.Pool {
		tokenAddrs := []string{hexutil.Encode(pair.Token0[:]), hexutil.Encode(pair.Token1[:])}
		p := entity.Pool{
			Address:   hexutil.Encode(pair.Pair[:]),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: lo.Map(tokenAddrs, func(addr string, _ int) *entity.PoolToken {
				return &entity.PoolToken{
					Address:   valueobject.ZeroToWrappedLower(addr, valueobject.ChainID(u.config.ChainID)),
					Swappable: true,
				}
			}),
			Extra: "{}",
		}
		extra := Extra{
			IsNative: lo.Map(tokenAddrs, func(addr string, _ int) bool {
				return valueobject.IsZero(addr)
			}),
		}
		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return entity.Pool{}
		}
		p.Extra = string(extraBytes)
		return p
	})
	if _, err := TrackPools(ctx, pools, u.ethrpcClient, u.config); err != nil {
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + len(pairs))
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
				"pools_len":   len(pairs),
				"offset":      offset,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

// getOffset gets index of the last pair that is fetched
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
