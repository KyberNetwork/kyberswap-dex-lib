package ekubo

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils/klog"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	dataFetcher  *dataFetchers
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		dataFetcher:  NewDataFetchers(ethrpcClient, config),
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	lg := klog.WithFields(ctx, klog.Fields{
		"dexId":       d.config.DexId,
		"poolAddress": p.Address,
	})
	defer func() {
		lg.Info("Finish updating state.")
	}()

	var err error

	var staticExtra StaticExtra
	if err = json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	ekuboPool, err := unmarshalPool([]byte(p.Extra), &staticExtra)
	if err != nil {
		return p, fmt.Errorf("unmarshalling extra: %w", err)
	}

	poolWithBlockNumber := &PoolWithBlockNumber{
		Pool: ekuboPool,
	}

	needRpcCall := len(params.Logs) == 0
	if !needRpcCall {
		if err := d.applyLogs(params.Logs, poolWithBlockNumber); err != nil {
			lg.Errorf("log application failed, falling back to RPC, error: %v", err)
			needRpcCall = true
		}
	}

	if needRpcCall {
		poolWithBlockNumber, err = d.forceUpdateState(ctx, staticExtra.PoolKey)
		if err != nil {
			return p, err
		}
	}

	extraBytes, err := json.Marshal(poolWithBlockNumber.GetState())
	if err != nil {
		return p, err
	}

	balances, err := poolWithBlockNumber.CalcBalances()
	if err != nil {
		return p, fmt.Errorf("calculating balances: %w", err)
	}

	p.Reserves = lo.Map(balances, func(v big.Int, _ int) string { return v.String() })
	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)
	p.BlockNumber = poolWithBlockNumber.blockNumber

	return p, nil
}

func (d *PoolTracker) applyLogs(logs []types.Log, pool *PoolWithBlockNumber) error {
	for _, log := range logs {
		if log.Removed {
			return ErrReorg
		}

		pool.blockNumber = log.BlockNumber

		var event pools.Event
		if d.config.Core.Cmp(log.Address) == 0 {
			if len(log.Topics) == 0 {
				event = pools.EventSwapped
			} else if log.Topics[0] == abis.PositionUpdatedEvent.ID {
				event = pools.EventPositionUpdated
			} else {
				continue
			}
		} else if d.config.Twamm.Cmp(log.Address) == 0 {
			if len(log.Topics) == 0 {
				event = pools.EventVirtualOrdersExecuted
			} else if log.Topics[0] == abis.OrderUpdatedEvent.ID {
				event = pools.EventOrderUpdated
			} else {
				continue
			}
		} else {
			continue
		}

		if err := pool.ApplyEvent(event, log.Data); err != nil {
			return fmt.Errorf("applying %v event: %w", event, err)
		}
	}

	pool.NewBlock()

	return nil
}

func (d *PoolTracker) forceUpdateState(ctx context.Context, poolKey *pools.PoolKey) (*PoolWithBlockNumber, error) {
	logger.WithFields(logger.Fields{
		"dexId":       d.config.DexId,
		"poolAddress": poolKey.StringId(),
	}).Info("update state from data fetcher")

	pools, err := d.dataFetcher.fetchPools(
		ctx,
		[]*pools.PoolKey{poolKey})
	if err != nil {
		return nil, fmt.Errorf("fetching pool state: %w", err)
	}

	return pools[0], nil
}
