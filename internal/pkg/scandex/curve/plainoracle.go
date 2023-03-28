package curve

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	curvePlainOracle "github.com/KyberNetwork/router-service/internal/pkg/core/curve-plain-oracle"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
)

const (
	PlainOraclePoolMethodBalances     = "balances"
	PlainOraclePoolMethodTotalSupply  = "totalSupply"
	PlainOraclePoolMethodInitialA     = "initial_A"
	PlainOraclePoolMethodInitialATime = "initial_A_time"
	PlainOraclePoolMethodFutureA      = "future_A"
	PlainOraclePoolMethodFutureATime  = "future_A_time"
	PlainOraclePoolMethodFee          = "fee"
	PlainOraclePoolMethodAdminFee     = "admin_fee"

	OracleMethodLatestAnswer = "latestAnswer"
)

var (
	PlainOraclePoolPrecision = constant.TenPowInt(18)
)

type PlainOraclePoolParameters struct {
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

func (t *Curve) getPlainOraclePoolExtra(ctx context.Context, pool entity.Pool) (interface{}, error) {
	var staticExtra = curvePlainOracle.PoolStaticExtra{}
	_ = json.Unmarshal([]byte(pool.StaticExtra), &staticExtra)
	if len(staticExtra.Oracle) == 0 {
		return nil, errors.New("staticExtra field is not updated")
	}

	var calls []*repository.CallParams
	var initialA, futureA, initialATime, futureATime, swapFee, adminFee, oracleLatestAnswer *big.Int

	callParamsFactory := repository.CallParamsFactory(abis.CurvePlainOraclePool, pool.Address)

	calls = append(
		calls,
		callParamsFactory(PlainOraclePoolMethodInitialA, &initialA, nil),
		callParamsFactory(PlainOraclePoolMethodInitialATime, &initialATime, nil),
		callParamsFactory(PlainOraclePoolMethodFutureA, &futureA, nil),
		callParamsFactory(PlainOraclePoolMethodFutureATime, &futureATime, nil),
		callParamsFactory(PlainOraclePoolMethodFee, &swapFee, nil),
		callParamsFactory(PlainOraclePoolMethodAdminFee, &adminFee, nil),
	)

	if err := t.scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, err
	}

	err := t.scanService.Call(ctx, &repository.CallParams{
		ABI:    abis.CurveOracle,
		Target: staticExtra.Oracle,
		Method: OracleMethodLatestAnswer,
		Output: &oracleLatestAnswer,
	})
	if err != nil {
		return nil, err
	}

	rates := []*big.Int{
		PlainOraclePoolPrecision,
		oracleLatestAnswer,
	}

	return curvePlainOracle.Extra{
		InitialA:     utils.SafeCastBigIntToString(initialA),
		FutureA:      utils.SafeCastBigIntToString(futureA),
		InitialATime: utils.SafeCastBigIntToInt64(initialATime),
		FutureATime:  utils.SafeCastBigIntToInt64(futureATime),
		SwapFee:      utils.SafeCastBigIntToString(swapFee),
		AdminFee:     utils.SafeCastBigIntToString(adminFee),
		Rates:        rates,
	}, nil
}

func (t *Curve) getPlainOraclePoolReserves(ctx context.Context, pool entity.Pool) ([]*big.Int, error) {
	var extra = curvePlainOracle.PoolStaticExtra{}
	err := json.Unmarshal([]byte(pool.StaticExtra), &extra)
	if err != nil {
		return nil, errors.New("failed to parse static extra")
	}

	if len(extra.LpToken) == 0 {
		return nil, errors.New("extra field is not updated")
	}

	var nTokens = len(pool.Tokens)
	var reserves = make([]*big.Int, nTokens+1)
	var tryCalls = make([]*repository.TryCallParams, 0)
	for i := range pool.Tokens {
		tryCalls = append(tryCalls, &repository.TryCallParams{
			ABI:    abis.CurvePlainOraclePool,
			Target: pool.Address,
			Method: PlainOraclePoolMethodBalances,
			Params: []interface{}{big.NewInt(int64(i))},
			Output: &reserves[i],
		})

	}
	tryCalls = append(tryCalls, &repository.TryCallParams{
		ABI:    abis.ERC20,
		Target: extra.LpToken,
		Method: PlainOraclePoolMethodTotalSupply,
		Params: nil,
		Output: &reserves[nTokens],
	})

	if err := t.scanService.TryAggregateForce(ctx, false, tryCalls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, err
	}

	return reserves, nil
}
