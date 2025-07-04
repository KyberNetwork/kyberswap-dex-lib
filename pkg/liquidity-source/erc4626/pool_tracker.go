package erc4626

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
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
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

	return &PoolTracker{
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
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	lg := t.logger.WithFields(logger.Fields{
		"address": p.Address,
	})
	lg.Info("Start updating state.")
	defer func() {
		lg.Info("Finish updating state.")
	}()

	vaultAddr := p.Tokens[0].Address
	vaultCfg := t.cfg.Vaults[vaultAddr]
	_, state, err := fetchAssetAndState(ctx, t.ethrpcClient, vaultAddr, vaultCfg, false, overrides)
	if err != nil {
		lg.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to fetch state")

		return p, err
	}

	extraBytes, err := json.Marshal(Extra{
		Gas:         Gas(vaultCfg.Gas),
		SwapTypes:   vaultCfg.SwapTypes,
		MaxDeposit:  uint256.MustFromBig(state.MaxDeposit),
		MaxRedeem:   uint256.MustFromBig(state.MaxRedeem),
		EntryFeeBps: state.EntryFeeBps,
		ExitFeeBps:  state.ExitFeeBps,
	})
	if err != nil {
		lg.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to marshal extra")
		return p, err
	}

	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{state.TotalSupply.String(), state.TotalAssets.String()}
	p.Extra = string(extraBytes)
	p.BlockNumber = state.blockNumber

	return p, nil
}
