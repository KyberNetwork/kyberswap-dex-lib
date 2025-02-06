package weighted

import (
	"context"
	"errors"
	"fmt"
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

	var staticExtra StaticExtra
	err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Errorf("failed to unmarshal StaticExtra data : %s", err.Error())

		return entity.Pool{}, err
	}

	if !staticExtra.IsPoolInitialized {
		isPoolInitialized, err := t.IsPoolInitialized(ctx, p.Address, overrides)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexId":       t.config.DexID,
				"dexType":     DexType,
				"poolAddress": p.Address,
			}).Error(err.Error())

			return entity.Pool{}, err
		}

		if !isPoolInitialized {
			logger.WithFields(logger.Fields{
				"dexId":       t.config.DexID,
				"dexType":     DexType,
				"poolAddress": p.Address,
			}).Warn("this pool still not be initialized")

			return p, nil
		}

		staticExtraBytes, err := json.Marshal(&staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexId":       t.config.DexID,
				"dexType":     DexType,
				"poolAddress": p.Address,
			}).Errorf("failed to marshal StaticExtra data : %s", err.Error())

			return entity.Pool{}, err
		}

		p.StaticExtra = string(staticExtraBytes)
	}

	res, shouldDisablePool, err := t.queryRPCData(ctx, p.Address, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Errorf(err.Error())

		return p, err
	}

	var (
		staticSwapFeePercentage, _    = uint256.FromBig(res.StaticSwapFeePercentage)
		aggregateSwapFeePercentage, _ = uint256.FromBig(res.AggregateSwapFeePercentage)

		balancesLiveScaled18 = lo.Map(res.BalancesLiveScaled18, func(v *big.Int, _ int) *uint256.Int {
			return uint256.MustFromBig(v)
		})
		tokenRates = lo.Map(res.TokenRates, func(v *big.Int, _ int) *uint256.Int {
			return uint256.MustFromBig(v)
		})
		decimalScalingFactors = lo.Map(res.DecimalScalingFactors, func(v *big.Int, _ int) *uint256.Int {
			return uint256.MustFromBig(v)
		})
		normalizedWeights = lo.Map(res.NormalizedWeights, func(v *big.Int, _ int) *uint256.Int {
			return uint256.MustFromBig(v)
		})
	)

	extraBytes, err := json.Marshal(&Extra{
		NormalizedWeights:          normalizedWeights,
		StaticSwapFeePercentage:    staticSwapFeePercentage,
		AggregateSwapFeePercentage: aggregateSwapFeePercentage,
		BalancesLiveScaled18:       balancesLiveScaled18,
		DecimalScalingFactors:      decimalScalingFactors,
		TokenRates:                 tokenRates,
		IsVaultPaused:              res.IsVaultPaused,
		IsPoolPaused:               res.IsPoolPaused,
		IsPoolInRecoveryMode:       res.IsPoolInRecoveryMode,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Errorf("failed to marshal extra data : %s", err.Error())

		return p, err
	}

	p.BlockNumber = res.BlockNumber
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	// Set all reserves to 0 to disable pool temporarily
	if shouldDisablePool {
		p.Reserves = lo.Map(p.Reserves, func(_ string, _ int) string {
			return "0"
		})
	} else {
		p.Reserves = lo.Map(res.BalancesRaw, func(v *big.Int, _ int) string {
			return v.String()
		})
	}
	return p, nil
}

func (t *PoolTracker) IsPoolInitialized(
	ctx context.Context,
	poolAddress string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (bool, error) {
	var isPoolInitialized bool

	req := t.ethrpcClient.R().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodIsPoolInitialized,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&isPoolInitialized})

	if _, err := req.Call(); err != nil {
		return false, fmt.Errorf("failed to check if pool is initialized : %s", err.Error())
	}

	return isPoolInitialized, nil
}

func (t *PoolTracker) queryRPCData(
	ctx context.Context,
	poolAddress string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*RpcResult, bool, error) {
	var (
		aggregateFeePercentages shared.AggregateFeePercentage
		hooksConfig             shared.HooksConfigRPC
		poolData                shared.PoolDataRPC

		normalizedWeights       []*big.Int
		staticSwapFeePercentage *big.Int

		isVaultPaused        bool
		isPoolPaused         bool
		isPoolInRecoveryMode bool
	)

	req := t.ethrpcClient.R().SetContext(ctx).SetRequireSuccess(true)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodGetAggregateFeePercentages,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&aggregateFeePercentages})

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodGetStaticSwapFeePercentage,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&staticSwapFeePercentage})

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodGetPoolData,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&poolData})

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodGetHooksConfig,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&hooksConfig})

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodIsVaultPaused,
	}, []interface{}{&isVaultPaused})

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodIsPoolPaused,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&isPoolPaused})

	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodIsPoolInRecoveryMode,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&isPoolInRecoveryMode})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetNormalizedWeights,
	}, []interface{}{&normalizedWeights})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, false, fmt.Errorf("failed to query RPC data : %s", err.Error())
	}

	var shouldDisablePool bool
	if hooksConfig.Data.HooksContract != (common.Address{}) {
		shouldDisablePool = true

		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": poolAddress,
		}).Warnf("this pool has a contract hook implemented at %s => should check it", hooksConfig.Data.HooksContract)
	}

	return &RpcResult{
		HooksConfig: shared.HooksConfig{
			EnableHookAdjustedAmounts:       hooksConfig.Data.EnableHookAdjustedAmounts,
			ShouldCallComputeDynamicSwapFee: hooksConfig.Data.ShouldCallComputeDynamicSwapFee,
			ShouldCallBeforeSwap:            hooksConfig.Data.ShouldCallBeforeSwap,
			ShouldCallAfterSwap:             hooksConfig.Data.ShouldCallAfterSwap,
		},
		BalancesRaw:                poolData.Data.BalancesRaw,
		BalancesLiveScaled18:       poolData.Data.BalancesLiveScaled18,
		TokenRates:                 poolData.Data.TokenRates,
		DecimalScalingFactors:      poolData.Data.DecimalScalingFactors,
		StaticSwapFeePercentage:    staticSwapFeePercentage,
		AggregateSwapFeePercentage: aggregateFeePercentages.AggregateSwapFeePercentage,
		NormalizedWeights:          normalizedWeights,
		IsVaultPaused:              isVaultPaused,
		IsPoolPaused:               isPoolPaused,
		IsPoolInRecoveryMode:       isPoolInRecoveryMode,
		BlockNumber:                res.BlockNumber.Uint64(),
	}, shouldDisablePool, nil
}
