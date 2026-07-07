package erc4626

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
		var extra Extra
		_ = json.Unmarshal([]byte(p.Extra), &extra)
		vaultCfg.Gas = GasCfg(extra.Gas)
	}
	req, applyResult := lazycall(ctx, &p, t.ethrpcClient, vaultAddr, vaultCfg, false, overrides)

	return req, applyResult, nil
}

func lazycall(
	ctx context.Context,
	pool *entity.Pool,
	ethrpcClient *ethrpc.Client,
	vaultAddr string,
	vaultCfg VaultCfg,
	fetchAsset bool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error)) {
	var (
		assetToken common.Address
		poolState  = PoolState{
			DepositRates: make([]*big.Int, len(PrefetchAmounts)),
			RedeemRates:  make([]*big.Int, len(PrefetchAmounts)),
		}
	)
	r := ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	req := poolpkg.LazyRequest{Request: r}
	if fetchAsset {
		req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodAsset,
		}, []any{&assetToken})
	}

	if vaultCfg.Gas.Deposit > 0 {
		req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodMaxDeposit,
			Params: []any{AddrDummy},
		}, []any{&poolState.MaxDeposit}).AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodTotalAssets,
		}, []any{&poolState.TotalAssets})

		for i, amt := range PrefetchAmounts {
			req.AddCall(&ethrpc.Call{
				ABI:    ABI,
				Target: vaultAddr,
				Method: ERC4626MethodPreviewDeposit,
				Params: []any{amt.ToBig()},
			}, []any{&poolState.DepositRates[i]})
		}
	}
	if vaultCfg.Gas.Redeem > 0 {
		req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodMaxRedeem,
			Params: []any{AddrDummy},
		}, []any{&poolState.MaxRedeem}).AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodTotalSupply,
		}, []any{&poolState.TotalSupply})

		for i, amt := range PrefetchAmounts {
			req.AddCall(&ethrpc.Call{
				ABI:    ABI,
				Target: vaultAddr,
				Method: ERC4626MethodPreviewRedeem,
				Params: []any{amt.ToBig()},
			}, []any{&poolState.RedeemRates[i]})
		}
	}
	return &req, func(blockNumber *big.Int) (entity.Pool, error) {
		if poolState.MaxDeposit == nil || poolState.MaxDeposit.Sign() == 0 {
			poolState.MaxDeposit = poolState.TotalAssets // fallback to a sensible value
		} else if poolState.MaxDeposit.Cmp(bignumber.MaxUint128) > 0 {
			poolState.MaxDeposit = nil // no limit
		}
		if poolState.MaxRedeem == nil || poolState.MaxRedeem.Sign() == 0 {
			poolState.MaxRedeem = poolState.TotalSupply // fallback to a sensible value
		} else if poolState.MaxRedeem.Cmp(bignumber.MaxUint128) > 0 {
			poolState.MaxRedeem = nil // no limit
		}

		if blockNumber != nil {
			poolState.BlockNumber = blockNumber.Uint64()
		} else {
			poolState.BlockNumber = pool.BlockNumber
		}
		return *pool, UpdateEntityState(pool, vaultCfg, &poolState)
	}
}
