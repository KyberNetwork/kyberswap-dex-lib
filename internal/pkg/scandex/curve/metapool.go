package curve

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	curveMeta "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/curve-meta"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

func (t *Curve) getMetaPoolData(ctx context.Context, pool entity.Pool) (interface{}, []*big.Int, error) {

	var calls = make([]*repository.CallParams, 0)
	var nCoins = len(pool.Tokens)
	var initialA, futureA, initialATime, futureATime, swapFee, adminFee *big.Int
	calls = append(calls, &repository.CallParams{
		ABI:    abis.CurveMeta,
		Target: pool.Address,
		Method: "initial_A",
		Params: nil,
		Output: &initialA,
	})
	calls = append(calls, &repository.CallParams{
		ABI:    abis.CurveMeta,
		Target: pool.Address,
		Method: "initial_A_time",
		Params: nil,
		Output: &initialATime,
	})
	calls = append(calls, &repository.CallParams{
		ABI:    abis.CurveMeta,
		Target: pool.Address,
		Method: "future_A",
		Params: nil,
		Output: &futureA,
	})
	calls = append(calls, &repository.CallParams{
		ABI:    abis.CurveMeta,
		Target: pool.Address,
		Method: "future_A_time",
		Params: nil,
		Output: &futureATime,
	})
	calls = append(calls, &repository.CallParams{
		ABI:    abis.CurveMeta,
		Target: pool.Address,
		Method: "fee",
		Params: nil,
		Output: &swapFee,
	})
	calls = append(calls, &repository.CallParams{
		ABI:    abis.CurveMeta,
		Target: pool.Address,
		Method: "admin_fee",
		Params: nil,
		Output: &adminFee,
	})

	var reserves = make([]*big.Int, nCoins+1)
	for j := range pool.Tokens {
		calls = append(calls, &repository.CallParams{
			ABI:    abis.CurveMeta,
			Target: pool.Address,
			Method: "balances",
			Params: []interface{}{big.NewInt(int64(j))},
			Output: &reserves[j],
		})
	}
	calls = append(calls, &repository.CallParams{
		ABI:    abis.ERC20,
		Target: pool.GetLpToken(),
		Method: "totalSupply",
		Params: nil,
		Output: &reserves[nCoins],
	})

	if err := t.scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, nil, err
	}
	extra := curveMeta.Extra{
		InitialA:     initialA.String(),
		FutureA:      futureA.String(),
		InitialATime: initialATime.Int64(),
		FutureATime:  futureATime.Int64(),
		SwapFee:      swapFee.String(),
		AdminFee:     adminFee.String(),
	}
	return extra, reserves, nil
}
