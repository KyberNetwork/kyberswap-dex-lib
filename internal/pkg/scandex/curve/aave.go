package curve

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	curveAave "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/curve-aave"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

func (t *Curve) getAavePoolExtra(ctx context.Context, pool entity.Pool) (interface{}, error) {

	var calls []*repository.CallParams
	var initialA, futureA, initialATime, futureATime, swapFee, adminFee, offpegFee *big.Int
	callParamsFactory := repository.CallParamsFactory(abis.CurveAave, pool.Address)
	calls = append(calls,
		callParamsFactory("initial_A", &initialA, nil),
		callParamsFactory("initial_A_time", &initialATime, nil),
		callParamsFactory("future_A", &futureA, nil),
		callParamsFactory("future_A_time", &futureATime, nil),
		callParamsFactory("fee", &swapFee, nil),
		callParamsFactory("admin_fee", &adminFee, nil),
		callParamsFactory("offpeg_fee_multiplier", &offpegFee, nil),
	)
	if err := t.scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall getAavePoolExtra(%v), err: %v", pool.Address, err)
		return nil, err
	}
	extra := curveAave.Extra{
		InitialA:            initialA.String(),
		FutureA:             futureA.String(),
		InitialATime:        initialATime.Int64(),
		FutureATime:         futureATime.Int64(),
		SwapFee:             swapFee.String(),
		AdminFee:            adminFee.String(),
		OffpegFeeMultiplier: offpegFee.String(),
	}
	return extra, nil
}

func (t *Curve) getAavePoolReserves(ctx context.Context, pool entity.Pool) ([]*big.Int, error) {

	nTokens := len(pool.Tokens)
	reserves := make([]*big.Int, 2*nTokens)
	tryCalls := make([]*repository.TryCallParams, 0)
	tryCallParamsFactory := repository.TryCallParamsFactory(abis.CurveAave, pool.Address)
	for j := range pool.Tokens {
		tryCalls = append(tryCalls,
			tryCallParamsFactory("balances", &reserves[2*j], []interface{}{big.NewInt(int64(j))}),
			tryCallParamsFactory("balances", &reserves[2*j+1], []interface{}{big.NewInt(int64(j))}),
		)
	}
	if err := t.scanService.TryAggregateForce(ctx, false, tryCalls); err != nil {
		logger.Errorf("failed to process multicall for getAavePoolReserves(%v), err: %v", pool.Address, err)
		return nil, err
	}
	res := make([]*big.Int, 0, nTokens)
	for i := 0; i < nTokens; i++ {
		if reserves[2*i] != nil {
			res = append(res, reserves[2*i])
		} else if reserves[2*i+1] != nil {
			res = append(res, reserves[2*i+1])
		} else {
			return nil, ErrCanNotGetBalances
		}
	}

	return res, nil
}
