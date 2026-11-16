package btrprop

import "time"

// Monad mainnet records (BTR pool v2, deployment records of 2026-11-13).
//
// The BTR Router is not a swap venue for an aggregator: KyberSwap's executor calls
// Pool.swap_qe directly, so msg.sender is the executor (the router never calls the
// pool). While any leg of a swap is SWAP_GATED, swap_qe requires msg.sender to be in
// the pool's swap gate allowlist (Pool.inSwapGate); the gate checks msg.sender only.
const (
	ChainIDMonad = 143

	// DefaultExecutor is KyberSwap's executor, the same address on every chain (prism-prop and
	// kipseli/prop key their per-taker quotes by it too). Both it and the router
	// (0x6131B5fae19EA4f9D964eAc0408E4408b66337b5) are in the Monad pool's swap gate; only the
	// executor is the pool's msg.sender, so only it is probed. Config.Executor overrides it.
	DefaultExecutor = "0x8f10b468b06c6fd214b65f87778827f7d113f996"

	// CreditPoolCore is BTR's single multi-asset AIMM book (tag `btr-core`), pool v2.
	CreditPoolCore = "0xbbbbbbbd7060b1ced79c381ead40d73fcb9ae0c6"
	// PoolFactory mints official pools (v2 factory). The updater lists the core only once
	// the factory marks it official.
	PoolFactory = "0xbbbbbbb16b9b5fa94ad8cd35e696bea55db9c8b0"

	// DefaultGas covers one swap_qe at Monad's cold-access pricing: the pool
	// call measured 155k-161k gas in Tenderly (USDC<->WMON).
	DefaultGas = 170_000
	// GasPerExtraHop is added per route node past the direct edge (a spoke-to-spoke route crosses
	// the hub). Measured on Monad in Tenderly (swap_qe frame): 8 direct hub-pair swaps averaged
	// 155746 gas, 6 spoke-to-spoke swaps 161239, so one extra hop costs 5493.
	GasPerExtraHop = 5_493
	// MaxAge drops a stale sampled ladder rather than quoting marks the keeper
	// has since moved. The keeper pushes about every 3 s and the tracker refreshes on its poll,
	// so a ladder is replaced within a few pushes: 15 s. Every sampled output is also
	// scaled down by the worst-case drift of the pool's staleness premium over MaxAge plus
	// InclusionAllowance (see guard.refine), so a longer window costs competitiveness, a shorter one only refetches.
	MaxAge = 15 * time.Second
	// InclusionAllowance extends the drift window past MaxAge for the time a swap takes to land.
	InclusionAllowance = 2 * time.Second
	// RPCTimeout bounds one tracker refresh.
	RPCTimeout = 60 * time.Second
)

// Risk-flag bits (BTR Solidity constants, dex-evm PoolConstantsLib).
const (
	HaltBit        uint16 = 1    // HALT_BIT (HALT_MASK is this bit alone)
	KneeBit        uint16 = 2    // KNEE_BIT (the leg's kneeBits word is live)
	SwapEnabledBit uint16 = 16   // SWAP_ENABLED_BIT
	SwapGatedBit   uint16 = 1024 // SWAP_GATED_BIT (swaps restricted to the pool's swap gate)
	HaltMask       uint16 = HaltBit
)
