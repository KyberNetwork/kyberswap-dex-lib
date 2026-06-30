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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	weighted "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/weighted"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config       *shared.Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(weighted.DexType, NewPoolTracker)

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
		"dexType":     weighted.DexType,
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

	var (
		rpcRes weighted.RpcResult
		flags  rpcFlags
	)
	req := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides).SetFrom(shared.AddrDummy)
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) }, p.Address, t.config.VaultExplorer,
		&staticExtra, t.config.ChainID, t.config.DexID, &rpcRes, &flags)

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		l.WithFields(klog.Fields{"error": err}).Error("failed to query RPC data")
		return p, errors.WithMessage(err, "failed to query RPC data")
	}

	rpcRes.IsPoolDisabled = flags.isVaultPaused || flags.isPoolPaused || flags.isPoolInRecoveryMode
	rpcRes.BlockNumber = res.BlockNumber.Uint64()

	return buildPoolState(p, &staticExtra, &rpcRes)
}

type rpcFlags struct {
	isVaultPaused        bool
	isPoolPaused         bool
	isPoolInRecoveryMode bool
}

func addRPCCalls(
	addFn func(*ethrpc.Call, []any),
	poolAddress, vaultExplorer string,
	staticExtra *shared.StaticExtra,
	chainID valueobject.ChainID,
	dexID string,
	rpcRes *weighted.RpcResult,
	flags *rpcFlags,
) {
	paramsPool := []any{common.HexToAddress(poolAddress)}
	addFn(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: vaultExplorer,
		Method: shared.VaultMethodGetHooksConfig,
		Params: paramsPool,
	}, []any{&rpcRes.HooksConfigRPC})
	addFn(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: vaultExplorer,
		Method: shared.VaultMethodGetStaticSwapFeePercentage,
		Params: paramsPool,
	}, []any{&rpcRes.StaticSwapFeePercentage})
	addFn(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: vaultExplorer,
		Method: shared.VaultMethodGetAggregateFeePercentages,
		Params: paramsPool,
	}, []any{&rpcRes.AggregateFeePercentageRPC})
	addFn(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: vaultExplorer,
		Method: shared.VaultMethodGetPoolData,
		Params: paramsPool,
	}, []any{&rpcRes.PoolDataRPC})
	addFn(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: vaultExplorer,
		Method: shared.VaultMethodIsVaultPaused,
	}, []any{&flags.isVaultPaused})
	addFn(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: vaultExplorer,
		Method: shared.VaultMethodIsPoolPaused,
		Params: paramsPool,
	}, []any{&flags.isPoolPaused})
	addFn(&ethrpc.Call{
		ABI:    shared.VaultExplorerABI,
		Target: vaultExplorer,
		Method: shared.VaultMethodIsPoolInRecoveryMode,
		Params: paramsPool,
	}, []any{&flags.isPoolInRecoveryMode})
	addFn(&ethrpc.Call{
		ABI:    *weighted.PoolABI,
		Target: poolAddress,
		Method: weighted.PoolMethodGetNormalizedWeights,
	}, []any{&rpcRes.NormalizedWeights})
	rpcRes.Buffers = shared.GetBufferTokens(addFn, chainID, dexID, staticExtra.BufferTokens)
}

func buildPoolState(
	p entity.Pool,
	staticExtra *shared.StaticExtra,
	rpcRes *weighted.RpcResult,
) (entity.Pool, error) {
	extra := weighted.Extra{Extra: &shared.Extra{}}
	extra.EnableHookAdjustedAmounts = rpcRes.HooksConfigData.EnableHookAdjustedAmounts
	extra.ShouldCallComputeDynamicSwapFee = rpcRes.HooksConfigData.ShouldCallComputeDynamicSwapFee
	extra.ShouldCallBeforeSwap = rpcRes.HooksConfigData.ShouldCallBeforeSwap
	extra.ShouldCallAfterSwap = rpcRes.HooksConfigData.ShouldCallAfterSwap
	extra.StaticSwapFeePercentage, _ = uint256.FromBig(rpcRes.StaticSwapFeePercentage)
	extra.AggregateSwapFeePercentage, _ = uint256.FromBig(rpcRes.AggregateSwapFeePercentage)
	extra.BalancesLiveScaled18 = shared.FromBigs(rpcRes.PoolData.BalancesLiveScaled18)
	extra.DecimalScalingFactors = shared.FromBigs(rpcRes.PoolData.DecimalScalingFactors)
	extra.TokenRates = shared.FromBigs(rpcRes.PoolData.TokenRates)
	extra.Buffers = rpcRes.Buffers()
	extra.NormalizedWeights = shared.FromBigs(rpcRes.NormalizedWeights)

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.BlockNumber = rpcRes.BlockNumber
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	if rpcRes.IsPoolDisabled || !shared.IsHookSupported(staticExtra.HookType) {
		p.Reserves = lo.Map(p.Reserves, func(_ string, _ int) string { return "0" })
	} else {
		p.Reserves = lo.Map(rpcRes.PoolData.BalancesRaw, func(v *big.Int, _ int) string { return v.String() })
	}

	return p, nil
}
