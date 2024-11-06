package usd0pp

import (
	"context"
	"errors"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	startTime := time.Now()
	logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address}).Info("Start getting new pool state")

	var paused bool
	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

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
