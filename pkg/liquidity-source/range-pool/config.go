package rangepool

import "github.com/ethereum/go-ethereum/common"

const (
	// DexType is the pool-type identifier for Range Pools, a concentrated-liquidity,
	// Balancer-V3-shaped DEX running on a custom, non-canonical Vault.
	DexType = "range-pool"
)

// Mainnet (chainId 1) addresses for the Range Pool deployment.
//
// Range runs on a custom, non-canonical Balancer V3 Vault, so it is not indexed
// by any subgraph; these addresses are hardcoded and pools are discovered on-chain
// via RangePoolFactory.getPools().
var (
	// VaultAddress is the custom, non-canonical Balancer V3 Vault.
	VaultAddress = common.HexToAddress("0x955244EDC797A1C1b04134b600f819aC23C76081")

	// RouterAddress is the standard Router: it is the ONLY contract exposing
	// swapSingleTokenExactIn/Out and querySwapSingleTokenExactIn/Out. It is the
	// target/approval address for swaps and the querySwap parity oracle.
	RouterAddress = common.HexToAddress("0x8726019313CD59D2e33dBE643aaC94A59D5518df")

	// FactoryAddress is the RangePoolFactory (pool discovery via getPools()).
	FactoryAddress = common.HexToAddress("0x5D6D1dC0D045a8DE284C7Ab5FE83aCd7bdc5d4E0")

	// HookAddress is the RangePoolHook.
	HookAddress = common.HexToAddress("0xf31e1f37E1f9C2C531e6bC3ad89fFc9206cE85d9")

	// SingleTokenRouterAddress adds/removes liquidity only: it exposes neither
	// swapSingleToken... nor querySwap... and MUST NOT be used for swaps.
	SingleTokenRouterAddress = common.HexToAddress("0x79C112F05Ce3C297De4105B5507c90169B3686F8")

	// Permit2Address is the canonical Permit2 contract (swap approval path).
	Permit2Address = common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3")

	// WETHAddress is mainnet WETH.
	WETHAddress = common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
)

// Config is the per-instance runtime configuration for the Range Pool connector.
type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
