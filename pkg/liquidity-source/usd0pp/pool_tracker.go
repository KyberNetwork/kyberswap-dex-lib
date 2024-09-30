package usd0pp

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"time"
)

var (
	ErrFailedToGetExtra = errors.New("failed to get extra")
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
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()
	logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address}).Info("Start getting new pool state")

	var paused bool
	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    usd0ppABI,
		Target: USD0PP,
		Method: usd0ppMethodPaused,
		Params: []interface{}{},
	}, []interface{}{&paused})
	_, err := calls.Call()
	if err != nil {
		return p, ErrFailedToGetExtra
	}

	var poolExtra PoolExtra
	if err := json.Unmarshal([]byte(p.Extra), &poolExtra); err != nil {
		return p, err
	}
	if poolExtra.Paused != paused {
		poolExtra.Paused = paused
		extraBytes, err := json.Marshal(poolExtra)
		if err != nil {
			return p, err
		}
		p.Extra = string(extraBytes)
	}
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
