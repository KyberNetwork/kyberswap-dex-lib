package lunarbase

import (
	"context"
	"fmt"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	if config.DexID == "" {
		config.DexID = DexType
	}
	if config.ChainID == 0 {
		config.ChainID = valueobject.ChainIDBase
	}
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}
}

// GetNewPoolState publishes only complete, canonical RPC snapshots. Event
// batches do not prove that no fee, pause, reserve, or admin update is missing.
// Their block numbers are freshness requirements, never state to replay on top
// of a snapshot. Duplicates, ordering, and removed logs cannot corrupt it.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	params pool.GetNewPoolStateParams) (entity.Pool, error) {
	minimum := p.BlockNumber
	address := common.HexToAddress(p.Address)
	for _, lg := range params.Logs {
		if lg.Address == address && !lg.Removed && lg.BlockNumber > minimum {
			minimum = lg.BlockNumber
		}
	}
	for number := range params.BlockHeaders {
		if number > minimum {
			minimum = number
		}
	}
	return t.refresh(ctx, p, nil, minimum)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(ctx context.Context, p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams) (entity.Pool, error) {
	minimum := p.BlockNumber
	address := common.HexToAddress(p.Address)
	for _, lg := range params.Logs {
		if lg.Address == address && !lg.Removed && lg.BlockNumber > minimum {
			minimum = lg.BlockNumber
		}
	}
	return t.refresh(ctx, p, params.Overrides, minimum)
}

func (t *PoolTracker) getNewPoolState(ctx context.Context, p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount) (entity.Pool, error) {
	return t.refresh(ctx, p, overrides, p.BlockNumber)
}

func (t *PoolTracker) refresh(ctx context.Context, p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount, minimum uint64) (entity.Pool, error) {
	state, err := fetchRPCState(ctx, p.Address, t.config.ChainID, t.ethrpcClient, overrides)
	if err != nil {
		return p, err
	}
	if state.blockNumber < minimum {
		return p, fmt.Errorf("%w: got %d, need at least %d", ErrSnapshotBehind, state.blockNumber, minimum)
	}
	updated := p
	if _, err = buildEntityPool(&updated, state); err != nil {
		return p, err
	}
	return updated, nil
}
