package ringswap

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

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
	if err := validateTokens(p.Tokens); err != nil {
		return nil, nil, err
	}
	req, applyResult := d.lazycall(ctx, &p, overrides)
	return req, applyResult, nil
}

func (d *PoolTracker) lazycall(
	ctx context.Context,
	p *entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error)) {
	rpc := newRPCData()

	r := d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	req := poolpkg.LazyRequest{Request: r}
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) }, p.Address, p.Tokens, rpc)

	return &req, func(blockNumber *big.Int) (entity.Pool, error) {
		if blockNumber != nil && p.BlockNumber > blockNumber.Uint64() {
			logger.WithFields(logger.Fields{
				"pool_id":           p.Address,
				"pool_block_number": p.BlockNumber,
				"data_block_number": blockNumber.Uint64(),
			}).Info("skip update: data block number is less than current pool block number")
			return *p, nil
		}
		return buildPoolState(*p, rpc, blockNumber)
	}
}
