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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
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

	res, err := t.queryRPCData(ctx, &p, staticExtra, overrides)
	if err != nil {
		l.WithFields(klog.Fields{"error": err}).Error("failed to query RPC data")
		return p, err
	}

	extra := Extra{Extra: &shared.Extra{}}
	extra.EnableHookAdjustedAmounts = res.HooksConfigData.EnableHookAdjustedAmounts
	extra.ShouldCallComputeDynamicSwapFee = res.HooksConfigData.ShouldCallComputeDynamicSwapFee
	extra.ShouldCallBeforeSwap = res.HooksConfigData.ShouldCallBeforeSwap
	extra.ShouldCallAfterSwap = res.HooksConfigData.ShouldCallAfterSwap
	extra.StaticSwapFeePercentage, _ = uint256.FromBig(res.StaticSwapFeePercentage)
	extra.AggregateSwapFeePercentage, _ = uint256.FromBig(res.AggregateSwapFeePercentage)
	extra.BalancesLiveScaled18 = shared.FromBigs(res.PoolData.BalancesLiveScaled18)
	extra.DecimalScalingFactors = shared.FromBigs(res.PoolData.DecimalScalingFactors)
	extra.TokenRates = shared.FromBigs(res.PoolData.TokenRates)
	extra.Buffers = res.Buffers()
	if staticExtra.HookType == shared.StableSurgeHookType {
		extra.MaxSurgeFeePercentage, _ = uint256.FromBig(res.MaxSurgeFeePercentage)
		extra.SurgeThresholdPercentage, _ = uint256.FromBig(res.SurgeThresholdPercentage)
		extra.IsRisky = extra.isRisky(p, t.config.ChainID)
	}
	extra.AmplificationParameter, _ = uint256.FromBig(res.Value)

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.WithFields(klog.Fields{"error": err}).Error("failed to marshal extra data")
		return p, err
	}

	p.BlockNumber = res.BlockNumber
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	if res.IsPoolDisabled || extra.IsRisky || !shared.IsHookSupported(staticExtra.HookType) {
		// set all reserves to 0 to disable pool
		p.Reserves = lo.Map(p.Reserves, func(_ string, _ int) string { return "0" })
	} else {
		p.Reserves = lo.Map(res.PoolData.BalancesRaw, func(v *big.Int, _ int) string { return v.String() })
	}

	return p, nil
}

func (t *PoolTracker) queryRPCData(ctx context.Context, p *entity.Pool, staticExtra shared.StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount) (*RpcResult, error) {
	var (
		rpcRes               RpcResult
		isVaultPaused        bool
		isPoolPaused         bool
		isPoolInRecoveryMode bool
	)

	req := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides)

	poolAddress := p.Address
	paramsPool := []any{common.HexToAddress(poolAddress)}
	req.AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodGetHooksConfig,
		Params: paramsPool,
	}, []any{&rpcRes.HooksConfigRPC}).AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodGetStaticSwapFeePercentage,
		Params: paramsPool,
	}, []any{&rpcRes.StaticSwapFeePercentage}).AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodGetAggregateFeePercentages,
		Params: paramsPool,
	}, []any{&rpcRes.AggregateFeePercentageRPC}).AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodGetPoolData,
		Params: paramsPool,
	}, []any{&rpcRes.PoolDataRPC}).AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodIsVaultPaused,
	}, []any{&isVaultPaused}).AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodIsPoolPaused,
		Params: paramsPool,
	}, []any{&isPoolPaused}).AddCall(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: t.config.VaultExplorer,
		Method: shared.VaultMethodIsPoolInRecoveryMode,
		Params: paramsPool,
	}, []any{&isPoolInRecoveryMode}).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetAmplificationParameter,
	}, []any{&rpcRes.AmplificationParameterRpc})
	if staticExtra.HookType == shared.StableSurgeHookType {
		req.AddCall(&ethrpc.Call{
			ABI:    stableSurgeABI,
			Target: staticExtra.Hook,
			Method: stableSurgeHookMethodGetMaxSurgeFeePercentage,
			Params: paramsPool,
		}, []any{&rpcRes.MaxSurgeFeePercentage}).AddCall(&ethrpc.Call{
			ABI:    stableSurgeABI,
			Target: staticExtra.Hook,
			Method: stableSurgeHookMethodGetSurgeThresholdPercentage,
			Params: paramsPool,
		}, []any{&rpcRes.SurgeThresholdPercentage})
	}
	rpcRes.Buffers = shared.GetBufferTokens(req, t.config.ChainID, t.config.DexID, staticExtra.BufferTokens)

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to query RPC data")
	}

	rpcRes.IsPoolDisabled = isVaultPaused || isPoolPaused || isPoolInRecoveryMode
	rpcRes.BlockNumber = res.BlockNumber.Uint64()

	return &rpcRes, nil
}

func (s SurgePercentages) isRisky(p entity.Pool, chainId valueobject.ChainID) bool {
	if s.MaxSurgeFeePercentage == nil || s.SurgeThresholdPercentage == nil ||
		s.MaxSurgeFeePercentage.Cmp(AcceptableMaxSurgeFeePercentage) <= 0 &&
			math.StableSurgeMedian.CalculateFeeSurgeRatio(s.MaxSurgeFeePercentage, s.SurgeThresholdPercentage).
				Cmp(AcceptableMaxSurgeFeeByImbalance) <= 0 {
		return false
	}

	var hasNative, hasStable bool
	for _, token := range p.Tokens {
		if !hasNative && valueobject.IsWrappedNative(token.Address, chainId) {
			if hasStable {
				return true
			}
			hasNative = true
		} else if !hasStable && stablesByChain[chainId][token.Address] {
			if hasNative {
				return true
			}
			hasStable = true
		}
	}
	return false
}
