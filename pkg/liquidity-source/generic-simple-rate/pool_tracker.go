package generic_simple_rate

import (
	"context"
	"encoding/json"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

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
	return t.getNewPoolState(ctx, p, params)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()
	logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address}).Info("Start getting new pool state")

	var (
		paused bool
		rate   *big.Int
	)

	calls := t.ethrpcClient.NewRequest()
	var poolExtra PoolExtra
	if err := json.Unmarshal([]byte(p.Extra), &poolExtra); err != nil {
		return p, err
	}

	ABI := GetABI(p.Exchange)
	if t.config.PausedMethod != "" {
		calls.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: p.Address,
			Method: t.config.PausedMethod,
			Params: []interface{}{},
		}, []interface{}{&paused})
	}

	if t.config.IsRateUpdatable {
		calls.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: p.Address,
			Method: t.config.RateMethod,
			Params: []interface{}{},
		}, []interface{}{&rate})
	}

	if len(calls.Calls) == 0 {
		return p, nil
	}

	resp, err := calls.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address}).Error("Failed to get new pool state")
		return p, nil
	}

	if t.config.PausedMethod != "" {
		poolExtra.Paused = paused
	}

	if t.config.IsRateUpdatable {
		poolExtra.Rate = uint256.MustFromBig(rate)
	}

	extraBytes, err := json.Marshal(poolExtra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      t.config.DexID,
				"pool_id":     p.Address,
				"type":        DexType,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	return p, nil
}
