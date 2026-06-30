package lazy

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	eclp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/eclp"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
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
	var staticExtra shared.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, nil, err
	}
	req, applyResult := t.lazycall(ctx, &p, staticExtra, overrides)
	return req, applyResult, nil
}

func (t *PoolTracker) lazycall(
	ctx context.Context,
	p *entity.Pool,
	staticExtra shared.StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error)) {
	var (
		rpcRes               eclp.RpcResult
		isVaultPaused        bool
		isPoolPaused         bool
		isPoolInRecoveryMode bool
	)
	r := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides).SetFrom(shared.AddrDummy)
	req := poolpkg.LazyRequest{Request: r}
	addFn := func(c *ethrpc.Call, output []any) { req.AddCall(c, output) }
	addRPCCalls(addFn, t.config.VaultExplorer, p.Address, &rpcRes, &isVaultPaused, &isPoolPaused, &isPoolInRecoveryMode)
	rpcRes.Buffers = shared.GetBufferTokens(addFn, t.config.ChainID, t.config.DexID, staticExtra.BufferTokens)

	return &req, func(blockNumber *big.Int) (entity.Pool, error) {
		rpcRes.IsPoolDisabled = isVaultPaused || isPoolPaused || isPoolInRecoveryMode
		if blockNumber != nil {
			rpcRes.BlockNumber = blockNumber.Uint64()
		} else {
			rpcRes.BlockNumber = p.BlockNumber
		}
		return buildPoolState(p, &rpcRes, staticExtra)
	}
}
