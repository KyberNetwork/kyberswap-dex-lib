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

// HookAddresses lists Premium's graduated-pool hooks. A hook is shared across every pool
// that graduated under the GraduationManager which deployed it - pool-specific state
// (memeIsCurrency0, paused, creator, platformTreasury) lives in poolConfig(poolId), not in
// per-pool bytecode. Both entries are needed: MemeFactory has rotated its graduation
// manager, and pools graduated under the retired one keep pointing at the older hook.
//
// Both charge the same 1% desk-token fee split 70/30 (FEE_BPS/CREATOR_SHARE_BPS read equal
// on-chain), so the fee math below covers both; they differ only in feeEscrow, which is a
// payout destination and not part of pricing.
var HookAddresses = []common.Address{
	// GraduationManager 0xee33bff08de96709dd6c877ee192688489e16596 (active)
	common.HexToAddress("0xfa225FE7b2404f8A361A15fa88AD515032726aCC"),
	// GraduationManager 0xe768b13282361a3571e0cd7bdb5c548183c40f46 (retired; $PRM's pool)
	common.HexToAddress("0x1af6269A7E53422406FF2410b8ED5590F610Aacc"),
}
