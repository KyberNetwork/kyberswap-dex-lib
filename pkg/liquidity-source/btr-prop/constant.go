package btrprop

import "time"

// Monad mainnet records (BTR pool v2, deployment records of 2026-11-13).
//
// The BTR Router is not a swap venue for an aggregator: the executor calls
// Pool.swap_qe directly, so msg.sender is the executor. While any leg of a
// swap is SWAP_GATED, swap_qe requires msg.sender to be in the pool's swap
// gate allowlist (Pool.inSwapGate), so the executor must be listed there.
const (
	ChainIDMonad = 143

	// CreditPoolCore is BTR's single multi-asset AIMM book (tag `btr-core`), pool v2.
	CreditPoolCore = "0xbbbbbbbd7060b1ced79c381ead40d73fcb9ae0c6"
	// PoolFactory mints official pools (v2 factory). The updater lists the core only once
	// the factory marks it official.
	PoolFactory = "0xbbbbbbb16b9b5fa94ad8cd35e696bea55db9c8b0"

	// DefaultGas covers one swap_qe at Monad's cold-access pricing: the pool
	// call measured 155k-161k gas in Tenderly (USDC<->WMON).
	DefaultGas = 170_000
	// MaxAge drops a stale sampled ladder rather than quoting marks the keeper
	// has since moved. Marks drift within seconds, so keep it well below 1 min.
	MaxAge = 15 * time.Second
	// RPCTimeout bounds one tracker refresh.
	RPCTimeout = 15 * time.Second
)

// Risk-flag bits (BTR Solidity constants, dex-evm PoolConstantsLib).
const (
	HaltBit        uint16 = 1    // HALT_BIT (HALT_MASK is this bit alone)
	KneeBit        uint16 = 2    // KNEE_BIT (the leg's kneeBits word is live)
	SwapEnabledBit uint16 = 16   // SWAP_ENABLED_BIT
	SwapGatedBit   uint16 = 1024 // SWAP_GATED_BIT (swaps restricted to the pool's swap gate)
	HaltMask       uint16 = HaltBit
)
