package eclp

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
	extra.Buffers = lo.Map(res.Buffers, func(v *shared.ExtraBufferRPC, _ int) *shared.ExtraBuffer {
		if v == nil {
			return nil
		}
		var totalAssets, totalSupply uint256.Int
		totalAssets.SetFromBig(v.TotalAssets)
		totalSupply.SetFromBig(v.TotalSupply)
		return &shared.ExtraBuffer{
			TotalAssets: totalAssets.AddUint64(&totalAssets, 1),
			TotalSupply: totalSupply.Add(&totalSupply, shared.DecimalsOffsetPow),
		}
	})
	extra.ECLPParams = res.ECLPParamsRpc.toInt256()

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.WithFields(klog.Fields{"error": err}).Error("failed to marshal extra data")
		return p, err
	}

	p.BlockNumber = res.BlockNumber
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	if !res.IsPoolDisabled && shared.IsHookSupported(staticExtra.HookType) {
		p.Reserves = lo.Map(res.PoolData.BalancesRaw, func(v *big.Int, _ int) string { return v.String() })
	} else { // set all reserves to 0 to disable pool temporarily
		p.Reserves = lo.Map(p.Reserves, func(_ string, _ int) string { return "0" })
	}
	return p, nil
}

func (t *PoolTracker) queryRPCData(ctx context.Context, poolAddress string, staticExtra shared.StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount) (*RpcResult, error) {
	var (
		rpcRes               RpcResult
		isVaultPaused        bool
		isPoolPaused         bool
		isPoolInRecoveryMode bool
	)
	rpcRes.Buffers = make([]*shared.ExtraBufferRPC, len(staticExtra.BufferTokens))

	req := t.ethrpcClient.R().SetContext(ctx).SetRequireSuccess(true)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

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
		Method: poolMethodGetECLPParams,
	}, []any{&rpcRes.ECLPParamsRpc})

	for i, token := range staticExtra.BufferTokens {
		if token != "" {
			rpcRes.Buffers[i] = &shared.ExtraBufferRPC{}
			req.AddCall(&ethrpc.Call{
				ABI:    shared.ERC4626ABI,
				Target: token,
				Method: shared.ERC4626MethodTotalAssets,
			}, []any{&rpcRes.Buffers[i].TotalAssets}).AddCall(&ethrpc.Call{
				ABI:    shared.ERC4626ABI,
				Target: token,
				Method: shared.ERC4626MethodTotalSupply,
			}, []any{&rpcRes.Buffers[i].TotalSupply})
		}
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to query RPC data")
	}

	rpcRes.IsPoolDisabled = isVaultPaused || isPoolPaused || isPoolInRecoveryMode
	rpcRes.BlockNumber = res.BlockNumber.Uint64()

	return &rpcRes, nil
}
