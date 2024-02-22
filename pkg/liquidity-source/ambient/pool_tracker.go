package ambient

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/logger"
)

type PoolTracker struct {
	cfg          Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	cfg Config,
) *PoolListUpdater {
	return &PoolListUpdater{
		cfg:      cfg,
		subgraph: graphqlPkg.NewWithTimeout(cfg.SubgraphURL, cfg.SubgraphRequestTimeout.Duration),
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")
	defer func() {
		logger.
			WithFields(
				logger.Fields{
					"pool_id":     p.Address,
					"duration_ms": time.Since(startTime).Milliseconds(),
				},
			).
			Info("Finished getting new pool state")
	}()

	return p, nil
}
