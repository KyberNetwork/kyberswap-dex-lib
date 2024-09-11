package pool

import (
	"context"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
)

type calcAmountOut struct {
	dexUseAEVM map[string]bool
}

func NewCalcAmountOut(dexUseAEVM map[string]bool) *calcAmountOut {
	return &calcAmountOut{dexUseAEVM: dexUseAEVM}
}

func (c *calcAmountOut) CalcAmountOut(ctx context.Context, pool poolpkg.IPoolSimulator, tokenAmountIn poolpkg.TokenAmount, tokenOut string, limit poolpkg.SwapLimit) (*poolpkg.CalcAmountOutResult, error) {
	if c.dexUseAEVM[pool.GetType()] {
		span, _ := tracer.StartSpanFromContext(ctx, "CalcAmountOut")
		span.SetTag("calcAmountOutAEVM", "aevm")
		span.SetTag("dex", pool.GetType())
		defer span.End()
	}

	if c := ctx.Value(metrics.CalcAmountOutCounterContextKey); c != nil {
		c.(*metrics.CalcAmountOutCounter).Inc(pool.GetType(), 1)
	}

	return poolpkg.CalcAmountOut(pool, tokenAmountIn, tokenOut, limit)
}

// [Deprecated] CalcAmountOut wrapper of (github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool).CalcAmountOut
func CalcAmountOut(ctx context.Context, pool poolpkg.IPoolSimulator, tokenAmountIn poolpkg.TokenAmount, tokenOut string, limit poolpkg.SwapLimit, dexUseAEVM map[string]bool) (res *poolpkg.CalcAmountOutResult, err error) {
	if dexUseAEVM[pool.GetType()] {
		span, _ := tracer.StartSpanFromContext(ctx, "CalcAmountOut")
		span.SetTag("calcAmountOutAEVM", "aevm")
		span.SetTag("dex", pool.GetType())
		defer span.End()
	}
	if c := ctx.Value(metrics.CalcAmountOutCounterContextKey); c != nil {
		c.(*metrics.CalcAmountOutCounter).Inc(pool.GetType(), 1)
	}

	return poolpkg.CalcAmountOut(pool, tokenAmountIn, tokenOut, limit)
}
