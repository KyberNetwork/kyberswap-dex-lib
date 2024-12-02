package pool

import (
	"context"
	"strconv"
	"time"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	"github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"

	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
)

type customFuncs struct {
	entity.ICustomFuncs
	dexUseAEVM map[string]bool
}

func NewCustomFuncs(dexUseAEVM map[string]bool) *customFuncs {
	return &customFuncs{
		ICustomFuncs: common.DefaultCustomFuncs,
		dexUseAEVM:   dexUseAEVM,
	}
}

func (c *customFuncs) CalcAmountOut(ctx context.Context, pool poolpkg.IPoolSimulator, tokenAmountIn poolpkg.TokenAmount, tokenOut string, limit poolpkg.SwapLimit) (*poolpkg.CalcAmountOutResult, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "CalcAmountOut")
	defer span.End()
	dexUseAEVM, exchange := c.dexUseAEVM[pool.GetType()], pool.GetExchange()
	span.SetTag("dexUseAEVM", strconv.FormatBool(dexUseAEVM))
	span.SetTag("dex", exchange)
	defer func(now time.Time) {
		metrics.RecordCalcAmountOutDuration(ctx, time.Since(now), dexUseAEVM, exchange)
	}(time.Now())

	if c := ctx.Value(metrics.CalcAmountOutCounterContextKey); c != nil {
		c.(*metrics.CalcAmountOutCounter).Inc(pool.GetExchange(), 1)
	}

	return c.ICustomFuncs.CalcAmountOut(ctx, pool, tokenAmountIn, tokenOut, limit)
}

func (c *customFuncs) ClonePool(ctx context.Context, pool poolpkg.IPoolSimulator) poolpkg.IPoolSimulator {
	span, ctx := tracer.StartSpanFromContext(ctx, "ClonePool")
	defer span.End()
	exchange := pool.GetExchange()
	span.SetTag("dex", exchange)
	defer func(now time.Time) {
		metrics.RecordClonePoolDuration(ctx, time.Since(now), exchange)
	}(time.Now())

	return c.ICustomFuncs.ClonePool(ctx, pool)
}

func (c *customFuncs) CloneSwapLimit(ctx context.Context, limit poolpkg.SwapLimit) poolpkg.SwapLimit {
	span, ctx := tracer.StartSpanFromContext(ctx, "CloneSwapLimit")
	defer span.End()
	span.SetTag("dex", limit.GetExchange())

	return c.ICustomFuncs.CloneSwapLimit(ctx, limit)
}

// CalcAmountOut wrapper of (github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool).CalcAmountOut
// Deprecated: use pathfinder-lib
func CalcAmountOut(ctx context.Context, pool poolpkg.IPoolSimulator, tokenAmountIn poolpkg.TokenAmount, tokenOut string, limit poolpkg.SwapLimit, dexUseAEVM map[string]bool) (res *poolpkg.CalcAmountOutResult, err error) {
	span, _ := tracer.StartSpanFromContext(ctx, "CalcAmountOut")
	defer span.End()

	if dexUseAEVM[pool.GetType()] {
		span.SetTag("dexUseAEVM", "true")
	} else {
		span.SetTag("dexUseAEVM", "false")
	}
	span.SetTag("dex", pool.GetType())

	if c := ctx.Value(metrics.CalcAmountOutCounterContextKey); c != nil {
		c.(*metrics.CalcAmountOutCounter).Inc(pool.GetType(), 1)
	}

	return poolpkg.CalcAmountOut(pool, tokenAmountIn, tokenOut, limit)
}
