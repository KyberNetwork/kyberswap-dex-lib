package v2

import (
	"context"
	"errors"
	"fmt"
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
// A pool the lens cannot serve, because its rounds disagree with the tracker's, the lens cannot
// read it even alone, or the whole call failed, is read the usual way instead, at the lens's block
// when there is one. Without UseLens every pool is read the usual way.
func (d *PoolTracker) GetNewPoolStates(ctx context.Context, pools []entity.Pool) ([]entity.Pool, []error) {
	return d.getNewPoolStates(ctx, pools, nil)
}

func (d *PoolTracker) getNewPoolStates(ctx context.Context, pools []entity.Pool,
	block *big.Int) ([]entity.Pool, []error) {
	out := make([]entity.Pool, len(pools))
	errs := make([]error, len(pools))
	if !d.config.UseLens {
		for i, p := range pools {
			out[i], errs[i] = d.readOne(ctx, p, block)
		}
		return out, errs
	}

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

	// The block the lens read at; none when no pool's read ran, each too large alone, and those are
	// read the usual way at the block asked.
	pinned := block
	if blockNumber != 0 {
		pinned = new(big.Int).SetUint64(blockNumber)
	}
	for k, pl := range plans {
		p := pools[pl.idx]
		newPool, err := d.fromLens(ctx, p, pl.staticExtra, pl.vaults, pinned, reads[k])
		if errors.Is(err, errLensUnusable) {
			logger.WithFields(logger.Fields{"poolAddress": p.Address, "error": err}).
				Warn("euler-swap-v2: lens reads unusable for pool, reading it alone")
			newPool, err = d.readOne(ctx, p, pinned)
		}
		if err != nil {
			newPool = p
		}
		out[pl.idx], errs[pl.idx] = newPool, err
	}

	return out, errs
}

// errLensUnusable means the lens's reads cannot give a pool its state, and it is to be read alone.
var errLensUnusable = errors.New("euler-swap-v2: lens reads unusable")

// fromLens gives p its state from what the lens read of it at block.
func (d *PoolTracker) fromLens(ctx context.Context, p entity.Pool, staticExtra StaticExtra,
	vaults []shared.VaultInfo, block *big.Int, read lensRead) (entity.Pool, error) {
	if read.err != nil {
		return p, fmt.Errorf("%w: %w", errLensUnusable, read.err)
	}
	replay := newLensReplay(block, read.reads)
	data, bn, err := d.getPoolData(ctx, p.Address, staticExtra.EulerAccount, staticExtra.EVC, vaults, nil, nil,
		replay.exec)
	if err == nil {
		err = replay.consumed()
	}
	if err != nil {
		return p, fmt.Errorf("%w: %w", errLensUnusable, err)
	}
	return d.updatePool(ctx, p, data, bn)
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
