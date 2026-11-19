package thogprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Proxy is the pool contract. Calls and token approvals always target it;
	// the implementation behind it can change.
	Proxy string `json:"proxy"`
	// PoolId is the one pool the contract serves. Discovery checks it against
	// getPoolIds() before listing the pool.
	PoolId string `json:"poolId"`
	// Zero selects the defaults: 300 ms/block, 15 s snapshot age, 2 s/attempt.
	BlockTimeMs      int64 `json:"blockTimeMs"`
	MaxSnapshotAgeMs int64 `json:"maxSnapshotAgeMs"`
	RequestTimeoutMs int64 `json:"requestTimeoutMs"`
	// Read through Multicall3 only, for providers that refuse a creation eth_call.
	DisableLens bool `json:"disableLens"`
}
