package lazy

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
	eclp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/eclp"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *shared.Config
	ethrpcClient *ethrpc.Client
}

var (
	_ pool.IBatchRPCPoolTracker = (*PoolTracker)(nil)
	_ = pooltrack.RegisterFactoryCE(eclp.DexType, NewPoolTracker)
)

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
		"dexType":     eclp.DexType,
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

	result, err := buildPoolState(&p, res, staticExtra)
	if err != nil {
		l.WithFields(klog.Fields{"error": err}).Error("failed to marshal extra data")
		return p, err
	}
	return result, nil
}

func buildPoolState(p *entity.Pool, res *eclp.RpcResult, staticExtra shared.StaticExtra) (entity.Pool, error) {
	extra := eclp.Extra{Extra: &shared.Extra{}}
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
	extra.ECLPParams = res.ToInt256()

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return *p, err
	}

	p.BlockNumber = res.BlockNumber
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	if res.IsPoolDisabled || !shared.IsHookSupported(staticExtra.HookType) {
		p.Reserves = lo.Map(p.Reserves, func(_ string, _ int) string { return "0" })
	} else {
		p.Reserves = lo.Map(res.PoolData.BalancesRaw, func(v *big.Int, _ int) string { return v.String() })
	}
	return *p, nil
}

func (t *PoolTracker) queryRPCData(ctx context.Context, p *entity.Pool, staticExtra shared.StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount) (*eclp.RpcResult, error) {
	var (
		rpcRes               eclp.RpcResult
		isVaultPaused        bool
		isPoolPaused         bool
		isPoolInRecoveryMode bool
	)

	req := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides).SetFrom(shared.AddrDummy)
	addRPCCalls(func(c *ethrpc.Call, output []any) { req.AddCall(c, output) },
		t.config.VaultExplorer, p.Address, &rpcRes, &isVaultPaused, &isPoolPaused, &isPoolInRecoveryMode)
	rpcRes.Buffers = shared.GetBufferTokens(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) }, t.config.ChainID, t.config.DexID, staticExtra.BufferTokens)

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to query RPC data")
	}

	rpcRes.IsPoolDisabled = isVaultPaused || isPoolPaused || isPoolInRecoveryMode
	rpcRes.BlockNumber = res.BlockNumber.Uint64()

	return &rpcRes, nil
}

func addRPCCalls(addFn func(*ethrpc.Call, []any), vaultExplorer, poolAddress string,
	rpcRes *eclp.RpcResult, isVaultPaused, isPoolPaused, isPoolInRecoveryMode *bool) {
	paramsPool := []any{common.HexToAddress(poolAddress)}
	addFn(&ethrpc.Call{ABI: shared.VaultExplorerABI, Target: vaultExplorer, Method: shared.VaultMethodGetHooksConfig, Params: paramsPool}, []any{&rpcRes.HooksConfigRPC})
	addFn(&ethrpc.Call{ABI: shared.VaultExplorerABI, Target: vaultExplorer, Method: shared.VaultMethodGetStaticSwapFeePercentage, Params: paramsPool}, []any{&rpcRes.StaticSwapFeePercentage})
	addFn(&ethrpc.Call{ABI: shared.VaultExplorerABI, Target: vaultExplorer, Method: shared.VaultMethodGetAggregateFeePercentages, Params: paramsPool}, []any{&rpcRes.AggregateFeePercentageRPC})
	addFn(&ethrpc.Call{ABI: shared.VaultExplorerABI, Target: vaultExplorer, Method: shared.VaultMethodGetPoolData, Params: paramsPool}, []any{&rpcRes.PoolDataRPC})
	addFn(&ethrpc.Call{ABI: shared.VaultExplorerABI, Target: vaultExplorer, Method: shared.VaultMethodIsVaultPaused}, []any{isVaultPaused})
	addFn(&ethrpc.Call{ABI: shared.VaultExplorerABI, Target: vaultExplorer, Method: shared.VaultMethodIsPoolPaused, Params: paramsPool}, []any{isPoolPaused})
	addFn(&ethrpc.Call{ABI: shared.VaultExplorerABI, Target: vaultExplorer, Method: shared.VaultMethodIsPoolInRecoveryMode, Params: paramsPool}, []any{isPoolInRecoveryMode})
	addFn(&ethrpc.Call{ABI: *eclp.PoolABI, Target: poolAddress, Method: eclp.PoolMethodGetECLPParams}, []any{&rpcRes.ECLPParamsRpc})
}
