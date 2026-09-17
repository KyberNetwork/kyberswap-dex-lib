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

// Per-chain addresses for the Range Pool deployment.
//
// Range runs on a custom, non-canonical Balancer V3 Vault, so it is not indexed
// by any subgraph; these addresses are hardcoded and pools are discovered on-chain
// via RangePoolFactory.getPools().
var (
	// HookMap is keyed by chain: Range's Hook (unlike the canonical Balancer V3
	// deployment) is redeployed per chain with a different address. Use
	// HookAddress(config.ChainID). The Vault is looked up via shared.Vault
	// (shared.VaultMap["range"] + shared.VaultOverrideMap["range"]) instead, so the
	// tracker and base.PoolSimulator.GetMetaInfo can never disagree on the address.
	HookMap = map[valueobject.ChainID]common.Address{
		valueobject.ChainIDEthereum:  common.HexToAddress("0xf31e1f37E1f9C2C531e6bC3ad89fFc9206cE85d9"),
		valueobject.ChainIDRobinhood: common.HexToAddress("0xA2577544B1172d397209B8378516c6caFc023252"),
	}
)

// VaultAddress returns the Vault deployed on the given chain for Range Pools.
func VaultAddress(chainID valueobject.ChainID) common.Address {
	return shared.Vault(chainID, DexType)
}

// HookAddress returns the RangePoolHook deployed on the given chain.
func HookAddress(chainID valueobject.ChainID) common.Address {
	return HookMap[chainID]
}

// Config is the per-instance runtime configuration for the Range Pool connector.
type Config struct {
	DexID          string              `json:"dexID"`
	ChainID        valueobject.ChainID `json:"chainID"`
	FactoryAddress string              `json:"factoryAddress"`
	NewPoolLimit   int                 `json:"newPoolLimit"`
}
