package frxusd

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	lg := logger.WithFields(logger.Fields{
		"address": p.Address,
	})
	lg.Info("start updating state.")
	defer func() {
		lg.Info("finish updating state.")
	}()

	vaultCfg, ok := t.cfg.Vaults[p.Address]
	if !ok {
		lg.Error("vault config not found")
		return p, nil
	}

	_, state, err := erc4626.FetchAssetAndState(ctx, t.ethrpcClient, p.Address, vaultCfg, false, nil)
	if err != nil {
		lg.WithFields(logger.Fields{"error": err}).Errorf("failed to fetch state")

		return p, err
	}

	return p, erc4626.UpdateEntityState(&p, vaultCfg, state)
}
