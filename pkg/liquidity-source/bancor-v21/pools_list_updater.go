package bancorv21

import (
	"context"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type (
	PoolsListUpdater struct {
		config *Config
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

func NewPoolsListUpdater(config *Config) *PoolsListUpdater {
	return &PoolsListUpdater{
		config: config,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = u.config.DexID
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new innerPools")
	// ctx = util.NewContextWithTimestamp(ctx)

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
		Exchange:     DexType,
		Type:         DexType,
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
