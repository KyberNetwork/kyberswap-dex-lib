package weighted

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	// DexType is the pool-type identifier for Range Pools, a concentrated-liquidity,
	// Balancer-V3-shaped weighted DEX running on a custom, non-canonical Vault.
	DexType = "range-v3-weighted"
)

// Mainnet (chainId 1) addresses for the Range Pool deployment.
//
// Range runs on a custom, non-canonical Balancer V3 Vault, so it is not indexed
// by any subgraph; these addresses are hardcoded and pools are discovered on-chain
// via RangePoolFactory.getPools().
var (
	// VaultAddress and RouterAddress are sourced from shared.VaultMap/BatchRouterMap
	// (keyed "range", like the existing "coinhane" non-canonical deployment) so the
	// tracker and base.PoolSimulator.GetMetaInfo can never disagree on the address.
	VaultAddress  = shared.VaultMap["range"]
	RouterAddress = shared.BatchRouterMap["range"][valueobject.ChainIDEthereum]

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
