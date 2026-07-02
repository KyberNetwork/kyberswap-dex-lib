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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	plain "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
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
	var staticExtra plain.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, nil, err
	}
	req, applyResult := t.lazycall(ctx, &p, staticExtra, overrides)
	return req, applyResult, nil
}

func (t *PoolTracker) lazycall(
	ctx context.Context,
	p *entity.Pool,
	staticExtra plain.StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error)) {
	lg := t.logger.WithFields(logger.Fields{"poolAddress": p.Address})
	mainRegistryAddress := t.mainRegistryAddress()
	numTokens := len(p.Tokens)
	d := &rpcData{
		balances:   make([]*big.Int, numTokens),
		balancesV1: make([]*big.Int, numTokens),
	}

	r := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).
		SetFrom(shared.AddrDummy) // poolMethodStoredRates behaves differently for tx.origin == 0
	req := poolpkg.LazyRequest{Request: r}
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) },
		p.Address, numTokens, staticExtra.LpToken, staticExtra.Oracle, mainRegistryAddress, d)

	return &req, func(blockNumber *big.Int) (entity.Pool, error) {
		if blockNumber != nil {
			p.BlockNumber = blockNumber.Uint64()
		}
		return buildPoolState(lg, *p, d)
	}
}
