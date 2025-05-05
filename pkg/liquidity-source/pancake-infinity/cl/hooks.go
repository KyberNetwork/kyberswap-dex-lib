package cl

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// HookOption represents different hook operation types
type HookOption int

const (
	BeforeInitialize HookOption = iota
	AfterInitialize
	BeforeAddLiquidity
	AfterAddLiquidity
	BeforeRemoveLiquidity
	AfterRemoveLiquidity
	BeforeSwap
	AfterSwap
	BeforeDonate
	AfterDonate
	BeforeSwapReturnDelta
	AfterSwapReturnDelta
	AfterAddLiquidityReturnDelta
	AfterRemoveLiquidityReturnDelta
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
	RFQ(context.Context, pool.RFQParams, *PoolMetaInfo, *pool.RFQResult) (any, error)
}

var Hooks = map[common.Address]Hook{}

func RegisterHooks(hook Hook, addresses ...common.Address) bool {
	for _, address := range addresses {
		Hooks[address] = hook
	}
	return true
}

func GetHook(hookAddress common.Address) Hook {
	if hook, ok := Hooks[hookAddress]; ok {
		return hook
	}

	return &BaseHook{
		Exchange: valueobject.ExchangePancakeInfinityCL,
	}
}

type BaseHook struct{ Exchange valueobject.Exchange }

func (h *BaseHook) GetExchange() string {
	if h != nil {
		return string(h.Exchange)
	}
	return DexType
}

func (*BaseHook) RFQ(context.Context, pool.RFQParams, *PoolMetaInfo, *pool.RFQResult) (any, error) {
	return nil, nil
}
