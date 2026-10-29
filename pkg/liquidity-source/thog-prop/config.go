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
	// Zero selects the defaults: 300 ms/block, 15 s snapshot age, 2 s/attempt.
	BlockTimeMs      int64 `json:"blockTimeMs"`
	MaxSnapshotAgeMs int64 `json:"maxSnapshotAgeMs"`
	RequestTimeoutMs int64 `json:"requestTimeoutMs"`
	// Use the page-first number-pinned reader for providers without creation eth_call.
	DisableLens bool `json:"disableLens"`
}
