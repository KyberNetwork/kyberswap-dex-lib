package premium

import "github.com/ethereum/go-ethereum/common"

// gasBeforeSwap / gasAfterSwap cover PremiumLaunchHook's beforeSwap/afterSwap callbacks.
// Unlike Fables (fee-override only, no deltas), this hook takes real BeforeSwapDelta /
// afterSwap fee on one side or the other of every trade (see hook.go), so both callbacks
// carry a nonzero gas estimate.
const (
	gasBeforeSwap = 90000
	gasAfterSwap  = 70000
)

// bps and feeBps mirror PremiumLaunchHook.FEE_BPS (100 = 1%) - the same 1% desk-token fee
// MemeCurve charges pre-graduation, charged again by the hook after graduation.
const (
	bps    = 10_000
	feeBps = 100
)

// HookAddresses lists Premium's graduated-pool hook. Unlike Fables (one immutable hook per
// pool), PremiumLaunchHook is a single immutable contract shared across every graduated
// Premium pool - pool-specific state (memeIsCurrency0, paused, creator, platformTreasury)
// lives in its poolConfig(poolId) mapping, not in per-pool bytecode.
var HookAddresses = []common.Address{
	common.HexToAddress("0x1af6269A7E53422406FF2410b8ED5590F610Aacc"),
}
