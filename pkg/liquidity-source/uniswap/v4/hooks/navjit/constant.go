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

// L2BlockByChain marks Arbitrum-stack chains, where block.number inside an eth_call (and so the
// block Multicall3 reports, which the v4 tracker passes on as HookParam.BlockNumber) is the L1
// block, not the chain's own. On 4663 at 2026-10-01: Multicall3.getBlockNumber() = 26,097,221 vs
// eth_blockNumber = arbBlockNumber() = 77,345,0xx. Pinning to the former reads a months-old
// state (or fails on a non-archive node), so on these chains Track pins to ArbSys.arbBlockNumber().
var L2BlockByChain = map[valueobject.ChainID]common.Address{
	valueobject.ChainIDRobinhood: common.HexToAddress("0x0000000000000000000000000000000000000064"),
}

// Rung sizes in whole USD. Buy rungs are USDG in; sell rungs are LOT in, converted at the
// tracked NAV. Dense enough that the chord error stays within a few bps at every size (the fill
// curve bends only with the constituent pools' slippage).
var rungsUsd = []uint64{1, 10, 100, 500, 1_000, 2_500, 5_000, 10_000, 20_000, 50_000}

// floatRungsBps: extra sell rungs as a fraction of the PoolManager's LOT balance. That balance is
// only an upper bound on a sell: inside the swap, _restoreStanding re-centres the standing position
// on NAV and takes whatever LOT no longer fits to the owner BEFORE _unwind checks the float, so the
// effective cap is lower by a swap-dependent amount (measured 2026-10-01 on LOT-two: balance 49.265,
// SellExceedsFloat(49.077, 49.1)). The ladder finds the real edge; these rungs keep it close.
var floatRungsBps = []int64{9_000, 9_700, 9_950}

const bps = 10_000
