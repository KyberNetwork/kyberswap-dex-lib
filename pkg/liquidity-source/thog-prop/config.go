package thogprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Proxy is ThogAMM's proxy contract -- always integrate the proxy
	// address, never its implementation (the doc's explicit instruction;
	// the implementation is upgradeable via Upgraded/BeaconUpgraded).
	Proxy string `json:"proxy"`
	// PoolId is the single fixed pool this proxy serves (doc's Pool ID).
	// getPoolIds()/getTokens() are still called at discovery time to
	// verify this against live state rather than trusting the constant
	// blindly.
	PoolId string `json:"poolId"`
}
