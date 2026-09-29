package b20

import (
	"github.com/ethereum/go-ethereum/common"
)

// MaxTotalFeeBps mirrors LaunchHook.sol's MAX_TOTAL_FEE_BPS -- a hard ceiling on
// base + anti-snipe surcharge, strictly below 100%.
const MaxTotalFeeBps = 9_900

const bps = 10_000

// HookAddresses is the single shared LaunchHook instance reused by every B20 launch
// (per-pool economics live in poolConfig, not in the hook's own address).
var HookAddresses = []common.Address{
	common.HexToAddress("0x985c14baa2A18316ffDA0AefB3a632faDFCA2acc"),
}
