package lazy

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	erc4626 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func (t *PoolTracker) LazyNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateParams,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error), error) {
	return t.lazyNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) LazyNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error), error) {
	return t.lazyNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) lazyNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error), error) {
	lg := t.logger.WithFields(logger.Fields{
		"address": p.Address,
	})
	lg.Info("Start updating state.")
	defer func() {
		lg.Info("Finish updating state.")
	}()

	vaultAddr := p.Tokens[0].Address
	vaultCfg, ok := t.cfg.Vaults[vaultAddr]
	if !ok { // manually added vault
		var extra erc4626.Extra
		_ = json.Unmarshal([]byte(p.Extra), &extra)
		vaultCfg.Gas = erc4626.GasCfg(extra.Gas)
	}
	// standard ERC4626: vault == share, so tokenAddr is empty and totalSupply targets the vault
	vaultCfg.Balance = erc4626.LoadBalanceSlot(&p)
	req, applyResult := Lazycall(ctx, &p, t.ethrpcClient, vaultAddr, "", vaultCfg, false, overrides)

	return req, applyResult, nil
}

// Lazycall builds the batch-RPC request and applyResult closure for a vault. vaultAddr is the entrypoint all
// logic getters target; tokenAddr is the share ERC20 whose totalSupply is read (empty => vault, i.e. standard
// ERC4626). Exported so decoupled-share integrations (e.g. erc7575) can reuse it with a share token != vault.
func Lazycall(
	ctx context.Context,
	pool *entity.Pool,
	ethrpcClient *ethrpc.Client,
	vaultAddr, tokenAddr string,
	vaultCfg erc4626.VaultCfg,
	fetchAsset bool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error)) {
	var assetToken common.Address
	poolState := erc4626.PoolState{
		DepositRates: make([]*big.Int, len(erc4626.PrefetchAmounts)),
		RedeemRates:  make([]*big.Int, len(erc4626.PrefetchAmounts)),
	}
	vaultCfg = decoupledShareCfg(vaultCfg, tokenAddr)
	// The probe overrides ride on the request; the batch consumer applies them (ILazyRequestWithOverrides).
	r := ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(vaultCfg.WithBalanceProbe(vaultAddr, overrides))
	req := poolpkg.LazyRequest{Request: r}
	add := func(c *ethrpc.Call, output []any) { req.AddCall(c, output) }
	addStateCalls(add, vaultAddr, tokenAddr, vaultCfg, fetchAsset, &assetToken, &poolState)
	erc4626.AddBalanceProbeCalls(add, vaultAddr, vaultCfg, &poolState)

	return &req, func(blockNumber *big.Int) (entity.Pool, error) {
		if err := poolState.ResolveLimits(vaultCfg); err != nil {
			return *pool, err
		}
		normalizePoolState(vaultCfg, &poolState) // resolves the probe read-back too
		if poolState.NeedsCleanRead() {          // markers may alias storage: read again without them, once per vault
			_, state, err := cleanReread(ctx, ethrpcClient, vaultAddr, tokenAddr, vaultCfg, fetchAsset, overrides, poolState.Slot)
			if err != nil {
				return *pool, err
			}
			poolState = *state // keeps the re-read block
		} else if blockNumber != nil {
			poolState.BlockNumber = blockNumber.Uint64()
		} else {
			poolState.BlockNumber = pool.BlockNumber
		}
		return *pool, UpdateEntityState(pool, vaultCfg, &poolState)
	}
}
