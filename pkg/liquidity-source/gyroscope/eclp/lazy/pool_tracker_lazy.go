package lazy

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	gyroeclp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/eclp"
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
	var staticExtra gyroeclp.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, nil, err
	}

	req, applyResult := t.lazycall(ctx, &p, &staticExtra, overrides)
	return req, applyResult, nil
}

func (t *PoolTracker) lazycall(
	ctx context.Context,
	p *entity.Pool,
	staticExtra *gyroeclp.StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount,
) (poolpkg.ILazyRequest, func(*big.Int) (entity.Pool, error)) {
	d := &gyroeclp.RPCResp{}

	r := t.ethrpcClient.R().SetContext(ctx).SetRequireSuccess(true).SetOverrides(overrides)
	req := poolpkg.LazyRequest{Request: r}
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) }, p.Address, staticExtra.Vault, staticExtra.PoolID, staticExtra.PoolTypeVer, d)

	return &req, func(blockNumber *big.Int) (entity.Pool, error) {
		if blockNumber != nil {
			d.BlockNumber = blockNumber.Uint64()
		}
		return buildPoolState(*p, staticExtra, d)
	}
}
