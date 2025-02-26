package uniswapv4

import (
	"github.com/ethereum/go-ethereum/common"
)

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
