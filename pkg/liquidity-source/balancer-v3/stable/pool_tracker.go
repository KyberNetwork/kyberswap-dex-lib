package stable

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils/klog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *shared.Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *shared.Config,
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
	l := klog.WithFields(ctx, klog.Fields{
		"dexId":       t.config.DexID,
		"dexType":     DexType,
		"poolAddress": p.Address,
	})
	l.Info("Start updating state ...")
	defer func() {
		l.Info("Finish updating state.")
	}()

	var staticExtra shared.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		l.WithFields(klog.Fields{"error": err}).Error("failed to unmarshal StaticExtra data")
		return entity.Pool{}, err
	}

	res, err := t.queryRPCData(ctx, p.Address, staticExtra, overrides)
	if err != nil {
		l.WithFields(klog.Fields{"error": err}).Error("failed to query RPC data")
		return p, err
	}

	var (
		amplificationParameter, _     = uint256.FromBig(res.AmplificationParameter)
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

		buffers = lo.Map(res.Buffers, func(v *shared.ExtraBufferRPC, _ int) *shared.ExtraBuffer {
			if v == nil {
				return nil
			}
			buffer := &shared.ExtraBuffer{}
			buffer.TotalAssets, _ = uint256.FromBig(v.TotalAssets)
			buffer.TotalAssets.AddUint64(buffer.TotalAssets, 1)
			buffer.TotalSupply, _ = uint256.FromBig(v.TotalSupply)
			buffer.TotalSupply.Add(buffer.TotalSupply, shared.DecimalsOffsetPow)
			return buffer
		})
	)

	extraBytes, err := json.Marshal(&Extra{
		AmplificationParameter:     amplificationParameter,
		StaticSwapFeePercentage:    staticSwapFeePercentage,
		AggregateSwapFeePercentage: aggregateSwapFeePercentage,
		BalancesLiveScaled18:       balancesLiveScaled18,
		DecimalScalingFactors:      decimalScalingFactors,
		TokenRates:                 tokenRates,
		Buffers:                    buffers,
	})
	if err != nil {
		l.WithFields(klog.Fields{"error": err}).Error("failed to marshal extra data")
		return p, err
	}

	p.BlockNumber = res.BlockNumber
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	if isPoolDisabled := res.IsVaultPaused || res.IsPoolPaused || res.IsPoolInRecoveryMode; !isPoolDisabled && shared.IsHookSupported(staticExtra.HookType) {
		p.Reserves = lo.Map(res.BalancesRaw, func(v *big.Int, _ int) string { return v.String() })
	} else { // set all reserves to 0 to disable pool temporarily
		p.Reserves = lo.Map(p.Reserves, func(_ string, _ int) string { return "0" })
	}

	return p, nil
}

func (t *PoolTracker) queryRPCData(ctx context.Context, poolAddress string, staticExtra shared.StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount) (*RpcResult, error) {
	var (
		aggregateFeePercentages shared.AggregateFeePercentage
		hooksConfig             shared.HooksConfigRPC
		poolData                shared.PoolDataRPC

		amplificationParameter  AmplificationParameter
		staticSwapFeePercentage *big.Int

		buffers = make([]*shared.ExtraBufferRPC, len(staticExtra.BufferTokens))

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
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&aggregateFeePercentages}).AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodGetStaticSwapFeePercentage,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&staticSwapFeePercentage}).AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodGetPoolData,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&poolData}).AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodGetHooksConfig,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&hooksConfig}).AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodIsVaultPaused,
	}, []any{&isVaultPaused}).AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodIsPoolPaused,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&isPoolPaused}).AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodIsPoolInRecoveryMode,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&isPoolInRecoveryMode}).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetAmplificationParameter,
	}, []any{&amplificationParameter})

	for i, token := range staticExtra.BufferTokens {
		if token != "" {
			buffers[i] = &shared.ExtraBufferRPC{}
			req.AddCall(&ethrpc.Call{
				ABI:    shared.ERC4626ABI,
				Target: token,
				Method: shared.ERC4626MethodTotalAssets,
			}, []any{&buffers[i].TotalAssets}).AddCall(&ethrpc.Call{
				ABI:    shared.ERC4626ABI,
				Target: token,
				Method: shared.ERC4626MethodTotalSupply,
			}, []any{&buffers[i].TotalSupply})
		}
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to query RPC data")
	}

	return &RpcResult{
		HooksConfig: shared.HooksConfig{
			EnableHookAdjustedAmounts:       hooksConfig.Data.EnableHookAdjustedAmounts,
			ShouldCallComputeDynamicSwapFee: hooksConfig.Data.ShouldCallComputeDynamicSwapFee,
			ShouldCallBeforeSwap:            hooksConfig.Data.ShouldCallBeforeSwap,
			ShouldCallAfterSwap:             hooksConfig.Data.ShouldCallAfterSwap,
		},
		Buffers:                    buffers,
		BalancesRaw:                poolData.Data.BalancesRaw,
		BalancesLiveScaled18:       poolData.Data.BalancesLiveScaled18,
		TokenRates:                 poolData.Data.TokenRates,
		DecimalScalingFactors:      poolData.Data.DecimalScalingFactors,
		StaticSwapFeePercentage:    staticSwapFeePercentage,
		AggregateSwapFeePercentage: aggregateFeePercentages.AggregateSwapFeePercentage,
		AmplificationParameter:     amplificationParameter.Value,
		IsVaultPaused:              isVaultPaused,
		IsPoolPaused:               isPoolPaused,
		IsPoolInRecoveryMode:       isPoolInRecoveryMode,
		BlockNumber:                res.BlockNumber.Uint64(),
	}, nil
}
