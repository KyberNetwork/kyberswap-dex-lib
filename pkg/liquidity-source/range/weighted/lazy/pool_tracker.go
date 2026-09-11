package lazy

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	rangeweighted "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/range/weighted"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

// PoolTracker is the primary, batchable tracker for Range Pools: LazyNewPoolState plans
// the same three RPC calls as rangeweighted.PoolTracker (the backup) via
// rangeweighted.AddRPCCalls/BuildPoolState, but into a poolpkg.LazyRequest instead of
// executing them immediately, so pool-service's worker can merge them with other pools'
// calls into one multicall. GetNewPoolState runs the same plan eagerly for callers that
// don't batch. Mirrors balancer/v3/weighted/lazy.
type PoolTracker struct {
	config       *rangeweighted.Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(rangeweighted.DexType, NewPoolTracker)
var _ poolpkg.IBatchRPCPoolTracker = (*PoolTracker)(nil)

func NewPoolTracker(config *rangeweighted.Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := log.Ctx(ctx).With().Str("dex", rangeweighted.DexType).Str("pool", p.Address).Logger()
	l.Info().Msg("Started getting new pool state")
	start := time.Now()

	var staticExtra rangeweighted.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var (
		dyn      rangeweighted.RangePoolDynamicDataResult
		cfg      rangeweighted.PoolConfigResult
		minTrade *big.Int
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	rangeweighted.AddRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) },
		rangeweighted.VaultAddress(t.config.ChainID), p.Address, &dyn, &cfg, &minTrade)

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	if resp.BlockNumber == nil || len(resp.Result) < 3 {
		return p, rangeweighted.ErrIncompleteState
	}
	for _, ok := range resp.Result {
		if !ok {
			return p, rangeweighted.ErrIncompleteState
		}
	}

	blockNumber := resp.BlockNumber.Uint64()
	if p.BlockNumber > blockNumber {
		l.Info().
			Uint64("poolBlockNumber", p.BlockNumber).
			Uint64("dataBlockNumber", blockNumber).
			Msg("skip update: data block number is less than current pool block number")
		return p, nil
	}

	newP, err := rangeweighted.BuildPoolState(p, &staticExtra, &dyn.Data, &cfg.PoolConfig, minTrade, blockNumber)
	if err != nil {
		return p, err
	}

	l.Info().
		Uint64("block", blockNumber).
		Int64("durationMs", time.Since(start).Milliseconds()).
		Msg("Finished getting new pool state")

	return newP, nil
}

func (t *PoolTracker) LazyNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error), error) {
	return t.lazyNewPoolState(ctx, &p)
}

func (t *PoolTracker) LazyNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateWithOverridesParams,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error), error) {
	return t.lazyNewPoolState(ctx, &p)
}

func (t *PoolTracker) lazyNewPoolState(
	ctx context.Context,
	p *entity.Pool,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error), error) {
	var staticExtra rangeweighted.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, nil, err
	}

	var (
		dyn      rangeweighted.RangePoolDynamicDataResult
		cfg      rangeweighted.PoolConfigResult
		minTrade *big.Int
	)

	r := t.ethrpcClient.R().SetContext(ctx)
	req := poolpkg.LazyRequest{Request: r}
	rangeweighted.AddRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) },
		rangeweighted.VaultAddress(t.config.ChainID), p.Address, &dyn, &cfg, &minTrade)

	return &req, func(blockNumber *big.Int) (entity.Pool, error) {
		bn := p.BlockNumber
		if blockNumber != nil {
			bn = blockNumber.Uint64()
		}
		return rangeweighted.BuildPoolState(*p, &staticExtra, &dyn.Data, &cfg.PoolConfig, minTrade, bn)
	}, nil
}
