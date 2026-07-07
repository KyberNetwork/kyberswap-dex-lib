package cl

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = poolfactory.RegisterFactoryCE(DexType, NewPoolFactory)

type PoolFactory struct {
	config              *Config
	ethrpcClient        *ethrpc.Client
	poolCreatedEventIds map[common.Hash]struct{}
}

func NewPoolFactory(config *Config, ethrpcClient *ethrpc.Client) *PoolFactory {
	return &PoolFactory{
		config:       config,
		ethrpcClient: ethrpcClient,
		poolCreatedEventIds: map[common.Hash]struct{}{
			shared.CLPoolManagerABI.Events["Initialize"].ID: {},
		},
	}
}

func (f *PoolFactory) DecodePoolCreated(event ethtypes.Log) (*entity.Pool, error) {
	p, err := shared.CLPoolManagerFilterer.ParseInitialize(event)
	if err != nil {
		return nil, err
	}

	return f.newPool(p, event.BlockNumber)
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	_, ok := f.poolCreatedEventIds[event]
	return ok
}

func (f *PoolFactory) newPool(p *abi.PancakeInfinityPoolManagerInitialize, blockNbr uint64) (*entity.Pool, error) {
	chainId := valueobject.ChainID(f.config.ChainID)

	swapFee, _ := p.Fee.Float64()
	params := p.Parameters[:]
	tickSpacing := GetTickSpacing(params)
	hasSwapPermissions := shared.HasSwapPermissions(params)

	extraBytes, _ := json.Marshal(Extra{
		Extra: &uniswapv3.Extra{
			SqrtPriceX96: p.SqrtPriceX96,
			TickSpacing:  tickSpacing,
			Tick:         p.Tick,
		},
	})
	staticExtra := StaticExtra{
		HasSwapPermissions: hasSwapPermissions,
		IsNative:           [2]bool{valueobject.IsZeroAddress(p.Currency0), valueobject.IsZeroAddress(p.Currency1)},
		Fee:                uint32(p.Fee.Uint64()),
		TickSpacing:        tickSpacing,
		Parameters:         hexutil.Encode(params),
		HooksAddress:       p.Hooks,
		PoolManagerAddress: common.HexToAddress(f.config.CLPoolManagerAddress),
		VaultAddress:       common.HexToAddress(f.config.VaultAddress),
		Permit2Address:     common.HexToAddress(f.config.Permit2Address),
		Multicall3Address:  common.HexToAddress(f.config.Multicall3Address),
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	exchange := ""
	if classifyStableHooks(context.Background(), f.ethrpcClient, f.config.StableHookFactories, []common.Address{p.Hooks})[p.Hooks] {
		exchange = valueobject.ExchangePancakeInfinityCLStable
	} else {
		hook, _ := GetHook(staticExtra.HooksAddress, &HookParam{Cfg: f.config})
		exchange = hook.GetExchange()
	}

	return &entity.Pool{
		Address:   hexutil.Encode(p.Id[:]),
		SwapFee:   swapFee,
		Exchange:  exchange,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: valueobject.ZeroToWrappedLower(p.Currency0.Hex(), chainId), Swappable: true},
			{Address: valueobject.ZeroToWrappedLower(p.Currency1.Hex(), chainId), Swappable: true},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
		BlockNumber: blockNbr,
	}, nil
}

// StableHookClassifier returns the subset of `hooks` registered under any of
// `stableHookFactories`. Wired in via RegisterStableHookClassifier from
// cl/hooks/stable to break the import cycle.
type StableHookClassifier func(
	ctx context.Context,
	rpcClient *ethrpc.Client,
	stableHookFactories []string,
	hooks []common.Address,
) map[common.Address]bool

var stableHookClassifier StableHookClassifier

func RegisterStableHookClassifier(c StableHookClassifier) bool {
	stableHookClassifier = c
	return true
}

// nil-safe wrapper: returns nil (which is a usable empty map for reads) when
// no classifier is registered or there's nothing to classify.
func classifyStableHooks(
	ctx context.Context,
	rpcClient *ethrpc.Client,
	stableHookFactories []string,
	hooks []common.Address,
) map[common.Address]bool {
	if stableHookClassifier == nil || rpcClient == nil ||
		len(stableHookFactories) == 0 || len(hooks) == 0 {
		return nil
	}
	return stableHookClassifier(ctx, rpcClient, stableHookFactories, hooks)
}

func (f *PoolFactory) DecodePoolAddressesFromFactoryLog(_ context.Context, log ethtypes.Log) ([]string, error) {
	if len(log.Topics) == 0 || valueobject.IsZeroAddress(log.Address) {
		return nil, nil
	}

	switch log.Topics[0] {
	case shared.CLPoolManagerABI.Events["Initialize"].ID,
		shared.CLPoolManagerABI.Events["Donate"].ID,
		shared.CLPoolManagerABI.Events["ModifyLiquidity"].ID,
		shared.CLPoolManagerABI.Events["ProtocolFeeUpdated"].ID,
		shared.CLPoolManagerABI.Events["DynamicLPFeeUpdated"].ID,
		shared.CLPoolManagerABI.Events["Swap"].ID: // these events have the pool address in topic1
		if len(log.Topics) < 2 {
			break
		}
		return []string{hexutil.Encode(log.Topics[1][:])}, nil
	}

	return nil, nil
}
