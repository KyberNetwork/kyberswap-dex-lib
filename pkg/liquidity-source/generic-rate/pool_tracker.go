package generic_rate

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config              *Config
	ethrpcClient        *ethrpc.Client
	rateTrackerRegistry map[valueobject.Exchange]IRateTracker
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		rateTrackerRegistry: map[valueobject.Exchange]IRateTracker{
			valueobject.ExchangeSkyPSM: &SkyPSMRateTracker{ethrpcClient: ethrpcClient},
		},
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
	_ map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	startTime := time.Now()
	logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address}).Info("Start getting new pool state")

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	blockTimestamp, err := req.GetCurrentBlockTimestamp()
	if err != nil {
		return p, err
	}

	swapFuncArgs, swapFuncData, err := t.getSwapFuncData(ctx, &p, blockTimestamp)
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(
		Extra{
			BlockTimestamp: blockTimestamp,
			SwapFuncArgs:   swapFuncArgs,
			SwapFuncByPair: swapFuncData,
		})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"dex_id":      t.config.DexID,
		"pool_id":     p.Address,
		"type":        p.Type,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Finished getting new pool state")

	return p, nil
}

func (t *PoolTracker) getSwapFuncData(
	ctx context.Context,
	p *entity.Pool,
	blockTimestamp uint64,
) ([]*uint256.Int, map[int]map[int]SwapFunc, error) {
	rateTracker, ok := t.rateTrackerRegistry[valueobject.Exchange(t.config.DexID)]
	if !ok {
		return nil, nil, nil
	}

	return rateTracker.GetSwapData(ctx, p, blockTimestamp)
}
