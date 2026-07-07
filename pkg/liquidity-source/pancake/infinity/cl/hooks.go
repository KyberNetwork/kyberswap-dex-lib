package cl

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	HookParam struct {
		Cfg         *Config
		RpcClient   *ethrpc.Client
		Pool        *entity.Pool
		HookExtra   []byte
		HookAddress common.Address
		BlockNumber *big.Int
	}
	BeforeSwapParams = uniswapv4.BeforeSwapParams
	BeforeSwapResult = uniswapv4.BeforeSwapResult
	AfterSwapParams  = uniswapv4.AfterSwapParams
	AfterSwapResult  = uniswapv4.AfterSwapResult
)

const FeeMax = uniswapv4.FeeMax

var (
	ValidateBeforeSwapResult = uniswapv4.ValidateBeforeSwapResult
	ValidateAfterSwapResult  = uniswapv4.ValidateAfterSwapResult
)

type Hook interface {
	GetExchange() string
	GetDynamicFee(ctx context.Context, params *HookParam, lpFee uint32) uint32
	GetReserves(context.Context, *HookParam) (entity.PoolReserves, error)
	Track(context.Context, *HookParam) ([]byte, error)
	BeforeSwap(swapHookParams *BeforeSwapParams) (*BeforeSwapResult, error)
	AfterSwap(swapHookParams *AfterSwapParams) (*AfterSwapResult, error)
	CloneState() Hook
	UpdateBalance(swapInfo any)
	AllowEmptyTicks() bool
}

type HookFactory func(param *HookParam) Hook

var HookFactories = map[common.Address]HookFactory{}

func RegisterHooksFactory(factory HookFactory, addresses ...common.Address) bool {
	for _, address := range addresses {
		HookFactories[address] = factory
	}
	return true
}

var stableHookFactory HookFactory

func RegisterStableHookFactory(f HookFactory) bool {
	stableHookFactory = f
	return true
}

// GetHook resolves a Hook in priority order:
//  1. by hook contract address (HookFactories) — static init-time config
//  2. stable-hook fast-path when entity.Pool.Exchange (or Cfg.DexID) is stable
//  3. RPC classification — only when param.RpcClient is set (in tracker)
//  4. fallback to BaseHook
func GetHook(hookAddress common.Address, param *HookParam) (Hook, bool) {
	if param == nil {
		param = &HookParam{}
	}
	param.HookAddress = hookAddress

	if f, ok := HookFactories[hookAddress]; ok && f != nil {
		return f(param), true
	}

	exchange := ""
	if param.Pool != nil {
		exchange = param.Pool.Exchange
	} else if param.Cfg != nil {
		exchange = param.Cfg.DexID
	}

	if stableHookFactory != nil &&
		(exchange == valueobject.ExchangePancakeInfinityCLStable || isStableHook(param, hookAddress)) {
		return stableHookFactory(param), true
	}

	return NewBaseHook(valueobject.ExchangePancakeInfinityCL, param), false
}

func isStableHook(param *HookParam, hookAddress common.Address) bool {
	if valueobject.IsZeroAddress(hookAddress) {
		return false
	}

	var stableHookFactories []string
	if cfg := param.Cfg; cfg != nil {
		stableHookFactories = cfg.StableHookFactories
	}
	return classifyStableHooks(
		context.Background(),
		param.RpcClient,
		stableHookFactories,
		[]common.Address{hookAddress},
	)[hookAddress]
}

type BaseHook struct{ Exchange string }

func NewBaseHook(exchange valueobject.Exchange, param *HookParam) *BaseHook {
	suffix := strings.TrimPrefix(string(exchange), DexType) // -dynamic
	thisExchange := string(exchange)                        // pancake-infinity-cl-dynamic
	if pool := param.Pool; pool != nil {
		thisExchange = pool.Exchange // pancake-infinity-cl / pancake-infinity-cl-dynamic / omni-cl / omni-cl-dynamic
	} else if cfg := param.Cfg; cfg != nil && cfg.DexID != "" {
		thisExchange = cfg.DexID // pancake-infinity-cl / omni-cl
	}
	if !strings.HasSuffix(thisExchange, suffix) {
		thisExchange += suffix // pancake-infinity-cl-dynamic / omni-cl-dynamic
	}
	return &BaseHook{Exchange: thisExchange}
}

func BaseFactory(exchange valueobject.Exchange) HookFactory {
	return func(param *HookParam) Hook {
		return NewBaseHook(exchange, param)
	}
}

func (h *BaseHook) GetExchange() string {
	if h != nil {
		return h.Exchange
	}
	return valueobject.ExchangePancakeInfinityCL
}

func (h *BaseHook) GetDynamicFee(_ context.Context, _ *HookParam, _ uint32) uint32 {
	return 0
}

func (h *BaseHook) GetReserves(context.Context, *HookParam) (entity.PoolReserves, error) {
	return nil, nil
}

func (h *BaseHook) Track(context.Context, *HookParam) ([]byte, error) {
	return nil, nil
}

func (h *BaseHook) BeforeSwap(_ *BeforeSwapParams) (*BeforeSwapResult, error) {
	return &BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}

func (h *BaseHook) AfterSwap(_ *AfterSwapParams) (*AfterSwapResult, error) {
	return &AfterSwapResult{
		HookFee: bignumber.ZeroBI,
	}, nil
}

func (h *BaseHook) CloneState() Hook {
	return h
}

func (h *BaseHook) UpdateBalance(_ any) {}

func (h *BaseHook) AllowEmptyTicks() bool { return false }
