package uniswapv4

import (
	"context"
	"errors"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type BeforeSwapParams struct {
	ExactIn         bool
	ZeroForOne      bool
	AmountSpecified *big.Int
}

type BeforeSwapResult struct {
	DeltaSpecific   *big.Int
	DeltaUnSpecific *big.Int
	SwapFee         FeeAmount
	Gas             int64
}

func (b *BeforeSwapResult) Validate() error {
	if b.DeltaSpecific == nil {
		return errors.New("delta specified is nil")
	}

	if b.DeltaUnSpecific == nil {
		return errors.New("delta unspecified is nil")
	}

	return nil
}

type AfterSwapResult struct {
	HookFee *big.Int
	Gas     int64
}

func (b *AfterSwapResult) Validate() error {
	if b.HookFee == nil {
		return errors.New("hook fee is nil")
	}

	return nil
}

type AfterSwapParams struct {
	*BeforeSwapParams
	AmountIn  *big.Int
	AmountOut *big.Int
}

type FeeAmount = constants.FeeAmount

const FeeMax = constants.FeeMax

// HookOption represents different hook operation types
type HookOption int

const (
	AfterRemoveLiquidityReturnsDelta HookOption = iota
	AfterAddLiquidityReturnsDelta
	AfterSwapReturnsDelta
	BeforeSwapReturnsDelta
	AfterDonate
	BeforeDonate
	AfterSwap
	BeforeSwap
	AfterRemoveLiquidity
	BeforeRemoveLiquidity
	AfterAddLiquidity
	BeforeAddLiquidity
	AfterInitialize
	BeforeInitialize
)

// hasPermission checks if the address has permission for a specific hook option
func hasPermission(address common.Address, hookOption HookOption) bool {
	// Convert last 2 bytes of the address to int64
	addressInt := uint64(address[common.AddressLength-1]) | uint64(address[common.AddressLength-2])<<8
	// Check if the bit at hookOption index is set
	return (addressInt & (1 << hookOption)) != 0
}

// HasSwapPermissions checks if the address has swap-related permissions
// adapted from https://github.com/Uniswap/sdks/blob/62d162a3bb2f4b9b800bd617ab6d8ee913d447a1/sdks/v4-sdk/src/utils/hook.ts#L85
func HasSwapPermissions(address common.Address) bool {
	// This implicitly encapsulates swap delta permissions
	return hasPermission(address, BeforeSwap) || hasPermission(address, AfterSwap)
}

type Hook interface {
	GetExchange() string
	GetReserves(context.Context, *HookParam) (entity.PoolReserves, error)
	Track(context.Context, *HookParam) (string, error)
	BeforeSwap(swapHookParams *BeforeSwapParams) (*BeforeSwapResult, error)
	AfterSwap(swapHookParams *AfterSwapParams) (*AfterSwapResult, error)
	CloneState() Hook
}

type HookParam struct {
	Cfg         *Config
	RpcClient   *ethrpc.Client
	Pool        *entity.Pool
	HookExtra   string
	HookAddress common.Address
}

type HookFactory func(param *HookParam) Hook

var HookFactories = map[common.Address]HookFactory{}

func RegisterHooks(hook Hook, addresses ...common.Address) bool {
	return RegisterHooksFactory(func(*HookParam) Hook {
		return hook
	}, addresses...)
}

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

type BaseHook struct{ Exchange valueobject.Exchange }

func (h *BaseHook) CloneState() Hook {
	return h
}

func (h *BaseHook) GetExchange() string {
	if h != nil {
		return string(h.Exchange)
	}
	return DexType
}

func (h *BaseHook) GetReserves(context.Context, *HookParam) (entity.PoolReserves, error) {
	return nil, nil
}

func (h *BaseHook) Track(context.Context, *HookParam) (string, error) {
	return "", nil
}

func (h *BaseHook) BeforeSwap(swapHookParams *BeforeSwapParams) (*BeforeSwapResult, error) {
	return &BeforeSwapResult{
		SwapFee:         0,
		DeltaSpecific:   new(big.Int),
		DeltaUnSpecific: new(big.Int),
		Gas:             0,
	}, nil
}

func (h *BaseHook) AfterSwap(_ *AfterSwapParams) (*AfterSwapResult, error) {
	return &AfterSwapResult{
		HookFee: new(big.Int),
		Gas:     0,
	}, nil
}
