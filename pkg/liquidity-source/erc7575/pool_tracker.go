package erc7575

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	erc4626lazy "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626/lazy"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client

	logger logger.Logger
}

var (
	_ poolpkg.IBatchRPCPoolTracker = (*PoolTracker)(nil)
	_                              = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)
)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
		logger:       logger.WithFields(logger.Fields{"dexId": cfg.DexId, "dexType": DexType}),
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	vaultAddr, shareToken, vaultCfg := t.resolve(&p)
	// totalSupply is read from the share token (Tokens[0]); all logic getters target the vault entrypoint
	_, state, err := erc4626lazy.FetchAssetAndState(ctx, t.ethrpcClient, vaultAddr, shareToken, vaultCfg, false, overrides)
	if err != nil {
		t.logger.WithFields(logger.Fields{"address": p.Address, "error": err}).Error("failed to fetch state")
		return p, err
	}

	return p, erc4626.UpdateEntityState(&p, vaultCfg, state)
}

// resolve returns the vault entrypoint (call target, from pool Address), the decoupled share token (from
// Tokens[0], where totalSupply is read) and the vault config (with manual-add fallback via Extra).
func (t *PoolTracker) resolve(p *entity.Pool) (vaultAddr, shareToken string, vaultCfg erc4626.VaultCfg) {
	vaultAddr = p.Address
	shareToken = p.Tokens[0].Address
	vaultCfg, ok := t.cfg.Vaults[vaultAddr]
	if !ok { // manually added vault
		var extra erc4626.Extra
		_ = json.Unmarshal([]byte(p.Extra), &extra)
		vaultCfg.Gas = erc4626.GasCfg(extra.Gas)
	}
	return vaultAddr, shareToken, vaultCfg
}
