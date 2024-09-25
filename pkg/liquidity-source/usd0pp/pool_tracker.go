package usd0pp

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"math/big"
	"time"
)

var (
	ErrFailedToGetExtra = errors.New("failed to get extra")
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
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address}).Info("Start getting new pool state")
	var (
		paused      bool
		totalSupply *big.Int
	)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    usd0ppABI,
		Target: USD0PP,
		Method: usd0ppMethodPaused,
		Params: []interface{}{},
	}, []interface{}{&paused})
	calls.AddCall(&ethrpc.Call{
		ABI:    usd0ppABI,
		Target: USD0PP,
		Method: usd0ppMethodTotalSupply,
		Params: nil,
	}, []interface{}{&totalSupply})

	if _, err := calls.Aggregate(); err != nil {
		logger.
			WithFields(
				logger.Fields{
					"liquiditySource": t.config.DexID,
					"poolAddress":     p.Address,
					"error":           err,
				}).
			Error("failed to get extra")

		return p, ErrFailedToGetExtra
	}

	var poolExtra PoolExtra
	if err := json.Unmarshal([]byte(p.Extra), &poolExtra); err != nil {
		return p, err
	}
	poolExtra.Paused = paused
	poolExtra.TotalSupply = totalSupply

	extraBytes, err := json.Marshal(poolExtra)
	if err != nil {
		return p, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":       p.Address,
				"total_supply": totalSupply.String(),
				"duration_ms":  time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	p.Extra = string(extraBytes)
	p.Reserves = []string{totalSupply.String(), totalSupply.String()}
	p.Timestamp = time.Now().Unix()

	return p, nil
}
