package quantamm

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

	var staticExtra StaticExtra
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

	// QuantAMM-specific fields
	if staticExtra.MaxTradeSizeRatio == nil {
		staticExtra.MaxTradeSizeRatio, _ = uint256.FromBig(res.ImmutableData.MaxTradeSizeRatio)
		if staticExtraBytes, err := json.Marshal(staticExtra); err == nil {
			p.StaticExtra = string(staticExtraBytes)
		}
	}
	// Split weights and multipliers from the packed arrays (ww..mm..)
	lenTokens := len(p.Tokens)
	firstHalf := min(4, lenTokens)
	weights := res.DynamicData.FirstFourWeightsAndMultipliers[:firstHalf]
	multipliers := res.DynamicData.FirstFourWeightsAndMultipliers[firstHalf:]
	if lenTokens > 4 {
		secondHalf := lenTokens - 4
		weights = append(weights, res.DynamicData.SecondFourWeightsAndMultipliers[:secondHalf]...)
		multipliers = append(multipliers, res.DynamicData.SecondFourWeightsAndMultipliers[secondHalf:]...)
	}
	extra.Weights = shared.FromBigs(weights)
	extra.Multipliers = shared.FromBigs(multipliers)
	extra.LastUpdateTime = res.DynamicData.LastUpdateTime
	extra.LastInteropTime = res.DynamicData.LastInteropTime

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.WithFields(klog.Fields{"error": err}).Error("failed to marshal extra data")
		return p, err
	}

	p.BlockNumber = res.BlockNumber
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	if res.IsPoolDisabled || !shared.IsHookSupported(staticExtra.HookType) {
		// set all reserves to 0 to disable pool
		p.Reserves = lo.Map(p.Reserves, func(_ string, _ int) string { return "0" })
	} else {
		p.Reserves = lo.Map(res.PoolData.BalancesRaw, func(v *big.Int, _ int) string { return v.String() })
	}

	return p, nil
}

func (t *PoolTracker) queryRPCData(ctx context.Context, p *entity.Pool, staticExtra StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount) (*RpcResult, error) {
	var (
		rpcRes        RpcResult
		isVaultPaused bool
	)

	req := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides).SetFrom(shared.AddrDummy)

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
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetQuantAMMWeightedPoolDynamicData,
	}, []any{&rpcRes.DynamicDataRpc})
	if staticExtra.MaxTradeSizeRatio == nil {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodGetQuantAMMWeightedPoolImmutableData,
		}, []any{&rpcRes.ImmutableDataRpc})
	}
	rpcRes.Buffers = shared.GetBufferTokens(req, t.config.ChainID, t.config.DexID, staticExtra.BufferTokens)

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to query RPC data")
	}

	rpcRes.IsPoolDisabled = isVaultPaused || !rpcRes.DynamicData.IsPoolInitialized || rpcRes.DynamicData.IsPoolPaused ||
		rpcRes.DynamicData.IsPoolInRecoveryMode
	rpcRes.BlockNumber = res.BlockNumber.Uint64()

	return &rpcRes, nil
}
