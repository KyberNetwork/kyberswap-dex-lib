package curve

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	curveBase "github.com/KyberNetwork/router-service/internal/pkg/core/curve-base"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	MainRegistryMethodGetParameters = "get_parameters"

	BasePoolMethodBalances     = "balances"
	BasePoolMethodTotalSupply  = "totalSupply"
	BasePoolMethodInitialA     = "initial_A"
	BasePoolMethodInitialATime = "initial_A_time"
	BasePoolMethodFutureA      = "future_A"
	BasePoolMethodFutureATime  = "future_A_time"
	BasePoolMethodFee          = "fee"
	BasePoolMethodAdminFee     = "admin_fee"
)

type BasePoolParameters struct {
	A              *big.Int
	FutureA        *big.Int
	Fee            *big.Int
	AdminFee       *big.Int
	FutureFee      *big.Int
	FutureAdminFee *big.Int
	FutureOwner    common.Address
	InitialA       *big.Int
	InitialATime   *big.Int
	FutureATime    *big.Int
}

func (t *Curve) getBasePoolExtraFromPoolDirectly(ctx context.Context, pool entity.Pool) (curveBase.Extra, error) {

	var calls = make([]*repository.TryCallParams, 0)
	var initialA, futureA, initialATime, futureATime, swapFee, adminFee *big.Int

	callParamsFactory := repository.TryCallParamsFactory(abis.CurveBase, pool.Address)

	calls = append(
		calls,
		callParamsFactory(BasePoolMethodInitialA, &initialA, nil),
		callParamsFactory(BasePoolMethodInitialATime, &initialATime, nil),
		callParamsFactory(BasePoolMethodFutureA, &futureA, nil),
		callParamsFactory(BasePoolMethodFutureATime, &futureATime, nil),
		callParamsFactory(BasePoolMethodFee, &swapFee, nil),
		callParamsFactory(BasePoolMethodAdminFee, &adminFee, nil),
	)

	if err := t.scanService.TryAggregateForce(ctx, false, calls); err != nil {
		logger.Errorf("failed to process tryAggregate, pool: %v, err: %v", pool.Address, err)
		return curveBase.Extra{}, err
	}

	return curveBase.Extra{
		InitialA:     utils.SafeCastBigIntToString(initialA),
		FutureA:      utils.SafeCastBigIntToString(futureA),
		InitialATime: utils.SafeCastBigIntToInt64(initialATime),
		FutureATime:  utils.SafeCastBigIntToInt64(futureATime),
		SwapFee:      utils.SafeCastBigIntToString(swapFee),
		AdminFee:     utils.SafeCastBigIntToString(adminFee),
	}, nil
}

func (t *Curve) getBasePoolExtraFromMainRegistry(ctx context.Context, pool entity.Pool) (curveBase.Extra, error) {
	if len(t.properties.AddressesFromProvider) == 0 {
		return curveBase.Extra{}, ErrNoMainRegistry
	}

	mainRegistry := t.properties.AddressesFromProvider[0]
	var poolInfos BasePoolParameters
	tryCalls := []*repository.TryCallParams{
		{
			ABI:    abis.CurveMainRegistry,
			Target: mainRegistry,
			Method: MainRegistryMethodGetParameters,
			Params: []interface{}{common.HexToAddress(pool.Address)},
			Output: &poolInfos,
		},
	}

	if err := t.scanService.TryAggregateForce(ctx, false, tryCalls); err != nil {
		logger.Errorf("failed to process tryAggregate, pool: %v, err: %v", pool.Address, err)
		return curveBase.Extra{}, err
	}

	return curveBase.Extra{
		InitialA:     utils.SafeCastBigIntToString(poolInfos.InitialA),
		FutureA:      utils.SafeCastBigIntToString(poolInfos.FutureA),
		InitialATime: utils.SafeCastBigIntToInt64(poolInfos.InitialATime),
		FutureATime:  utils.SafeCastBigIntToInt64(poolInfos.FutureATime),
		SwapFee:      utils.SafeCastBigIntToString(poolInfos.Fee),
		AdminFee:     utils.SafeCastBigIntToString(poolInfos.AdminFee),
	}, nil
}

func (t *Curve) getBasePoolExtra(ctx context.Context, pool entity.Pool) (curveBase.Extra, error) {
	extraFromPool, err := t.getBasePoolExtraFromPoolDirectly(ctx, pool)
	if err != nil {
		return t.getBasePoolExtraFromMainRegistry(ctx, pool)
	}

	extraFromMainRegistry, err := t.getBasePoolExtraFromMainRegistry(ctx, pool)
	if err != nil {
		return extraFromPool, nil
	}

	var extra curveBase.Extra

	// Merge information from pool and main registry, prefer data from pool
	extra.InitialA = utils.CoalesceEmptyString(extraFromPool.InitialA, extraFromMainRegistry.InitialA)
	extra.FutureA = utils.CoalesceEmptyString(extraFromPool.FutureA, extraFromMainRegistry.FutureA)
	extra.InitialATime = utils.CoalesceZero(extraFromPool.InitialATime, extraFromMainRegistry.InitialATime)
	extra.FutureATime = utils.CoalesceZero(extraFromPool.FutureATime, extraFromMainRegistry.FutureATime)
	extra.SwapFee = utils.CoalesceEmptyString(extraFromPool.SwapFee, extraFromMainRegistry.SwapFee)
	extra.AdminFee = utils.CoalesceEmptyString(extraFromPool.AdminFee, extraFromMainRegistry.AdminFee)

	return extra, nil
}

func (t *Curve) getBasePoolReserves(ctx context.Context, pool entity.Pool) ([]*big.Int, error) {

	var extra = curveBase.PoolStaticExtra{}
	_ = json.Unmarshal([]byte(pool.StaticExtra), &extra)
	if len(extra.LpToken) == 0 {
		return nil, errors.New("extra field is not updated")
	}
	var nTokens = len(pool.Tokens)
	var reserves = make([]*big.Int, 2*nTokens+1)
	var tryCalls = make([]*repository.TryCallParams, 0)
	for j := range pool.Tokens {
		tryCalls = append(
			tryCalls,
			&repository.TryCallParams{
				ABI:    abis.CurveBase,
				Target: pool.Address,
				Method: BasePoolMethodBalances,
				Params: []interface{}{big.NewInt(int64(j))},
				Output: &reserves[2*j],
			},
			&repository.TryCallParams{
				ABI:    abis.CurveBaseV1,
				Target: pool.Address,
				Method: BasePoolMethodBalances,
				Params: []interface{}{big.NewInt(int64(j))},
				Output: &reserves[2*j+1],
			},
		)
	}

	tryCalls = append(
		tryCalls, &repository.TryCallParams{
			ABI:    abis.ERC20,
			Target: extra.LpToken,
			Method: BasePoolMethodTotalSupply,
			Params: nil,
			Output: &reserves[2*nTokens],
		},
	)

	if err := t.scanService.TryAggregateForce(ctx, false, tryCalls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, err
	}
	var res = make([]*big.Int, 0)

	for i := 0; i < nTokens; i++ {
		if reserves[2*i] != nil {
			res = append(res, reserves[2*i])
		} else {
			res = append(res, reserves[2*i+1])
		}
	}
	res = append(res, reserves[2*nTokens])
	return res, nil
}
