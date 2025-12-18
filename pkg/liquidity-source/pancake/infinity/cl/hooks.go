package cl

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
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
	Track(context.Context, *HookParam) ([]byte, error)
	BeforeSwap(swapHookParams *BeforeSwapParams) (*BeforeSwapResult, error)
	AfterSwap(swapHookParams *AfterSwapParams) (*AfterSwapResult, error)
	CloneState() Hook
	UpdateBalance(swapInfo any)
	// ModifyTicks some hook need ModifyLiquidity before swap, we can use this method in the NewPoolSimulator for simplicity
	// instead of calling BeforeSwap in CalcAmountOut.
	ModifyTicks(extraTickUint256 *uniswapv3.ExtraTickU256) error
}

type HookFactory func(param *HookParam) Hook

var HookFactories = map[common.Address]HookFactory{}

func RegisterHooksFactory(factory HookFactory, addresses ...common.Address) bool {
	for _, address := range addresses {
		HookFactories[address] = factory
	}
	return true
}

func GetHook(hookAddress common.Address, param *HookParam) (hook Hook, ok bool) {
	hookFactory, ok := HookFactories[hookAddress]
	if hookFactory == nil {
		hook = (*BaseHook)(nil)
	} else {
		if param == nil {
			param = &HookParam{}
		}
		param.HookAddress = hookAddress
		hook = hookFactory(param)
	}
	return hook, ok
}

type BaseHook struct {
	Exchange string
}

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
	return &BaseHook{
		Exchange: thisExchange,
	}
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

func (h *BaseHook) ModifyTicks(extraTickUint256 *uniswapv3.ExtraTickU256) error {
	return nil
}
