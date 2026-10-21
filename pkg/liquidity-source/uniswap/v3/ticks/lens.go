package ticks

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

// HasState reports whether a tracker has given the pool its state. Only a tracker writes its
// liquidity, 0 included; a pool factory may fill in the price and tick from the creation event, but
// never the liquidity, and never the ticks.
func HasState(p entity.Pool) bool {
	var extra struct {
		Liquidity json.RawMessage `json:"liquidity"`
	}
	return len(p.Extra) > 0 && json.Unmarshal([]byte(p.Extra), &extra) == nil &&
		len(extra.Liquidity) > 0 && string(extra.Liquidity) != "null"
}

// BatchHead is the newest block of a batch, nil when it carries none.
func BatchHead(headers map[uint64]entity.BlockHeader) *big.Int {
	if len(headers) == 0 {
		return nil
	}
	return new(big.Int).SetUint64(lo.Max(lo.Keys(headers)))
}

// LensBootstrap reads a pool's state whole through the lens at block (nil for the latest), and
// returns it with the block the lens read it at.
type LensBootstrap func(ctx context.Context, block *big.Int) (entity.Pool, uint64, error)

// StateFromLens gives a pool that has no state yet, such as one a backfill created from its
// creation log, its whole state from the lens at the newest block of the batch. Every log in the
// batch is at or below that block, so they are not applied: the snapshot already holds them.
//
// A node behind the batch does not have that block yet, and a batch without headers names none. The
// node's own newest block then does instead, if it holds every log the pool has seen, and update
// applies the batch's later logs on top. When the guard declines the read, or it fails, the pool is
// left without state, recording the newest of its logs a snapshot must hold, and its next update
// tries again; the batch never waits for it.
func StateFromLens(ctx context.Context, guard *ephemeral.Guard, p entity.Pool, param pool.GetNewPoolStateParams,
	bootstrap LensBootstrap, update func(ctx context.Context, p entity.Pool) (entity.Pool, error),
	l logger.Logger) entity.Pool {
	head, due := BatchHead(param.BlockHeaders), lensDueOf(p, param.Logs)
	var newPool entity.Pool
	ran, err := guard.Try(ctx, p.Address, func(ctx context.Context) error {
		var err error
		if head != nil {
			if newPool, _, err = bootstrap(ctx, head); err == nil {
				return nil
			}
		}
		var read uint64
		if newPool, read, err = bootstrap(ctx, nil); err != nil {
			return err
		}
		if read < due {
			return fmt.Errorf("node at block %d, behind block %d of the pool's logs", read, due)
		}
		newPool, err = update(ctx, newPool)
		return err
	})
	if !ran || err != nil {
		l.WithFields(logger.Fields{"ran": ran, "error": err}).
			Warn("pool without state not read through the lens; leaving it for its next update")
		return WithLensDue(p, param.Logs)
	}
	return newPool
}
