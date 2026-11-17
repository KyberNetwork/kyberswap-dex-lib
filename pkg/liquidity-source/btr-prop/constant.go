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

	// DefaultGas is the router's per-swap gas for a direct edge. The router's total estimate is
	// 193831 + 1.25*Gas (fixed executor/router overhead plus its 1.25 buffer). Measured on Monad in
	// Tenderly: 14 sampled routes, direct hub-pair txs averaged 499624 total gas, so
	// (499624-193831)/1.25 = 244635 (rounded up to the integer that floors to 499624).
	DefaultGas = 244_635
	// GasPerExtraHop is added per route node past the direct edge (a spoke-to-spoke route crosses
	// the hub). Measured on Monad in Tenderly (swap_qe frame): 8 direct hub-pair swaps averaged
	// 155746 gas, 6 spoke-to-spoke swaps 161239, so one extra hop costs 5493.
	GasPerExtraHop = 5_493
	// MaxAge drops a sampled ladder older than this at quote time rather than quoting marks the
	// keeper has since moved. It is a drop rule only: the haircut is sized from the quote's actual
	// age (see Extra.factor). Kyber's route snapshots of the pool measured 2.1 to 7.3 s old
	// (median 4.0 s, 360 routes, 2026-11-16), so 10 s drops none of them.
	MaxAge = 10 * time.Second
	// InclusionAllowance is the time a routed swap takes to land, added to the quote's age when
	// sizing the haircut. Measured on Monad over the 7 days to 2026-11-16 (5061 KyberSwap router
	// swaps, ClientData build timestamp to landing block): p50 5 s, 75% within 9 s, p90 18 s. The
	// pool's premium resets on every keeper push, so a swap landing later is short of the haircut
	// only when no push happened in between: replayed on 24 h of pushes, under 1.5% of fills on
	// the volatile legs. Mark moves over the delay are the trader's slippage, as on any venue.
	InclusionAllowance = 9 * time.Second
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
