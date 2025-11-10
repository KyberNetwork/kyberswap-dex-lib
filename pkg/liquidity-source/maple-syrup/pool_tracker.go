package maplesyrup

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

type PoolTracker struct {
	erc4626.PoolTracker
	cfg          *Config
	ethrpcClient *ethrpc.Client

	logger logger.Logger
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	lg := logger.WithFields(logger.Fields{
		"dexId":   cfg.DexId,
		"dexType": DexType,
	})

	erc4626Tracker := erc4626.NewPoolTracker(&erc4626.Config{
		DexId: cfg.DexId,
		Vaults: lo.MapValues(cfg.Vaults, func(vCfg VaultCfg, key string) erc4626.VaultCfg {
			return erc4626.VaultCfg{Gas: vCfg.Gas, SwapTypes: vCfg.SwapTypes}
		}),
	}, ethrpcClient)

	return &PoolTracker{
		PoolTracker:  *erc4626Tracker,
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
		logger:       lg,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	lg := t.logger.WithFields(logger.Fields{
		"address": p.Address,
	})
	lg.Info("Start updating state.")
	defer func() {
		lg.Info("Finish updating state.")
	}()

	p, err := t.PoolTracker.GetNewPoolState(ctx, p, params)
	if err != nil {
		return p, err
	}

	var (
		active       bool
		liquidityCap *big.Int
	)
	_, err = t.ethrpcClient.NewRequest().
		SetContext(ctx).
		SetBlockNumber(big.NewInt(int64(p.BlockNumber))).
		AddCall(&ethrpc.Call{
			ABI:    poolManagerABI,
			Target: t.cfg.Vaults[p.Address].PoolManager,
			Method: poolManagerMethodActive,
		}, []any{&active}).
		AddCall(&ethrpc.Call{
			ABI:    poolManagerABI,
			Target: t.cfg.Vaults[p.Address].PoolManager,
			Method: poolManagerMethodLiquidityCap,
		}, []any{&liquidityCap}).TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	var extra Extra
	if err = json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}
	extra.Active = active
	extra.LiquidityCap = uint256.MustFromBig(liquidityCap)
	extra.Router = t.cfg.Vaults[p.Address].Router

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)

	return p, nil
}
