package getroute

import (
	"context"
	"sync"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// chargeExtraFee is a decorator for aggregator which handle charging extra fee logic
type chargeExtraFee struct {
	aggregator IAggregator

	mu sync.RWMutex
}

func NewChargeExtraFee(
	aggregator IAggregator,
) *chargeExtraFee {
	return &chargeExtraFee{
		aggregator: aggregator,
	}
}

func (c *chargeExtraFee) Aggregate(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummary, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[getroutev2] chargeExtraFee.Aggregate")
	defer span.Finish()

	if params.ExtraFee.IsChargeFeeByCurrencyIn() {
		return c.chargeFeeByCurrencyIn(ctx, params)
	}

	if params.ExtraFee.IsChargeFeeByCurrencyOut() {
		return c.chargeFeeByCurrencyOut(ctx, params)
	}

	return c.aggregator.Aggregate(ctx, params)
}

func (c *chargeExtraFee) ApplyConfig(config Config) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.aggregator.ApplyConfig(config)
}

func (c *chargeExtraFee) chargeFeeByCurrencyIn(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummary, error) {
	// Step 1: calculate amountIn after fee
	amountIn := params.AmountIn
	amountInAfterFee := business.CalcAmountInAfterFee(amountIn, params.ExtraFee)

	// Step 2: update amountIn after charged fee
	params.AmountIn = amountInAfterFee

	// Step 3: aggregate
	routeSummary, err := c.aggregator.Aggregate(ctx, params)
	if err != nil {
		return nil, err
	}

	// Step 4: update route summary with amountIn before fee
	amountInUSDBigFloat := business.CalcAmountUSD(amountIn, params.TokenIn.Decimals, params.TokenInPriceUSD)
	amountInUSD, _ := amountInUSDBigFloat.Float64()

	routeSummary.AmountIn = amountIn
	routeSummary.AmountInUSD = amountInUSD

	return routeSummary, nil
}

func (c *chargeExtraFee) chargeFeeByCurrencyOut(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummary, error) {
	// Step 1: aggregate
	routeSummary, err := c.aggregator.Aggregate(ctx, params)
	if err != nil {
		return nil, err
	}

	// Step 2: calculate amountOut after fee
	amountOutAfterFee := business.CalcAmountInAfterFee(routeSummary.AmountOut, params.ExtraFee)

	// Step 3: update route summary with amountOut after fee
	amountOutAfterFeeUSDBigFloat := business.CalcAmountUSD(amountOutAfterFee, params.TokenOut.Decimals, params.TokenOutPriceUSD)
	amountOutAfterFeeUSD, _ := amountOutAfterFeeUSDBigFloat.Float64()

	routeSummary.AmountOut = amountOutAfterFee
	routeSummary.AmountOutUSD = amountOutAfterFeeUSD

	return routeSummary, nil
}
