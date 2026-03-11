package lunarbase

import (
	"context"
	"strings"
	"time"

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

	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Overrides)
}

func (t *PoolTracker) GetDependencies(_ context.Context, p entity.Pool) ([]string, bool, error) {
	return []string{strings.ToLower(p.Address)}, true, nil
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	state, err := fetchRPCState(ctx, t.config, t.ethrpcClient, overrides)
	if err != nil {
		return p, err
	}

	p.Reserves = entity.PoolReserves{
		state.reserveX.String(),
		state.reserveY.String(),
	}
	p.BlockNumber = state.blockNumber
	p.Timestamp = time.Now().Unix()

	updatedPool, err := buildEntityPool(t.config, state)
	if err != nil {
		return p, err
	}

	p.Extra = updatedPool.Extra
	p.StaticExtra = updatedPool.StaticExtra
	p.Tokens = updatedPool.Tokens

	return p, nil
}
