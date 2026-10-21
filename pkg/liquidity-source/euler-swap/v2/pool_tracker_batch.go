package v2

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ pool.IBatchPoolStateTracker = (*PoolTracker)(nil)

// GetNewPoolStates reads every pool through the state lens, in one call where GetNewPoolState
// takes up to four round trips per pool, then decodes each pool exactly as the single-pool path
// does: the lens records each call's raw result, and getPoolData replays them in place of its
// multicalls.
//
// A pool the lens cannot serve, because its rounds disagree with the tracker's or the whole call
// failed, is read the usual way instead, at the lens' block when there is one.
func (d *PoolTracker) GetNewPoolStates(ctx context.Context, pools []entity.Pool) ([]entity.Pool, []error) {
	return d.getNewPoolStates(ctx, pools, nil)
}

func (d *PoolTracker) getNewPoolStates(ctx context.Context, pools []entity.Pool,
	block *big.Int) ([]entity.Pool, []error) {
	out := make([]entity.Pool, len(pools))
	errs := make([]error, len(pools))

	type planned struct {
		idx         int
		staticExtra StaticExtra
		vaults      []shared.VaultInfo
	}
	var (
		plans  []planned
		inputs []poolInput
	)
	for i, p := range pools {
		staticExtra, vaults, err := poolVaults(p)
		if err != nil {
			out[i], errs[i] = p, err
			continue
		}
		plans = append(plans, planned{idx: i, staticExtra: staticExtra, vaults: vaults})
		inputs = append(inputs, lensInput(p.Address, staticExtra, vaults))
	}

	blockNumber, reads, err := d.readLens(ctx, inputs, block, 0)
	if err != nil {
		logger.WithFields(logger.Fields{"pools": len(inputs), "error": err}).
			Warn("euler-swap-v2: state lens failed, reading pools one by one")
		for _, pl := range plans {
			out[pl.idx], errs[pl.idx] = d.readOne(ctx, pools[pl.idx], block)
		}
		return out, errs
	}

	pinned := new(big.Int).SetUint64(blockNumber)
	for k, pl := range plans {
		p := pools[pl.idx]
		replay := newLensReplay(pinned, reads[k])

		data, bn, err := d.getPoolData(ctx, p.Address, pl.staticExtra.EulerAccount, pl.staticExtra.EVC, pl.vaults,
			nil, nil, replay.exec)
		if err == nil {
			err = replay.consumed()
		}
		if err != nil {
			logger.WithFields(logger.Fields{"poolAddress": p.Address, "error": err}).
				Warn("euler-swap-v2: lens reads unusable for pool, reading it alone")
			out[pl.idx], errs[pl.idx] = d.readOne(ctx, p, pinned)
			continue
		}

		newPool, err := d.updatePool(ctx, p, data, bn)
		if err != nil {
			out[pl.idx], errs[pl.idx] = p, err
			continue
		}
		out[pl.idx] = newPool
	}

	return out, errs
}

// readOne is the single-pool path, pinned to block when one is given.
func (d *PoolTracker) readOne(ctx context.Context, p entity.Pool, block *big.Int) (entity.Pool, error) {
	staticExtra, vaults, err := poolVaults(p)
	if err != nil {
		return p, err
	}
	data, bn, err := d.getPoolData(ctx, p.Address, staticExtra.EulerAccount, staticExtra.EVC, vaults, nil, block,
		rpcRoundExec)
	if err != nil {
		return p, err
	}
	newPool, err := d.updatePool(ctx, p, data, bn)
	if err != nil {
		return p, err
	}
	return newPool, nil
}
