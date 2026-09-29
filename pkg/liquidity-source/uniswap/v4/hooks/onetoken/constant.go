package onetoken

import (
	"github.com/ethereum/go-ethereum/common"
)

// MaxFeeBps mirrors the OneToken hook's MAX_FEE_BPS: a hard 20% ceiling on any fee
// the owner can set (base, per-pool override, or snipe tax), so the pool can never
// be priced as a sell-blocking honeypot. feePipsPerBps converts our 1/10_000 bps
// unit to Uniswap v4's 1/1_000_000 pip fee unit.
const (
	MaxFeeBps     = 2_000
	feePipsPerBps = 100
)

// HookAddresses is the shared OneToken hook instance reused across every launch, one
// per chain. All factory generations on a chain point at the same hook, so these
// three cover every OneToken v4 pool. Per-pool economics (launchTime, fee override)
// live in the hook's storage keyed by poolId, not in the hook address. The deployed
// contract is verified on each chain's explorer (its verified name is RampHook).
var HookAddresses = []common.Address{
	common.HexToAddress("0x63e0Ff2e9c38dB24C56A075B69653D99c412C880"), // Base (8453)
	common.HexToAddress("0xD5C0249cC9F32F4696a6357d66B5317870D5C880"), // BNB Smart Chain (56)
	common.HexToAddress("0xFc1088203547710CdD8a45ba9ec80369cc254880"), // Robinhood (4663)
}
