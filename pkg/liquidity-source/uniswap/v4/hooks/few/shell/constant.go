package fewshell

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// HookAddresses lists every FewV2ShellHook deployment. Append new chains or redeployments here.
//
// The hook is not a proxy, and its v4 permissions are mined into the low 14 bits of its address: 0x2088 is
// beforeInitialize | beforeSwap | beforeSwapReturnDelta, so the swap is fully replaced by what BeforeSwap returns.
var HookAddresses = []common.Address{
	// Ethereum. PoolManager 0x000000000004444c5dc75cb358380d2e3de08a90, useNativeCurrencyForWeth = false (a WETH
	// pool keeps WETH as its currency, never native ETH).
	common.HexToAddress("0xb39aa0fea3d8940c1106059460934ccfb0212088"),
}

const (
	// feeNumerator / feeDenominator are FewV2Math's 997/1000: the 0.30% fee of a FewV2 pair.
	feeNumerator   = 997
	feeDenominator = 1000

	// ppm is the denominator of the protocol fee rate (ProtocolFeeLibrary.PIPS_DENOMINATOR).
	ppm = 1_000_000

	// protocolFeeMultiplier is FewV2ShellHookBase.protocolFeeMultiplier: the hook charges
	// slot0's directional protocol fee times 25.
	protocolFeeMultiplier = 25

	// hookGas is the gas of the PoolManager's beforeSwap call into the hook (the wrap, the pair swap, the unwrap,
	// the settle, the fee take and the balance checks), as the median of five mainnet swaps through the hook:
	// 212955, 292181, 300528, 311319 and 324425 (Tenderly traces of
	// 0xbc1a97..29a6, 0x07828c..eac, 0x5a79195f..c589, 0x36a1f0c6..7248, 0x43a9c234..e042, the call from the PoolManager).
	// The v4 simulator adds its own PoolManager base gas on top.
	hookGas int64 = 300_000

	// trackInterval is how long a shell pool's tracked state may stand with no event before it is read again. The
	// pair's reserves, the PoolManager's inventory and the wrappers' backing change with trades that emit no event
	// for the v4 pool (a swap on the FewV2 pair, a transfer, a wrap), so no PoolManager log tells the tracker. Every
	// quote reads that state, and a stale read quotes a price the pair no longer gives. 30 s is about 2.5 mainnet
	// blocks: close enough to the pair that a quote is rarely off, and cheap, since a shell pool costs one
	// multicall per interval while the pool-service schedule allows 20 updates per second for all v4 pools and
	// there are few shell pools. The hook has no max age and refuses no quote on a stale state, so this is a
	// freshness target, not a safety limit.
	trackInterval = 30 * time.Second
)
