package curve

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	curveTricrypto "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/curve-tricrypto"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

func (t *Curve) getTricryptoPoolData(ctx context.Context, pool entity.Pool) (interface{}, []*big.Int, error) {

	var (
		calls       = make([]*repository.CallParams, 0)
		nCoins      = len(pool.Tokens)
		priceScale  = make([]*big.Int, nCoins-1)
		priceOracle = make([]*big.Int, nCoins-1)
		lastPrices  = make([]*big.Int, nCoins-1)
		reserves    = make([]*big.Int, nCoins)
	)
	var (
		A, D, gamma, feeGamma, midFee, outFee, futureAGammaTime, futureAGamma, initialAGammaTime, initialAGamma *big.Int
		lastPriceTimestamp, lpSupply, xcpProfit, virtualPrice, allowedExtraProfit, adjustmentStep, maHalfTime   *big.Int
	)

	callParamsFactory := repository.CallParamsFactory(abis.CurveTricrypto, pool.Address)
	calls = append(calls,
		callParamsFactory("A", &A, nil),
		callParamsFactory("D", &D, nil),
		callParamsFactory("gamma", &gamma, nil),
		callParamsFactory("fee_gamma", &feeGamma, nil),
		callParamsFactory("mid_fee", &midFee, nil),
		callParamsFactory("out_fee", &outFee, nil),
		callParamsFactory("future_A_gamma_time", &futureAGammaTime, nil),
		callParamsFactory("future_A_gamma", &futureAGamma, nil),
		callParamsFactory("initial_A_gamma_time", &initialAGammaTime, nil),
		callParamsFactory("initial_A_gamma", &initialAGamma, nil),
		callParamsFactory("last_prices_timestamp", &lastPriceTimestamp, nil),
		callParamsFactory("xcp_profit", &xcpProfit, nil),
		callParamsFactory("virtual_price", &virtualPrice, nil),
		callParamsFactory("allowed_extra_profit", &allowedExtraProfit, nil),
		callParamsFactory("adjustment_step", &adjustmentStep, nil),
		callParamsFactory("ma_half_time", &maHalfTime, nil),
		callParamsFactory("balances", &reserves[nCoins-1], []interface{}{big.NewInt(int64(nCoins - 1))}),
	)

	for i := 0; i < nCoins-1; i += 1 {
		calls = append(calls,
			callParamsFactory("price_scale", &priceScale[i], []interface{}{big.NewInt(int64(i))}),
			callParamsFactory("price_oracle", &priceOracle[i], []interface{}{big.NewInt(int64(i))}),
			callParamsFactory("last_prices", &lastPrices[i], []interface{}{big.NewInt(int64(i))}),
			callParamsFactory("balances", &reserves[i], []interface{}{big.NewInt(int64(i))}),
		)
	}

	if len(pool.GetLpToken()) > 0 {
		calls = append(
			calls, &repository.CallParams{
				ABI:    abis.ERC20,
				Target: pool.GetLpToken(),
				Method: "totalSupply",
				Params: nil,
				Output: &lpSupply,
			},
		)
	}

	// Execute multicall, require all calls success
	if err := t.scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, nil, err
	}

	var priceScaleStr = make([]string, nCoins-1)
	var lastPricesStr = make([]string, nCoins-1)
	var priceOracleStr = make([]string, nCoins-1)
	for i := 0; i < nCoins-1; i += 1 {
		priceScaleStr[i] = priceScale[i].String()
		lastPricesStr[i] = lastPrices[i].String()
		priceOracleStr[i] = priceOracle[i].String()
	}
	extra := curveTricrypto.Extra{
		A:                   A.String(),
		Gamma:               gamma.String(),
		D:                   D.String(),
		FeeGamma:            feeGamma.String(),
		MidFee:              midFee.String(),
		OutFee:              outFee.String(),
		FutureAGammaTime:    futureAGammaTime.Int64(),
		FutureAGamma:        futureAGamma.String(),
		InitialAGammaTime:   initialAGammaTime.Int64(),
		InitialAGamma:       initialAGamma.String(),
		LastPricesTimestamp: lastPriceTimestamp.Int64(),
		PriceScale:          priceScaleStr,
		LastPrices:          lastPricesStr,
		PriceOracle:         priceOracleStr,
		LpSupply:            lpSupply.String(),
		XcpProfit:           xcpProfit.String(),
		VirtualPrice:        virtualPrice.String(),
		AllowedExtraProfit:  allowedExtraProfit.String(),
		AdjustmentStep:      adjustmentStep.String(),
		MaHalfTime:          maHalfTime.String(),
	}
	return extra, reserves, nil
}
