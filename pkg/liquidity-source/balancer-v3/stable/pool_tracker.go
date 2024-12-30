package stable

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

var ErrReserveNotFound = errors.New("reserve not found")

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dexId":       t.config.DexID,
		"dexType":     DexType,
		"poolAddress": p.Address,
	}).Info("Start updating state ...")

	defer func() {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Info("Finish updating state.")
	}()

	var extra Extra
	err := json.Unmarshal([]byte(p.Extra), &extra)
	if err != nil {
		return p, err
	}

	res, err := t.queryRPC(ctx, p.Address, overrides)
	if err != nil {
		return p, err
	}

	if len(p.Reserves) != len(res.BalancesRaw) {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error("can not fetch reserves")
		return p, err
	}

	var (
		amplificationParameter, _     = uint256.FromBig(res.AmplificationParameter)
		staticSwapFeePercentage, _    = uint256.FromBig(res.StaticSwapFeePercentage)
		aggregateSwapFeePercentage, _ = uint256.FromBig(res.AggregateSwapFeePercentage)
		balancesLiveScaled18          = lo.Map(res.BalancesLiveScaled18, func(v *big.Int, _ int) *uint256.Int {
			r, _ := uint256.FromBig(v)
			return r
		})
		tokenRates = lo.Map(res.TokenRates, func(v *big.Int, _ int) *uint256.Int {
			r, _ := uint256.FromBig(v)
			return r
		})
	)

	extra.AmplificationParameter = amplificationParameter
	extra.StaticSwapFeePercentage = staticSwapFeePercentage
	extra.AggregateSwapFeePercentage = aggregateSwapFeePercentage
	extra.BalancesLiveScaled18 = balancesLiveScaled18
	extra.TokenRates = tokenRates
	extra.IsPaused = res.IsPoolPaused
	extra.IsInRecoveryMode = res.IsPoolInRecoveryMode

	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())

		return p, err
	}

	p.BlockNumber = res.BlockNumber
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = lo.Map(res.TokenRates, func(v *big.Int, _ int) string {
		return v.String()
	})

	return p, nil
}

func (t *PoolTracker) queryRPC(
	ctx context.Context,
	poolAddress string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*RpcResult, error) {
	var (
		aggregateFeePercentages AggregateFeePercentage
		stablePoolDynamicData   StablePoolDynamicData
		poolTokenInfo           PoolTokenInfo
	)

	req := t.ethrpcClient.R().SetContext(ctx).SetRequireSuccess(true)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: shared.PoolMethodGetAggregateFeePercentages,
	}, []interface{}{&aggregateFeePercentages})
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetStablePoolDynamicData,
	}, []interface{}{&stablePoolDynamicData})
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: shared.PoolMethodGetTokenInfo,
	}, []interface{}{&poolTokenInfo})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": poolAddress,
		}).Error(err.Error())

		return nil, err
	}

	return &RpcResult{
		Tokens:                     poolTokenInfo.Tokens,
		BalancesRaw:                poolTokenInfo.BalancesRaw,
		BalancesLiveScaled18:       stablePoolDynamicData.BalancesLiveScaled18,
		TokenRates:                 stablePoolDynamicData.TokenRates,
		StaticSwapFeePercentage:    stablePoolDynamicData.StaticSwapFeePercentage,
		AggregateSwapFeePercentage: aggregateFeePercentages.AggregateSwapFeePercentage,
		AmplificationParameter:     stablePoolDynamicData.AmplificationParameter,
		IsPoolPaused:               stablePoolDynamicData.IsPoolPaused,
		IsPoolInRecoveryMode:       stablePoolDynamicData.IsPoolInRecoveryMode,
		BlockNumber:                res.BlockNumber.Uint64(),
	}, nil
}
