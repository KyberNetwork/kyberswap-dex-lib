package navjit

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// HookAddresses lists every NavJitHook deployment. Permissions are mined into the low 14 bits:
// 0x…eAe0 & 0x3FFF = 0x2AE0 = beforeInitialize | beforeAddLiquidity | beforeRemoveLiquidity |
// beforeSwap | afterSwap | beforeDonate, and NO return-delta bit — the hook never edits a swap's
// deltas, it fills the swap with liquidity it places for that swap alone. Append V2 here once it
// is deployed.
var HookAddresses = []common.Address{
	// Robinhood Chain (4663), NavJitHookV15. PoolManager 0x8366a39cc670b4001a1121b8f6a443a643e40951.
	common.HexToAddress("0x99A670d2103E1e4Ebd53a483f725DD651a08eAe0"),
}

// QuoterByChain is the stock Uniswap V4Quoter per chain. The ladder is read from it, so the
// plugin quotes exactly what the deployed hook would fill at the tracked block.
var QuoterByChain = map[valueobject.ChainID]common.Address{
	valueobject.ChainIDRobinhood: common.HexToAddress("0x8Dc178eFB8111BB0973Dd9d722ebeFF267c98F94"),
}

// maxAgeSec bounds the ladder's age under StaleCheck. pool-service re-tracks every 10s; constituent
// pools and NAV move the fill curve, not this pool's own events, so an untracked ladder drifts.
const maxAgeSec = 60

// Rung sizes in whole USD. Buy rungs are USDG in; sell rungs are LOT in, converted at the
// tracked NAV. The 30k/40k rungs split the widest segment: without them the chord under-quoted
// the live venues by up to ~9 bps between 20k and 50k (2026-10-02).
var rungsUsd = []uint64{1, 10, 100, 500, 1_000, 2_500, 5_000, 10_000, 20_000, 30_000, 40_000, 50_000}

// buyHaircutPpm is taken off every buy (USDG in) output, in parts per million. The buy fill
// curve is not concave: the mint rounds to whole constituent units, which puts ~±1 bp ripples on
// it, and a chord between two rungs can sit above a ripple. Dense live sweeps against the
// V4Quoter (2026-10-02, three venues) put the worst chord over-quote at +1.0 bp; the haircut
// covers that with margin. Sells showed no over-quote (worst +0.000 bp), so they are not cut.
const buyHaircutPpm = 150

const ppm = 1_000_000

// floatRungsBps: extra sell rungs as a fraction of the PoolManager's LOT balance. That balance is
// only an upper bound on a sell: inside the swap, _restoreStanding re-centres the standing position
// on NAV and takes whatever LOT no longer fits to the owner BEFORE _unwind checks the float, so the
// effective cap is lower by a swap-dependent amount (measured 2026-10-01 on LOT-two: balance 49.265,
// SellExceedsFloat(49.077, 49.1)). The ladder finds the real edge; these rungs keep it close.
var floatRungsBps = []int64{9_000, 9_700, 9_950}
