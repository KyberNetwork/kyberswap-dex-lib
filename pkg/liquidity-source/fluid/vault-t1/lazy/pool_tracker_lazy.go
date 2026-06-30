package lazy

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	vaultT1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/vault-t1"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ poolpkg.IBatchRPCPoolTracker = (*PoolTracker)(nil)

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
	req, applyResult := t.lazycall(ctx, &p, overrides)
	return req, applyResult, nil
}

func (t *PoolTracker) lazycall(
	ctx context.Context,
	p *entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error)) {
	d := &rpcData{output: &vaultT1.Swap{}}

	r := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides)
	req := poolpkg.LazyRequest{Request: r}
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) }, p.Address, t.config.VaultLiquidationResolver, d)

	return &req, func(blockNumber *big.Int) (entity.Pool, error) {
		if d.output == nil {
			logger.WithFields(logger.Fields{"dexType": vaultT1.DexType, "pool": p.Address}).Error("nil output from GetSwapForProtocol")
			return *p, nil
		}
		return buildPoolState(*p, d, blockNumber)
	}
}
