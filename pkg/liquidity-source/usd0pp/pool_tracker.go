package usd0pp

import (
	"context"
	"encoding/json"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"time"
)

type (
	PoolTracker struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}
)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address}).Info("Start getting new pool state")

	found, isPaused, log := findLatestPausedOrUnpausedEvent(params.Logs)
	if !found {
		logger.
			WithFields(
				logger.Fields{
					"dex_id":  t.config.DexID,
					"pool_id": p.Address,
				},
			).
			Info("skip update: no paused/unpaused event found")
		return p, nil
	}

	if p.BlockNumber > log.BlockNumber {
		logger.
			WithFields(
				logger.Fields{
					"dex_id":            t.config.DexID,
					"pool_id":           p.Address,
					"pool_block_number": p.BlockNumber,
					"data_block_number": log.BlockNumber,
				},
			).
			Info("skip update: data block number is less than current pool block number")
		return p, nil
	}

	var poolExtra PoolExtra
	if err := json.Unmarshal([]byte(p.Extra), &poolExtra); err != nil {
		return p, err
	}

	poolExtra.Paused = isPaused
	extraBytes, err := json.Marshal(poolExtra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = log.BlockNumber
	p.Timestamp = time.Now().Unix()

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      t.config.DexID,
				"pool_id":     p.Address,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	return p, nil
}
