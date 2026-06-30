package aavev3

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ poolpkg.IBatchRPCPoolTracker = (*PoolTracker)(nil)

func (d *PoolTracker) LazyNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateParams,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error), error) {
	return d.lazyNewPoolState(ctx, p, params, nil)
}

func (d *PoolTracker) LazyNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error), error) {
	return d.lazyNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (d *PoolTracker) lazyNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error), error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, nil, err
	}

	req, applyResult := d.lazycall(ctx, &p, staticExtra.AavePoolAddress, overrides)
	return req, applyResult, nil
}

func (d *PoolTracker) lazycall(
	ctx context.Context,
	p *entity.Pool,
	poolAddress string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error)) {
	rd := newRPCData()

	r := d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	req := poolpkg.LazyRequest{Request: r}
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) },
		poolAddress, p.Tokens[0].Address, p.Tokens[1].Address, rd)

	return &req, func(blockNumber *big.Int) (entity.Pool, error) {
		return buildPoolState(*p, rd, blockNumber)
	}
}
