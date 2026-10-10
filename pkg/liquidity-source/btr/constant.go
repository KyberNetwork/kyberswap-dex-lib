package btr

import "time"

// Monad mainnet records (BTR deployment records, verified 2026-10-08).
//
// The BTR Router holds no swap lane, so an aggregator must NOT route through
// Router.swap: the call reaches Pool.swap_qe with msg.sender = Router, which
// has no lane and reverts NotAuthorized. The executor calls Pool.swap_qe
// directly and needs lane 0x400, or SWAP_GATED must be cleared by BTR governance.
const (
	ChainIDMonad = 143

	// CreditPoolCore is BTR's single multi-asset AIMM book (tag `btr-core`).
	CreditPoolCore = "0xbbbbbbb04f5b762a4cdd1d89e341e2537e3267e4"
	// PoolFactory mints official pools.
	PoolFactory = "0xbbbbbbbbbd5e955e20f323ca1f95f8191fd0e0bf"

	// DefaultGas is a conservative gas figure for one swap_qe at Monad's
	// cold-access pricing.
	DefaultGas = 350_000
	// MaxAge drops a stale sampled ladder rather than quoting marks the keeper
	// has since moved.
	MaxAge = time.Minute
	// RPCTimeout bounds one tracker refresh.
	RPCTimeout = 15 * time.Second
)

// Risk-flag bits (BTR Solidity constants).
const (
	SwapEnabledBit uint16 = 16   // SWAP_ENABLED_BIT
	SwapGatedBit   uint16 = 1024 // SWAP_GATED_BIT (lane-0x400 allowlist)
	HaltMask       uint16 = 5    // HALT_MASK (HALT_BIT | FEED_HALT_BIT)
	HaltBit        uint16 = 1    // HALT_BIT
)
