package bancor_v21

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
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

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = u.config.DexID
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new innerPools")
	ctx = util.NewContextWithTimestamp(ctx)

	extra := Extra{}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("marshal extra failed")
		return nil, metadataBytes, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      dexID,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new innerPools")

	onePool := entity.Pool{
		Address:      u.config.BancorNetworkAddress,
		ReserveUsd:   0,
		AmplifiedTvl: 0,
		SwapFee:      0,
		Exchange:     DexTypeBancorV21,
		Type:         DexTypeBancorV21,
		Timestamp:    time.Now().Unix(),
		Reserves:     nil,
		Tokens:       nil,
		Extra:        string(extraBytes),
		StaticExtra:  "",
		TotalSupply:  "",
		BlockNumber:  0,
	}
	return []entity.Pool{onePool}, metadataBytes, nil
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

// getBatchSize
// @params length number of pairs (factory tracked)
// @params limit number of pairs to be fetched in one run
// @params offset index of the last pair has been fetched
// @returns batchSize
func getBatchSize(length int, limit int, offset int) int {
	if offset == length {
		return 0
	}

	if offset+limit >= length {
		return length - offset
	}

	return limit
}
